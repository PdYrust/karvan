package pasarguarddb

import (
	"context"
	"errors"
	"flag"
	"slices"
	"testing"
	"time"

	"github.com/PdYrust/karvan/internal/panel"
	"github.com/PdYrust/karvan/internal/panel/paneldb"
	"github.com/PdYrust/karvan/internal/panel/paneldb/paneldbtest"
)

func view(t *testing.T, tg paneldb.Target, fn func(q paneldb.Querier, srv paneldb.Server) error) {
	t.Helper()
	ctx := context.Background()
	r, err := paneldb.OpenReader(ctx, tg)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	if err := r.View(ctx, func(q paneldb.Querier) error { return fn(q, r.Server()) }); err != nil {
		t.Fatal(err)
	}
}

func exec(t *testing.T, tg paneldb.Target, stmts ...[]any) {
	t.Helper()
	ctx := context.Background()
	w, err := paneldb.OpenWriter(ctx, tg)
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	err = w.Chunk(ctx, func(x paneldb.Execer) error {
		for _, s := range stmts {
			if _, err := x.ExecContext(ctx, s[0].(string), s[1:]...); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

// eachRevision runs fn on the fixtures of every known revision, on every
// database server.
func eachRevision(t *testing.T, fn func(t *testing.T, tg paneldb.Target)) {
	t.Helper()
	for _, rev := range Revisions {
		t.Run(rev, func(t *testing.T) { paneldbtest.Each(t, "testdata", rev, fn) })
	}
}

func TestInspectFixtures(t *testing.T) {
	eachRevision(t, func(t *testing.T, tg paneldb.Target) {
		view(t, tg, func(q paneldb.Querier, srv paneldb.Server) error {
			s, err := Inspect(context.Background(), q, srv)
			if err != nil {
				return err
			}
			if s.Checked {
				t.Error("a known revision needs no schema check")
			}
			if want := tg.Dialect == paneldb.MySQL; s.AdminsFoldCase != want {
				t.Errorf("AdminsFoldCase = %v, want %v", s.AdminsFoldCase, want)
			}
			if len(s.UnknownChildren) > 0 {
				t.Errorf("unknown children: %+v", s.UnknownChildren)
			}
			return nil
		})
	})
}

var update = flag.Bool("update", false, "rewrite schema.json from the fixtures")

func TestKnownSchema(t *testing.T) {
	eachRevision(t, func(t *testing.T, tg paneldb.Target) {
		paneldbtest.KnownSpec(t, tg, "schema.json", columns, inserts, *update)
	})
}

func inspectRevision(t *testing.T, tg paneldb.Target, stmts ...string) (Schema, error) {
	t.Helper()
	exec(t, tg, append([][]any{{"UPDATE alembic_version SET version_num = 'c9b48df42f10'"}}, toStmts(stmts)...)...)
	ctx := context.Background()
	r, err := paneldb.OpenReader(ctx, tg)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	var s Schema
	err = r.View(ctx, func(q paneldb.Querier) error {
		var err error
		s, err = Inspect(ctx, q, r.Server())
		return err
	})
	return s, err
}

func toStmts(stmts []string) [][]any {
	out := make([][]any, len(stmts))
	for i, s := range stmts {
		out[i] = []any{s}
	}
	return out
}

// A revision Karvan does not know is used when what Karvan reads and
// writes has the known shape. A new nullable column changes nothing.
func TestInspectChecksOtherRevisions(t *testing.T) {
	eachRevision(t, func(t *testing.T, tg paneldb.Target) {
		s, err := inspectRevision(t, tg, "ALTER TABLE users ADD COLUMN nickname VARCHAR(64)")
		if err != nil || !s.Checked || s.Revision != "c9b48df42f10" {
			t.Fatalf("schema = %+v, err = %v", s, err)
		}
	})
}

func TestInspectRefusesOtherRevisionsThatDiffer(t *testing.T) {
	eachRevision(t, func(t *testing.T, tg paneldb.Target) {
		_, err := inspectRevision(t, tg, "ALTER TABLE users DROP COLUMN note")
		var ve *panel.VersionError
		if !errors.As(err, &ve) || ve.Kind != panel.PasarGuard || !slices.Equal(ve.Diffs, []string{"users.note missing"}) {
			t.Fatalf("err = %v", err)
		}
	})
}

const proxySettings = `{"vmess":{"id":"3f2504e0-4f89-11d3-9a0c-0305e82c3301"},"vless":{"id":"3f2504e0-4f89-11d3-9a0c-0305e82c3302"},` +
	`"trojan":{"password":"tp"},"shadowsocks":{"password":"ssssssssssssssssssssss","method":"aes-128-gcm"}}`

func seedShop(t *testing.T, tg paneldb.Target) {
	t.Helper()
	now := paneldb.TimeArg(tg.Dialect, time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC))
	groups := paneldb.Ident(tg.Dialect, "groups")
	exec(t, tg,
		[]any{"INSERT INTO admins (id, username, hashed_password, created_at, used_traffic, role_id, status) VALUES (?, ?, ?, ?, ?, ?, ?)",
			10, "Shop", "$2b$12$x", now, 0, OperatorRole, "active"},
		[]any{"INSERT INTO users (id, username, status, used_traffic, created_at, admin_id, data_limit_reset_strategy, proxy_settings) VALUES (?, ?, ?, ?, ?, ?, ?, ?)",
			20, "Ali", "active", 7, now, 10, "no_reset", proxySettings},
		[]any{"INSERT INTO " + groups + " (id, name, is_disabled) VALUES (?, ?, ?)", 30, "Main", false},
		[]any{"INSERT INTO " + groups + " (id, name, is_disabled) VALUES (?, ?, ?)", 31, "Old", true},
		[]any{"INSERT INTO inbounds (id, tag) VALUES (?, ?)", 40, "VLESS TCP"},
		[]any{"INSERT INTO inbounds_groups_association (inbound_id, group_id) VALUES (?, ?)", 40, 30},
	)
}

func TestLoadIndex(t *testing.T) {
	eachRevision(t, func(t *testing.T, tg paneldb.Target) {
		seedShop(t, tg)
		view(t, tg, func(q paneldb.Querier, _ paneldb.Server) error {
			x, err := LoadIndex(context.Background(), q)
			if err != nil {
				return err
			}
			if x.Users["Ali"] == nil || x.Users["ali"] != nil || x.Users["Ali"].AdminID != 10 {
				t.Errorf("users: %+v", x.Users)
			}
			if x.Credentials["shadowsocks:ssssssssssssssssssssss"] != 20 || x.Credentials["vmess:3f2504e0-4f89-11d3-9a0c-0305e82c3301"] != 20 {
				t.Error("credentials not indexed")
			}
			return nil
		})
	})
}

func TestTargets(t *testing.T) {
	eachRevision(t, func(t *testing.T, tg paneldb.Target) {
		seedShop(t, tg)
		view(t, tg, func(q paneldb.Querier, _ paneldb.Server) error {
			targets, err := Targets(context.Background(), q, tg.Dialect)
			if err == nil && (len(targets) != 2 || targets[0].Name != "Main" || len(targets[0].Inbounds) != 1 || !targets[1].Disabled) {
				t.Errorf("targets: %+v", targets)
			}
			return err
		})
	})
}

// The default Xray core lists its inbounds; a WireGuard core is one
// inbound named by interface_name. Found on the running panel.
func TestInboundProtocols(t *testing.T) {
	eachRevision(t, func(t *testing.T, tg paneldb.Target) {
		now := paneldb.TimeArg(tg.Dialect, time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC))
		exec(t, tg, []any{"INSERT INTO core_configs (id, created_at, name, config, type) VALUES (?, ?, ?, ?, ?)",
			2, now, "wg", `{"interface_name": " wg0 ", "listen_port": 51820, "address": ["10.77.0.1/24"]}`, "wg"})
		view(t, tg, func(q paneldb.Querier, _ paneldb.Server) error {
			got, err := InboundProtocols(context.Background(), q)
			if err == nil && (len(got) != 2 || got["Shadowsocks TCP"] != "shadowsocks" || got["wg0"] != "wireguard") {
				t.Errorf("protocols: %v", got)
			}
			return err
		})
	})
}

func TestRolesAndAdmins(t *testing.T) {
	eachRevision(t, func(t *testing.T, tg paneldb.Target) {
		seedShop(t, tg)
		view(t, tg, func(q paneldb.Querier, _ paneldb.Server) error {
			ctx := context.Background()
			roles, err := Roles(ctx, q)
			if err != nil {
				return err
			}
			if len(roles) != 3 || !roles[0].IsOwner || roles[OperatorRole-1].Name != "operator" {
				t.Errorf("roles: %+v", roles)
			}
			admins, err := FindAdmins(ctx, q, []string{"Shop"})
			if a := admins["Shop"]; err == nil && (a.ID != 10 || a.IsOwner) {
				t.Errorf("admin: %+v", a)
			}
			return err
		})
	})
}

// passkeyTables stand in, on 5.4.1, for the tables PasarGuard 5.5 adds
// (revision 9a5c2e7f1b4d), keyed to admins with ON DELETE CASCADE. The
// 5.5.1 fixtures have the real ones, so these leave them alone.
var passkeyTables = []string{
	"CREATE TABLE IF NOT EXISTS admin_passkeys (id BIGINT NOT NULL PRIMARY KEY, admin_id BIGINT NOT NULL, " +
		"credential_id VARCHAR(64) NOT NULL, public_key VARCHAR(64) NOT NULL, name VARCHAR(128) NOT NULL, " +
		"FOREIGN KEY (admin_id) REFERENCES admins (id) ON DELETE CASCADE)",
	"CREATE TABLE IF NOT EXISTS passkey_challenges (id BIGINT NOT NULL PRIMARY KEY, challenge VARCHAR(64) NOT NULL, " +
		"admin_id BIGINT, kind VARCHAR(16) NOT NULL, expires_at TIMESTAMP NOT NULL, " +
		"FOREIGN KEY (admin_id) REFERENCES admins (id) ON DELETE CASCADE)",
}

// The passkey tables are PasarGuard's own, not an add-on's.
func TestInspectKnowsPasskeyTables(t *testing.T) {
	eachRevision(t, func(t *testing.T, tg paneldb.Target) {
		s, err := inspectRevision(t, tg, passkeyTables...)
		if err != nil || !s.Checked || len(s.UnknownChildren) > 0 {
			t.Fatalf("schema = %+v, err = %v", s, err)
		}
	})
}

// A table that is not PasarGuard's still stops undo.
func TestInspectReportsUnknownChildren(t *testing.T) {
	eachRevision(t, func(t *testing.T, tg paneldb.Target) {
		s, err := inspectRevision(t, tg, "CREATE TABLE addon_notes (id BIGINT NOT NULL PRIMARY KEY, admin_id BIGINT, "+
			"FOREIGN KEY (admin_id) REFERENCES admins (id))")
		u := s.UnknownChildren
		if err != nil || len(u) != 1 || u[0].Table != "addon_notes" || u[0].Column != "admin_id" || u[0].RefTable != "admins" {
			t.Fatalf("schema = %+v, err = %v", s, err)
		}
	})
}
