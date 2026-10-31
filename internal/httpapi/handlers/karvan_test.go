package handlers_test

import (
	"context"
	"crypto/tls"
	"encoding/pem"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/PdYrust/karvan/internal/httpapi"
	"github.com/PdYrust/karvan/internal/httpapi/address"
	"github.com/PdYrust/karvan/internal/paneltls"
	"github.com/PdYrust/karvan/internal/secret"
	"github.com/PdYrust/karvan/internal/settings"
)

// panelTLS gives the panel its certificates, under its own key, from the
// test bridge's domains.
func panelTLS(t *testing.T, e *env) *paneltls.Manager {
	t.Helper()
	keyFile := filepath.Join(t.TempDir(), "secret.key")
	if err := secret.CreateKeyFile(keyFile); err != nil {
		t.Fatal(err)
	}
	key, err := secret.LoadKeyFile(keyFile)
	if err != nil {
		t.Fatal(err)
	}
	return &paneltls.Manager{Settings: settings.New(e.db), Key: key, Domain: e.bridge.PanelCertificate,
		Answers: e.bridge.PanelAnswers, Log: slog.New(slog.DiscardHandler)}
}

func newHTTPSEnv(t *testing.T, ttl time.Duration) *addressEnv {
	t.Helper()
	a := newAddressEnv(t, ttl)
	a.login("")
	return a
}

// atHTTPSFor is this browser on the HTTPS address of port, asking for
// the server name name ("" for none, as for an IP address).
func (a *addressEnv) atHTTPSFor(port int, name string) *env {
	b := *a.env
	tr := &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true, MinVersion: tls.VersionTLS12, ServerName: name}}
	c := *a.client
	c.Transport = tr
	b.client, b.url, b.csrf = &c, "https://127.0.0.1:"+strconv.Itoa(port), ""
	b.fetchCSRF()
	return &b
}

// karvanEnv is a panel where no peer gets plain HTTP as is, as seen from
// a browser on the server's IP over HTTPS, signed in.
type karvanEnv struct {
	*addressEnv
	ip *env
}

func newKarvanEnv(t *testing.T) *karvanEnv {
	t.Helper()
	a := newAddressEnvWith(t, 0, nil)
	if err := settings.New(a.db).SetHTTP(context.Background(), settings.HTTP{Port: a.port, BasePath: base}); err != nil {
		t.Fatal(err)
	}
	ip := a.atHTTPSFor(a.port, "")
	ip.login("")
	return &karvanEnv{addressEnv: a, ip: ip}
}

type karvanBody struct {
	DomainID int64 `json:"domainId"`
	Plain    bool  `json:"plain"`
	Shared   bool  `json:"shared"`
}

// put changes the Karvan domain and checks it was taken.
func (k *karvanEnv) put(body map[string]any, want karvanBody) {
	k.t.Helper()
	res := k.ip.do(http.MethodPut, api+"/settings/karvan-domain", body)
	var got karvanBody
	res.json(k.t, &got)
	if res.status != http.StatusOK || got != want {
		k.t.Fatalf("put %v: %d %s, want %+v", body, res.status, res.body, want)
	}
	var again karvanBody
	k.ip.do(http.MethodGet, api+"/settings/karvan-domain", nil).json(k.t, &again)
	h, err := settings.New(k.db).HTTP(context.Background())
	if err != nil || again != want || h.TLS != (settings.TLS{DomainID: want.DomainID, Plain: want.Plain}) {
		k.t.Fatalf("after put %v: get %+v, saved %+v, %v", body, again, h.TLS, err)
	}
}

// domain registers name with a certificate of its own, or none, and
// returns its id and the certificate's DER.
func (k *karvanEnv) domain(name string, withCert bool) (int64, []byte) {
	k.t.Helper()
	at := &addressEnv{env: k.ip}
	if !withCert {
		return at.addDomain(k.t, name, "", ""), nil
	}
	cert, key := testCert(k.t, name)
	b, _ := pem.Decode([]byte(cert))
	return at.addDomain(k.t, name, cert, key), b.Bytes
}

// served is "self" when a handshake for name gets Karvan's own
// certificate, "domain" for the one with DER want, else the subject.
func (k *karvanEnv) served(name string, want []byte) string {
	k.t.Helper()
	conn, err := tls.Dial("tcp", "127.0.0.1:"+strconv.Itoa(k.port), &tls.Config{InsecureSkipVerify: true,
		MinVersion: tls.VersionTLS12, ServerName: name})
	if err != nil {
		k.t.Fatalf("handshake for %q: %v", name, err)
	}
	defer conn.Close()
	leaf := conn.ConnectionState().PeerCertificates[0]
	switch {
	case want != nil && string(leaf.Raw) == string(want):
		return "domain"
	case leaf.Subject.CommonName == "Karvan":
		return "self"
	}
	return leaf.Subject.CommonName
}

// wantServed checks the certificate of a handshake for each name: the
// domain, another one, and none (the server's IP).
func (k *karvanEnv) wantServed(want []byte, names map[string]string) {
	k.t.Helper()
	for name, w := range names {
		if got := k.served(name, want); got != w {
			k.t.Errorf("handshake for %q got %s, want %s", name, got, w)
		}
	}
}

// plain is the status of a plain HTTP GET of the panel asking for host.
func (k *karvanEnv) plain(host string) int {
	k.t.Helper()
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("http://127.0.0.1:%d/%s/", k.port, base), nil)
	if err != nil {
		k.t.Fatal(err)
	}
	req.Host = fmt.Sprintf("%s:%d", host, k.port)
	res, err := (&http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}).Do(req)
	if err != nil {
		k.t.Fatal(err)
	}
	res.Body.Close()
	return res.StatusCode
}

// Use for Karvan serves the domain's certificate for its name at once;
// every other handshake, the server's IP included, keeps Karvan's own.
// Enabling another domain switches to it, and 0 stops. Each change is
// audited.
func TestKarvanDomainIsPickedByServerName(t *testing.T) {
	k := newKarvanEnv(t)
	a, aDER := k.domain("a.example.com", true)
	b, bDER := k.domain("b.example.com", true)
	k.put(map[string]any{"plain": false}, karvanBody{})
	k.wantServed(aDER, map[string]string{"a.example.com": "self", "": "self"})

	k.put(map[string]any{"domainId": a}, karvanBody{DomainID: a})
	k.wantServed(aDER, map[string]string{"a.example.com": "domain", "A.EXAMPLE.COM": "domain", "b.example.com": "self", "": "self"})
	if got := k.plain("a.example.com"); got != http.StatusTemporaryRedirect {
		t.Fatalf("plain HTTP for a domain with a certificate: %d", got)
	}

	k.put(map[string]any{"domainId": b}, karvanBody{DomainID: b})
	k.wantServed(bDER, map[string]string{"b.example.com": "domain", "a.example.com": "self", "": "self"})

	k.put(map[string]any{"domainId": 0}, karvanBody{})
	k.wantServed(bDER, map[string]string{"b.example.com": "self", "a.example.com": "self", "": "self"})
	rows := auditRows(t, k.env, "settings.https")
	want := []string{`{"domain":"` + strconv.FormatInt(a, 10) + `","previous":"0"}`,
		`{"domain":"` + strconv.FormatInt(b, 10) + `","previous":"` + strconv.FormatInt(a, 10) + `"}`,
		`{"domain":"0","previous":"` + strconv.FormatInt(b, 10) + `"}`}
	if strings.Join(rows, " ") != strings.Join(want, " ") {
		t.Fatalf("audited %v, want %v", rows, want)
	}
}

// The enabled domain without a certificate gets Karvan's own for its
// name. With plain HTTP chosen, a plain request for its name is answered,
// any other is still sent on to HTTPS; once the domain gets a certificate
// it is served at once and plain HTTP stops.
func TestKarvanDomainWithoutACertificate(t *testing.T) {
	k := newKarvanEnv(t)
	id, _ := k.domain("panel.example.com", false)
	k.put(map[string]any{"domainId": id}, karvanBody{DomainID: id})
	k.wantServed(nil, map[string]string{"panel.example.com": "self", "": "self"})
	if got := k.plain("panel.example.com"); got != http.StatusTemporaryRedirect {
		t.Fatalf("own certificate: plain HTTP got %d", got)
	}

	k.put(map[string]any{"plain": true}, karvanBody{DomainID: id, Plain: true})
	k.wantServed(nil, map[string]string{"panel.example.com": "self", "": "self"})
	for host, want := range map[string]int{"panel.example.com": http.StatusOK, "other.example.com": http.StatusTemporaryRedirect,
		"127.0.0.1": http.StatusTemporaryRedirect} {
		if got := k.plain(host); got != want {
			t.Errorf("plain HTTP for %s: %d, want %d", host, got, want)
		}
	}
	if got := auditRows(t, k.env, "settings.https"); len(got) != 2 || got[1] != `{"plain":"true"}` {
		t.Fatalf("audited %v", got)
	}

	cert, key := testCert(t, "panel.example.com")
	res := k.ip.do(http.MethodPut, api+"/domains/"+strconv.FormatInt(id, 10)+"/certificate",
		map[string]any{"certificate": map[string]any{"kind": "pasted", "certPem": cert, "keyPem": key}})
	if res.status != http.StatusOK {
		t.Fatalf("give it a certificate: %d %s", res.status, res.body)
	}
	b, _ := pem.Decode([]byte(cert))
	k.wantServed(b.Bytes, map[string]string{"panel.example.com": "domain", "": "self"})
	if got := k.plain("panel.example.com"); got != http.StatusTemporaryRedirect {
		t.Fatalf("plain HTTP with a certificate: %d", got)
	}
}

// Each refused change says why and changes nothing; the domain enabled
// for Karvan cannot be deleted until it is stopped.
func TestKarvanDomainIsChecked(t *testing.T) {
	k := newKarvanEnv(t)
	id, _ := k.domain("panel.example.com", true)
	ip, _ := k.domain("203.0.113.9", false)
	for _, c := range []struct {
		body         map[string]any
		status       int
		code, reason string
	}{
		{map[string]any{}, 400, "bad_request", ""},
		{map[string]any{"domainId": -1}, 400, "bad_request", ""},
		{map[string]any{"tls": "domain"}, 400, "bad_request", ""},
		{map[string]any{"domainId": 999}, 400, "karvan_invalid", "unknown"},
		{map[string]any{"domainId": ip}, 400, "karvan_invalid", "ip"},
	} {
		res := k.ip.do(http.MethodPut, api+"/settings/karvan-domain", c.body)
		var e struct {
			Error struct {
				Code   string `json:"code"`
				Field  string `json:"field"`
				Reason string `json:"reason"`
			} `json:"error"`
		}
		res.json(t, &e)
		if res.status != c.status || e.Error.Code != c.code || e.Error.Reason != c.reason {
			t.Errorf("%v: %d %s", c.body, res.status, res.body)
		}
	}
	k.put(map[string]any{"domainId": id}, karvanBody{DomainID: id})
	del := api + "/domains/" + strconv.FormatInt(id, 10)
	if res := k.ip.do(http.MethodDelete, del, nil); res.status != http.StatusConflict || res.errorCode(t) != "domain_karvan" {
		t.Fatalf("delete the Karvan domain: %d %s", res.status, res.body)
	}
	k.put(map[string]any{"domainId": 0}, karvanBody{})
	if res := k.ip.do(http.MethodDelete, del, nil); res.status != http.StatusNoContent {
		t.Fatalf("delete after stop: %d %s", res.status, res.body)
	}
	if res := newAddressEnv(t, 0).do(http.MethodGet, api+"/settings/karvan-domain", nil); res.status != http.StatusUnauthorized {
		t.Fatalf("signed out: %d", res.status)
	}
}

// A panel on a shared bridge port keeps its domain until it has a port
// of its own; the plain HTTP choice can still change.
func TestKarvanDomainStaysOnASharedPort(t *testing.T) {
	e, d := newEnvDeps(t)
	srv := &address.Server{Bind: "127.0.0.1", Log: d.Logger,
		HTTP: func(h http.Handler) *http.Server { return &http.Server{Handler: h, ReadHeaderTimeout: 5 * time.Second} }}
	d.Address, d.PanelTLS = srv, panelTLS(t, e)
	srv.Handler = func(b string) (http.Handler, error) {
		dd := d
		dd.BasePath = b
		return httpapi.New(dd)
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	current := address.Site{Port: freePort(t), BasePath: base, Shared: true}
	go func() { done <- srv.Serve(ctx, current, nil) }()
	t.Cleanup(func() { cancel(); <-done })
	for srv.Status().Current != current {
		time.Sleep(5 * time.Millisecond)
	}
	h, err := srv.Handler(base)
	if err != nil {
		t.Fatal(err)
	}
	ts := httptest.NewServer(h)
	t.Cleanup(ts.Close)
	e.url = ts.URL
	e.login("")
	d.PanelTLS.Set(settings.TLS{DomainID: 1, Shared: true})
	res := e.do(http.MethodPut, api+"/settings/karvan-domain", map[string]any{"domainId": 0})
	if res.status != http.StatusConflict || res.errorCode(t) != "karvan_shared" || d.PanelTLS.Panel().DomainID != 1 {
		t.Fatalf("stop on a shared port: %d %s", res.status, res.body)
	}
	if res := e.do(http.MethodPut, api+"/settings/karvan-domain", map[string]any{"plain": true}); res.status != http.StatusOK ||
		!d.PanelTLS.Panel().Plain {
		t.Fatalf("plain on a shared port: %d %s", res.status, res.body)
	}
}

// addDomain registers name, with the pasted certificate when given.
func (a *addressEnv) addDomain(t *testing.T, name, cert, key string) int64 {
	t.Helper()
	choice := map[string]any{"kind": "none", "plainOk": true}
	if cert != "" {
		choice = map[string]any{"kind": "pasted", "certPem": cert, "keyPem": key}
	}
	res := a.do(http.MethodPost, api+"/domains", map[string]any{"name": name, "certificate": choice})
	var d struct {
		ID int64 `json:"id"`
	}
	res.json(t, &d)
	if res.status != http.StatusCreated {
		t.Fatalf("add domain %s: %d %s", name, res.status, res.body)
	}
	return d.ID
}

func auditRows(t *testing.T, e *env, action string) []string {
	t.Helper()
	rows, err := e.db.QueryContext(context.Background(), "SELECT detail FROM audit_events WHERE action = ? AND outcome = 'success' ORDER BY id", action)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var d string
		if err := rows.Scan(&d); err != nil {
			t.Fatal(err)
		}
		out = append(out, d)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return out
}
