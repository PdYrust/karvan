package handlers_test

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/PdYrust/karvan/internal/audit"
	"github.com/PdYrust/karvan/internal/discovery"
)

// Saved panels that turn out to be copies of the real one: an unused one
// leaves Karvan's list (its folder stays), one a job still points at is
// kept with the finding that asks to remove it.
func TestScanDropsSavedCopies(t *testing.T) {
	m := newMigrationEnv(t, 0)
	ctx := context.Background()
	mainEnv, unused, used, mainID, usedID := savedCopies(t, m)
	var got struct {
		Panels []struct {
			ID       int64               `json:"id"`
			EnvPath  string              `json:"envPath"`
			Ready    bool                `json:"ready"`
			Findings []discovery.Finding `json:"findings"`
		} `json:"panels"`
	}
	res := m.do(http.MethodPost, api+"/panels/scan", map[string]any{"extra": []string{mainEnv, unused, used}})
	res.json(t, &got)
	if res.status != http.StatusOK || len(got.Panels) != 2 || got.Panels[0].ID != mainID || !got.Panels[0].Ready {
		t.Fatalf("scan: %d %s", res.status, res.body)
	}
	kept := got.Panels[1]
	if kept.ID != usedID || kept.Ready || !slices.Contains(kept.Findings, discovery.Finding{Code: discovery.PanelCopy, Detail: filepath.Dir(mainEnv)}) {
		t.Fatalf("kept copy: %+v", kept)
	}
	if after, err := m.runs.Panels(ctx); err != nil || len(after) != 2 {
		t.Fatalf("saved panels: %+v %v", after, err)
	}
	if _, err := os.Stat(unused); err != nil {
		t.Fatalf("the copy's folder was touched: %v", err)
	}
	if n := len(m.audited(audit.ActionPanelCopyRemoved, string(audit.Success))); n != 1 {
		t.Fatalf("%d audit entries for the removed copy", n)
	}
	m.do(http.MethodGet, api+"/panels", nil).json(t, &got)
	if len(got.Panels) != 2 || got.Panels[1].ID != usedID || got.Panels[1].Ready {
		t.Fatalf("list: %+v", got)
	}
}

// copyEnv copies the ENV file src to dir/name/.env, the way an update
// script leaves a backup folder.
func copyEnv(t *testing.T, src, dir, name string) string {
	t.Helper()
	body, err := os.ReadFile(src)
	if err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(dir, name, ".env")
	if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, body, 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

// savedCopies saves two backup copies of the test panel, one of them used
// by a job.
func savedCopies(t *testing.T, m *migrationEnv) (mainEnv, unused, used string, mainID, usedID int64) {
	t.Helper()
	ctx := context.Background()
	rows, err := m.runs.Panels(ctx)
	if err != nil {
		t.Fatal(err)
	}
	mainEnv, mainID = rows[0].EnvPath, rows[0].ID
	parent := t.TempDir()
	unused, used = copyEnv(t, mainEnv, parent, "pasarguard.backup.1"), copyEnv(t, mainEnv, parent, "pasarguard.backup.2")
	if _, err := m.runs.RegisterPanel(ctx, unused, "pasarguard.backup.1", "pasarguard"); err != nil {
		t.Fatal(err)
	}
	if usedID, err = m.runs.RegisterPanel(ctx, used, "pasarguard.backup.2", "pasarguard"); err != nil {
		t.Fatal(err)
	}
	m.exec("UPDATE jobs SET panel_id = ? WHERE id = ?", usedID, m.create())
	return mainEnv, unused, used, mainID, usedID
}
