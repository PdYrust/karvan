package marzbandb

import (
	"context"
	"testing"
	"time"

	"github.com/PdYrust/karvan/internal/panel"
	"github.com/PdYrust/karvan/internal/panel/marzban"
	"github.com/PdYrust/karvan/internal/panel/paneldb"
	"github.com/PdYrust/karvan/internal/panel/paneldb/paneldbtest"
)

// seed writes one admin, two users and their proxies with plain SQL, the
// way the panel would have.
func seed(t *testing.T, tg paneldb.Target) {
	t.Helper()
	ctx := context.Background()
	w, err := paneldb.OpenWriter(ctx, tg)
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	d := w.Server().Dialect
	created := time.Date(2025, 3, 1, 10, 0, 0, 123456000, time.UTC)
	err = w.Batch(ctx, func(x paneldb.Execer) error {
		stmts := []struct {
			q    string
			args []any
		}{
			{"INSERT INTO admins (id, username, hashed_password, created_at, is_sudo, users_usage) VALUES (?, ?, ?, ?, ?, ?)",
				[]any{1, "Reseller", "$2b$12$x", paneldb.TimeArg(d, created), false, 0}},
			{"INSERT INTO users (id, username, status, used_traffic, created_at, admin_id, data_limit_reset_strategy) VALUES (?, ?, ?, ?, ?, ?, ?)",
				[]any{1, "ali", "active", 5, paneldb.TimeArg(d, created), 1, "no_reset"}},
			{"INSERT INTO users (id, username, status, used_traffic, created_at, data_limit_reset_strategy) VALUES (?, ?, ?, ?, ?, ?)",
				[]any{2, "orphan", "disabled", 0, paneldb.TimeArg(d, created), "no_reset"}},
			{"INSERT INTO proxies (user_id, type, settings) VALUES (?, ?, ?)",
				[]any{1, marzban.TypeName("vless"), `{"id": "3F2504E0-4F89-11D3-9A0C-0305E82C3301", "flow": ""}`}},
			{"INSERT INTO proxies (user_id, type, settings) VALUES (?, ?, ?)",
				[]any{2, marzban.TypeName("trojan"), `{"password": "shared", "flow": ""}`}},
		}
		for _, s := range stmts {
			if _, err := x.ExecContext(ctx, s.q, s.args...); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestLoadIndexAndFindAdmins(t *testing.T) {
	paneldbtest.Each(t, "testdata", Revision, func(t *testing.T, tg paneldb.Target) {
		seed(t, tg)
		ctx := context.Background()
		r, err := paneldb.OpenReader(ctx, tg)
		if err != nil {
			t.Fatal(err)
		}
		defer r.Close()
		err = r.View(ctx, func(q paneldb.Querier) error {
			s, err := Inspect(ctx, q, r.Server())
			if err != nil {
				return err
			}
			x, err := LoadIndex(ctx, q, marzban.NewRules(s.UsersFoldCase))
			if err != nil {
				return err
			}
			ali := x.Users["ali"]
			want := paneldb.Stored(r.Server().Dialect, time.Date(2025, 3, 1, 10, 0, 0, 123456000, time.UTC))
			if ali == nil || ali.AdminID != 1 || !ali.CreatedAt.Equal(want) {
				t.Errorf("ali = %+v", ali)
			}
			if x.Users["orphan"] == nil || x.Users["orphan"].AdminID != 0 {
				t.Error("user without admin not read")
			}
			if x.Credentials["vless:3f2504e0-4f89-11d3-9a0c-0305e82c3301"] != 1 {
				t.Error("UUID not canonicalized into the index")
			}
			if x.Credentials["trojan:shared"] != 2 {
				t.Error("trojan password not indexed")
			}
			admins, err := FindAdmins(ctx, q, []string{"reseller", "Reseller", "nobody"})
			if err != nil {
				return err
			}
			if _, ok := admins["Reseller"]; !ok {
				t.Error("exact admin name not found")
			}
			if _, ok := admins["reseller"]; ok != s.AdminsFoldCase {
				t.Errorf("lower-case lookup found=%v, collation folds=%v", ok, s.AdminsFoldCase)
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	})
}

var _ panel.Rules = marzban.Rules{}
