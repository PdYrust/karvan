package handlers_test

import (
	"net/http"
	"testing"

	"github.com/PdYrust/karvan/internal/migration/run"
)

// The overview sums up the jobs: one ran and moved its users, the other
// is still a draft.
func TestOverviewSumsUpMigrations(t *testing.T) {
	m := newMigrationEnv(t, 2)
	ran := m.create()
	v := m.plan(ran)
	if res := m.do(http.MethodPost, api+"/migrations/"+ran+"/run", map[string]any{"planVersion": v, "reload": "manual"}); res.status != http.StatusAccepted {
		t.Fatalf("run: %d %s", res.status, res.body)
	}
	m.runs.Wait()
	m.create()

	var ov struct {
		Migrations run.Totals `json:"migrations"`
	}
	res := m.do(http.MethodGet, api+"/overview", nil)
	if res.status != http.StatusOK {
		t.Fatalf("overview: %d %s", res.status, res.body)
	}
	res.json(t, &ov)
	got := ov.Migrations
	if got.Jobs[run.StateDone] != 1 || got.Jobs[run.StateChecked] != 1 || got.Moved != 2 || got.Skipped != 0 || got.LastEnded == nil {
		t.Fatalf("migrations %+v", got)
	}
}
