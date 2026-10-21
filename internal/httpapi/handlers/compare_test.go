package handlers_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/PdYrust/karvan/internal/migration/run"
)

// endOfCompare is a compare's last event: its build stage done, or a
// stage failed.
func endOfCompare(e run.Event) bool {
	return e.Type == run.EventStage && (e.Data["state"] == "failed" || e.Stage == run.StageBuild && e.Data["state"] == "done")
}

// compareLog starts a compare, waits for it and returns the data of its
// last log event: why it failed.
func compareLog(t *testing.T, m *migrationEnv, id string) map[string]any {
	t.Helper()
	if res := m.do(http.MethodPost, api+"/migrations/"+id+"/compare", nil); res.status != http.StatusAccepted {
		t.Fatalf("compare: %d %s", res.status, res.body)
	}
	m.runs.Wait()
	past, _, stop, err := m.runs.Events(context.Background(), id, 0)
	if err != nil {
		t.Fatal(err)
	}
	stop()
	for i := len(past) - 1; i >= 0; i-- {
		if past[i].Type == run.EventLog {
			return past[i].Data
		}
	}
	t.Fatalf("no log event in %+v", past)
	return nil
}

// Compare answers at once and reads in the background: the stream shows
// users read of total, then the build.
func TestCompareRunsInTheBackground(t *testing.T) {
	m := newMigrationEnv(t, 3)
	res := m.do(http.MethodPost, api+"/migrations", map[string]any{"sourceUrl": m.src.URL, "username": m.src.Username,
		"password": m.src.Password, "panelId": m.panelID})
	var job struct{ ID string }
	res.json(t, &job)
	res = m.do(http.MethodPost, api+"/migrations/"+job.ID+"/compare", nil)
	var accepted struct{ ID string }
	res.json(t, &accepted)
	if res.status != http.StatusAccepted || accepted.ID != job.ID {
		t.Fatalf("compare: %d %s", res.status, res.body)
	}
	events := readUntil(t, m, job.ID, "", endOfCompare)
	read := events[1]
	if read.Stage != run.StageRead || read.Type != run.EventProgress || read.Data["done"] != 3.0 || read.Data["total"] != 3.0 || read.Data["pass"] != 1.0 {
		t.Fatalf("first progress %+v", read)
	}
	if last := events[len(events)-1]; last.Stage != run.StageBuild || last.Data["state"] != "done" {
		t.Fatalf("last event %+v", last)
	}
	var sum run.Summary
	m.do(http.MethodGet, api+"/migrations/"+job.ID+"/summary", nil).json(t, &sum)
	if sum.State != run.StateChecked || sum.Users != 3 {
		t.Fatalf("summary %+v", sum)
	}
	if res := m.do(http.MethodPost, api+"/migrations/nope/compare", nil); res.status != http.StatusNotFound {
		t.Fatalf("unknown job: %d", res.status)
	}
}
