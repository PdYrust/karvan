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

// addCertificate stores a pasted certificate for name and returns its id.
func (m *migrationEnv) addCertificate(t *testing.T, name string) int64 {
	t.Helper()
	cert, key := testCert(t, "old.example.com")
	res := m.do(http.MethodPost, api+"/certificates", map[string]any{"name": name, "certPem": cert, "keyPem": key})
	var c struct {
		ID int64 `json:"id"`
	}
	res.json(t, &c)
	if res.status != http.StatusCreated || c.ID == 0 {
		t.Fatalf("add certificate: %d %s", res.status, res.body)
	}
	return c.ID
}

// A wrong site names its field as a code; a taken name, a port that
// would mix HTTP and HTTPS and an unknown panel are 4xx, not 500.
// Certificate files and PEM are not part of the body any more.
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
	m.siteFails(t, http.MethodPost, sites, with(first, "name", "x", "certificateId", 999), 400, "site_invalid", "certificateId")
	m.siteFails(t, http.MethodPost, sites, with(first, "name", "x", "mode", "both"), 400, "site_invalid", "mode")
	for _, field := range []string{"certFile", "keyFile", "certPem", "keyPem"} {
		m.siteFails(t, http.MethodPost, sites, with(first, "name", "x", field, "/c.pem"), 400, "bad_request", "")
	}
	m.siteFails(t, http.MethodPost, sites, with(first, "oldPath", "other"), 409, "name_taken", "")
	m.siteFails(t, http.MethodPost, sites, with(first, "name", "tls", "oldPath", "other", "certificateId", m.addCertificate(t, "tls")),
		409, "port_mixed", "")
}

type siteOut struct {
	ID          int64    `json:"id"`
	Mode        string   `json:"mode"`
	Cert        *certOut `json:"cert"`
	Certificate *struct {
		ID    int64   `json:"id"`
		Name  string  `json:"name"`
		Facts certOut `json:"facts"`
	} `json:"certificate"`
	Batches  []map[string]any `json:"batches"`
	Warnings []string         `json:"warnings"`
	Stats    *map[string]int  `json:"stats"`
}

type certOut struct {
	Domains  []string `json:"domains"`
	Issuer   string   `json:"issuer"`
	NotAfter string   `json:"notAfter"`
	Warnings []string `json:"warnings"`
}

// A site picks its certificate by id; the answer carries the
// certificate's facts and never a key, a PEM or a path.
func TestSiteCertificateByID(t *testing.T) {
	m := newMigrationEnv(t, 0)
	id := m.addCertificate(t, "tls")
	res := m.do(http.MethodPost, api+"/legacy-sites", with(baseSite(m, t, "tls"), "certificateId", id))
	var s siteOut
	res.json(t, &s)
	t.Cleanup(func() { _ = m.bridge.DeleteSite(t.Context(), s.ID) })
	for _, leak := range []string{"PRIVATE KEY", "CERTIFICATE-----", "keyPem", "certPem", "certFile", "hasKey"} {
		if strings.Contains(string(res.body), leak) {
			t.Fatalf("answer carries %s: %s", leak, res.body)
		}
	}
	if res.status != http.StatusCreated || s.Mode != "manual" || s.Certificate == nil || s.Certificate.ID != id ||
		s.Certificate.Name != "tls" || !slices.Equal(s.Certificate.Facts.Warnings, []string{"expires_soon"}) ||
		s.Cert == nil || s.Cert.Issuer != "old.example.com" || len(s.Batches) != 0 || len(s.Warnings) != 0 {
		t.Fatalf("add: %d %s", res.status, res.body)
	}
	list := m.do(http.MethodGet, api+"/legacy-sites", nil)
	if strings.Contains(string(list.body), "PRIVATE KEY") || !strings.Contains(string(list.body), `"certificateId":`) {
		t.Fatalf("list: %s", list.body)
	}
}

type siteReason struct {
	Error struct {
		Code   string `json:"code"`
		Field  string `json:"field"`
		Reason string `json:"reason"`
	} `json:"error"`
}

// A rule made from batches takes its old panel and destination from
// them; refusals name the field and, for batchIds, the reason.
func TestBatchRuleAnswers(t *testing.T) {
	m := newMigrationEnv(t, 1)
	_, b := doneBatch(t, m)
	sites := api + "/legacy-sites"
	rule := map[string]any{"name": "rule", "mode": "batches", "batchIds": []int64{b.ID}, "port": freePort(t), "plainHttpOk": true}
	for _, c := range []struct {
		body          map[string]any
		field, reason string
	}{
		{with(rule, "batchIds", []int64{b.ID, 999}), "batchIds", "unknown"},
		{with(rule, "batchIds", []int64{}), "batchIds", ""},
		{with(rule, "panelId", m.panelID), "panelId", ""},
		{with(rule, "sourceUrl", m.src.URL), "sourceUrl", ""},
	} {
		res := m.do(http.MethodPost, sites, c.body)
		var e siteReason
		res.json(t, &e)
		if res.status != 400 || e.Error.Code != "site_invalid" || e.Error.Field != c.field || e.Error.Reason != c.reason {
			t.Errorf("%v: %d %s", c.body, res.status, res.body)
		}
	}
	res := m.do(http.MethodPost, sites, rule)
	var s siteOut
	res.json(t, &s)
	t.Cleanup(func() { _ = m.bridge.DeleteSite(t.Context(), s.ID) })
	if res.status != http.StatusCreated || s.Mode != "batches" || len(s.Batches) != 1 || s.Batches[0]["name"] != b.Name ||
		s.Batches[0]["state"] != "active" || !strings.Contains(string(res.body), `"panelId":`+strconv.FormatInt(m.panelID, 10)) {
		t.Fatalf("add: %d %s", res.status, res.body)
	}
	path := sites + "/" + strconv.FormatInt(s.ID, 10)
	m.siteFails(t, http.MethodPatch, path, with(rule, "mode", "manual"), 400, "site_invalid", "mode")
	if res := m.do(http.MethodPatch, path, with(rule, "name", "renamed")); res.status != http.StatusOK {
		t.Fatalf("patch: %d %s", res.status, res.body)
	}
}

// A site whose certificate was not adopted answers 409 until a save
// picks a certificate; the refused save changes nothing and is audited
// as a failure (D54).
func TestSiteNotAdoptedAnswers409(t *testing.T) {
	m := newMigrationEnv(t, 0)
	cert, _ := testCert(t, "old.example.com")
	m.exec(`INSERT INTO legacy_sites (id, name, panel_id, source_kind, source_url, old_path, domains, listen_port, cert_pem,
		plain_http_ok, created_at, updated_at) VALUES (5, 'own', ?, 'marzban', ?, 'sub', '["old.example.com"]', ?, ?, 1, 1, 1)`,
		m.panelID, m.src.URL, freePort(t), cert)
	t.Cleanup(func() { _ = m.bridge.DeleteSite(t.Context(), 5) })
	const cols = "SELECT name || cert_pem || COALESCE(certificate_id, '-') FROM legacy_sites WHERE id = 5"
	var before, after string
	_ = m.db.QueryRow(cols).Scan(&before)
	path := api + "/legacy-sites/5"
	body := baseSite(m, t, "renamed")
	m.siteFails(t, http.MethodPatch, path, body, 409, "certificate_not_adopted", "")
	m.siteFails(t, http.MethodPatch, path, with(body, "certificateId", 0), 409, "certificate_not_adopted", "")
	if _ = m.db.QueryRow(cols).Scan(&after); after != before {
		t.Fatalf("a refused save changed the row: %q", after)
	}
	if got := m.audited("legacy_site.save", "failure"); len(got) != 2 {
		t.Fatalf("failure audits %v", got)
	}
	id := m.addCertificate(t, "picked")
	res := m.do(http.MethodPatch, path, with(body, "certificateId", id))
	var s siteOut
	res.json(t, &s)
	if res.status != http.StatusOK || s.Certificate == nil || s.Certificate.ID != id || len(s.Warnings) != 0 {
		t.Fatalf("pick: %d %s", res.status, res.body)
	}
	if _ = m.db.QueryRow(cols).Scan(&after); after != "renamed"+strconv.FormatInt(id, 10) {
		t.Fatalf("after the pick: %q", after)
	}
}

func (m *migrationEnv) inspect(t *testing.T, body map[string]any) certOut {
	t.Helper()
	res := m.do(http.MethodPost, api+"/certificates/inspect", body)
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
	m.siteFails(t, http.MethodPost, api+"/certificates/inspect", map[string]any{"certFile": "cert.pem", "keyFile": "/k"}, 400, "site_invalid", "certFile")
	if res := m.do(http.MethodPost, api+"/certificates/inspect", map[string]any{}); res.status != http.StatusBadRequest {
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
