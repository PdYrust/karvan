package pasarguarddb

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
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
// names under the column's own collation, keyed by the name asked for,
// with their role, own limits and how many users they own.
func FindAdmins(ctx context.Context, q paneldb.Querier, names []string) (map[string]panel.ExistingAdmin, error) {
	out := map[string]panel.ExistingAdmin{}
	for _, n := range names {
		var a panel.ExistingAdmin
		var overrides []byte
		err := q.QueryRowContext(ctx, "SELECT a.id, a.username, r.is_owner, a.role_id, a.permission_overrides, "+
			"(SELECT count(*) FROM users u WHERE u.admin_id = a.id) FROM admins a "+
			"JOIN admin_roles r ON r.id = a.role_id WHERE a.username = ?", n).Scan(&a.ID, &a.Username, &a.IsOwner, &a.RoleID, &overrides, &a.Users)
		switch {
		case errors.Is(err, sql.ErrNoRows):
			continue
		case err != nil:
			return nil, fmt.Errorf("find admin: %w", err)
		}
		if a.Overrides, err = decodeOverrides(overrides); err != nil {
			return nil, fmt.Errorf("admin %s limits: %w", a.Username, err)
		}
		out[n] = a
	}
	return out, nil
}

func decodeOverrides(raw []byte) (*panel.RoleLimits, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return nil, nil
	}
	var l panel.RoleLimits
	if err := json.Unmarshal(raw, &l); err != nil {
		return nil, err
	}
	return &l, nil
}

// Role is an admin role the operator can give the moved admin.
type Role = panel.Role

// OperatorRole is the built-in reseller role (scope OWN), the default for
// a moved admin.
const OperatorRole = 3

// Roles lists the destination's admin roles with what they allow.
func Roles(ctx context.Context, q paneldb.Querier) ([]Role, error) {
	rows, err := q.QueryContext(ctx, "SELECT id, name, is_owner, limits, features, access FROM admin_roles ORDER BY id")
	if err != nil {
		return nil, fmt.Errorf("read admin roles: %w", err)
	}
	defer rows.Close()
	var out []Role
	for rows.Next() {
		var r Role
		var limits, features, access []byte
		if err := rows.Scan(&r.ID, &r.Name, &r.IsOwner, &limits, &features, &access); err != nil {
			return nil, err
		}
		if err := decodeRole(&r, limits, features, access); err != nil {
			return nil, fmt.Errorf("role %s: %w", r.Name, err)
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func decodeRole(r *Role, limits, features, access []byte) error {
	r.Features = panel.RoleFeatures{ResetStrategy: true, NextPlan: true}
	for _, f := range []struct {
		raw []byte
		dst any
	}{{limits, &r.Limits}, {features, &r.Features}} {
		if len(f.raw) > 0 && string(f.raw) != "null" {
			if err := json.Unmarshal(f.raw, f.dst); err != nil {
				return err
			}
		}
	}
	var a struct {
		RequireTemplate bool     `json:"require_template"`
		Groups          *[]int64 `json:"allowed_group_ids"`
	}
	if len(access) > 0 && string(access) != "null" {
		if err := json.Unmarshal(access, &a); err != nil {
			return err
		}
	}
	r.RequireTemplate = a.RequireTemplate
	if a.Groups != nil {
		r.Groups = make([]string, 0, len(*a.Groups))
		for _, id := range *a.Groups {
			r.Groups = append(r.Groups, strconv.FormatInt(id, 10))
		}
	}
	return nil
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

// HostCounts counts the enabled hosts behind each group, by group id.
func HostCounts(ctx context.Context, q paneldb.Querier, d paneldb.Dialect) (map[string]int, error) {
	rows, err := q.QueryContext(ctx, "SELECT ig.group_id, count(h.id) FROM inbounds_groups_association ig "+
		"JOIN inbounds i ON i.id = ig.inbound_id JOIN hosts h ON h.inbound_tag = i.tag "+
		"WHERE h.is_disabled IS NULL OR h.is_disabled = ? GROUP BY ig.group_id", false)
	if err != nil {
		return nil, fmt.Errorf("count hosts: %w", err)
	}
	defer rows.Close()
	out := map[string]int{}
	for rows.Next() {
		var id int64
		var n int
		if err := rows.Scan(&id, &n); err != nil {
			return nil, err
		}
		out[strconv.FormatInt(id, 10)] = n
	}
	return out, rows.Err()
}
