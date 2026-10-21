package run

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/PdYrust/karvan/internal/panel"
)

// A compare that changes the outcome names what changed and raises the
// plan version, so the version the operator reviewed no longer runs
// (B3). A compare that finds nothing new keeps the version.
func TestCompareNamesWhatChanged(t *testing.T) {
	h := newHarness(t, panel.Marzban, panel.PasarGuard, pasarGuardDB(t), 3)
	j := h.job(nil)
	ctx := context.Background()
	_, _, v, _ := h.svc.View(j.ID)
	if _, err := h.svc.Check(ctx, j.ID); err != nil {
		t.Fatal(err)
	}
	if c, version, at, err := h.svc.Changes(j.ID); err != nil || c.Changed() || version != v || at.IsZero() {
		t.Fatalf("nothing changed, yet %+v version %d %v", c, version, err)
	}
	h.src.Mutate(func(users *[]map[string]any) {
		(*users)[0]["username"] = "user999"
		(*users)[1]["proxies"].(map[string]any)["vmess"] = map[string]any{"id": "00000000-0000-4000-9000-999999999999"}
	})
	if _, err := h.svc.Check(ctx, j.ID); err != nil {
		t.Fatal(err)
	}
	c, version, _, err := h.svc.Changes(j.ID)
	var got []string
	for _, u := range c.Users {
		got = append(got, u.Key+" "+u.Change)
	}
	if err != nil || version != v+1 || !slices.Equal(got, []string{"user000 gone", "user001 changed", "user999 new"}) {
		t.Fatalf("changes %v version %d %v", got, version, err)
	}
	if err := h.svc.Run(ctx, j.ID, Options{Version: v, Reload: ReloadManual}); !errors.Is(err, ErrStaleVer) {
		t.Fatalf("a run of the reviewed version: %v", err)
	}
	if _, _, _, err := h.svc.Changes(h.created().ID); !errors.Is(err, ErrNotChecked) {
		t.Fatalf("changes before a compare: %v", err)
	}
}
