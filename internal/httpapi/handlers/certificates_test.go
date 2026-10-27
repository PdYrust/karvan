package handlers_test

import (
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

type ruleOut struct {
	Kind string `json:"kind"`
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

type inUseOut struct {
	Error struct {
		Code  string    `json:"code"`
		Rules []ruleOut `json:"rules"`
	} `json:"error"`
}

func wantInUse(t *testing.T, res response, code string, site int64) {
	t.Helper()
	var e inUseOut
	res.json(t, &e)
	if res.status != http.StatusConflict || e.Error.Code != code || len(e.Error.Rules) != 1 ||
		e.Error.Rules[0] != (ruleOut{Kind: "panel", ID: site, Name: "old"}) {
		t.Fatalf("want 409 %s: %d %s", code, res.status, res.body)
	}
}

// addSite stores a plain HTTP site for old.example.com and returns its id.
func (m *migrationEnv) addSite(t *testing.T) int64 {
	t.Helper()
	res := m.do(http.MethodPost, api+"/legacy-sites", baseSite(m, t, "old"))
	var s siteOut
	res.json(t, &s)
	if res.status != http.StatusCreated {
		t.Fatalf("add site: %d %s", res.status, res.body)
	}
	t.Cleanup(func() { _ = m.bridge.DeleteSite(t.Context(), s.ID) })
	return s.ID
}

// Domains are added in their normal form, refused with a field when
// wrong, and kept while a rule holds them.
func TestDomainRoutes(t *testing.T) {
	m := newMigrationEnv(t, 0)
	domains := api + "/domains"
	res := m.do(http.MethodPost, domains, map[string]any{"name": " Old.Example.COM. "})
	var d struct {
		ID       int64    `json:"id"`
		Name     string   `json:"name"`
		Warnings []string `json:"warnings"`
	}
	res.json(t, &d)
	if res.status != http.StatusCreated || d.Name != "old.example.com" || len(d.Warnings) != 1 || d.Warnings[0] != "no_certificate" {
		t.Fatalf("add: %d %s", res.status, res.body)
	}
	wantError(t, m.do(http.MethodPost, domains, map[string]any{"name": "old.example.com"}), http.StatusConflict, "name_taken")
	m.siteFails(t, http.MethodPost, domains, map[string]any{"name": "a..b"}, http.StatusBadRequest, "domain_invalid", "name")
	wantError(t, m.do(http.MethodPost, domains, map[string]any{"name": "a.com", "x": 1}), http.StatusBadRequest, "bad_request")
	site := m.addSite(t)
	wantInUse(t, m.do(http.MethodDelete, domains+"/"+strconv.FormatInt(d.ID, 10), nil), "domain_in_use", site)
	var free struct {
		ID int64 `json:"id"`
	}
	m.do(http.MethodPost, domains, map[string]any{"name": "free.example.com"}).json(t, &free)
	path := domains + "/" + strconv.FormatInt(free.ID, 10)
	token := m.csrf
	m.csrf = ""
	wantError(t, m.do(http.MethodPost, domains, map[string]any{"name": "b.example.com"}), http.StatusForbidden, "csrf")
	wantError(t, m.do(http.MethodDelete, path, nil), http.StatusForbidden, "csrf")
	m.csrf = token
	if res := m.do(http.MethodDelete, path, nil); res.status != http.StatusNoContent {
		t.Fatalf("delete: %d %s", res.status, res.body)
	}
	wantError(t, m.do(http.MethodDelete, path, nil), http.StatusNotFound, "not_found")
	wantError(t, m.do(http.MethodDelete, domains+"/x", nil), http.StatusBadRequest, "bad_request")
	list := m.do(http.MethodGet, domains, nil)
	if list.status != http.StatusOK || !strings.Contains(string(list.body), `"rules":[{"kind":"panel","id":`) ||
		strings.Contains(string(list.body), "free.example.com") {
		t.Fatalf("list: %d %s", list.status, list.body)
	}
	if ok, failed := len(m.audited("domain.add", "success")), len(m.audited("domain.add", "failure")); ok != 2 || failed != 2 {
		t.Fatalf("add audits %d %d", ok, failed)
	}
	if ok, failed := len(m.audited("domain.delete", "success")), len(m.audited("domain.delete", "failure")); ok != 1 || failed != 2 {
		t.Fatalf("delete audits %d %d", ok, failed)
	}
}

type certificateOut struct {
	ID     int64  `json:"id"`
	Kind   string `json:"kind"`
	HasKey bool   `json:"hasKey"`
}

// certFields checks a certificate failure's field and reason.
func certFails(t *testing.T, res response, field, reason string) {
	t.Helper()
	var e struct {
		Error struct {
			Code   string `json:"code"`
			Field  string `json:"field"`
			Reason string `json:"reason"`
		} `json:"error"`
	}
	res.json(t, &e)
	if res.status != http.StatusBadRequest || e.Error.Code != "certificate_invalid" || e.Error.Field != field || e.Error.Reason != reason {
		t.Errorf("want certificate_invalid %s %s: %d %s", field, reason, res.status, res.body)
	}
}

// Certificates are added as files or pasted, changed, refused with a
// field and reason when wrong, and kept while a rule uses them.
func TestCertificateRoutes(t *testing.T) {
	m := newMigrationEnv(t, 0)
	certs := api + "/certificates"
	cert, key := testCert(t, "old.example.com")
	_, otherKey := testCert(t, "old.example.com")
	dir := t.TempDir()
	certFile, keyFile := filepath.Join(dir, "cert.pem"), filepath.Join(dir, "key.pem")
	for p, data := range map[string]string{certFile: cert, keyFile: key} {
		if err := os.WriteFile(p, []byte(data), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	var file, pasted certificateOut
	res := m.do(http.MethodPost, certs, map[string]any{"name": "file", "certFile": certFile, "keyFile": keyFile})
	res.json(t, &file)
	if res.status != http.StatusCreated || file.Kind != "file" || !file.HasKey {
		t.Fatalf("add files: %d %s", res.status, res.body)
	}
	res = m.do(http.MethodPost, certs, map[string]any{"name": "pasted", "certPem": cert, "keyPem": key})
	res.json(t, &pasted)
	if res.status != http.StatusCreated || pasted.Kind != "pasted" || !pasted.HasKey {
		t.Fatalf("add pasted: %d %s", res.status, res.body)
	}
	path := certs + "/" + strconv.FormatInt(pasted.ID, 10)
	if res := m.do(http.MethodPatch, path, map[string]any{"name": "renamed", "certPem": cert}); res.status != http.StatusOK ||
		!strings.Contains(string(res.body), `"hasKey":true`) {
		t.Fatalf("change without the key: %d %s", res.status, res.body)
	}
	certFails(t, m.do(http.MethodPost, certs, map[string]any{"name": "", "certPem": cert, "keyPem": key}), "name", "format")
	certFails(t, m.do(http.MethodPost, certs, map[string]any{"name": "x", "certFile": "cert.pem", "keyFile": keyFile}), "certFile", "format")
	certFails(t, m.do(http.MethodPost, certs, map[string]any{"name": "x", "certFile": dir + "/none.pem", "keyFile": keyFile}), "certFile", "unreadable")
	certFails(t, m.do(http.MethodPost, certs, map[string]any{"name": "x", "certPem": "x", "keyPem": key}), "certPem", "no_cert")
	certFails(t, m.do(http.MethodPatch, path, map[string]any{"name": "x", "certPem": cert, "keyPem": otherKey}), "keyPem", "key_mismatch")
	wantError(t, m.do(http.MethodPost, certs, map[string]any{"name": "file", "certPem": cert, "keyPem": key}), http.StatusConflict, "name_taken")
	wantError(t, m.do(http.MethodPatch, certs+"/999", map[string]any{"name": "x", "certPem": cert, "keyPem": key}), http.StatusNotFound, "not_found")
	wantError(t, m.do(http.MethodPatch, certs+"/x", map[string]any{"name": "x"}), http.StatusBadRequest, "bad_request")
	wantError(t, m.do(http.MethodPost, certs, map[string]any{"name": "x", "certPem": strings.Repeat("a", 256<<10)}), http.StatusBadRequest, "bad_request")
	site := m.addSite(t)
	m.exec("UPDATE legacy_sites SET certificate_id = ? WHERE id = ?", file.ID, site)
	wantInUse(t, m.do(http.MethodDelete, certs+"/"+strconv.FormatInt(file.ID, 10), nil), "certificate_in_use", site)
	token := m.csrf
	m.csrf = ""
	wantError(t, m.do(http.MethodPost, certs, map[string]any{"name": "y", "certPem": cert, "keyPem": key}), http.StatusForbidden, "csrf")
	wantError(t, m.do(http.MethodPatch, path, map[string]any{"name": "y", "certPem": cert}), http.StatusForbidden, "csrf")
	wantError(t, m.do(http.MethodDelete, path, nil), http.StatusForbidden, "csrf")
	wantError(t, m.do(http.MethodPost, certs+"/inspect", map[string]any{"certificateId": pasted.ID}), http.StatusForbidden, "csrf")
	m.csrf = token
	if res := m.do(http.MethodDelete, path, nil); res.status != http.StatusNoContent {
		t.Fatalf("delete: %d %s", res.status, res.body)
	}
	wantError(t, m.do(http.MethodDelete, path, nil), http.StatusNotFound, "not_found")
	for action, want := range map[string][2]int{"certificate.add": {2, 5}, "certificate.save": {1, 2}, "certificate.delete": {1, 2}} {
		if ok, failed := len(m.audited(action, "success")), len(m.audited(action, "failure")); ok != want[0] || failed != want[1] {
			t.Errorf("%s audits %d %d", action, ok, failed)
		}
	}
}

// The inspect route moved under certificates and checks a stored one too;
// the old route is gone.
func TestInspectMovedToCertificates(t *testing.T) {
	m := newMigrationEnv(t, 0)
	cert, key := testCert(t, "old.example.com")
	var c certificateOut
	m.do(http.MethodPost, api+"/certificates", map[string]any{"name": "pasted", "certPem": cert, "keyPem": key}).json(t, &c)
	got := m.inspect(t, map[string]any{"certificateId": c.ID, "domains": []string{"new.example.com"}})
	if strings.Join(got.Warnings, ",") != "domain_mismatch,expires_soon" {
		t.Fatalf("inspect a stored certificate: %+v", got)
	}
	m.siteFails(t, http.MethodPost, api+"/certificates/inspect", map[string]any{"certificateId": 999}, 400, "site_invalid", "certificateId")
	if res := m.do(http.MethodPost, api+"/legacy-sites/cert", map[string]any{"certPem": cert}); res.status != http.StatusNotFound {
		t.Fatalf("old route: %d %s", res.status, res.body)
	}
}

// No private key is in any answer, log line or audit entry.
func TestCertificateKeysStayOutOfAnswersLogsAndAudit(t *testing.T) {
	logs := &lockedBuffer{}
	testLog = logs
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(logs, &slog.HandlerOptions{Level: slog.LevelDebug})))
	t.Cleanup(func() { testLog = io.Discard; slog.SetDefault(prev) })
	m := newMigrationEnv(t, 0)
	cert, key := testCert(t, "old.example.com")
	_, otherKey := testCert(t, "old.example.com")
	keyBody := strings.Split(strings.TrimSpace(key), "\n")[1]
	var answers []response
	add := m.do(http.MethodPost, api+"/certificates", map[string]any{"name": "pasted", "certPem": cert, "keyPem": key})
	var c certificateOut
	add.json(t, &c)
	path := api + "/certificates/" + strconv.FormatInt(c.ID, 10)
	answers = append(answers, add,
		m.do(http.MethodPatch, path, map[string]any{"name": "pasted", "certPem": cert, "keyPem": otherKey}),
		m.do(http.MethodPatch, path, map[string]any{"name": "pasted", "certPem": cert, "keyPem": key}),
		m.do(http.MethodGet, api+"/certificates", nil),
		m.do(http.MethodPost, api+"/certificates/inspect", map[string]any{"certificateId": c.ID}),
		m.do(http.MethodPost, api+"/certificates/inspect", map[string]any{"certPem": cert, "keyPem": key}))
	m.addSite(t)
	m.do(http.MethodPost, api+"/domains", map[string]any{"name": "old.example.com"})
	answers = append(answers, m.do(http.MethodGet, api+"/domains", nil))
	audit := strings.Join(append(m.audited("certificate.add", "success"), m.audited("certificate.save", "failure")...), "\n")
	for _, text := range []string{logs.String(), audit} {
		if strings.Contains(text, "PRIVATE KEY") || strings.Contains(text, keyBody) {
			t.Fatalf("a key is in the logs or the audit log:\n%s", text)
		}
	}
	for _, a := range answers {
		if a.status >= 300 && a.status != http.StatusBadRequest || strings.Contains(string(a.body), "PRIVATE KEY") ||
			strings.Contains(string(a.body), keyBody) || strings.Contains(string(a.body), `"keyPem":`) {
			t.Fatalf("answer %d %s", a.status, a.body)
		}
	}
}
