package bridge

import (
	"context"
	"fmt"
	"net/http"
	"reflect"
	"testing"
	"time"

	"github.com/PdYrust/karvan/internal/migration/batch"
)

// run ends job done at ended (unix ms) with users moved by it, and makes
// its batch the way a finished migration does.
func (e *env) run(s *batch.Store, job string, ended int64, users ...string) int64 {
	e.t.Helper()
	ctx := context.Background()
	if _, err := e.b.DB.ExecContext(ctx, "INSERT INTO jobs (id, state, source_kind, source_origin, panel_id, safe_mode, created_at, "+
		"updated_at, ended_at) VALUES (?, 'done', 'marzban', ?, 1, 1, 1, 1, ?)", job, e.site.origin, ended); err != nil {
		e.t.Fatal(err)
	}
	for _, u := range users {
		if _, err := e.b.DB.ExecContext(ctx, "UPDATE ledger SET job_id = ? WHERE id = ?", job, e.users[u].ID); err != nil {
			e.t.Fatal(err)
		}
	}
	id, err := s.Ensure(ctx, job, "fake-dest", batch.Facts{OldPath: "sub", OldHosts: []string{"old.example.com"}, SourceIP: "127.0.0.1"})
	if err != nil || id == 0 {
		e.t.Fatalf("batch of %s: %d %v", job, id, err)
	}
	return id
}

// serveRule saves a rule and serves it.
func (e *env) serveRule(s Site) Site {
	e.t.Helper()
	ctx := context.Background()
	saved, err := e.b.st.Save(ctx, s)
	if err != nil {
		e.t.Fatal(err)
	}
	sites, err := e.b.st.Sites(ctx)
	if err != nil {
		e.t.Fatal(err)
	}
	e.b.mu.Lock()
	e.b.load(sites, nil)
	e.b.mu.Unlock()
	return mustSite(e.t, e, saved.ID)
}

// want checks the answer to each user's old link on the env's port, each
// from its own address, so no per-address limit is reached.
func (e *env) want(codes map[string]int) {
	e.t.Helper()
	for name, code := range codes {
		e.asked++
		ip := fmt.Sprintf("198.51.100.%d", 100+e.asked)
		if got, _ := e.get("/sub/"+e.oldToken(name, e.clock.now()), ip); got != code {
			e.t.Errorf("%s: %d, want %d", name, got, code)
		}
	}
}

// One rule made from run 1 serves runs 2 and 3 of the same old panel with
// no edit. Rolling back run 2, the latest run then, puts back a backup
// taken after run 1: run 2's users get the 404, run 1's stay served, and
// run 3 made after it is served too.
func TestOneRuleServesEveryRunOfItsOldPanel(t *testing.T) {
	e := newEnv(t, "ali", "bob", "cyrus")
	ctx := context.Background()
	runs := batch.New(e.b.DB, e.clock.now)
	r1 := e.run(runs, "run-1", 1000, "ali")
	s := fromBatches("old panel", r1)
	s.Port = 2
	rule := e.serveRule(s)
	e.port = 2
	e.want(map[string]int{"ali": http.StatusOK, "bob": http.StatusNotFound})
	e.run(runs, "run-2", 3000, "bob")
	e.want(map[string]int{"ali": http.StatusOK, "bob": http.StatusOK, "cyrus": http.StatusNotFound})
	hold, err := runs.Restoring(ctx, "fake-dest", "run-2", time.UnixMilli(2000))
	if err != nil {
		t.Fatal(err)
	}
	if err := hold.Done(ctx); err != nil {
		t.Fatal(err)
	}
	e.want(map[string]int{"ali": http.StatusOK, "bob": http.StatusNotFound})
	e.run(runs, "run-3", 5000, "cyrus")
	e.want(map[string]int{"ali": http.StatusOK, "bob": http.StatusNotFound, "cyrus": http.StatusOK})
	if after := mustSite(t, e, rule.ID); !reflect.DeepEqual(after, rule) {
		t.Fatalf("the rule changed: %+v, was %+v", after, rule)
	}
}

// A user of another old panel moved onto the same destination is never
// served by the rule, and the old panel is not asked about it.
func TestAnotherOldPanelIsNotMixedIn(t *testing.T) {
	e := newEnv(t, "ali", "zed")
	runs := batch.New(e.b.DB, e.clock.now)
	r1 := e.run(runs, "run-1", 1000, "ali")
	if _, err := e.b.DB.Exec("UPDATE ledger SET source_origin = 'https://elsewhere.example.com' WHERE id = ?", e.users["zed"].ID); err != nil {
		t.Fatal(err)
	}
	if _, err := e.b.DB.Exec("INSERT INTO jobs (id, state, source_kind, source_origin, panel_id, safe_mode, created_at, updated_at, " +
		"ended_at) VALUES ('other', 'done', 'marzban', 'https://elsewhere.example.com', 1, 1, 1, 1, 2000)"); err != nil {
		t.Fatal(err)
	}
	if _, err := e.b.DB.Exec("UPDATE ledger SET job_id = 'other' WHERE id = ?", e.users["zed"].ID); err != nil {
		t.Fatal(err)
	}
	if _, err := runs.Ensure(context.Background(), "other", "fake-dest", batch.Facts{}); err != nil {
		t.Fatal(err)
	}
	s := fromBatches("old panel", r1)
	s.Port = 2
	e.serveRule(s)
	e.port = 2
	calls := e.oldCalls()
	e.want(map[string]int{"zed": http.StatusNotFound})
	if e.oldCalls() != calls || e.tokens() != 0 {
		t.Fatalf("old calls %d -> %d, tokens %d", calls, e.oldCalls(), e.tokens())
	}
	e.want(map[string]int{"ali": http.StatusOK})
}
