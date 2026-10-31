package run

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/PdYrust/karvan/internal/migration/engine"
	"github.com/PdYrust/karvan/internal/migration/plan"
	"github.com/PdYrust/karvan/internal/panel"
)

// The pre-flight report says what a run of the current plan would do,
// before anything runs.
func TestCheckReportOfTheCurrentPlan(t *testing.T) {
	h := newHarness(t, panel.Marzban, panel.PasarGuard, pasarGuardDB(t), 3)
	j := h.job(func(p *plan.Plan) { p.Users = map[string]plan.UserDecision{"user001": {Exclude: true}} })
	rep, err := h.svc.CheckReport(context.Background(), j.ID)
	if err != nil {
		t.Fatal(err)
	}
	_, _, v, _ := h.svc.View(j.ID)
	header := fmt.Sprintf("%s %s v%d from %s %s to %s %s as %s, %d blockers, stable %v", rep.Schema, rep.JobID, rep.PlanVersion,
		rep.Source.Kind, rep.Source.Admin, rep.Destination.Kind, rep.Destination.Name, rep.Admin.Username, rep.Blockers, rep.Stable)
	if want := fmt.Sprintf("karvan.check/v1 %s v%d from marzban reseller to pasarguard dest as reseller, 0 blockers, stable true", j.ID, v); header != want || rep.CheckedAt.IsZero() {
		t.Fatalf("header %q", header)
	}
	if rep.Totals != (CheckTotals{Users: 3, Moving: 2, Skipped: 1}) {
		t.Fatalf("totals %+v", rep.Totals)
	}
	var lines []string
	for _, u := range rep.Users {
		lines = append(lines, fmt.Sprintf("%s moving %v to %v applied %v fixed %v open %v", u.Key, u.Moving, u.Targets, u.Applied, u.Fixed, u.Conflicts))
	}
	if want := "[user000 moving true to [1] applied [] fixed [] open [] user001 moving false to [1] applied [excluded] fixed [] open [] " +
		"user002 moving true to [1] applied [] fixed [] open []]"; fmt.Sprint(lines) != want {
		t.Fatalf("users %q", lines)
	}
}

func TestCheckReportNeedsACheck(t *testing.T) {
	h := newHarness(t, panel.Marzban, panel.PasarGuard, pasarGuardDB(t), 1)
	if _, err := h.svc.CheckReport(context.Background(), h.created().ID); !errors.Is(err, ErrNotChecked) {
		t.Fatalf("before a compare: %v", err)
	}
	if _, err := h.svc.CheckReport(context.Background(), "nope"); !errors.Is(err, ErrNoJob) {
		t.Fatalf("an unknown job: %v", err)
	}
}

// Users moved before count as skipped and as already moved.
func TestCheckTotalsAlreadyMoved(t *testing.T) {
	var tt CheckTotals
	tt.count(&engine.Item{Conflicts: []engine.Conflict{{Type: engine.AlreadyMoved, Fix: engine.OptSkip}}})
	tt.count(&engine.Item{Include: true})
	if tt != (CheckTotals{Users: 2, Moving: 1, Skipped: 1, AlreadyMoved: 1}) {
		t.Fatalf("totals %+v", tt)
	}
}
