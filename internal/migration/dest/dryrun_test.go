package dest

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/PdYrust/karvan/internal/migration/engine"
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

// sqlalchemyURL turns a test target back into the URL a panel ENV holds.
func sqlalchemyURL(tg paneldb.Target, kind panel.Kind) string {
	switch tg.Dialect {
	case paneldb.SQLite:
		return "sqlite:///" + tg.Path
	case paneldb.MySQL:
		drv := "mysql+pymysql"
		if kind == panel.PasarGuard {
			drv = "mysql+asyncmy"
		}
		return fmt.Sprintf("%s://%s:%s@%s:%d/%s", drv, tg.User, url.QueryEscape(tg.Password.Reveal()), tg.Host, tg.Port, tg.Database)
	}
	return fmt.Sprintf("postgresql+asyncpg://%s:%s@%s:%d/%s", tg.User, url.QueryEscape(tg.Password.Reveal()), tg.Host, tg.Port, tg.Database)
}

// panelDir writes an ENV (and for Marzban an Xray config) for tg.
func panelDir(t *testing.T, tg paneldb.Target, kind panel.Kind) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), string(kind))
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	env := "SQLALCHEMY_DATABASE_URL = \"" + sqlalchemyURL(tg, kind) + "\"\n"
	if kind == panel.Marzban {
		xray := filepath.Join(dir, "xray_config.json")
		cfg := `{"inbounds": [{"tag": "VLESS TCP", "protocol": "vless"}, {"tag": "VMess WS", "protocol": "vmess"}]}`
		if err := os.WriteFile(xray, []byte(cfg), 0o600); err != nil {
			t.Fatal(err)
		}
		env += "XRAY_JSON = \"" + xray + "\"\n"
	}
	path := filepath.Join(dir, ".env")
	if err := os.WriteFile(path, []byte(env), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func seedPasarGuardGroup(t *testing.T, tg paneldb.Target) {
	t.Helper()
	ctx := context.Background()
	w, err := paneldb.OpenWriter(ctx, tg)
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	err = w.Batch(ctx, func(x paneldb.Execer) error {
		if _, err := x.ExecContext(ctx, "INSERT INTO inbounds (id, tag) VALUES (?, ?)", 1, "Shadowsocks TCP"); err != nil {
			return err
		}
		if _, err := x.ExecContext(ctx, "INSERT INTO "+paneldb.Ident(tg.Dialect, "groups")+" (id, name, is_disabled) VALUES (?, ?, ?)", 1, "Main", false); err != nil {
			return err
		}
		_, err := x.ExecContext(ctx, "INSERT INTO inbounds_groups_association (inbound_id, group_id) VALUES (?, ?)", 1, 1)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
}

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
	if to == panel.PasarGuard {
		seedPasarGuardGroup(t, tg)
	}
	before := paneldbtest.Checksum(t, tg)
	d, err := Open(ctx, panelDir(t, tg, to))
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
