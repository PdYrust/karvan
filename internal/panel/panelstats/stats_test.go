package panelstats_test

import (
	"context"
	"testing"
	"time"

	"github.com/PdYrust/karvan/internal/panel/marzbandb"
	"github.com/PdYrust/karvan/internal/panel/paneldb"
	"github.com/PdYrust/karvan/internal/panel/panelstats"
	"github.com/PdYrust/karvan/internal/panel/pasarguarddb"
)

type kind struct {
	name, dir, rev string
	schema         panelstats.Schema
}

var kinds = []kind{
	{"marzban", "../marzbandb/testdata", marzbandb.Revision, marzbandb.Stats},
	{"pasarguard", "../pasarguarddb/testdata", pasarguarddb.Revision, pasarguarddb.Stats},
}

var now = time.Date(2026, 10, 17, 12, 0, 0, 0, time.UTC)

type user struct {
	name, status, agent string
	admin               int // index into the admins
	updates             int // PasarGuard subscription updates
	used                int64
	created             time.Duration // before now
	online, expire      *time.Duration
	subAt               time.Duration
}

func ago(d time.Duration) *time.Duration { return &d }

var users = []user{
	{name: "u1", status: "active", admin: 0, used: 100, created: 2 * time.Hour, online: ago(time.Minute), expire: ago(-3 * 24 * time.Hour), agent: "v2rayNG/1.8.5", subAt: time.Hour, updates: 2},
	{name: "u2", status: "active", admin: 0, used: 200, created: 10 * 24 * time.Hour, online: ago(3 * time.Hour), expire: ago(-30 * 24 * time.Hour), agent: "Hiddify/2.0", subAt: 2 * 24 * time.Hour, updates: 1},
	{name: "u3", status: "disabled", admin: 1, used: 50, created: time.Hour},
	{name: "u4", status: "expired", admin: 1, created: 40 * 24 * time.Hour, online: ago(30 * 24 * time.Hour), expire: ago(24 * time.Hour)},
	{name: "u5", status: "on_hold", admin: 0, created: 3 * 24 * time.Hour},
}

// seed writes two admins, the users above, a node, its hourly usage and
// the panel's totals, the way each panel stores them.
func seed(t *testing.T, tg paneldb.Target, k kind) {
	t.Helper()
	ctx := context.Background()
	w, err := paneldb.OpenWriter(ctx, tg)
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	d := tg.Dialect
	at := func(before time.Duration) any { return paneldb.TimeArg(d, now.Add(-before)) }
	err = w.Chunk(ctx, func(x paneldb.Execer) error {
		admins := make([]int64, 2)
		for i, name := range []string{"reseller1", "reseller2"} {
			q := "INSERT INTO admins (username, hashed_password, created_at) VALUES (?, 'x', ?)"
			if k.schema.Updates {
				q = "INSERT INTO admins (username, hashed_password, created_at, role_id) VALUES (?, 'x', ?, 3)"
			}
			if admins[i], err = paneldb.InsertID(ctx, x, d, q, name, at(0)); err != nil {
				return err
			}
		}
		for _, u := range users {
			if err := insertUser(ctx, x, d, k, u, admins[u.admin]); err != nil {
				return err
			}
		}
		node, err := paneldb.InsertID(ctx, x, d, "INSERT INTO nodes (name, address, port, api_port, status, created_at, uplink, downlink) VALUES ('de-1', '10.0.0.2', 62050, 62051, 'connected', ?, 40, 60)", at(0))
		if err != nil {
			return err
		}
		for _, r := range []struct {
			before   time.Duration
			node     any
			up, down int64
		}{{3 * time.Hour, node, 5, 10}, {2 * time.Hour, node, 7, 3}, {2 * time.Hour, nil, 1, 1}, {48 * time.Hour, node, 99, 99}} {
			if _, err := x.ExecContext(ctx, "INSERT INTO node_usages (created_at, node_id, uplink, downlink) VALUES (?, ?, ?, ?)", at(r.before), r.node, r.up, r.down); err != nil {
				return err
			}
		}
		_, err = x.ExecContext(ctx, "UPDATE "+paneldb.Ident(d, "system")+" SET uplink = 10, downlink = 20")
		return err
	})
	if err != nil {
		t.Fatalf("seed: %v", err)
	}
}

func insertUser(ctx context.Context, x paneldb.Execer, d paneldb.Dialect, k kind, u user, admin int64) error {
	at := func(before *time.Duration) any {
		if before == nil {
			return nil
		}
		return paneldb.TimeArg(d, now.Add(-*before))
	}
	expire := at(u.expire)
	if k.schema.ExpireUnix && u.expire != nil {
		expire = now.Add(-*u.expire).Unix()
	}
	// The status is a literal: PostgreSQL turns it into its enum type.
	cols := "username, status, used_traffic, created_at, admin_id, online_at, expire"
	vals := "?, '" + u.status + "', ?, ?, ?, ?, ?"
	args := []any{u.name, u.used, at(&u.created), admin, at(u.online), expire}
	if k.schema.Updates {
		// MySQL cannot give a JSON column a default.
		cols += ", proxy_settings"
		vals += ", '{}'"
	}
	if !k.schema.Updates && u.agent != "" {
		cols += ", sub_last_user_agent, sub_updated_at"
		vals += ", ?, ?"
		args = append(args, u.agent, at(&u.subAt))
	}
	id, err := paneldb.InsertID(ctx, x, d, "INSERT INTO users ("+cols+") VALUES ("+vals+")", args...)
	if err != nil || !k.schema.Updates || u.agent == "" {
		return err
	}
	// PasarGuard keeps every update, not only the last one.
	for i := range u.updates {
		if _, err := x.ExecContext(ctx, "INSERT INTO user_subscription_updates (user_id, created_at, user_agent) VALUES (?, ?, ?)",
			id, at(ago(u.subAt+time.Duration(i)*time.Minute)), u.agent); err != nil {
			return err
		}
	}
	return nil
}

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
