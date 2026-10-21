package handlers_test

import (
	"net/http"
	"testing"

	"github.com/PdYrust/karvan/internal/migration/run"
)

// A run whose re-check finds the old panel changed ends with done in
// state checked: nothing was written, the changes say what moved, and
// the new plan version runs (B4).
func TestStaleRunGoesBackToChecked(t *testing.T) {
	m := newMigrationEnv(t, 3)
	id := m.create()
	v := m.plan(id)
	m.src.Mutate(func(users *[]map[string]any) { *users = (*users)[1:] })
	if res := m.do(http.MethodPost, api+"/migrations/"+id+"/run", map[string]any{"planVersion": v, "reload": "manual"}); res.status != http.StatusAccepted {
		t.Fatalf("run: %d %s", res.status, res.body)
	}
	events := readEvents(t, m, id, "")
	if done := events[len(events)-1]; done.Data["state"] != run.StateChecked {
		t.Fatalf("done %+v", done)
	}
	if got := changes(t, m, id, ""); got != "changed true version 2 total 1 map[changed:0 gone:1 new:0] users [gone user000] next \"\"" {
		t.Fatalf("changes %s", got)
	}
	if res := m.do(http.MethodGet, api+"/migrations/"+id+"/report", nil); res.status != http.StatusNotFound {
		t.Fatalf("report of a run that wrote nothing: %d", res.status)
	}
	if res := m.do(http.MethodPost, api+"/migrations/"+id+"/run", map[string]any{"planVersion": v + 1, "reload": "manual"}); res.status != http.StatusAccepted {
		t.Fatalf("run the new version: %d %s", res.status, res.body)
	}
	m.runs.Wait()
	var rep run.Report
	m.do(http.MethodGet, api+"/migrations/"+id+"/report", nil).json(t, &rep)
	if rep.State != run.StateDone || rep.Counts.Moved != 2 {
		t.Fatalf("report %s %+v", rep.State, rep.Counts)
	}
}
