package run

import (
	"context"
	"slices"
	"testing"
	"time"

	"github.com/PdYrust/karvan/internal/migration/engine"
	"github.com/PdYrust/karvan/internal/migration/plan"
	"github.com/PdYrust/karvan/internal/panel"
	"github.com/PdYrust/karvan/internal/panel/paneldb"
	"github.com/PdYrust/karvan/internal/panel/pasarguarddb"
)

func insertAdmin(t *testing.T, tg paneldb.Target, name string) {
	t.Helper()
	ctx := context.Background()
	w, err := paneldb.OpenWriter(ctx, tg)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = w.Close() }()
	err = w.Chunk(ctx, func(x paneldb.Execer) error {
		_, err := pasarguarddb.Writer{Dialect: tg.Dialect}.InsertAdmin(ctx, x, panel.Admin{Username: name}, "$2b$12$x", 1, time.Now())
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
}

// A plan edit that renames the admin to a taken name, or merges it into
// another admin, shows at once instead of at the next read.
func TestAPlanEditLooksUpANewAdminName(t *testing.T) {
	tg := pasarGuardDB(t)
	h := newHarness(t, panel.Marzban, panel.PasarGuard, tg, 2)
	j := h.job(nil)
	insertAdmin(t, tg, "root")
	ctx := context.Background()
	_, p, v, _ := h.svc.View(j.ID)
	p.Admin = plan.AdminPlan{Username: "root"}
	res, v, err := h.svc.SetPlan(ctx, j.ID, v, p)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Conflicts) != 1 || res.Conflicts[0].Type != engine.AdminTaken || res.Conflicts[0].Detail["suggestion"] != "root_2" {
		t.Fatalf("a rename to a taken name: %+v", res.Conflicts)
	}
	p.Admin = plan.AdminPlan{Mode: plan.AdminMerge, Username: "root"}
	res, _, err = h.svc.SetPlan(ctx, j.ID, v, p)
	if err != nil || res.Login().Merge == nil || slices.ContainsFunc(res.Conflicts, func(c engine.Conflict) bool { return c.Type == engine.AdminTaken }) {
		t.Fatalf("a merge into root: %+v %v", res.Conflicts, err)
	}
	// D35: the old admin field is stored as the login's entry in admins.
	if _, p, _, _ = h.svc.View(j.ID); p.Admin != (plan.AdminPlan{}) || p.Admins[j.snapshot.Admin.Username].Username != "root" {
		t.Fatalf("stored plan %+v", p)
	}
}
