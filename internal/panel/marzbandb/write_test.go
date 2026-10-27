package marzbandb

import (
	"context"
	"slices"
	"testing"
	"time"

	"github.com/PdYrust/karvan/internal/panel"
	"github.com/PdYrust/karvan/internal/panel/paneldb"
	"github.com/PdYrust/karvan/internal/panel/paneldb/paneldbtest"
	"github.com/PdYrust/karvan/internal/secret"
)

var targets = []panel.Target{
	{ID: "vless/VLESS TCP", Name: "VLESS TCP", Protocol: "vless"},
	{ID: "vless/VLESS WS", Name: "VLESS WS", Protocol: "vless"},
	{ID: "vmess/VMess WS", Name: "VMess WS", Protocol: "vmess"},
}

func movedUser(name string) panel.User {
	exp := time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC)
	online := time.Date(2026, 10, 1, 9, 30, 0, 0, time.UTC)
	week := 7 * 24 * time.Hour
	days := 30
	return panel.User{
		Username: name, Status: panel.StatusActive, ResetStrategy: panel.ResetMonth,
		UsedTraffic: 123456789, DataLimit: 10 << 30, Expire: &exp, Note: "note",
		CreatedAt: time.Date(2025, 5, 6, 7, 8, 9, 654321000, time.UTC), OnlineAt: &online, AutoDeleteIn: &days,
		NextPlan: &panel.NextPlan{DataLimit: 5 << 30, Expire: &week, AddRemainingTraffic: true, FireOnEither: true},
		Credentials: panel.Credentials{
			VMess:       &panel.VMess{ID: "3f2504e0-4f89-11d3-9a0c-0305e82c3301"},
			VLESS:       &panel.VLESS{ID: "3f2504e0-4f89-11d3-9a0c-0305e82c3302", Flow: "xtls-rprx-vision"},
			Trojan:      &panel.Trojan{Password: secret.NewValue("trojan secret with spaces & ünïcode")},
			Shadowsocks: &panel.Shadowsocks{Password: secret.NewValue("ss"), Method: "chacha20-ietf-poly1305"},
		},
	}
}

func chunk(t *testing.T, w *paneldb.Writer, fn func(x paneldb.Execer) error) {
	t.Helper()
	if err := w.Chunk(context.Background(), fn); err != nil {
		t.Fatal(err)
	}
}

// writeOne writes the admin and a user, plus a row the panel itself adds.
func writeOne(t *testing.T, db *paneldb.Writer, w Writer, u panel.User, now time.Time) (adminID, userID int64) {
	t.Helper()
	ctx, d := context.Background(), w.Dialect
	chunk(t, db, func(x paneldb.Execer) error {
		var err error
		if adminID, err = w.InsertAdmin(ctx, x, panel.Admin{Username: "shop", UsedTraffic: 5}, "$2a$12$hash", now); err != nil {
			return err
		}
		userID, err = w.InsertUser(ctx, x, UserRow{User: u, AdminID: adminID, Targets: []string{"vless/VLESS TCP"},
			UsageLog: &UsageLog{Traffic: 999, ResetAt: now}}, now)
		if err != nil {
			return err
		}
		_, err = x.ExecContext(ctx, "INSERT INTO notification_reminders (user_id, type, expires_at, created_at) VALUES (?, ?, ?, ?)",
			userID, "expiration_date", paneldb.TimeArg(d, now), paneldb.TimeArg(d, now))
		return err
	})
	return adminID, userID
}

func checkWritten(t *testing.T, got *Written, u panel.User, adminID int64, d paneldb.Dialect) {
	t.Helper()
	if got == nil || got.Username != u.Username || got.UsedTraffic != u.UsedTraffic || got.AdminID != adminID ||
		!got.CreatedAt.Equal(paneldb.Stored(d, u.CreatedAt)) {
		t.Fatalf("read back: %+v", got)
	}
	c, want := got.Credentials, u.Credentials
	same := c.VMess.ID == want.VMess.ID && c.VLESS.ID == want.VLESS.ID && c.VLESS.Flow == want.VLESS.Flow &&
		c.Trojan.Password.Reveal() == want.Trojan.Password.Reveal() && c.Shadowsocks.Method == want.Shadowsocks.Method
	if !same {
		t.Fatalf("credentials changed on the way: %+v", c)
	}
	if ex := got.Excluded["vless"]; len(ex) != 1 || ex[0] != "VLESS WS" || len(got.Excluded["vmess"]) != 1 {
		t.Fatalf("exclusions: %v", got.Excluded)
	}
}

func TestWriteReadBackAndRemove(t *testing.T) {
	paneldbtest.Each(t, "testdata", Revision, func(t *testing.T, tg paneldb.Target) {
		ctx := context.Background()
		db, err := paneldb.OpenWriter(ctx, tg)
		if err != nil {
			t.Fatal(err)
		}
		defer db.Close()
		d := db.Server().Dialect
		w := NewWriter(d, targets)
		chunk(t, db, func(x paneldb.Execer) error { return w.EnsureInbounds(ctx, x) })
		clean := paneldbtest.Checksum(t, tg)

		now, u := time.Now().UTC(), movedUser("moved_user")
		adminID, userID := writeOne(t, db, w, u, now)
		var got *Written
		chunk(t, db, func(x paneldb.Execer) error {
			back, err := ReadUsers(ctx, x, d, []int64{userID})
			got = back[userID]
			return err
		})
		checkWritten(t, got, u, adminID, d)

		chunk(t, db, func(x paneldb.Execer) error {
			removed, skipped, err := RemoveUsers(ctx, x, d, []panel.Row{{ID: userID, Username: u.Username, CreatedAt: u.CreatedAt}, {ID: 999999, Username: "x"}})
			if err != nil {
				return err
			}
			if len(removed) != 1 || len(skipped) != 1 {
				t.Errorf("removed %v skipped %v", removed, skipped)
			}
			ok, err := RemoveAdmin(ctx, x, d, panel.Row{ID: adminID, Username: "shop", CreatedAt: now})
			if !ok {
				t.Error("admin not removed")
			}
			return err
		})
		if paneldbtest.Checksum(t, tg) != clean {
			t.Fatal("removing what was written did not restore the database")
		}
	})
}

func TestRemoveLeavesChangedRows(t *testing.T) {
	tg := paneldbtest.Load(t, paneldbtest.Fixture{Flavor: "sqlite", Path: "testdata/" + Revision + ".sqlite.sql"})
	ctx := context.Background()
	db, err := paneldb.OpenWriter(ctx, tg)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	w := NewWriter(paneldb.SQLite, targets)
	u := movedUser("someone")
	var id int64
	chunk(t, db, func(x paneldb.Execer) error {
		if err := w.EnsureInbounds(ctx, x); err != nil {
			return err
		}
		var err error
		id, err = w.InsertUser(ctx, x, UserRow{User: u}, time.Now())
		return err
	})
	// An admin renames the user on the panel before a rollback.
	chunk(t, db, func(x paneldb.Execer) error {
		_, err := x.ExecContext(ctx, "UPDATE users SET username = 'renamed' WHERE id = ?", id)
		return err
	})
	chunk(t, db, func(x paneldb.Execer) error {
		removed, skipped, err := RemoveUsers(ctx, x, paneldb.SQLite, []panel.Row{{ID: id, Username: "someone", CreatedAt: u.CreatedAt}})
		if err == nil && (len(removed) != 0 || len(skipped) != 1) {
			t.Errorf("a changed row must be left alone: removed %v", removed)
		}
		return err
	})
}

// Like get_or_create_inbound: a new inbound row comes with its default
// host, or Marzban would give that inbound no links; an existing row
// gets nothing, and running twice adds nothing.
func TestEnsureInboundsAddsTheDefaultHost(t *testing.T) {
	paneldbtest.Each(t, "testdata", Revision, func(t *testing.T, tg paneldb.Target) {
		ctx := context.Background()
		db, err := paneldb.OpenWriter(ctx, tg)
		if err != nil {
			t.Fatal(err)
		}
		defer db.Close()
		chunk(t, db, func(x paneldb.Execer) error {
			_, err := x.ExecContext(ctx, "INSERT INTO inbounds (tag) VALUES (?)", "VMess WS")
			return err
		})
		w := NewWriter(db.Server().Dialect, targets)
		for range 2 {
			chunk(t, db, func(x paneldb.Execer) error { return w.EnsureInbounds(ctx, x) })
		}
		chunk(t, db, func(x paneldb.Execer) error {
			rows, err := x.QueryContext(ctx, "SELECT inbound_tag, remark, address, alpn FROM hosts ORDER BY inbound_tag")
			if err != nil {
				return err
			}
			defer rows.Close()
			var got []string
			for rows.Next() {
				var tag, remark, addr, alpn string
				if err := rows.Scan(&tag, &remark, &addr, &alpn); err != nil {
					return err
				}
				if remark != DefaultHostRemark || addr != "{SERVER_IP}" || alpn != "none" {
					t.Errorf("host %s: %q %q %q", tag, remark, addr, alpn)
				}
				got = append(got, tag)
			}
			if want := []string{"VLESS TCP", "VLESS WS"}; !slices.Equal(got, want) {
				t.Errorf("hosts for %v, want %v", got, want)
			}
			return rows.Err()
		})
	})
}
