package handlers_test

import (
	"net/http"
	"strconv"
	"strings"
	"testing"
)

type redirectOut struct {
	ID          int64    `json:"id"`
	Name        string   `json:"name"`
	Domains     []string `json:"domains"`
	FromPath    string   `json:"fromPath"`
	ToURL       string   `json:"toUrl"`
	Certificate *struct {
		ID int64 `json:"id"`
	} `json:"certificate"`
	Warnings []string `json:"warnings"`
}

type redirectError struct {
	Error struct {
		Code   string `json:"code"`
		Field  string `json:"field"`
		Reason string `json:"reason"`
	} `json:"error"`
}

func baseRedirect(t *testing.T, name string) map[string]any {
	return map[string]any{"name": name, "domains": []string{"old.example.com"}, "port": freePort(t), "fromPath": "/go",
		"match": "prefix", "toUrl": "https://new.example.com/"}
}

func (m *migrationEnv) redirectFails(t *testing.T, method, path string, body map[string]any, status int, code, field, reason string) {
	t.Helper()
	res := m.do(method, path, body)
	var e redirectError
	res.json(t, &e)
	if res.status != status || e.Error.Code != code || e.Error.Field != field || e.Error.Reason != reason {
		t.Errorf("%v: %d %s", body, res.status, res.body)
	}
}

// Redirect rules are added, listed, changed and deleted, each audited.
func TestRedirectRoutes(t *testing.T) {
	m := newMigrationEnv(t, 0)
	list := api + "/link-redirects"
	body := baseRedirect(t, " go ")
	res := m.do(http.MethodPost, list, body)
	var d redirectOut
	res.json(t, &d)
	if res.status != http.StatusCreated || d.Name != "go" || d.FromPath != "/go" || d.Certificate != nil || d.Warnings == nil {
		t.Fatalf("add: %d %s", res.status, res.body)
	}
	t.Cleanup(func() { _ = m.bridge.DeleteRedirect(t.Context(), d.ID) })
	m.wantRedirects(t, d.ID)
	path := list + "/" + strconv.FormatInt(d.ID, 10)
	cert := m.certify(t, "tls.example.com")
	res = m.do(http.MethodPatch, path, with(body, "name", "go", "toUrl", "https://other.example.com/x", "domains", []string{"tls.example.com"},
		"https", true))
	res.json(t, &d)
	if res.status != http.StatusOK || d.ToURL != "https://other.example.com/x" || d.Certificate == nil || d.Certificate.ID != cert {
		t.Fatalf("change: %d %s", res.status, res.body)
	}
	wantError(t, m.do(http.MethodPatch, list+"/999", body), http.StatusNotFound, "not_found")
	wantError(t, m.do(http.MethodPatch, list+"/x", body), http.StatusBadRequest, "bad_request")
	if res := m.do(http.MethodDelete, path, nil); res.status != http.StatusNoContent {
		t.Fatalf("delete: %d %s", res.status, res.body)
	}
	wantError(t, m.do(http.MethodDelete, path, nil), http.StatusNotFound, "not_found")
	m.wantRedirectAudits(t)
}

// wantRedirectAudits checks the entries of TestRedirectRoutes: two saves
// and a failed one, a delete and a failed one.
func (m *migrationEnv) wantRedirectAudits(t *testing.T) {
	t.Helper()
	if s := m.audited("redirect.save", "success"); len(s) != 2 || !strings.Contains(s[0], `"redirect":" go "`) {
		t.Fatalf("save audit %v", s)
	}
	if len(m.audited("redirect.save", "failure")) != 1 || len(m.audited("redirect.delete", "success")) != 1 ||
		len(m.audited("redirect.delete", "failure")) != 1 {
		t.Fatal("audit entries")
	}
}

// wantRedirects checks that the list holds only redirect id.
func (m *migrationEnv) wantRedirects(t *testing.T, id int64) {
	t.Helper()
	var got struct {
		Redirects []redirectOut `json:"redirects"`
	}
	res := m.do(http.MethodGet, api+"/link-redirects", nil)
	res.json(t, &got)
	if res.status != http.StatusOK || len(got.Redirects) != 1 || got.Redirects[0].ID != id {
		t.Fatalf("list: %d %s", res.status, res.body)
	}
}

// A wrong field is 400 redirect_invalid with the field, a loop has its
// reason, and conflicts are the rule codes.
func TestRedirectErrorsAreCodes(t *testing.T) {
	m := newMigrationEnv(t, 0)
	list := api + "/link-redirects"
	first := baseRedirect(t, "first")
	res := m.do(http.MethodPost, list, first)
	var d redirectOut
	res.json(t, &d)
	if res.status != http.StatusCreated {
		t.Fatalf("add: %d %s", res.status, res.body)
	}
	t.Cleanup(func() { _ = m.bridge.DeleteRedirect(t.Context(), d.ID) })
	port := strconv.Itoa(first["port"].(int))
	for field, v := range map[string]any{"name": "", "domains": []string{"a..b"}, "port": 0, "fromPath": "/a/../b",
		"match": "glob", "toUrl": "//evil.example.com", "certificateId": 999} {
		m.redirectFails(t, http.MethodPost, list, with(first, "name", "x", field, v), 400, "redirect_invalid", field, "")
	}
	m.redirectFails(t, http.MethodPost, list, with(first, "name", "x", "fromPath", "/loop", "toUrl", "http://old.example.com:"+port+"/go/a"),
		400, "redirect_invalid", "toUrl", "loop")
	m.redirectFails(t, http.MethodPost, list, with(first, "x", 1), 400, "bad_request", "", "")
	m.redirectFails(t, http.MethodPost, list, with(first, "fromPath", "/other"), 409, "name_taken", "", "")
	m.redirectFails(t, http.MethodPost, list, with(first, "name", "x"), 409, "site_overlap", "", "")
	m.certify(t, "tls.example.com")
	m.redirectFails(t, http.MethodPost, list, with(first, "name", "x", "fromPath", "/t", "domains", []string{"tls.example.com"}),
		409, "port_mixed", "", "")
	m.redirectFails(t, http.MethodPost, list, with(first, "name", "x", "domains", []string{"new.example.org"}),
		400, "redirect_invalid", "domains", "not_registered")
	m.redirectFails(t, http.MethodPost, list, with(first, "name", "x", "port", testPanelPort), 409, "port_taken", "", "")
}
