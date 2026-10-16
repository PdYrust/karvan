package marzbandb

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/PdYrust/karvan/internal/panel"
	"github.com/PdYrust/karvan/internal/panel/marzban"
	"github.com/PdYrust/karvan/internal/panel/paneldb"
)

// Writer writes moved admins and users into a Marzban database at
// Revision, the way the panel's own crud code lays them out.
type Writer struct {
	Dialect paneldb.Dialect
	// Inbounds are the destination's user inbound tags per protocol, from
	// its Xray config. A user's exclusions are these minus its targets.
	Inbounds map[string][]string
}

// NewWriter builds a writer from the destination's targets.
func NewWriter(d paneldb.Dialect, targets []panel.Target) Writer {
	w := Writer{Dialect: d, Inbounds: map[string][]string{}}
	for _, t := range targets {
		w.Inbounds[t.Protocol] = append(w.Inbounds[t.Protocol], t.Name)
	}
	return w
}

// UserRow is one user to write.
type UserRow struct {
	User    panel.User
	AdminID int64
	Targets []string // "<protocol>/<tag>"
	// UsageLog, when set, is the one user_usage_logs row (D19).
	UsageLog *UsageLog
}

// UsageLog keeps lifetime traffic and the reset schedule.
type UsageLog struct {
	Traffic int64
	ResetAt time.Time
}

// InsertAdmin creates the moved admin and returns its id.
func (w Writer) InsertAdmin(ctx context.Context, x paneldb.Execer, a panel.Admin, hash string, now time.Time) (int64, error) {
	var webhook any
	if a.DiscordWebhook != "" {
		webhook = a.DiscordWebhook
	}
	id, err := paneldb.InsertID(ctx, x, w.Dialect,
		"INSERT INTO admins (username, hashed_password, created_at, is_sudo, telegram_id, discord_webhook, users_usage) VALUES (?, ?, ?, ?, ?, ?, ?)",
		a.Username, hash, paneldb.TimeArg(w.Dialect, now), false, a.TelegramID, webhook, a.UsedTraffic)
	if err != nil {
		return 0, fmt.Errorf("insert admin %s: %w", a.Username, err)
	}
	return id, nil
}

// EnsureInbounds adds inbounds rows for tags that have none yet; Marzban's
// exclusions reference them, and the panel creates them lazily the same
// way (get_or_create_inbound, crud.py:62).
func (w Writer) EnsureInbounds(ctx context.Context, x paneldb.Execer) error {
	for _, proto := range marzban.Protocols {
		for _, tag := range w.Inbounds[proto] {
			if _, err := x.ExecContext(ctx, "INSERT INTO inbounds (tag) SELECT ? WHERE NOT EXISTS (SELECT 1 FROM inbounds WHERE tag = ?)", tag, tag); err != nil {
				return fmt.Errorf("add inbound %q: %w", tag, err)
			}
		}
	}
	return nil
}

// InsertUser writes one user with its proxies, exclusions, next plan and
// usage log, and returns its id.
func (w Writer) InsertUser(ctx context.Context, x paneldb.Execer, r UserRow, now time.Time) (int64, error) {
	u, d := r.User, w.Dialect
	id, err := paneldb.InsertID(ctx, x, d,
		"INSERT INTO users (username, status, used_traffic, data_limit, expire, created_at, admin_id, "+
			"data_limit_reset_strategy, note, sub_updated_at, sub_last_user_agent, online_at, on_hold_timeout, "+
			"on_hold_expire_duration, auto_delete_in_days, last_status_change) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
		u.Username, string(u.Status), u.UsedTraffic, nullPositive(u.DataLimit), unixOrNull(u.Expire),
		paneldb.TimeArg(d, u.CreatedAt), nullPositive(r.AdminID), string(u.ResetStrategy), nullString(u.Note),
		paneldb.NullTimeArg(d, u.SubUpdatedAt), nullString(u.SubUserAgent), paneldb.NullTimeArg(d, u.OnlineAt),
		paneldb.NullTimeArg(d, u.OnHoldTimeout), seconds(u.OnHoldExpire), u.AutoDeleteIn, paneldb.TimeArg(d, now))
	if err != nil {
		return 0, fmt.Errorf("insert user %s: %w", u.Username, err)
	}
	if err := w.insertProxies(ctx, x, id, r); err != nil {
		return 0, err
	}
	if err := w.insertExtras(ctx, x, id, r); err != nil {
		return 0, err
	}
	return id, nil
}

func (w Writer) insertProxies(ctx context.Context, x paneldb.Execer, userID int64, r UserRow) error {
	for _, proto := range marzban.Protocols {
		raw, err := marzban.ProxySettings(r.User.Credentials, proto)
		if err != nil {
			return err
		}
		if raw == nil {
			continue
		}
		pid, err := paneldb.InsertID(ctx, x, w.Dialect, "INSERT INTO proxies (user_id, type, settings) VALUES (?, ?, ?)",
			userID, marzban.TypeName(proto), string(raw))
		if err != nil {
			return fmt.Errorf("insert %s proxy of %s: %w", proto, r.User.Username, err)
		}
		for _, tag := range w.excluded(proto, r.Targets) {
			if _, err := x.ExecContext(ctx, "INSERT INTO exclude_inbounds_association (proxy_id, inbound_tag) VALUES (?, ?)", pid, tag); err != nil {
				return fmt.Errorf("exclude %q for %s: %w", tag, r.User.Username, err)
			}
		}
	}
	return nil
}

// excluded is every inbound of proto the user's targets do not name.
func (w Writer) excluded(proto string, targets []string) []string {
	var out []string
	for _, tag := range w.Inbounds[proto] {
		if !slices.Contains(targets, proto+"/"+tag) {
			out = append(out, tag)
		}
	}
	return out
}

func (w Writer) insertExtras(ctx context.Context, x paneldb.Execer, userID int64, r UserRow) error {
	if np := r.User.NextPlan; np != nil {
		if _, err := x.ExecContext(ctx, "INSERT INTO next_plans (user_id, data_limit, expire, add_remaining_traffic, fire_on_either) VALUES (?, ?, ?, ?, ?)",
			userID, np.DataLimit, seconds(np.Expire), np.AddRemainingTraffic, np.FireOnEither); err != nil {
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

func unixOrNull(t *time.Time) any {
	if t == nil {
		return nil
	}
	return t.Unix()
}

func seconds(d *time.Duration) any {
	if d == nil {
		return nil
	}
	return int64(d.Seconds())
}
