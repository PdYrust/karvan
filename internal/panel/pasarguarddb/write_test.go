package pasarguarddb

import (
	"context"
	"testing"
	"time"

	"github.com/PdYrust/karvan/internal/panel"
	"github.com/PdYrust/karvan/internal/panel/paneldb"
	"github.com/PdYrust/karvan/internal/panel/paneldb/paneldbtest"
	"github.com/PdYrust/karvan/internal/secret"
)

func movedUser() panel.User {
	exp := time.Date(2027, 2, 3, 4, 5, 6, 0, time.UTC)
	hwid := 2
	return panel.User{
		Username: "Moved.User", Status: panel.StatusDisabled, ResetStrategy: panel.ResetWeek, UsedTraffic: 42, DataLimit: 1 << 30,
		Expire: &exp, CreatedAt: time.Date(2025, 5, 6, 7, 8, 9, 123456000, time.UTC), HWIDLimit: &hwid, Note: "vip",
		NextPlan: &panel.NextPlan{DataLimit: 2 << 30},
		Credentials: panel.Credentials{
			VMess:       &panel.VMess{ID: "3f2504e0-4f89-11d3-9a0c-0305e82c3301"},
			VLESS:       &panel.VLESS{ID: "3f2504e0-4f89-11d3-9a0c-0305e82c3302"},
			Trojan:      &panel.Trojan{Password: secret.NewValue(`tp "quoted" \ back`)},
			Shadowsocks: &panel.Shadowsocks{Password: secret.NewValue("a-shadowsocks-password-22+"), Method: "xchacha20-poly1305"},
			Hysteria:    &panel.Hysteria{Auth: secret.NewValue("hy")},
			WireGuard:   &panel.WireGuard{PrivateKey: secret.NewValue("priv"), PublicKey: "pub", PeerIPs: []string{"10.8.0.9/32"}},
		},
	}
}

func checkWritten(t *testing.T, got *Written, u panel.User, adminID int64, d paneldb.Dialect) {
	t.Helper()
	if got == nil || got.Username != u.Username || got.AdminID != adminID || len(got.GroupIDs) != 1 || !got.CreatedAt.Equal(paneldb.Stored(d, u.CreatedAt)) {
		t.Fatalf("read back: %+v", got)
	}
	c, want := got.Credentials, u.Credentials
	same := c.Trojan.Password.Reveal() == want.Trojan.Password.Reveal() && c.Shadowsocks.Method == want.Shadowsocks.Method &&
		c.VLESS.ID == want.VLESS.ID && c.Hysteria.Auth.Reveal() == want.Hysteria.Auth.Reveal() && c.WireGuard.PublicKey == want.WireGuard.PublicKey
	if !same || len(c.WireGuard.PeerIPs) != 0 {
		t.Fatalf("credentials changed on the way: %+v", c)
	}
}

func remove(t *testing.T, db *paneldb.Writer, u panel.User, userID, adminID int64, now time.Time) {
	t.Helper()
	ctx, d := context.Background(), db.Server().Dialect
	err := db.Chunk(ctx, func(x paneldb.Execer) error {
		removed, _, err := RemoveUsers(ctx, x, d, []panel.Row{{ID: userID, Username: u.Username, CreatedAt: u.CreatedAt}})
		if err != nil {
			return err
		}
		if len(removed) != 1 {
			t.Errorf("removed %v", removed)
		}
		ok, err := RemoveAdmin(ctx, x, d, panel.Row{ID: adminID, Username: "newshop", CreatedAt: now})
		if !ok {
			t.Error("admin not removed")
		}
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestWriteReadBackAndRemove(t *testing.T) {
	paneldbtest.Each(t, "testdata", Revision, func(t *testing.T, tg paneldb.Target) {
		seedShop(t, tg)
		clean := paneldbtest.Checksum(t, tg)
		ctx := context.Background()
		db, err := paneldb.OpenWriter(ctx, tg)
		if err != nil {
			t.Fatal(err)
		}
		defer db.Close()
		d := db.Server().Dialect
		w := Writer{Dialect: d}
		now, u := time.Now().UTC(), movedUser()
		var adminID, userID int64
		err = db.Chunk(ctx, func(x paneldb.Execer) error {
			var err error
			if adminID, err = w.InsertAdmin(ctx, x, panel.Admin{Username: "newshop", UsedTraffic: 9}, "$2a$12$h", OperatorRole, now); err != nil {
				return err
			}
			userID, err = w.InsertUser(ctx, x, UserRow{User: u, AdminID: adminID, GroupIDs: []int64{30}, UsageLog: &UsageLog{Traffic: 7, ResetAt: now}})
			return err
		})
		if err != nil {
			t.Fatal(err)
		}
		var got *Written
		view(t, tg, func(q paneldb.Querier, _ paneldb.Server) error {
			back, err := ReadUsers(ctx, q, []int64{userID})
			got = back[userID]
			return err
		})
		checkWritten(t, got, u, adminID, d)
		remove(t, db, u, userID, adminID, now)
		if paneldbtest.Checksum(t, tg) != clean {
			t.Fatal("removing what was written did not restore the database")
		}
	})
}

func TestReadUsersTakesMoreIDsThanAStatementBinds(t *testing.T) {
	paneldbtest.Each(t, "testdata", Revision, func(t *testing.T, tg paneldb.Target) {
		ids := make([]int64, 70_000) // more than SQLite (32766), MySQL and PostgreSQL (65535) bind at once
		for i := range ids {
			ids[i] = int64(1_000_000 + i)
		}
		view(t, tg, func(q paneldb.Querier, _ paneldb.Server) error {
			back, err := ReadUsers(context.Background(), q, ids)
			if err == nil && len(back) != 0 {
				t.Fatalf("read back %d users that do not exist", len(back))
			}
			return err
		})
	})
}
