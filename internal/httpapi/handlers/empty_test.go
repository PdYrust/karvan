package handlers_test

import (
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"
)

// wantLists fails unless each named list field of the body is []: a
// fresh install has no rows, and a page must still get [] (a nil Go slice
// encodes as null).
func wantLists(t *testing.T, what string, res response, empty ...string) {
	t.Helper()
	body := string(res.body)
	if res.status >= http.StatusBadRequest {
		t.Fatalf("%s: %d %s", what, res.status, body)
	}
	for _, k := range empty {
		if !strings.Contains(body, `"`+k+`":[]`) {
			t.Errorf("%s: %q is not [] in %s", what, k, body)
		}
	}
}

// Every list of the bridge pages is [] on an empty store, never null.
func TestEmptyStoreListsAreNotNull(t *testing.T) {
	e := newEnv(t)
	if res := e.login(""); res.status != http.StatusOK {
		t.Fatalf("login: %d", res.status)
	}
	wantLists(t, "domains", e.do(http.MethodGet, api+"/domains", nil), "domains")
	wantLists(t, "certificates", e.do(http.MethodGet, api+"/certificates", nil), "certificates")
	wantLists(t, "legacy-sites", e.do(http.MethodGet, api+"/legacy-sites", nil), "sites")
	wantLists(t, "link-redirects", e.do(http.MethodGet, api+"/link-redirects", nil), "redirects")
	wantLists(t, "batches", e.do(http.MethodGet, api+"/batches", nil), "batches")
	wantLists(t, "scan", e.do(http.MethodPost, api+"/certificates/scan", nil), "found")
	wantLists(t, "acme settings", e.do(http.MethodGet, api+"/settings/acme", nil))
}

// A new domain with no certificate, rule or order has its lists as [].
func TestNewDomainListsAreNotNull(t *testing.T) {
	e := newEnv(t)
	if res := e.login(""); res.status != http.StatusOK {
		t.Fatalf("login: %d", res.status)
	}
	res := e.do(http.MethodPost, api+"/domains", map[string]any{"name": "a.example.com", "certificate": plain})
	wantLists(t, "add", res, "certificates", "pins", "rules")
	var d struct{ ID int64 }
	res.json(t, &d)
	wantLists(t, "list", e.do(http.MethodGet, api+"/domains", nil), "certificates", "pins", "rules")
	wantLists(t, "acme", e.do(http.MethodGet, api+"/domains/"+strconv.FormatInt(d.ID, 10)+"/acme", nil))
}

// The HTTPS settings of a fresh install have their lists as [].
func TestEmptyHTTPSListsAreNotNull(t *testing.T) {
	a := newHTTPSEnv(t, time.Minute)
	wantLists(t, "https", a.do(http.MethodGet, api+"/settings/https", nil), "sharedPorts")
}
