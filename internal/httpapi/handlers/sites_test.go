package handlers_test

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"
)

// testCert makes a self-signed certificate for name, valid for an hour,
// and its key, as PEM.
func testCert(t *testing.T, name string) (string, string) {
	t.Helper()
	k, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	tpl := &x509.Certificate{SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: name}, DNSNames: []string{name},
		Issuer: pkix.Name{CommonName: name}, NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().Add(time.Hour)}
	der, err := x509.CreateCertificate(rand.Reader, tpl, tpl, &k.PublicKey, k)
	if err != nil {
		t.Fatal(err)
	}
	kb, err := x509.MarshalPKCS8PrivateKey(k)
	if err != nil {
		t.Fatal(err)
	}
	return string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})),
		string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: kb}))
}

type siteError struct {
	Error struct {
		Code  string `json:"code"`
		Field string `json:"field"`
	} `json:"error"`
}

func (m *migrationEnv) siteFails(t *testing.T, method, path string, body map[string]any, status int, code, field string) {
	t.Helper()
	res := m.do(method, path, body)
	var e siteError
	res.json(t, &e)
	if res.status != status || e.Error.Code != code || e.Error.Field != field {
		t.Errorf("%v: %d %s", body, res.status, res.body)
	}
}

func baseSite(m *migrationEnv, t *testing.T, name string) map[string]any {
	return map[string]any{"name": name, "panelId": m.panelID, "sourceKind": "marzban", "sourceUrl": m.src.URL,
		"oldPath": "sub", "domains": []string{"old.example.com"}, "port": freePort(t), "plainHttpOk": true}
}

func with(s map[string]any, kv ...any) map[string]any {
	out := map[string]any{}
	for k, v := range s {
		out[k] = v
	}
	for i := 0; i+1 < len(kv); i += 2 {
		out[kv[i].(string)] = kv[i+1]
	}
	return out
}

// A wrong site names its field as a code; a taken name, a port that
// would mix HTTP and HTTPS and an unknown panel are 4xx, not 500.
func TestSiteErrorsAreCodes(t *testing.T) {
	m := newMigrationEnv(t, 0)
	sites := api + "/legacy-sites"
	first := baseSite(m, t, "first")
	if res := m.do(http.MethodPost, sites, first); res.status != http.StatusCreated {
		t.Fatalf("add: %d %s", res.status, res.body)
	}
	t.Cleanup(func() { _ = m.bridge.DeleteSite(t.Context(), 1) })
	m.siteFails(t, http.MethodPost, sites, with(first, "oldPath", "a/b"), 400, "site_invalid", "oldPath")
	m.siteFails(t, http.MethodPost, sites, with(first, "name", "x", "panelId", 999), 400, "site_invalid", "panelId")
	m.siteFails(t, http.MethodPost, sites, with(first, "certFile", "cert.pem", "keyFile", "/k.pem"), 400, "site_invalid", "certFile")
	m.siteFails(t, http.MethodPost, sites, with(first, "name", "x", "keyPem", "key"), 400, "site_invalid", "keyPem")
	m.siteFails(t, http.MethodPost, sites, with(first, "oldPath", "other"), 409, "name_taken", "")
	m.siteFails(t, http.MethodPost, sites, with(first, "name", "tls", "oldPath", "other", "certFile", "/c.pem", "keyFile", "/k.pem"),
		409, "port_mixed", "")
	cert, _ := testCert(t, "old.example.com")
	m.siteFails(t, http.MethodPost, sites, with(first, "name", "nokey", "port", freePort(t), "certPem", cert), 400, "site_invalid", "keyPem")
}

type siteOut struct {
	ID     int64           `json:"id"`
	HasKey bool            `json:"hasKey"`
	Cert   *certOut        `json:"cert"`
	Stats  *map[string]int `json:"stats"`
}

type certOut struct {
	Domains  []string `json:"domains"`
	Issuer   string   `json:"issuer"`
	NotAfter string   `json:"notAfter"`
	Warnings []string `json:"warnings"`
}

// A pasted key goes in and never comes back out; the certificate's facts do.
func TestPastedKeyIsWriteOnly(t *testing.T) {
	m := newMigrationEnv(t, 0)
	cert, key := testCert(t, "old.example.com")
	site := with(baseSite(m, t, "tls"), "certPem", cert, "keyPem", key)
	res := m.do(http.MethodPost, api+"/legacy-sites", site)
	var s siteOut
	res.json(t, &s)
	t.Cleanup(func() { _ = m.bridge.DeleteSite(t.Context(), s.ID) })
	if res.status != http.StatusCreated || !s.HasKey || strings.Contains(string(res.body), "PRIVATE KEY") ||
		strings.Contains(string(res.body), "keyPem") {
		t.Fatalf("add: %d %s", res.status, res.body)
	}
	if s.Cert == nil || !slices.Equal(s.Cert.Domains, []string{"old.example.com"}) || s.Cert.Issuer != "old.example.com" ||
		s.Cert.NotAfter == "" || !slices.Equal(s.Cert.Warnings, []string{"expires_soon"}) {
		t.Fatalf("cert facts: %+v", s.Cert)
	}
	delete(site, "keyPem")
	path := api + "/legacy-sites/" + strconv.FormatInt(s.ID, 10)
	if res := m.do(http.MethodPatch, path, site); res.status != http.StatusOK || !strings.Contains(string(res.body), `"hasKey":true`) {
		t.Fatalf("a change without the key: %d %s", res.status, res.body)
	}
	list := m.do(http.MethodGet, api+"/legacy-sites", nil)
	if strings.Contains(string(list.body), "PRIVATE KEY") || !strings.Contains(string(list.body), `"warnings":["expires_soon"]`) {
		t.Fatalf("list: %s", list.body)
	}
}

func (m *migrationEnv) inspect(t *testing.T, body map[string]any) certOut {
	t.Helper()
	res := m.do(http.MethodPost, api+"/legacy-sites/cert", body)
	if res.status != http.StatusOK {
		t.Fatalf("inspect %v: %d %s", body, res.status, res.body)
	}
	var c certOut
	res.json(t, &c)
	return c
}

// A certificate can be checked before the site is saved: pasted, or as
// files on this server. Only facts come back.
func TestInspectCertBeforeSaving(t *testing.T) {
	m := newMigrationEnv(t, 0)
	cert, key := testCert(t, "old.example.com")
	_, otherKey := testCert(t, "old.example.com")
	if c := m.inspect(t, map[string]any{"certPem": cert, "keyPem": key, "domains": []string{"old.example.com"}}); !slices.Equal(c.Warnings, []string{"expires_soon"}) {
		t.Errorf("good pair: %+v", c)
	}
	c := m.inspect(t, map[string]any{"certPem": cert, "keyPem": otherKey, "domains": []string{"new.example.com"}})
	if !slices.Equal(c.Warnings, []string{"key_mismatch", "domain_mismatch", "expires_soon"}) {
		t.Errorf("wrong key and domain: %+v", c)
	}
	dir := t.TempDir()
	certFile, keyFile := filepath.Join(dir, "cert.pem"), filepath.Join(dir, "key.pem")
	if err := os.WriteFile(certFile, []byte(cert), 0o600); err != nil {
		t.Fatal(err)
	}
	if c := m.inspect(t, map[string]any{"certFile": certFile, "keyFile": keyFile}); !slices.Equal(c.Warnings, []string{"no_key", "expires_soon"}) {
		t.Errorf("missing key file: %+v", c)
	}
	if c := m.inspect(t, map[string]any{"certFile": "/dev/zero", "keyFile": "/dev/zero"}); !slices.Equal(c.Warnings, []string{"unreadable"}) {
		t.Errorf("a device: %+v", c)
	}
	m.siteFails(t, http.MethodPost, api+"/legacy-sites/cert", map[string]any{"certFile": "cert.pem", "keyFile": "/k"}, 400, "site_invalid", "certFile")
	if res := m.do(http.MethodPost, api+"/legacy-sites/cert", map[string]any{}); res.status != http.StatusBadRequest {
		t.Errorf("nothing to inspect: %d", res.status)
	}
}

// A token list over 16 MiB is refused with its own code.
func TestTokenListTooLarge(t *testing.T) {
	m := newMigrationEnv(t, 0)
	res := m.do(http.MethodPost, api+"/legacy-sites", baseSite(m, t, "old"))
	var s siteOut
	res.json(t, &s)
	t.Cleanup(func() { _ = m.bridge.DeleteSite(t.Context(), s.ID) })
	path := api + "/legacy-sites/" + strconv.FormatInt(s.ID, 10) + "/tokens"
	if got := uploadText(t, m, path, strings.Repeat("a", 16<<20+1)); !strings.Contains(got, `"too_large"`) {
		t.Fatalf("large list: %s", got)
	}
}

// waitUpload reads a token list's progress until its checks end, as the
// web panel does.
func waitUpload(t *testing.T, m *migrationEnv, path string) response {
	t.Helper()
	for range 200 {
		res := m.do(http.MethodGet, path, nil)
		if !strings.Contains(string(res.body), `"state":"running"`) {
			return res
		}
		time.Sleep(25 * time.Millisecond)
	}
	t.Fatal("the token list is still being checked")
	return response{}
}
