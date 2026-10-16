package marzbandb

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/PdYrust/karvan/internal/panel"
	"github.com/PdYrust/karvan/internal/panel/marzban"
	"github.com/PdYrust/karvan/internal/panel/paneldb"
)

// LoadIndex reads every user and credential on the destination. Keys follow
// rules, so lookups use the destination's collation.
func LoadIndex(ctx context.Context, q paneldb.Querier, rules panel.Rules) (*panel.Index, error) {
	x := panel.NewIndex()
	rows, err := q.QueryContext(ctx, "SELECT id, username, admin_id, created_at, sub_revoked_at FROM users")
	if err != nil {
		return nil, fmt.Errorf("read users: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var (
			u         panel.ExistingUser
			admin     sql.NullInt64
			created   paneldb.NullTime
			revokedAt paneldb.NullTime
		)
		if err := rows.Scan(&u.ID, &u.Username, &admin, &created, &revokedAt); err != nil {
			return nil, fmt.Errorf("read users: %w", err)
		}
		u.AdminID, u.CreatedAt, u.SubRevokedAt = admin.Int64, created.Time, revokedAt.Ptr()
		x.Add(rules.UserKey(u.Username), &u)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read users: %w", err)
	}
	return x, loadCredentials(ctx, q, x)
}

func loadCredentials(ctx context.Context, q paneldb.Querier, x *panel.Index) error {
	rows, err := q.QueryContext(ctx, "SELECT user_id, type, settings FROM proxies ORDER BY id")
	if err != nil {
		return fmt.Errorf("read proxies: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var (
			userID   int64
			typeName string
			raw      []byte
		)
		if err := rows.Scan(&userID, &typeName, &raw); err != nil {
			return fmt.Errorf("read proxies: %w", err)
		}
		proto, ok := marzban.ProtocolOf(typeName)
		if !ok {
			continue
		}
		var c panel.Credentials
		if err := marzban.SetProxy(&c, proto, raw); err != nil {
			return fmt.Errorf("read proxies of user %d: %w", userID, err)
		}
		x.AddCredentials(c, userID)
	}
	return rows.Err()
}

// FindAdmins returns the destination admins whose username equals one of
// names under the column's own collation, keyed by the name asked for.
// The database compares, because MySQL collations also fold accents.
func FindAdmins(ctx context.Context, q paneldb.Querier, names []string) (map[string]panel.ExistingAdmin, error) {
	out := map[string]panel.ExistingAdmin{}
	for _, n := range names {
		var a panel.ExistingAdmin
		err := q.QueryRowContext(ctx, "SELECT id, username, is_sudo FROM admins WHERE username = ?", n).
			Scan(&a.ID, &a.Username, &a.IsOwner)
		switch {
		case errors.Is(err, sql.ErrNoRows):
			continue
		case err != nil:
			return nil, fmt.Errorf("find admin: %w", err)
		}
		out[n] = a
	}
	return out, nil
}
