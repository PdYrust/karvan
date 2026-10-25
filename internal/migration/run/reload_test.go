package run

import (
	"context"
	"testing"
	"time"

	"github.com/PdYrust/karvan/internal/panel"
	"github.com/PdYrust/karvan/internal/panel/paneldb"
	"github.com/PdYrust/karvan/internal/panel/pasarguarddb"
)

// fakeReload answers every reload with the same result.
type fakeReload ReloadResult

func (f fakeReload) Reload(context.Context, ReloadInput) ReloadResult { return ReloadResult(f) }

// addFullAdmin gives the destination the full admin the panel API methods sign in as.
func addFullAdmin(t *testing.T, tg paneldb.Target) {
	t.Helper()
	ctx := context.Background()
	w, err := paneldb.OpenWriter(ctx, tg)
	if err != nil {
		t.Fatal(err)
	}
	err = w.Batch(ctx, func(x paneldb.Execer) error {
		_, err := pasarguarddb.Writer{Dialect: tg.Dialect}.InsertAdmin(ctx, x, panel.Admin{Username: "root"}, "$2b$12$x", 1, time.Now())
		return err
	})
	_ = w.Close()
	if err != nil {
		t.Fatal(err)
	}
}

// reloadEvents runs a checked job with the method and returns the data of
// its two reload stage events.
func reloadEvents(t *testing.T, h *harness, method string) (start, end map[string]any) {
	t.Helper()
	ctx := context.Background()
	j := h.job(nil)
	_, _, v, _ := h.svc.View(j.ID)
	if err := h.svc.Run(ctx, j.ID, Options{Version: v, Reload: method}); err != nil {
		t.Fatal(err)
	}
	h.svc.Wait()
	past, _, stop, err := h.svc.Events(ctx, j.ID, 0)
	if err != nil {
		t.Fatal(err)
	}
	stop()
	var got []map[string]any
	for _, e := range past {
		if e.Stage == StageReload && e.Type == EventStage {
			got = append(got, e.Data)
		}
	}
	if len(got) != 2 {
		t.Fatalf("reload stage events: %v", got)
	}
	return got[0], got[1]
}

// The reload stage names its method, and its end says whether every
// written user was reloaded, so the live log can tell a manual or partial
// reload from a complete one.
func TestReloadEventsSayHowTheReloadWent(t *testing.T) {
	cases := []struct {
		name, method string
		admin        bool
		result       fakeReload
		complete     bool
	}{
		{name: "manual", method: ReloadManual, complete: false},
		{name: "every user", method: ReloadTouch, admin: true, result: fakeReload{Method: ReloadTouch, OK: 2}, complete: true},
		{name: "some users", method: ReloadTouch, admin: true, result: fakeReload{Method: ReloadTouch, OK: 1, Failed: []string{"u1: 500"}}, complete: false},
		{name: "no panel API", method: ReloadTouch, complete: false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			tg := pasarGuardDB(t)
			h := newHarness(t, panel.Marzban, panel.PasarGuard, tg, 2)
			h.svc.Reloaders[ReloadTouch] = c.result
			if c.admin {
				addFullAdmin(t, tg)
			}
			start, end := reloadEvents(t, h, c.method)
			if start["state"] != "start" || start["method"] != c.method {
				t.Fatalf("start: %v", start)
			}
			if end["state"] != "done" || end["method"] != c.method || end["complete"] != c.complete {
				t.Fatalf("end: %v, want complete %v", end, c.complete)
			}
		})
	}
}
