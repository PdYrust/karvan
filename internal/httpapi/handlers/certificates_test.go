package handlers_test

import (
	"fmt"
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

// plain is the certificate choice of a domain without one.
var plain = map[string]any{"kind": "none", "plainOk": true}

// domainID is the id of a registered domain.
func (m *migrationEnv) domainID(t *testing.T, name string) int64 {
	t.Helper()
	var id int64
	if err := m.db.QueryRow("SELECT id FROM domains WHERE name = ?", name).Scan(&id); err != nil {
		t.Fatal(err)
	}
	return id
}

// Domains are added in their normal form with their certificate, refused
// with a field when wrong, and kept while a rule holds them.
func TestDomainRoutes(t *testing.T) {
	m := newMigrationEnv(t, 0)
	domains := api + "/domains"
	res := m.do(http.MethodPost, domains, map[string]any{"name": " New.Example.COM. ", "certificate": plain})
	var d struct {
		ID          int64    `json:"id"`
		Name        string   `json:"name"`
		Certificate any      `json:"certificate"`
		Pins        []any    `json:"pins"`
		Warnings    []string `json:"warnings"`
	}
	res.json(t, &d)
	if res.status != http.StatusCreated || d.Name != "new.example.com" || d.Certificate != nil || d.Pins == nil ||
		len(d.Warnings) != 1 || d.Warnings[0] != "no_certificate" {
		t.Fatalf("add: %d %s", res.status, res.body)
	}
	wantError(t, m.do(http.MethodPost, domains, map[string]any{"name": "new.example.com", "certificate": plain}), http.StatusConflict, "name_taken")
	m.siteFails(t, http.MethodPost, domains, map[string]any{"name": "a..b", "certificate": plain}, http.StatusBadRequest, "domain_invalid", "name")
	m.siteFails(t, http.MethodPost, domains, map[string]any{"name": "a.com"}, http.StatusBadRequest, "domain_invalid", "certificate")
	m.siteFails(t, http.MethodPost, domains, map[string]any{"name": "a.com", "certificate": map[string]any{"kind": "none"}},
		http.StatusBadRequest, "domain_invalid", "plainOk")
	wantError(t, m.do(http.MethodPost, domains, map[string]any{"name": "a.com", "x": 1}), http.StatusBadRequest, "bad_request")
	site := m.addSite(t)
	old := domains + "/" + strconv.FormatInt(m.domainID(t, "old.example.com"), 10)
	wantInUse(t, m.do(http.MethodDelete, old, nil), "domain_in_use", site)
	path := domains + "/" + strconv.FormatInt(d.ID, 10)
	token := m.csrf
	m.csrf = ""
	wantError(t, m.do(http.MethodPost, domains, map[string]any{"name": "b.example.com", "certificate": plain}), http.StatusForbidden, "csrf")
	wantError(t, m.do(http.MethodPut, path+"/certificate", map[string]any{"certificate": plain}), http.StatusForbidden, "csrf")
	wantError(t, m.do(http.MethodDelete, path, nil), http.StatusForbidden, "csrf")
	m.csrf = token
	if res := m.do(http.MethodDelete, path, nil); res.status != http.StatusNoContent {
		t.Fatalf("delete: %d %s", res.status, res.body)
	}
	wantError(t, m.do(http.MethodDelete, path, nil), http.StatusNotFound, "not_found")
	wantError(t, m.do(http.MethodDelete, domains+"/x", nil), http.StatusBadRequest, "bad_request")
	list := m.do(http.MethodGet, domains, nil)
	if list.status != http.StatusOK || !strings.Contains(string(list.body), `"rules":[{"kind":"panel","id":`) ||
		strings.Contains(string(list.body), "new.example.com") {
		t.Fatalf("list: %d %s", list.status, list.body)
	}
	if ok, failed := len(m.audited("domain.add", "success")), len(m.audited("domain.add", "failure")); ok != 1 || failed != 4 {
		t.Fatalf("add audits %d %d", ok, failed)
	}
	if ok, failed := len(m.audited("domain.delete", "success")), len(m.audited("domain.delete", "failure")); ok != 1 || failed != 2 {
		t.Fatalf("delete audits %d %d", ok, failed)
	}
}

// A domain's certificate changes by PUT: refused with a code while an
// HTTPS rule holds the domain and the change is none, or when the
// certificate does not cover it, or for Let's Encrypt without the
// subscriber agreement on the first order.
func TestDomainCertificateRoute(t *testing.T) {
	m := newMigrationEnv(t, 0)
	m.certify(t, "tls.example.com")
	res := m.do(http.MethodPost, api+"/legacy-sites", with(baseSite(m, t, "tls"), "domains", []string{"tls.example.com"}))
	var s siteOut
	res.json(t, &s)
	t.Cleanup(func() { _ = m.bridge.DeleteSite(t.Context(), s.ID) })
	path := api + "/domains/" + strconv.FormatInt(m.domainID(t, "tls.example.com"), 10) + "/certificate"
	var inUse inUseOut
	m.do(http.MethodPut, path, map[string]any{"certificate": plain}).json(t, &inUse)
	if inUse.Error.Code != "domain_in_use" || len(inUse.Error.Rules) != 1 || inUse.Error.Rules[0].ID != s.ID {
		t.Fatalf("none under an HTTPS rule: %+v", inUse)
	}
	other, otherKey := testCert(t, "other.example.com")
	certFails(t, m.do(http.MethodPut, path, map[string]any{"certificate": map[string]any{"kind": "pasted", "certPem": other, "keyPem": otherKey}}),
		"certificate", "not_covered")
	m.siteFails(t, http.MethodPut, path, map[string]any{"certificate": map[string]any{"kind": "acme", "method": "http-01"}},
		http.StatusBadRequest, "acme_invalid", "agreeTos")
	wantError(t, m.do(http.MethodPut, api+"/domains/999/certificate", map[string]any{"certificate": plain}), http.StatusNotFound, "not_found")
	cert, key := testCert(t, "tls.example.com")
	res = m.do(http.MethodPut, path, map[string]any{"certificate": map[string]any{"kind": "pasted", "certPem": cert, "keyPem": key}})
	if res.status != http.StatusOK || !strings.Contains(string(res.body), `"name":"tls.example.com 2"`) {
		t.Fatalf("change: %d %s", res.status, res.body)
	}
	if ok, failed := len(m.audited("domain.certificate", "success")), len(m.audited("domain.certificate", "failure")); ok != 1 || failed != 4 {
		t.Fatalf("audits %d %d", ok, failed)
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
	m.exec("UPDATE certificates SET origin = 'acme' WHERE id = ?", file.ID)
	wantError(t, m.do(http.MethodPatch, certs+"/"+strconv.FormatInt(file.ID, 10), map[string]any{"name": "file", "certFile": certFile,
		"keyFile": keyFile}), http.StatusConflict, "certificate_managed")
	for action, want := range map[string][2]int{"certificate.add": {2, 5}, "certificate.save": {1, 3}, "certificate.delete": {1, 2}} {
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
	answers = append(answers, m.do(http.MethodPost, api+"/domains", map[string]any{"name": "new.example.com",
		"certificate": map[string]any{"kind": "pasted", "certPem": cert, "keyPem": key}}))
	answers = append(answers, m.do(http.MethodPut, api+"/domains/"+strconv.FormatInt(m.domainID(t, "old.example.com"), 10)+"/certificate",
		map[string]any{"certificate": map[string]any{"kind": "pasted", "certPem": cert, "keyPem": key}}))
	answers = append(answers, m.do(http.MethodGet, api+"/domains", nil))
	domainAudit := strings.Join(append(m.audited("domain.add", "failure"), m.audited("domain.certificate", "success")...), "\n")
	if strings.Contains(domainAudit, "PRIVATE KEY") || strings.Contains(domainAudit, keyBody) {
		t.Fatalf("a key is in the domain audit entries:\n%s", domainAudit)
	}
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

// The scan takes an empty body, answers found and skipped lists, is
// audited with the counts only and needs the CSRF token.
func TestCertificateScanRoute(t *testing.T) {
	m := newMigrationEnv(t, 0)
	scan := api + "/certificates/scan"
	var out struct {
		Found   []map[string]any `json:"found"`
		Skipped []struct {
			Place  string `json:"place"`
			Reason string `json:"reason"`
		} `json:"skipped"`
	}
	res := m.do(http.MethodPost, scan, nil)
	res.json(t, &out)
	if res.status != http.StatusOK || out.Found == nil || len(out.Skipped) < 2 || out.Skipped[0].Place != "/root/cert" {
		t.Fatalf("scan: %d %s", res.status, res.body)
	}
	got := m.audited("certificate.scan", "success")
	want := fmt.Sprintf(`{"found":"%d","skipped":"%d"}`, len(out.Found), len(out.Skipped))
	if len(got) != 1 || got[0] != want {
		t.Errorf("audit %v, want %s", got, want)
	}
	token := m.csrf
	m.csrf = ""
	wantError(t, m.do(http.MethodPost, scan, nil), http.StatusForbidden, "csrf")
	m.csrf = token
}
