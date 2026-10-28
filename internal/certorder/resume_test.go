package certorder

import (
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/pem"
	"log/slog"
	"testing"

	"github.com/PdYrust/karvan/internal/bridge"
	"github.com/PdYrust/karvan/internal/secret"
)

// resumeCase is one order as a crash left it.
type resumeCase struct {
	name, method, state, caStatus string
	gone, sealedKey               bool
	want                          string
	code                          string
}

// seed stores the order of c for a new plain domain, with its CA order,
// and returns the domain id.
func (e *orderEnv) seed(c resumeCase) int64 {
	e.t.Helper()
	d, err := e.br.AddDomain(e.t.Context(), c.name, bridge.CertChoice{Kind: bridge.ChoiceNone, PlainOK: true})
	if err != nil {
		e.t.Fatal(err)
	}
	var url, authz, chal, token string
	var ca *fakeOrder
	if c.caStatus != "" {
		ca = e.ca.add(c.name, c.caStatus)
		url, authz, chal, token = e.ca.url("/orders/"+ca.id), e.ca.url("/authz/"+ca.id), e.ca.url("/chal/"+ca.id+"-"+c.method), ca.token
	}
	if c.gone {
		url = e.ca.url("/orders/999")
	}
	res, err := e.db.Exec("INSERT INTO acme_orders (domain_id, account_id, method, state, order_url, authz_url, challenge_url, token, "+
		"created_at, updated_at) VALUES (?, 1, ?, ?, ?, ?, ?, ?, 1, 1)", d.ID, c.method, c.state, url, authz, chal, token)
	if err != nil {
		e.t.Fatal(err)
	}
	id, _ := res.LastInsertId()
	if c.sealedKey {
		e.sealOrderKey(id, ca)
	}
	return d.ID
}

// sealOrderKey seals a key in order id, as issuing does before finalize,
// and lets the CA issue for it, as if finalize had gone through.
func (e *orderEnv) sealOrderKey(id int64, ca *fakeOrder) {
	e.t.Helper()
	sealed, err := e.svc.sealNewKey("acme_orders", id)
	if err != nil {
		e.t.Fatal(err)
	}
	if _, err := e.db.Exec("UPDATE acme_orders SET key_sealed = ? WHERE id = ?", sealed, id); err != nil {
		e.t.Fatal(err)
	}
	raw, err := e.key.Open(sealed, secret.AAD("acme_orders", "key_sealed", id))
	if err != nil {
		e.t.Fatal(err)
	}
	b, _ := pem.Decode(raw)
	k, _ := x509.ParsePKCS8PrivateKey(b.Bytes)
	e.ca.issue(ca, []string{ca.name}, &k.(*ecdsa.PrivateKey).PublicKey)
}

// D69: at start, each running order goes on from where the CA has it.
func TestResumeInEachState(t *testing.T) {
	e := newOrderEnv(t)
	e.dnsDone(e.addACME("setup.example.com", DNS01)) // account 1, registered
	cases := []resumeCase{
		{name: "starting.example.com", method: DNS01, state: "starting", want: "waiting_dns"},
		{name: "http.example.com", method: HTTP01, state: "validating", caStatus: "pending", want: "done"},
		{name: "dns.example.com", method: DNS01, state: "validating", caStatus: "pending", want: "waiting_dns"},
		{name: "waiting.example.com", method: DNS01, state: "waiting_dns", caStatus: "pending", want: "waiting_dns"},
		{name: "ready.example.com", method: HTTP01, state: "issuing", caStatus: "ready", want: "done"},
		{name: "valid.example.com", method: HTTP01, state: "issuing", caStatus: "valid", sealedKey: true, want: "done"},
		{name: "gone.example.com", method: HTTP01, state: "validating", gone: true, want: "failed", code: CodeExpired},
		{name: "invalid.example.com", method: DNS01, state: "issuing", caStatus: "invalid", want: "failed", code: CodeExpired},
	}
	ids := make([]int64, len(cases))
	for i, c := range cases {
		ids[i] = e.seed(c)
	}
	s := e.newService(slog.New(slog.DiscardHandler))
	t.Cleanup(func() { s.stop(); s.wg.Wait() })
	s.resume(t.Context())
	for i, c := range cases {
		o := e.waitState(ids[i], c.want)
		if c.code != "" && (o.Error == nil || o.Error.Code != c.code) {
			t.Errorf("%s: %+v", c.name, o.Error)
		}
		d := e.domain(ids[i])
		if (c.want == "done") != (d.Certificate != nil && d.Certificate.Origin == bridge.OriginACME) {
			t.Errorf("%s: certificate %+v", c.name, d.Certificate)
		}
	}
	if n := e.ca.count("finalize"); n != 3 { // setup, http and ready; valid was finalized before the crash
		t.Errorf("finalize asked %d times", n)
	}
}
