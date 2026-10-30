package marzbandb

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"maps"
	"slices"

	"github.com/PdYrust/karvan/internal/panel"
	"github.com/PdYrust/karvan/internal/panel/marzban"
	"github.com/PdYrust/karvan/internal/panel/paneldb"
)

// Written is a user as it reads back from the database.
type Written struct {
	panel.Row
	AdminID     int64
	UsedTraffic int64
	Credentials panel.Credentials
	Excluded    map[string][]string // protocol to excluded inbound tags
}

// readChunk bounds the ids one statement binds: SQLite takes 32766
// parameters, MySQL and PostgreSQL 65535, and a job may verify more users.
const readChunk = 1000

// ReadUsers reads ids back, for the verify stage and for settling.
func ReadUsers(ctx context.Context, q paneldb.Querier, d paneldb.Dialect, ids []int64) (map[int64]*Written, error) {
	out := map[int64]*Written{}
	if len(ids) > readChunk {
		for part := range slices.Chunk(ids, readChunk) {
			got, err := ReadUsers(ctx, q, d, part)
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
	rows, err := q.QueryContext(ctx, "SELECT id, username, created_at, admin_id, used_traffic FROM users WHERE id IN ("+in+")", args...)
	if err != nil {
		return nil, fmt.Errorf("read back users: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var (
			w       Written
			created paneldb.NullTime
			admin   sql.NullInt64
		)
		if err := rows.Scan(&w.ID, &w.Username, &created, &admin, &w.UsedTraffic); err != nil {
			return nil, err
		}
		w.CreatedAt, w.AdminID, w.Excluded = created.Time, admin.Int64, map[string][]string{}
		out[w.ID] = &w
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, readProxies(ctx, q, in, args, out)
}

func readProxies(ctx context.Context, q paneldb.Querier, in string, args []any, out map[int64]*Written) error {
	rows, err := q.QueryContext(ctx, "SELECT p.user_id, p.type, p.settings, e.inbound_tag FROM proxies p "+
		"LEFT JOIN exclude_inbounds_association e ON e.proxy_id = p.id WHERE p.user_id IN ("+in+") ORDER BY p.id, e.inbound_tag", args...)
	if err != nil {
		return fmt.Errorf("read back proxies: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var (
			uid      int64
			typeName string
			raw      []byte
			tag      sql.NullString
		)
		if err := rows.Scan(&uid, &typeName, &raw, &tag); err != nil {
			return err
		}
		w, proto := out[uid], ""
		if p, ok := marzban.ProtocolOf(typeName); ok && w != nil {
			proto = p
			if err := marzban.SetProxy(&w.Credentials, proto, raw); err != nil {
				return err
			}
		}
		if tag.Valid && proto != "" {
			w.Excluded[proto] = append(w.Excluded[proto], tag.String)
		}
	}
	return rows.Err()
}

// RemoveUsers deletes rows (and everything that hangs off them, children
// first) in the open transaction. A row whose id, username or created_at
// no longer match is left alone and reported. Parent rows are locked first,
// so the panel cannot add a child while this runs.
func RemoveUsers(ctx context.Context, x paneldb.Execer, d paneldb.Dialect, rows []panel.Row) (removed []int64, skipped []panel.Row, err error) {
	ids := make([]int64, len(rows))
	for i, r := range rows {
		ids[i] = r.ID
	}
	if err := paneldb.LockRows(ctx, x, d, "users", ids); err != nil {
		return nil, nil, err
	}
	have, err := ReadUsers(ctx, x, d, ids)
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
	stmts := []string{
		"DELETE FROM exclude_inbounds_association WHERE proxy_id IN (SELECT id FROM proxies WHERE user_id IN (" + in + "))",
		"DELETE FROM proxies WHERE user_id IN (" + in + ")",
		"DELETE FROM user_usage_logs WHERE user_id IN (" + in + ")",
		"DELETE FROM node_user_usages WHERE user_id IN (" + in + ")",
		"DELETE FROM notification_reminders WHERE user_id IN (" + in + ")",
		"DELETE FROM next_plans WHERE user_id IN (" + in + ")",
		"DELETE FROM users WHERE id IN (" + in + ")",
	}
	for _, s := range stmts {
		if _, err := x.ExecContext(ctx, s, args...); err != nil {
			return nil, nil, fmt.Errorf("remove users: %w", err)
		}
	}
	return removed, skipped, nil
}

// RemoveAdmin deletes an admin Karvan created, only if it owns no user
// any more. It reports whether it was removed.
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
	if err := x.QueryRowContext(ctx, "SELECT count(*) FROM users WHERE admin_id = ?", a.ID).Scan(&owned); err != nil {
		return false, err
	}
	if owned > 0 {
		return false, nil
	}
	for _, s := range []string{"DELETE FROM admin_usage_logs WHERE admin_id = ?", "DELETE FROM admins WHERE id = ?"} {
		if _, err := x.ExecContext(ctx, s, a.ID); err != nil {
			return false, fmt.Errorf("remove admin: %w", err)
		}
	}
	return true, nil
}
