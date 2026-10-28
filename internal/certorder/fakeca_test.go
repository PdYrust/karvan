package certorder

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

// problem is an ACME problem answer.
type problem struct {
	status     int
	typ        string // without the urn:ietf:params:acme:error: prefix
	detail     string
	retryAfter string
}

// fakeOrder is one order the fake CA holds.
type fakeOrder struct {
	id, name, token, chalType string
	status, authz, chal       string // order, authorization and challenge status
	chalErr                   *problem
	cert                      []byte // the issued chain, PEM
	expires                   time.Time
}

// fakeCA is an ACME server on httptest (D72): directory, nonce, account,
// order, authorization, challenge, finalize and certificate, with
// problem answers. Validation is the test's validate function.
type fakeCA struct {
	t        *testing.T
	srv      *httptest.Server
	mu       sync.Mutex
	nonce    int
	thumb    string
	orders   map[string]*fakeOrder
	seq      int
	fail     map[string]problem // by endpoint: account, order, orders, authz, chal, finalize, cert
	validate func(o *fakeOrder, keyAuth string) *problem
	now      func() time.Time
	life     time.Duration
	reuse    bool // authorizations are already valid
	calls    map[string]int
	gate     chan struct{} // when set, newOrder waits for it
	inFlight int
	maxIn    int
	caKey    *ecdsa.PrivateKey
	caCert   *x509.Certificate
	caPEM    []byte
}

func newFakeCA(t *testing.T, now func() time.Time) *fakeCA {
	t.Helper()
	k, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	tpl := &x509.Certificate{SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "fake CA"}, IsCA: true,
		BasicConstraintsValid: true, KeyUsage: x509.KeyUsageCertSign, NotBefore: now().Add(-time.Hour), NotAfter: now().Add(10 * 365 * 24 * time.Hour)}
	der, err := x509.CreateCertificate(rand.Reader, tpl, tpl, &k.PublicKey, k)
	if err != nil {
		t.Fatal(err)
	}
	ca, _ := x509.ParseCertificate(der)
	f := &fakeCA{t: t, orders: map[string]*fakeOrder{}, fail: map[string]problem{}, now: now, life: 90 * 24 * time.Hour,
		calls: map[string]int{}, caKey: k, caCert: ca, caPEM: pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})}
	f.srv = httptest.NewServer(http.HandlerFunc(f.serve))
	t.Cleanup(f.srv.Close)
	return f
}

func (f *fakeCA) url(p string) string { return f.srv.URL + p }

func (f *fakeCA) set(endpoint string, p problem) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.fail[endpoint] = p
}

func (f *fakeCA) clear(endpoint string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.fail, endpoint)
}

func (f *fakeCA) count(endpoint string) int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.calls[endpoint]
}

func (f *fakeCA) serve(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	f.nonce++
	w.Header().Set("Replay-Nonce", fmt.Sprintf("nonce-%d", f.nonce))
	f.mu.Unlock()
	switch r.URL.Path {
	case "/dir":
		writeJSON(w, 200, map[string]any{"newNonce": f.url("/nonce"), "newAccount": f.url("/account"), "newOrder": f.url("/order"),
			"revokeCert": f.url("/revoke"), "keyChange": f.url("/key"), "meta": map[string]any{"termsOfService": f.url("/tos")}})
	case "/nonce":
		w.WriteHeader(http.StatusOK)
	default:
		f.post(w, r)
	}
}

// jws is a request body; the fake reads its parts and checks no signature.
type jws struct {
	Protected string `json:"protected"`
	Payload   string `json:"payload"`
}

func (f *fakeCA) post(w http.ResponseWriter, r *http.Request) {
	var body jws
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "bad body", http.StatusBadRequest)
		return
	}
	payload, _ := base64.RawURLEncoding.DecodeString(body.Payload)
	endpoint, id, _ := strings.Cut(strings.TrimPrefix(r.URL.Path, "/"), "/")
	f.mu.Lock()
	f.calls[endpoint]++
	p, failing := f.fail[endpoint]
	f.mu.Unlock()
	if failing {
		writeProblem(w, p)
		return
	}
	switch endpoint {
	case "account":
		f.account(w, body.Protected)
	case "order":
		f.newOrder(w, payload)
	case "orders":
		f.withOrder(w, id, func(o *fakeOrder) { f.writeOrder(w, 200, o) })
	case "authz":
		f.withOrder(w, id, func(o *fakeOrder) { f.authz(w, o, payload) })
	case "chal":
		oid, typ, _ := strings.Cut(id, "-")
		f.withOrder(w, oid, func(o *fakeOrder) { f.accept(w, o, typ) })
	case "finalize":
		f.withOrder(w, id, func(o *fakeOrder) { f.finalize(w, o, payload) })
	case "cert":
		f.withOrder(w, id, func(o *fakeOrder) {
			w.Header().Set("Content-Type", "application/pem-certificate-chain")
			_, _ = w.Write(o.cert)
		})
	default:
		writeProblem(w, problem{status: 404, typ: "malformed", detail: "no such resource"})
	}
}

func (f *fakeCA) withOrder(w http.ResponseWriter, id string, fn func(o *fakeOrder)) {
	f.mu.Lock()
	o := f.orders[id]
	f.mu.Unlock()
	if o == nil {
		writeProblem(w, problem{status: 404, typ: "malformed", detail: "no such order"})
		return
	}
	fn(o)
}

func (f *fakeCA) account(w http.ResponseWriter, protected string) {
	raw, _ := base64.RawURLEncoding.DecodeString(protected)
	var h struct {
		JWK struct {
			Crv string `json:"crv"`
			X   string `json:"x"`
			Y   string `json:"y"`
		} `json:"jwk"`
	}
	_ = json.Unmarshal(raw, &h)
	sum := sha256.Sum256(fmt.Appendf(nil, `{"crv":"%s","kty":"EC","x":"%s","y":"%s"}`, h.JWK.Crv, h.JWK.X, h.JWK.Y))
	f.mu.Lock()
	f.thumb = base64.RawURLEncoding.EncodeToString(sum[:])
	f.mu.Unlock()
	w.Header().Set("Location", f.url("/acct/1"))
	writeJSON(w, http.StatusCreated, map[string]any{"status": "valid"})
}

func (f *fakeCA) newOrder(w http.ResponseWriter, payload []byte) {
	f.mu.Lock()
	gate := f.gate
	f.inFlight++
	f.maxIn = max(f.maxIn, f.inFlight)
	f.mu.Unlock()
	if gate != nil {
		<-gate
	}
	var req struct {
		Identifiers []struct{ Value string } `json:"identifiers"`
	}
	_ = json.Unmarshal(payload, &req)
	f.mu.Lock()
	f.inFlight--
	f.seq++
	o := &fakeOrder{id: fmt.Sprint(f.seq), name: req.Identifiers[0].Value, token: fmt.Sprintf("tok%d_x-Y", f.seq),
		status: "pending", authz: "pending", chal: "pending", expires: f.now().Add(7 * 24 * time.Hour)}
	if f.reuse {
		o.status, o.authz, o.chal = "ready", "valid", "valid"
	}
	f.orders[o.id] = o
	f.mu.Unlock()
	w.Header().Set("Location", f.url("/orders/"+o.id))
	f.writeOrder(w, http.StatusCreated, o)
}

func (f *fakeCA) writeOrder(w http.ResponseWriter, status int, o *fakeOrder) {
	f.mu.Lock()
	body := map[string]any{"status": o.status, "expires": o.expires.Format(time.RFC3339),
		"identifiers": []map[string]string{{"type": "dns", "value": o.name}}, "authorizations": []string{f.url("/authz/" + o.id)},
		"finalize": f.url("/finalize/" + o.id)}
	if o.cert != nil {
		body["certificate"] = f.url("/cert/" + o.id)
	}
	f.mu.Unlock()
	writeJSON(w, status, body)
}

func (f *fakeCA) authz(w http.ResponseWriter, o *fakeOrder, payload []byte) {
	var req struct{ Status string }
	_ = json.Unmarshal(payload, &req)
	f.mu.Lock()
	if req.Status == "deactivated" {
		o.authz, o.status = "deactivated", "invalid"
	}
	chals := []map[string]any{}
	for _, typ := range []string{"http-01", "dns-01"} {
		c := map[string]any{"type": typ, "url": f.url("/chal/" + o.id + "-" + typ), "token": o.token, "status": "pending"}
		if o.chalType == "" || typ == o.chalType {
			c["status"] = o.chal
		}
		if o.chalErr != nil && typ == o.chalType {
			c["error"] = map[string]any{"type": "urn:ietf:params:acme:error:" + o.chalErr.typ, "detail": o.chalErr.detail}
		}
		chals = append(chals, c)
	}
	body := map[string]any{"status": o.authz, "identifier": map[string]string{"type": "dns", "value": strings.TrimPrefix(o.name, "*.")},
		"challenges": chals, "wildcard": strings.HasPrefix(o.name, "*.")}
	f.mu.Unlock()
	writeJSON(w, 200, body)
}

// accept validates a challenge at once with the test's validate function.
func (f *fakeCA) accept(w http.ResponseWriter, o *fakeOrder, typ string) {
	f.mu.Lock()
	ka := o.token + "." + f.thumb
	o.chalType = typ
	validate := f.validate
	f.mu.Unlock()
	var p *problem
	if validate != nil {
		p = validate(o, ka)
	}
	f.mu.Lock()
	if p != nil {
		o.chal, o.authz, o.status, o.chalErr = "invalid", "invalid", "invalid", p
	} else {
		o.chal, o.authz, o.status = "valid", "valid", "ready"
	}
	f.mu.Unlock()
	f.mu.Lock()
	body := map[string]any{"type": typ, "url": f.url("/chal/" + o.id + "-" + typ), "token": o.token, "status": o.chal}
	f.mu.Unlock()
	writeJSON(w, 200, body)
}

func (f *fakeCA) finalize(w http.ResponseWriter, o *fakeOrder, payload []byte) {
	var req struct{ CSR string }
	_ = json.Unmarshal(payload, &req)
	der, _ := base64.RawURLEncoding.DecodeString(req.CSR)
	csr, err := x509.ParseCertificateRequest(der)
	if err != nil || o.status != "ready" {
		writeProblem(w, problem{status: 403, typ: "orderNotReady", detail: "not ready"})
		return
	}
	f.issue(o, csr.DNSNames, csr.PublicKey)
	f.writeOrder(w, 200, o)
}

// issue signs a certificate for names and pub and makes o valid.
func (f *fakeCA) issue(o *fakeOrder, names []string, pub any) {
	now := f.now()
	tpl := &x509.Certificate{SerialNumber: big.NewInt(now.UnixNano()), Subject: pkix.Name{CommonName: names[0]},
		DNSNames: names, NotBefore: now, NotAfter: now.Add(f.life), KeyUsage: x509.KeyUsageDigitalSignature,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}}
	leaf, err := x509.CreateCertificate(rand.Reader, tpl, f.caCert, pub, f.caKey)
	if err != nil {
		f.t.Errorf("fake CA issue: %v", err)
	}
	f.mu.Lock()
	o.cert = append(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: leaf}), f.caPEM...)
	o.status, o.authz, o.chal = "valid", "valid", "valid"
	f.mu.Unlock()
}

// add makes an order for name at the CA, in status (pending, ready,
// valid or invalid), as if Karvan had made it before a restart.
func (f *fakeCA) add(name, status string) *fakeOrder {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.seq++
	o := &fakeOrder{id: fmt.Sprint(f.seq), name: name, token: fmt.Sprintf("tok%d_x-Y", f.seq), status: status, authz: "pending",
		chal: "pending", expires: f.now().Add(7 * 24 * time.Hour)}
	if status != "pending" {
		o.authz, o.chal = "valid", "valid"
	}
	f.orders[o.id] = o
	return o
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeProblem(w http.ResponseWriter, p problem) {
	w.Header().Set("Content-Type", "application/problem+json")
	if p.retryAfter != "" {
		w.Header().Set("Retry-After", p.retryAfter)
	}
	w.WriteHeader(p.status)
	_ = json.NewEncoder(w).Encode(map[string]any{"type": "urn:ietf:params:acme:error:" + p.typ, "detail": p.detail, "status": p.status})
}

func (f *fakeCA) authzState(id string) string {
	f.mu.Lock()
	defer f.mu.Unlock()
	if o := f.orders[id]; o != nil {
		return o.authz
	}
	return ""
}
