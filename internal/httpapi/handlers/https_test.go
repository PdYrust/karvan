package handlers_test

import (
	"context"
	"crypto/tls"
	"log/slog"
	"net"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/PdYrust/karvan/internal/paneltls"
	"github.com/PdYrust/karvan/internal/secret"
	"github.com/PdYrust/karvan/internal/settings"
)

type httpsBody struct {
	TLS         string `json:"tls"`
	DomainID    int64  `json:"domainId"`
	Port        int    `json:"port"`
	Fingerprint string `json:"fingerprint"`
	Pending     *struct {
		TLS     string `json:"tls"`
		Port    int    `json:"port"`
		FromNew bool   `json:"fromNew"`
	} `json:"pending"`
}

// panelTLS gives the panel its certificates, under its own key.
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
	return &paneltls.Manager{Settings: settings.New(e.db), Key: key, Domain: e.bridge.PanelCertificate, Log: slog.New(slog.DiscardHandler)}
}

func newHTTPSEnv(t *testing.T, ttl time.Duration) *addressEnv {
	t.Helper()
	a := newAddressEnv(t, ttl)
	a.login("")
	return a
}

func (e *env) https(basePath string) httpsBody {
	e.t.Helper()
	var h httpsBody
	res := e.do(http.MethodGet, "/"+basePath+"/api/v1/settings/https", nil)
	if res.status != http.StatusOK {
		e.t.Fatalf("https: %d %s", res.status, res.body)
	}
	res.json(e.t, &h)
	return h
}

func (a *addressEnv) putHTTPS(body map[string]any) response {
	a.t.Helper()
	if _, ok := body["password"]; !ok {
		body["password"] = password
	}
	return a.do(http.MethodPut, api+"/settings/https", body)
}

// atHTTPS is this browser on the HTTPS address of port.
func (a *addressEnv) atHTTPS(port int) *env { return a.atHTTPSFor(port, "") }

// atHTTPSFor is this browser on the HTTPS address of port, asking for
// the server name name.
func (a *addressEnv) atHTTPSFor(port int, name string) *env {
	b := *a.env
	tr := &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true, MinVersion: tls.VersionTLS12, ServerName: name}}
	c := *a.client
	c.Transport = tr
	b.client, b.url, b.csrf = &c, "https://127.0.0.1:"+strconv.Itoa(port), ""
	b.fetchCSRF()
	return &b
}

// startSelf asks for self mode on the same port and checks it waits and
// cannot be confirmed over plain HTTP. It returns the fingerprint.
func (a *addressEnv) startSelf() string {
	a.t.Helper()
	res := a.putHTTPS(map[string]any{"tls": "self"})
	var n httpsBody
	res.json(a.t, &n)
	if res.status != http.StatusOK || n.TLS != "self" || n.Port != a.port || len(n.Fingerprint) != 95 ||
		res.header.Get("Cache-Control") != "no-store" {
		a.t.Fatalf("start: %d %s", res.status, res.body)
	}
	if h := a.https(base); h.Pending == nil || h.Pending.TLS != "self" || h.Pending.FromNew || h.Fingerprint != n.Fingerprint {
		a.t.Fatalf("waiting = %+v", h)
	}
	if p := a.address(base).Pending; p == nil || p.TLS != "self" || p.Host != "" {
		a.t.Fatalf("address waiting = %+v", p)
	}
	if res := a.do(http.MethodPost, api+"/settings/address/confirm", nil); res.errorCode(a.t) != "not_new_address" {
		a.t.Fatalf("confirm over plain HTTP: %d %s", res.status, res.body)
	}
	return n.Fingerprint
}

// A change to HTTPS keeps the old address until it is confirmed from
// the HTTPS one.
func TestHTTPSIsConfirmedFromTheNewAddress(t *testing.T) {
	a := newHTTPSEnv(t, 0)
	if h := a.https(base); h.TLS != "off" || h.Port != a.port || h.Pending != nil || h.Fingerprint != "" {
		t.Fatalf("before = %+v", h)
	}
	fp := a.startSelf()
	next := a.atHTTPS(a.port)
	next.login("") // the HTTPS session cookie is another one
	if h := next.https(base); h.Pending == nil || !h.Pending.FromNew || h.Fingerprint != fp {
		t.Fatalf("at the new address = %+v", h)
	}
	if res := next.do(http.MethodPost, api+"/settings/address/confirm", nil); res.status != http.StatusNoContent {
		t.Fatalf("confirm: %d %s", res.status, res.body)
	}
	if h, err := a.saved(); err != nil || h.TLS != (settings.TLS{Mode: settings.TLSSelf}) || h.Port != a.port {
		t.Fatalf("saved = %+v, %v", h, err)
	}
	if got := len(auditRows(t, a.env, "settings.https")); got != 1 {
		t.Fatalf("settings.https audited %d times", got)
	}
}

// Every refused change says why, and nothing waits after it.
func TestHTTPSChangeIsChecked(t *testing.T) {
	a := newHTTPSEnv(t, 0)
	plain := a.addDomain(t, "plain.example.com", "", "")
	cert, key := testCert(t, "panel.example.com")
	good := a.addDomain(t, "panel.example.com", cert, key)
	busy, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer busy.Close()
	for _, c := range []struct {
		body                map[string]any
		status              int
		code, field, reason string
	}{
		{map[string]any{"tls": "maybe"}, 400, "https_invalid", "tls", "unknown"},
		{map[string]any{"tls": "domain"}, 400, "https_invalid", "domainId", "unknown"},
		{map[string]any{"tls": "domain", "domainId": 999}, 400, "https_invalid", "domainId", "unknown"},
		{map[string]any{"tls": "domain", "domainId": plain}, 400, "https_invalid", "domainId", "no_certificate"},
		{map[string]any{"tls": "self", "shared": true}, 409, "port_mixed", "", ""},
		{map[string]any{"tls": "domain", "domainId": good, "shared": true}, 409, "port_mixed", "", ""},
		{map[string]any{"tls": "self", "port": busy.Addr().(*net.TCPAddr).Port}, 409, "port_busy", "", ""},
		{map[string]any{"tls": "off"}, 400, "bad_request", "", ""},
		{map[string]any{"tls": "self", "port": 70000}, 400, "bad_request", "", ""},
		{map[string]any{"tls": "self", "password": "wrong"}, 403, "invalid_credentials", "", ""},
	} {
		res := a.putHTTPS(c.body)
		var e struct {
			Error struct {
				Code   string `json:"code"`
				Field  string `json:"field"`
				Reason string `json:"reason"`
			} `json:"error"`
		}
		res.json(t, &e)
		if res.status != c.status || e.Error.Code != c.code || e.Error.Field != c.field || e.Error.Reason != c.reason {
			t.Errorf("%v: %d %s", c.body, res.status, res.body)
		}
		if a.srv.Status().Pending != nil {
			t.Fatalf("%v left a change waiting", c.body)
		}
	}
	res := a.putHTTPS(map[string]any{"tls": "domain", "domainId": good})
	if res.status != http.StatusOK || strings.Contains(string(res.body), "fingerprint") {
		t.Fatalf("domain mode: %d %s", res.status, res.body)
	}
	if p := a.address(base).Pending; p == nil || p.TLS != "domain" || p.Host != "panel.example.com" {
		t.Fatalf("address waiting = %+v", p)
	}
}

// confirmAt confirms the waiting change from b, an HTTPS browser.
func confirmAt(t *testing.T, b *env) {
	t.Helper()
	b.login("")
	if res := b.do(http.MethodPost, api+"/settings/address/confirm", nil); res.status != http.StatusNoContent {
		t.Fatalf("confirm: %d %s", res.status, res.body)
	}
}

// Use for the Karvan panel serves a domain's certificate once it is
// confirmed from the new address, and Stop goes back to the self-signed
// one the same way; each change waits for its confirm.
func TestPanelDomainModeTurnsOnAndOff(t *testing.T) {
	a := newHTTPSEnv(t, 0)
	a.startSelf()
	confirmAt(t, a.atHTTPS(a.port))
	at := a.atHTTPS(a.port)
	at.login("")
	cert, key := testCert(t, "panel.example.com")
	id := (&addressEnv{env: at}).addDomain(t, "panel.example.com", cert, key)
	use := at.do(http.MethodPut, api+"/settings/https", map[string]any{"tls": "domain", "domainId": id, "password": password})
	if use.status != http.StatusOK {
		t.Fatalf("use: %d %s", use.status, use.body)
	}
	if h, _ := a.saved(); h.TLS.Mode != settings.TLSSelf {
		t.Fatalf("saved before the confirm = %+v", h)
	}
	confirmAt(t, a.atHTTPSFor(a.port, "panel.example.com"))
	if h, err := a.saved(); err != nil || h.TLS != (settings.TLS{Mode: settings.TLSDomain, DomainID: id}) {
		t.Fatalf("saved = %+v, %v", h, err)
	}
	on := a.atHTTPSFor(a.port, "panel.example.com")
	on.login("")
	stop := on.do(http.MethodPut, api+"/settings/https", map[string]any{"tls": "self", "password": password})
	if stop.status != http.StatusOK {
		t.Fatalf("stop: %d %s", stop.status, stop.body)
	}
	confirmAt(t, a.atHTTPS(a.port))
	if h, err := a.saved(); err != nil || h.TLS != (settings.TLS{Mode: settings.TLSSelf}) {
		t.Fatalf("saved after stop = %+v, %v", h, err)
	}
	if got := len(auditRows(t, a.env, "settings.https")); got != 3 {
		t.Fatalf("settings.https audited %d times", got)
	}
}

// A change to HTTPS that is not confirmed in time leaves the old address.
func TestUnconfirmedHTTPSChangeLapses(t *testing.T) {
	a := newHTTPSEnv(t, 100*time.Millisecond)
	if res := a.putHTTPS(map[string]any{"tls": "self"}); res.status != http.StatusOK {
		t.Fatalf("start: %d %s", res.status, res.body)
	}
	deadline := time.Now().Add(5 * time.Second)
	for a.srv.Status().Pending != nil {
		if time.Now().After(deadline) {
			t.Fatal("the change did not lapse")
		}
		time.Sleep(20 * time.Millisecond)
	}
	if h := a.https(base); h.TLS != "off" || h.Pending != nil {
		t.Fatalf("after it lapsed = %+v", h)
	}
	if _, err := a.saved(); err == nil {
		t.Fatal("a lapsed change was saved")
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
	rows, err := e.db.QueryContext(context.Background(), "SELECT detail FROM audit_events WHERE action = ? AND outcome = 'success'", action)
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
