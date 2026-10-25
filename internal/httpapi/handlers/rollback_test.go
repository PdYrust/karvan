package handlers_test

import (
	"net/http"
	"testing"

	"github.com/PdYrust/karvan/internal/migration/run"
)

// failedOverTheAPI runs a job of m's users to its end, makes sure a done
// job is refused, then marks it failed, as if the run had failed after its
// writes.
func failedOverTheAPI(t *testing.T, m *migrationEnv) string {
	t.Helper()
	id := m.create()
	v := m.plan(id)
	if res := m.do(http.MethodPost, api+"/migrations/"+id+"/run", map[string]any{"planVersion": v, "reload": "manual"}); res.status != http.StatusAccepted {
		t.Fatalf("run: %d %s", res.status, res.body)
	}
	m.runs.Wait()
	if res := m.do(http.MethodPost, api+"/migrations/"+id+"/rollback", nil); res.status != http.StatusConflict || res.errorCode(t) != "not_rollbackable" {
		t.Fatalf("a done job: %d %s", res.status, res.body)
	}
	if _, err := m.db.Exec("UPDATE jobs SET state = ? WHERE id = ?", run.StateFailed, id); err != nil {
		t.Fatal(err)
	}
	return id
}

// A rollback on request answers 202 and runs in the background; once it
// ends, the job and its listed counts say what the undo took back, and a
// second one is refused.
func TestRollbackOnRequestOverTheAPI(t *testing.T) {
	m := newMigrationEnv(t, 3)
	id := failedOverTheAPI(t, m)
	res := m.do(http.MethodPost, api+"/migrations/"+id+"/rollback", nil)
	var accepted struct{ ID string }
	res.json(t, &accepted)
	if res.status != http.StatusAccepted || accepted.ID != id {
		t.Fatalf("rollback: %d %s", res.status, res.body)
	}
	m.runs.Wait()
	page := listJobs(t, m, "")
	if len(page.Jobs) != 1 {
		t.Fatalf("jobs %+v", page.Jobs)
	}
	if j := page.Jobs[0]; j.State != run.StateRolledBack || j.Counts == nil || j.Counts.Undone != 3 || j.Counts.Moved != 3 || j.EndedAt == nil {
		t.Fatalf("listed job %+v counts %+v", j, j.Counts)
	}
	var job struct{ State string }
	m.do(http.MethodGet, api+"/migrations/"+id, nil).json(t, &job)
	if job.State != run.StateRolledBack {
		t.Fatalf("job state %s", job.State)
	}
	if again := m.do(http.MethodPost, api+"/migrations/"+id+"/rollback", nil); again.status != http.StatusConflict || again.errorCode(t) != "not_rollbackable" {
		t.Fatalf("second rollback: %d %s", again.status, again.body)
	}
}
