//go:build paneltest

package panelctl_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"path/filepath"
	"testing"
	"time"

	"github.com/PdYrust/karvan/internal/migration/livetest"
	"github.com/PdYrust/karvan/internal/migration/migrationtest"
	"github.com/PdYrust/karvan/internal/migration/plan"
	"github.com/PdYrust/karvan/internal/migration/run"
	"github.com/PdYrust/karvan/internal/panel"
	"github.com/PdYrust/karvan/internal/panel/paneldb"
	"github.com/PdYrust/karvan/internal/panelctl"
)

// Karvan stops and starts each running panel through its own compose file.
func TestLiveStopAndStart(t *testing.T) {
	for _, kind := range []panel.Kind{panel.Marzban, panel.PasarGuard} {
		t.Run(string(kind), func(t *testing.T) {
			l := livetest.OpenLive(t, kind)
			c, err := panelctl.Opener{Mode: panelctl.ModeAuto}.Open(context.Background(), filepath.Dir(l.Env))
			if err != nil {
				t.Fatal(err)
			}
			if len(c.Services) != 1 || c.Services[0] != string(kind) {
				t.Fatalf("services %v", c.Services)
			}
			if err := c.Stop(context.Background()); err != nil {
				t.Fatal(err)
			}
			if up(l.API.BaseURL, 2*time.Second) {
				t.Fatal("the panel still answers after stop")
			}
			if err := c.Start(context.Background()); err != nil {
				t.Fatal(err)
			}
			if !up(l.API.BaseURL, 90*time.Second) {
				t.Fatal("the panel did not come back")
			}
		})
	}
}

// up waits for the panel to answer anything over HTTP.
func up(base string, wait time.Duration) bool {
	c := &http.Client{Timeout: time.Second}
	for end := time.Now().Add(wait); time.Now().Before(end); time.Sleep(300 * time.Millisecond) {
		if resp, err := c.Get(base + "/"); err == nil {
			_ = resp.Body.Close()
			return true
		}
	}
	return false
}

// Safe Mode on failure: Karvan stops the running
// Marzban, puts the whole backup back and starts it again, with nobody
// typing a command.
func TestLiveBackupRollback(t *testing.T) {
	l := livetest.OpenLive(t, panel.Marzban)
	src := migrationtest.NewSource(t, panel.Marzban, 6)
	t.Cleanup(src.Close)
	livetest.Renamed(src, "rb_", func(i int, u map[string]any) {
		u["expire"] = 1900000000
		u["proxies"] = map[string]any{"vless": map[string]any{"id": fmt.Sprintf("00000000-0000-4c00-8000-%012d", i), "flow": ""}}
	})
	m := livetest.NewMover(t, l)
	m.Svc.ChunkSize = func(paneldb.Dialect) int { return 2 }
	m.Svc.Fault = func(step string, n int) error {
		if step == "ledger" && n == 2 {
			return errors.New("injected")
		}
		return nil
	}
	before := users(t, l)
	rep := m.Run(src, run.Options{Reload: run.ReloadManual, SafeMode: true, Rollback: run.RollbackBackup}, func(p *plan.Plan) {
		p.Admin.Username = "rb_admin"
	})
	if rep.State != run.StateRolledBack || rep.Counts.Undone != 0 || !restored(rep) {
		t.Fatalf("state %s undone %d stages %+v error %q", rep.State, rep.Counts.Undone, rep.Stages, rep.Error)
	}
	if !up(l.API.BaseURL, 90*time.Second) {
		t.Fatal("the panel did not come back")
	}
	if after := users(t, l); after != before {
		t.Fatalf("users before %d, after %d", before, after)
	}
}

func restored(rep *run.Report) bool {
	for _, s := range rep.Stages {
		if s.Stage == run.StageRestore {
			return s.OK
		}
	}
	return false
}

func users(t *testing.T, l *livetest.Live) int {
	t.Helper()
	ctx := context.Background()
	r, err := paneldb.OpenReader(ctx, l.Target)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	var n int
	if err := r.View(ctx, func(q paneldb.Querier) error {
		return q.QueryRowContext(ctx, "SELECT count(*) FROM users").Scan(&n)
	}); err != nil {
		t.Fatal(err)
	}
	return n
}
