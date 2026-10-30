package pasarguarddb

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"maps"
	"slices"

	"github.com/PdYrust/karvan/internal/panel"
	"github.com/PdYrust/karvan/internal/panel/paneldb"
	"github.com/PdYrust/karvan/internal/panel/pasarguard"
)

// Written is a user as it reads back from the database.
type Written struct {
	panel.Row
	AdminID     int64
	UsedTraffic int64
	Credentials panel.Credentials
	GroupIDs    []int64
}

// readChunk bounds the ids one statement binds: SQLite takes 32766
// parameters, MySQL and PostgreSQL 65535, and a job may verify more users.
const readChunk = 1000

// ReadUsers reads ids back, for the verify stage and for settling.
func ReadUsers(ctx context.Context, q paneldb.Querier, ids []int64) (map[int64]*Written, error) {
	out := map[int64]*Written{}
	if len(ids) > readChunk {
		for part := range slices.Chunk(ids, readChunk) {
			got, err := ReadUsers(ctx, q, part)
			if err != nil {
				return nil, err
			}
			maps.Copy(out, got)
		}
		return out, nil
	}
	if len(ids) == 0 {
		return out, nil
	}
	in, args := paneldb.In(ids)
	rows, err := q.QueryContext(ctx, "SELECT id, username, created_at, admin_id, used_traffic, proxy_settings FROM users WHERE id IN ("+in+")", args...)
	if err != nil {
		return nil, fmt.Errorf("read back users: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var (
			w       Written
			created paneldb.NullTime
			admin   sql.NullInt64
			raw     []byte
		)
		if err := rows.Scan(&w.ID, &w.Username, &created, &admin, &w.UsedTraffic, &raw); err != nil {
			return nil, err
		}
		w.CreatedAt, w.AdminID = created.Time, admin.Int64
		if w.Credentials, err = pasarguard.DecodeProxySettings(raw); err != nil {
			return nil, err
		}
		out[w.ID] = &w
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, readGroups(ctx, q, in, args, out)
}

func readGroups(ctx context.Context, q paneldb.Querier, in string, args []any, out map[int64]*Written) error {
	rows, err := q.QueryContext(ctx, "SELECT user_id, groups_id FROM users_groups_association WHERE user_id IN ("+in+") ORDER BY groups_id", args...)
	if err != nil {
		return fmt.Errorf("read back groups: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var uid, gid int64
		if err := rows.Scan(&uid, &gid); err != nil {
			return err
		}
		if w := out[uid]; w != nil {
			w.GroupIDs = append(w.GroupIDs, gid)
		}
	}
	return rows.Err()
}

// RemoveUsers deletes rows and everything hanging off them, children
// first, in the open transaction, skipping rows whose identity changed.
func RemoveUsers(ctx context.Context, x paneldb.Execer, d paneldb.Dialect, rows []panel.Row) (removed []int64, skipped []panel.Row, err error) {
	ids := make([]int64, len(rows))
	for i, r := range rows {
		ids[i] = r.ID
	}
	if err := paneldb.LockRows(ctx, x, d, "users", ids); err != nil {
		return nil, nil, err
	}
	have, err := ReadUsers(ctx, x, ids)
	if err != nil {
		return nil, nil, err
	}
	for _, r := range rows {
		w, ok := have[r.ID]
		if ok && w.Username == r.Username && w.CreatedAt.Equal(paneldb.Stored(d, r.CreatedAt)) {
			removed = append(removed, r.ID)
		} else {
			skipped = append(skipped, r)
		}
	}
	if len(removed) == 0 {
		return nil, skipped, nil
	}
	in, args := paneldb.In(removed)
	for _, c := range children {
		if c.RefTable != "users" {
			continue
		}
		if _, err := x.ExecContext(ctx, "DELETE FROM "+c.Table+" WHERE "+c.Column+" IN ("+in+")", args...); err != nil {
			return nil, nil, fmt.Errorf("remove users from %s: %w", c.Table, err)
		}
	}
	if _, err := x.ExecContext(ctx, "DELETE FROM users WHERE id IN ("+in+")", args...); err != nil {
		return nil, nil, fmt.Errorf("remove users: %w", err)
	}
	return removed, skipped, nil
}

// RemoveAdmin deletes an admin Karvan created, only if it owns no user
// and no API key (someone made one meanwhile). Its passkeys and passkey
// challenges, on revisions that have them, go with it by ON DELETE CASCADE.
func RemoveAdmin(ctx context.Context, x paneldb.Execer, d paneldb.Dialect, a panel.Row) (bool, error) {
	var name string
	var created paneldb.NullTime
	err := x.QueryRowContext(ctx, "SELECT username, created_at FROM admins WHERE id = ?", a.ID).Scan(&name, &created)
	if errors.Is(err, sql.ErrNoRows) || err == nil && (name != a.Username || !created.Time.Equal(paneldb.Stored(d, a.CreatedAt))) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	var owned int
	if err := x.QueryRowContext(ctx, "SELECT (SELECT count(*) FROM users WHERE admin_id = ?) + (SELECT count(*) FROM api_keys WHERE admin_id = ?)", a.ID, a.ID).Scan(&owned); err != nil {
		return false, err
	}
	if owned > 0 {
		return false, nil
	}
	for _, s := range []string{"DELETE FROM admin_usage_logs WHERE admin_id = ?", "DELETE FROM admin_notification_reminders WHERE admin_id = ?", "DELETE FROM admins WHERE id = ?"} {
		if _, err := x.ExecContext(ctx, s, a.ID); err != nil {
			return false, fmt.Errorf("remove admin: %w", err)
		}
	}
	return true, nil
}
