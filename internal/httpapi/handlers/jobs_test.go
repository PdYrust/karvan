package handlers_test

import (
	"net/http"
	"net/url"
	"strconv"
	"testing"

	"github.com/PdYrust/karvan/internal/migration/run"
)

type jobPage struct {
	Jobs []run.JobRow `json:"jobs"`
	Next string       `json:"next"`
}

func listJobs(t *testing.T, m *migrationEnv, query string) jobPage {
	t.Helper()
	res := m.do(http.MethodGet, api+"/migrations"+query, nil)
	if res.status != http.StatusOK {
		t.Fatalf("%s: %d %s", query, res.status, res.body)
	}
	var page jobPage
	res.json(t, &page)
	return page
}

func TestListMigrations(t *testing.T) {
	m := newMigrationEnv(t, 2)
	first := m.create()
	second := m.create()

	page := listJobs(t, m, "?limit=1")
	if len(page.Jobs) != 1 || page.Jobs[0].ID != second || page.Next == "" {
		t.Fatalf("first page: %+v", page)
	}
	if j := page.Jobs[0]; !j.LoginHeld || j.State != run.StateChecked || j.Panel.ID != m.panelID || j.Source.Admin != "reseller" {
		t.Fatalf("listed job %+v", j)
	}
	page = listJobs(t, m, "?limit=1&cursor="+url.QueryEscape(page.Next))
	if len(page.Jobs) != 1 || page.Jobs[0].ID != first || page.Next != "" {
		t.Fatalf("second page: %+v", page)
	}
	for query, want := range map[string]int{
		"?state=done": 0, "?state=checked,created": 2, "?panel=" + strconv.FormatInt(m.panelID, 10): 2, "?panel=999": 0,
	} {
		if n := len(listJobs(t, m, query).Jobs); n != want {
			t.Errorf("%s: %d jobs", query, n)
		}
	}
}

// A job that has not run lists its counts and end time as null, so every
// row has the same keys.
func TestListedDraftHasNullCounts(t *testing.T) {
	m := newMigrationEnv(t, 1)
	m.create()
	var page struct {
		Jobs []map[string]any `json:"jobs"`
	}
	m.do(http.MethodGet, api+"/migrations", nil).json(t, &page)
	if len(page.Jobs) != 1 {
		t.Fatalf("jobs %v", page.Jobs)
	}
	for _, key := range []string{"counts", "endedAt"} {
		if v, ok := page.Jobs[0][key]; !ok || v != nil {
			t.Errorf("%s: %v, present %v", key, v, ok)
		}
	}
}

func TestListMigrationsRejectsBadQueries(t *testing.T) {
	e := newEnv(t)
	if res := e.do(http.MethodGet, api+"/migrations", nil); res.status != http.StatusUnauthorized {
		t.Fatalf("signed out: %d", res.status)
	}
	e.login("")
	for _, bad := range []string{"?limit=0", "?limit=500", "?limit=x", "?panel=x", "?panel=0", "?state=lost", "?state=done,", "?cursor=nodot"} {
		if res := e.do(http.MethodGet, api+"/migrations"+bad, nil); res.status != http.StatusBadRequest || res.errorCode(t) != "bad_request" {
			t.Errorf("%s: %d", bad, res.status)
		}
	}
}
