package dest

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/PdYrust/karvan/internal/migration/engine"
	"github.com/PdYrust/karvan/internal/migration/migrationtest"
	"github.com/PdYrust/karvan/internal/migration/plan"
	"github.com/PdYrust/karvan/internal/panel"
	"github.com/PdYrust/karvan/internal/panel/fakepanel"
	"github.com/PdYrust/karvan/internal/panel/marzbanapi"
	"github.com/PdYrust/karvan/internal/panel/marzbandb"
	"github.com/PdYrust/karvan/internal/panel/paneldb"
	"github.com/PdYrust/karvan/internal/panel/paneldb/paneldbtest"
	"github.com/PdYrust/karvan/internal/panel/pasarguardapi"
	"github.com/PdYrust/karvan/internal/panel/pasarguarddb"
	"github.com/PdYrust/karvan/internal/panel/source"
	"github.com/PdYrust/karvan/internal/secret"
)

func sourceOf(t *testing.T, kind panel.Kind) (panel.Source, func()) {
	t.Helper()
	sleep := func(context.Context, time.Duration) error { return nil }
	if kind == panel.Marzban {
		var users []map[string]any
		for i := range 30 {
			users = append(users, fakepanel.MarzbanUser(fmt.Sprintf("user%02d", i), i, nil))
		}
		f := fakepanel.NewMarzban(fakepanel.MarzbanAdmin{Username: "reseller", Password: "pw"}, users, fakepanel.Faults{})
		c, _ := source.New(source.Options{BaseURL: f.URL, Sleep: sleep})
		return marzbanapi.New(c, "reseller", secret.NewValue("pw")), f.Close
	}
	var users []map[string]any
	for i := range 30 {
		users = append(users, fakepanel.PasarGuardUser(int64(i+1), fmt.Sprintf("user%02d", i), nil))
	}
	f := fakepanel.NewPasarGuard(fakepanel.PasarGuardAdmin{ID: 7, Username: "shop", Password: "pw"}, users, fakepanel.Faults{})
	c, _ := source.New(source.Options{BaseURL: f.URL, Sleep: sleep})
	return pasarguardapi.New(c, pasarguardapi.Login{Username: "shop", Password: secret.NewValue("pw")}), f.Close
}

func dryRun(t *testing.T, from panel.Kind, tg paneldb.Target, to panel.Kind) {
	t.Helper()
	ctx := context.Background()
	env := migrationtest.PanelEnv(t, tg, to)
	before := paneldbtest.Checksum(t, tg)
	d, err := Open(ctx, env)
	if err != nil {
		t.Fatal(err)
	}
	src, stop := sourceOf(t, from)
	defer stop()
	a, err := src.CurrentAdmin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	users, err := src.Users(ctx)
	if err != nil {
		t.Fatal(err)
	}
	facts, err := d.Facts(ctx, []string{a.Username})
	if err != nil {
		t.Fatal(err)
	}
	if len(facts.Targets) == 0 {
		t.Fatal("no targets on the destination")
	}
	in := engine.Inputs{Source: engine.Source{Kind: from, Admin: a, Users: users}, Dest: facts, Moved: map[string]engine.Moved{},
		Generate: func(string) panel.Credentials { return panel.Credentials{} }, Generated: map[string]panel.Credentials{}}
	r, err := engine.Build(plan.Plan{Now: time.Now().UTC(), Targets: []string{facts.Targets[0].ID}}, in)
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Items) != 30 {
		t.Fatalf("%d items", len(r.Items))
	}
	if after := paneldbtest.Checksum(t, tg); after != before {
		t.Fatal("a dry run changed the destination")
	}
}

func TestDryRunWritesNothing(t *testing.T) {
	for _, from := range []panel.Kind{panel.Marzban, panel.PasarGuard} {
		t.Run(string(from)+"-to-marzban", func(t *testing.T) {
			paneldbtest.Each(t, "../../panel/marzbandb/testdata", marzbandb.Revision, func(t *testing.T, tg paneldb.Target) {
				dryRun(t, from, tg, panel.Marzban)
			})
		})
		t.Run(string(from)+"-to-pasarguard", func(t *testing.T) {
			paneldbtest.Each(t, "../../panel/pasarguarddb/testdata", pasarguarddb.Revision, func(t *testing.T, tg paneldb.Target) {
				dryRun(t, from, tg, panel.PasarGuard)
			})
		})
	}
}
