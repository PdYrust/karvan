package pasarguarddb

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/PdYrust/karvan/internal/panel"
	"github.com/PdYrust/karvan/internal/panel/paneldb"
	"github.com/PdYrust/karvan/internal/panel/pasarguard"
)

// Writer writes moved admins and users into a PasarGuard database at
// Revision, the way the panel's own crud code lays them out.
//
// WireGuard peer IPs are not allocated here. PasarGuard's allocator
// (app/db/crud/wireguard.py) works across cores, namespaces and reserved
// offsets under row locks; copying it would race the panel. Users keep
// their keys with no peer IP, and the panel assigns one on its own update
// path, which the reload step goes through (ADR 0005).
type Writer struct {
	Dialect paneldb.Dialect
}

// UserRow is one user to write.
type UserRow struct {
	User     panel.User
	AdminID  int64
	GroupIDs []int64
	UsageLog *UsageLog
}

// UsageLog keeps lifetime traffic and the reset schedule.
type UsageLog struct {
	Traffic int64
	ResetAt time.Time
}

// InsertAdmin creates the moved admin with roleID and returns its id.
func (w Writer) InsertAdmin(ctx context.Context, x paneldb.Execer, a panel.Admin, hash string, roleID int64, now time.Time) (int64, error) {
	var overrides any
	if a.Overrides != nil {
		b, err := json.Marshal(a.Overrides)
		if err != nil {
			return 0, err
		}
		overrides = string(b)
	}
	id, err := paneldb.InsertID(ctx, x, w.Dialect,
		"INSERT INTO admins (username, hashed_password, created_at, telegram_id, discord_webhook, used_traffic, note, role_id, status, "+
			"permission_overrides) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
		a.Username, hash, paneldb.TimeArg(w.Dialect, now), a.TelegramID, nullString(a.DiscordWebhook), a.UsedTraffic,
		nullString(a.Note), roleID, "active", overrides)
	if err != nil {
		return 0, fmt.Errorf("insert admin %s: %w", a.Username, err)
	}
	return id, nil
}

// InsertUser writes one user with its groups, next plan and usage log.
func (w Writer) InsertUser(ctx context.Context, x paneldb.Execer, r UserRow) (int64, error) {
	u, d := r.User, w.Dialect
	c := u.Credentials
	if c.WireGuard != nil {
		wg := *c.WireGuard
		wg.PeerIPs = nil
		c.WireGuard = &wg
	}
	settings, err := pasarguard.EncodeProxySettings(c)
	if err != nil {
		return 0, err
	}
	var admin any
	if r.AdminID > 0 {
		admin = r.AdminID
	}
	id, err := paneldb.InsertID(ctx, x, d,
		"INSERT INTO users (username, status, used_traffic, data_limit, created_at, admin_id, data_limit_reset_strategy, "+
			"note, online_at, on_hold_timeout, on_hold_expire_duration, auto_delete_in_days, expire, proxy_settings, hwid_limit) "+
			"VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
		u.Username, string(u.Status), u.UsedTraffic, nullPositive(u.DataLimit), paneldb.TimeArg(d, u.CreatedAt), admin,
		string(u.ResetStrategy), nullString(u.Note), paneldb.NullTimeArg(d, u.OnlineAt), paneldb.NullTimeArg(d, u.OnHoldTimeout),
		seconds(u.OnHoldExpire), u.AutoDeleteIn, paneldb.NullTimeArg(d, u.Expire), string(settings), u.HWIDLimit)
	if err != nil {
		return 0, fmt.Errorf("insert user %s: %w", u.Username, err)
	}
	for _, g := range r.GroupIDs {
		if _, err := x.ExecContext(ctx, "INSERT INTO users_groups_association (user_id, groups_id) VALUES (?, ?)", id, g); err != nil {
			return 0, fmt.Errorf("add %s to group %d: %w", u.Username, g, err)
		}
	}
	return id, w.insertExtras(ctx, x, id, r)
}

func (w Writer) insertExtras(ctx context.Context, x paneldb.Execer, userID int64, r UserRow) error {
	if np := r.User.NextPlan; np != nil {
		if _, err := x.ExecContext(ctx, "INSERT INTO next_plans (user_id, data_limit, expire, add_remaining_traffic) VALUES (?, ?, ?, ?)",
			userID, np.DataLimit, seconds(np.Expire), np.AddRemainingTraffic); err != nil {
			return fmt.Errorf("insert next plan of %s: %w", r.User.Username, err)
		}
	}
	if l := r.UsageLog; l != nil {
		if _, err := x.ExecContext(ctx, "INSERT INTO user_usage_logs (user_id, used_traffic_at_reset, reset_at) VALUES (?, ?, ?)",
			userID, l.Traffic, paneldb.TimeArg(w.Dialect, l.ResetAt)); err != nil {
			return fmt.Errorf("insert usage log of %s: %w", r.User.Username, err)
		}
	}
	return nil
}

// GroupIDs parses target ids (group ids as text).
func GroupIDs(targets []string) ([]int64, error) {
	out := make([]int64, 0, len(targets))
	for _, t := range targets {
		n, err := strconv.ParseInt(t, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("group id %q: %w", t, err)
		}
		out = append(out, n)
	}
	return out, nil
}

func nullPositive(n int64) any {
	if n <= 0 {
		return nil
	}
	return n
}

func nullString(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func seconds(d *time.Duration) any {
	if d == nil {
		return nil
	}
	return int64(d.Seconds())
}
