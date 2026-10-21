package handlers_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/PdYrust/karvan/internal/migration/run"
)

// The pre-flight report downloads as JSON and CSV before anything runs
// (B7).
func TestCheckReportDownloads(t *testing.T) {
	m := newMigrationEnv(t, 3)
	id := m.create()
	v := m.plan(id)
	var rep run.CheckReport
	res := m.do(http.MethodGet, api+"/migrations/"+id+"/check-report", nil)
	res.json(t, &rep)
	if res.status != http.StatusOK || rep.Schema != "karvan.check/v1" || rep.PlanVersion != v || rep.Totals.Moving != 3 || len(rep.Users) != 3 {
		t.Fatalf("check-report: %d %s", res.status, res.body)
	}
	file := m.do(http.MethodGet, api+"/migrations/"+id+"/check-report.csv", nil)
	if file.status != http.StatusOK || strings.Count(string(file.body), "\n") != 4 || file.header.Get("Cache-Control") != "no-store" ||
		file.header.Get("Content-Disposition") != `attachment; filename="karvan-check-`+id+`.csv"` {
		t.Fatalf("check-report.csv: %d %v %q", file.status, file.header, file.body)
	}
	created := m.do(http.MethodPost, api+"/migrations", map[string]any{"sourceUrl": m.src.URL, "username": m.src.Username,
		"password": m.src.Password, "panelId": m.panelID})
	var job struct{ ID string }
	created.json(t, &job)
	for path, code := range map[string]string{"/migrations/" + job.ID + "/check-report": "not_checked", "/migrations/nope/check-report.csv": "not_found"} {
		if res := m.do(http.MethodGet, api+path, nil); res.errorCode(t) != code {
			t.Errorf("%s: %d %s", path, res.status, res.body)
		}
	}
}
