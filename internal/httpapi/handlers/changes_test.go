package handlers_test

import (
	"fmt"
	"net/http"
	"testing"
)

type changesPage struct {
	PlanVersion int            `json:"planVersion"`
	Changed     bool           `json:"changed"`
	Counts      map[string]int `json:"counts"`
	Users       []struct {
		Key    string `json:"key"`
		Change string `json:"change"`
		Before *struct {
			Username string `json:"username"`
		} `json:"before"`
		After *struct {
			Username string `json:"username"`
		} `json:"after"`
	} `json:"users"`
	Total int    `json:"total"`
	Next  string `json:"next"`
}

// String sums up a page: what changed, the version, and each user with
// its name before or after.
func (p changesPage) String() string {
	users := []string{}
	for _, u := range p.Users {
		name := ""
		if u.Before != nil {
			name = u.Before.Username
		}
		if u.After != nil {
			name += ">" + u.After.Username
		}
		users = append(users, u.Change+" "+name)
	}
	return fmt.Sprintf("changed %v version %d total %d %v users %v next %q", p.Changed, p.PlanVersion, p.Total, p.Counts, users, p.Next)
}

func changes(t *testing.T, m *migrationEnv, id, query string) string {
	t.Helper()
	res := m.do(http.MethodGet, api+"/migrations/"+id+"/changes"+query, nil)
	if res.status != http.StatusOK {
		t.Fatalf("changes: %d %s", res.status, res.body)
	}
	var page changesPage
	res.json(t, &page)
	return page.String()
}

// After a compare, GET changes says what the read changed, a page at a
// time, and the plan version the next run must send (B3).
func TestChangesAfterACompare(t *testing.T) {
	m := newMigrationEnv(t, 3)
	id := m.create()
	v := m.plan(id)
	want := fmt.Sprintf("changed false version %d total 0 map[changed:0 gone:0 new:0] users [] next \"\"", v)
	if got := changes(t, m, id, ""); got != want {
		t.Fatalf("after the first compare: %s", got)
	}
	m.src.Mutate(func(users *[]map[string]any) { (*users)[0]["username"] = "user999" })
	if res := m.do(http.MethodPost, api+"/migrations/"+id+"/compare", nil); res.status != http.StatusAccepted {
		t.Fatalf("compare: %d %s", res.status, res.body)
	}
	m.runs.Wait()
	for query, want := range map[string]string{
		"?limit=1":          "changed true version %d total 2 map[changed:0 gone:1 new:1] users [gone user000] next \"1\"",
		"?cursor=1&limit=1": "changed true version %d total 2 map[changed:0 gone:1 new:1] users [new >user999] next \"\"",
	} {
		if got := changes(t, m, id, query); got != fmt.Sprintf(want, v+1) {
			t.Errorf("%s: %s", query, got)
		}
	}
	run := m.do(http.MethodPost, api+"/migrations/"+id+"/run", map[string]any{"planVersion": v, "reload": "manual"})
	if run.status != http.StatusPreconditionFailed || run.errorCode(t) != "stale_plan" {
		t.Fatalf("a run of the reviewed version: %d %s", run.status, run.body)
	}
}

func TestChangesNeedACheckedJob(t *testing.T) {
	m := newMigrationEnv(t, 1)
	created := m.do(http.MethodPost, api+"/migrations", map[string]any{"sourceUrl": m.src.URL, "username": m.src.Username,
		"password": m.src.Password, "panelId": m.panelID})
	var job struct{ ID string }
	created.json(t, &job)
	for path, code := range map[string]string{"/migrations/" + job.ID + "/changes": "not_checked", "/migrations/nope/changes": "not_found"} {
		if res := m.do(http.MethodGet, api+path, nil); res.errorCode(t) != code {
			t.Errorf("%s: %d %s", path, res.status, res.body)
		}
	}
}
