package run

import (
	"context"
	"slices"
	"testing"
	"time"

	"github.com/PdYrust/karvan/internal/migration/engine"
	"github.com/PdYrust/karvan/internal/panel"
	"github.com/PdYrust/karvan/internal/panel/marzbandb"
	"github.com/PdYrust/karvan/internal/panel/paneldb"
	"github.com/PdYrust/karvan/internal/panel/paneldb/paneldbtest"
	"github.com/PdYrust/karvan/internal/panel/source"
	"github.com/PdYrust/karvan/internal/secret"
)

func marzbanDB(t *testing.T) paneldb.Target {
	t.Helper()
	return paneldbtest.Load(t, paneldbtest.Fixture{Flavor: "sqlite", Path: "../../panel/marzbandb/testdata/" + marzbandb.Revision + ".sqlite.sql"})
}

// A run whose own re-check finds the panels changed writes nothing and
// goes back to checked with the changes and the login: the operator
// reviews them and runs the new version.
func TestStaleRunGoesBackToChecked(t *testing.T) {
	h := newHarness(t, panel.Marzban, panel.Marzban, marzbanDB(t), 6)
	j := h.job(nil)
	_, _, v, _ := h.svc.View(j.ID)
	h.src.Mutate(func(users *[]map[string]any) { *users = (*users)[1:] })
	if rep := h.run(j, true); rep != nil || h.count("users") != 0 || h.count("admins") != 0 {
		t.Fatalf("a stale run wrote or reported: %+v, %d users", rep, h.count("users"))
	}
	got := trail(t, h, j.ID)
	if end := got[len(got)-4:]; !slices.Equal(end, []string{"check stage start", "check stage failed", "check log plan_stale", " done checked"}) {
		t.Fatalf("events %q", got)
	}
	c, version, _, _ := h.svc.Changes(j.ID)
	if j.State() != StateChecked || !j.holdsLogin() || version != v+1 || len(c.Users) != 1 || c.Users[0].Change != engine.UserGone {
		t.Fatalf("state %s, version %d, changes %+v", j.State(), version, c)
	}
	if rep := h.run(j, true); rep.State != StateDone || rep.Counts.Moved != 5 || h.count("users") != 5 {
		t.Fatalf("the new version: %s, %+v", rep.State, rep.Counts)
	}
}

// A password generated for a run that went back to checked was never
// set, so there is nothing to hand out.
func TestStaleRunDropsTheUnusedPassword(t *testing.T) {
	h := newHarness(t, panel.PasarGuard, panel.Marzban, marzbanDB(t), 2)
	ctx := context.Background()
	j, err := h.svc.Create(ctx, Login{URL: h.src.URL, Kind: panel.PasarGuard, APIKey: secret.NewValue(h.src.APIKey)}, h.panel,
		source.Options{Sleep: func(context.Context, time.Duration) error { return nil }})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := h.svc.Check(ctx, j.ID); err != nil {
		t.Fatal(err)
	}
	h.src.Mutate(func(users *[]map[string]any) { *users = (*users)[1:] })
	_ = h.run(j, false)
	if pw, _ := h.svc.TakePasswords(j.ID); len(pw) != 0 || j.State() != StateChecked {
		t.Fatalf("%d passwords after a stale run in state %s", len(pw), j.State())
	}
}
