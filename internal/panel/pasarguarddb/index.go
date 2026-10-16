package pasarguarddb

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/PdYrust/karvan/internal/panel"
	"github.com/PdYrust/karvan/internal/panel/paneldb"
	"github.com/PdYrust/karvan/internal/panel/pasarguard"
)

// LoadIndex reads every user and credential on the destination.
func LoadIndex(ctx context.Context, q paneldb.Querier) (*panel.Index, error) {
	x := panel.NewIndex()
	rows, err := q.QueryContext(ctx, "SELECT id, username, admin_id, created_at, sub_revoked_at, proxy_settings FROM users ORDER BY id")
	if err != nil {
		return nil, fmt.Errorf("read users: %w", err)
	}
	defer rows.Close()
	rules := pasarguard.Rules{}
	for rows.Next() {
		var (
			u         panel.ExistingUser
			admin     sql.NullInt64
			created   paneldb.NullTime
			revokedAt paneldb.NullTime
			raw       []byte
		)
		if err := rows.Scan(&u.ID, &u.Username, &admin, &created, &revokedAt, &raw); err != nil {
			return nil, fmt.Errorf("read users: %w", err)
		}
		u.AdminID, u.CreatedAt, u.SubRevokedAt = admin.Int64, created.Time, revokedAt.Ptr()
		c, err := pasarguard.DecodeProxySettings(raw)
		if err != nil {
			return nil, fmt.Errorf("read user %d: %w", u.ID, err)
		}
		x.Add(rules.UserKey(u.Username), &u)
		x.AddCredentials(c, u.ID)
	}
	return x, rows.Err()
}

// FindAdmins returns the destination admins whose username equals one of
// names under the column's own collation, keyed by the name asked for.
// IsOwner comes from the admin's role.
func FindAdmins(ctx context.Context, q paneldb.Querier, names []string) (map[string]panel.ExistingAdmin, error) {
	out := map[string]panel.ExistingAdmin{}
	for _, n := range names {
		var a panel.ExistingAdmin
		err := q.QueryRowContext(ctx, "SELECT a.id, a.username, r.is_owner FROM admins a "+
			"JOIN admin_roles r ON r.id = a.role_id WHERE a.username = ?", n).Scan(&a.ID, &a.Username, &a.IsOwner)
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

// Role is an admin role the operator can give the moved admin.
type Role struct {
	ID      int64
	Name    string
	IsOwner bool
}

// OperatorRole is the built-in reseller role (scope OWN), the default for
// a moved admin.
const OperatorRole = 3

// Roles lists the destination's admin roles.
func Roles(ctx context.Context, q paneldb.Querier) ([]Role, error) {
	rows, err := q.QueryContext(ctx, "SELECT id, name, is_owner FROM admin_roles ORDER BY id")
	if err != nil {
		return nil, fmt.Errorf("read admin roles: %w", err)
	}
	defer rows.Close()
	var out []Role
	for rows.Next() {
		var r Role
		if err := rows.Scan(&r.ID, &r.Name, &r.IsOwner); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// InboundProtocols maps every inbound tag in the panel's core configs to
// its protocol, so Karvan knows which protocols a group serves. An Xray
// core lists its inbounds; a WireGuard core is one inbound named by its
// interface_name (app/core/wireguard.py).
func InboundProtocols(ctx context.Context, q paneldb.Querier) (map[string]string, error) {
	rows, err := q.QueryContext(ctx, "SELECT type, config FROM core_configs ORDER BY id")
	if err != nil {
		return nil, fmt.Errorf("read core configs: %w", err)
	}
	defer rows.Close()
	out := map[string]string{}
	for rows.Next() {
		var kind string
		var raw []byte
		if err := rows.Scan(&kind, &raw); err != nil {
			return nil, err
		}
		var cfg struct {
			Interface string `json:"interface_name"`
			Inbounds  []struct {
				Tag      string `json:"tag"`
				Protocol string `json:"protocol"`
			} `json:"inbounds"`
		}
		if json.Unmarshal(raw, &cfg) != nil {
			continue // a core config Karvan cannot read serves no known protocol
		}
		if kind == "wg" {
			if tag := strings.TrimSpace(cfg.Interface); tag != "" {
				out[tag] = "wireguard"
			}
			continue
		}
		for _, in := range cfg.Inbounds {
			out[in.Tag] = in.Protocol
		}
	}
	return out, rows.Err()
}

// Targets lists the groups moved users can join, with their inbound tags
// in Name order. Disabled groups are listed too and marked, because the
// operator may still pick one on purpose.
func Targets(ctx context.Context, q paneldb.Querier, d paneldb.Dialect) ([]panel.Target, error) {
	rows, err := q.QueryContext(ctx, "SELECT g.id, g.name, g.is_disabled, i.tag FROM "+paneldb.Ident(d, "groups")+" g "+
		"LEFT JOIN inbounds_groups_association a ON a.group_id = g.id "+
		"LEFT JOIN inbounds i ON i.id = a.inbound_id ORDER BY g.id, i.tag")
	if err != nil {
		return nil, fmt.Errorf("read groups: %w", err)
	}
	defer rows.Close()
	var out []panel.Target
	for rows.Next() {
		var (
			id       int64
			name     string
			disabled bool
			tag      sql.NullString
		)
		if err := rows.Scan(&id, &name, &disabled, &tag); err != nil {
			return nil, err
		}
		gid := fmt.Sprint(id)
		if len(out) == 0 || out[len(out)-1].ID != gid {
			out = append(out, panel.Target{ID: gid, Name: name, Disabled: disabled})
		}
		if tag.Valid {
			last := &out[len(out)-1]
			last.Inbounds = append(last.Inbounds, tag.String)
		}
	}
	return out, rows.Err()
}
