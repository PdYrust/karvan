package run

import (
	"context"
	"errors"
	"testing"

	"github.com/PdYrust/karvan/internal/migration/engine"
	"github.com/PdYrust/karvan/internal/panel"
)

// D32: a plan save with targets is refused on either panel and leaves the
// plan and its version as they were.
func TestSetPlanRefusesTargets(t *testing.T) {
	for _, to := range []panel.Kind{panel.Marzban, panel.PasarGuard} {
		t.Run(string(to), func(t *testing.T) {
			tg := pasarGuardDB(t)
			if to == panel.Marzban {
				tg = marzbanDB(t)
			}
			h := newHarness(t, panel.Marzban, to, tg, 1)
			j := h.job(nil)
			_, p, v, _ := h.svc.View(j.ID)
			p.Targets = []string{"1"}
			var pe *engine.PlanError
			if _, _, err := h.svc.SetPlan(context.Background(), j.ID, v, p); !errors.As(err, &pe) || pe.Where != "targets" || pe.Problem != "not_allowed" {
				t.Fatalf("err = %v", err)
			}
			if _, got, now, _ := h.svc.View(j.ID); now != v || got.Targets != nil {
				t.Errorf("version %d targets %v after a refused save", now, got.Targets)
			}
		})
	}
}

// D32: a draft plan stored with targets is read with them cleared; a
// finished job shows its plan as it was stored.
func TestDraftPlanTargetsAreCleared(t *testing.T) {
	h := newHarness(t, panel.Marzban, panel.PasarGuard, pasarGuardDB(t), 1)
	j := h.job(nil)
	jj, err := h.svc.job(j.ID)
	if err != nil {
		t.Fatal(err)
	}
	jj.plan.Targets = []string{"1"}
	if _, p, _, _ := h.svc.View(j.ID); p.Targets != nil {
		t.Errorf("draft targets %v, want none", p.Targets)
	}
	jj.plan.Targets, jj.state = []string{"1"}, StateDone
	if _, p, _, _ := h.svc.View(j.ID); len(p.Targets) != 1 {
		t.Errorf("finished job targets %v, want the stored ones", p.Targets)
	}
}
