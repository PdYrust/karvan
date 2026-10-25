package handlers_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/PdYrust/karvan/internal/migration/run"
)

func TestRestoreNeedsTheTypedID(t *testing.T) {
	m := newMigrationEnv(t, 1)
	path := api + "/safe-mode/backups/0123456789abcdef/restore"
	if res := m.do(http.MethodPost, path, map[string]any{"confirm": "yes"}); res.status != http.StatusBadRequest || res.errorCode(t) != "confirm_required" {
		t.Fatalf("wrong confirm: %d %s", res.status, res.body)
	}
	if res := m.do(http.MethodPost, path, map[string]any{"confirm": "0123456789abcdef"}); res.status != http.StatusNotFound {
		t.Fatalf("unknown backup: %d %s", res.status, res.body)
	}
}

// nopControl stops and starts nothing.
type nopControl struct{}

func (nopControl) Stop(context.Context) error  { return nil }
func (nopControl) Start(context.Context) error { return nil }

// A restore answers 202 with the job and its last event before the
// restore, runs in the background and leaves the job rolled back; without
// panel control it answers 409 with the reason at once.
func TestRestoreInTheBackground(t *testing.T) {
	m := newMigrationEnv(t, 2)
	id := m.create()
	if res := m.do(http.MethodPost, api+"/migrations/"+id+"/run", map[string]any{"planVersion": m.plan(id), "reload": "manual"}); res.status != http.StatusAccepted {
		t.Fatalf("run: %d %s", res.status, res.body)
	}
	m.runs.Wait()
	var bs struct {
		Backups []run.BackupInfo `json:"backups"`
	}
	m.do(http.MethodGet, api+"/safe-mode/backups", nil).json(t, &bs)
	path := api + "/safe-mode/backups/" + bs.Backups[0].ID + "/restore"
	confirm := map[string]any{"confirm": bs.Backups[0].ID}
	if res := m.do(http.MethodPost, path, confirm); res.status != http.StatusConflict || res.errorCode(t) != "panel_control_unavailable" {
		t.Fatalf("without control: %d %s", res.status, res.body)
	}
	m.runs.Control = func(context.Context, string) (run.PanelControl, error) { return nopControl{}, nil }
	res := m.do(http.MethodPost, path, confirm)
	var got run.Restoring
	res.json(t, &got)
	if res.status != http.StatusAccepted || got.JobID != id || got.After == 0 {
		t.Fatalf("restore: %d %s", res.status, res.body)
	}
	m.runs.Wait()
	if state, _ := m.runs.JobState(context.Background(), id); state != run.StateRolledBack {
		t.Fatalf("job state %s", state)
	}
}
