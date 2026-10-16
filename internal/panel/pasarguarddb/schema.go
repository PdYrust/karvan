// Package pasarguarddb is the PasarGuard destination: it reads and writes a
// PasarGuard database directly, following the panel's own schema.
package pasarguarddb

import (
	"context"

	"github.com/PdYrust/karvan/internal/panel"
	"github.com/PdYrust/karvan/internal/panel/paneldb"
)

// Revision is the alembic head of PasarGuard 5.4.1. Newer revisions are
// added one by one, each with its own fixture (owner decision 3).
const Revision = "48a6bcb8bba1"

// columns are the columns Karvan reads or writes, per table, at Revision.
var columns = map[string][]string{
	"admins": {"id", "username", "hashed_password", "created_at", "password_reset_at", "telegram_id",
		"discord_webhook", "used_traffic", "note", "role_id", "status", "data_limit", "last_status_change"},
	"admin_roles": {"id", "name", "is_owner"},
	"users": {"id", "username", "status", "used_traffic", "data_limit", "created_at", "admin_id",
		"data_limit_reset_strategy", "sub_revoked_at", "note", "online_at", "edit_at", "on_hold_timeout",
		"on_hold_expire_duration", "auto_delete_in_days", "last_status_change", "expire", "proxy_settings",
		"hwid_limit"},
	"groups":                      {"id", "name", "is_disabled"},
	"users_groups_association":    {"user_id", "groups_id"},
	"inbounds":                    {"id", "tag"},
	"inbounds_groups_association": {"inbound_id", "group_id"},
	"next_plans":                  {"id", "user_id", "data_limit", "expire", "add_remaining_traffic", "user_template_id"},
	"user_usage_logs":             {"id", "user_id", "used_traffic_at_reset", "reset_at"},
	"jwt":                         {"id", "secret_key"},
	"settings":                    {"id", "subscription"},
	"wireguard_subnets":           {"id", "network", "next_offset", "free_offsets"},
	"core_configs":                {"id", "config", "type"},
}

// children are the rows that hang off a user or an admin, in delete order
// (D35, D37). Some cascade in the database; Karvan deletes them anyway, so
// the order is the same on every backend.
var children = []paneldb.ForeignKey{
	{Table: "users_groups_association", Column: "user_id", RefTable: "users"},
	{Table: "next_plans", Column: "user_id", RefTable: "users"},
	{Table: "node_user_usages", Column: "user_id", RefTable: "users"},
	{Table: "notification_reminders", Column: "user_id", RefTable: "users"},
	{Table: "user_subscription_updates", Column: "user_id", RefTable: "users"},
	{Table: "user_usage_logs", Column: "user_id", RefTable: "users"},
	{Table: "user_hwids", Column: "user_id", RefTable: "users"},
	{Table: "users", Column: "admin_id", RefTable: "admins"},
	{Table: "admin_usage_logs", Column: "admin_id", RefTable: "admins"},
	{Table: "api_keys", Column: "admin_id", RefTable: "admins"},
	{Table: "admin_notification_reminders", Column: "admin_id", RefTable: "admins"},
}

// Schema is what Inspect learned about a destination database.
type Schema struct {
	Server          paneldb.Server
	Revision        string
	AdminsFoldCase  bool
	UnknownChildren []paneldb.ForeignKey
}

// Inspect checks that the database is PasarGuard at Revision with the
// expected columns, and reads the admin collation and foreign keys. User
// names are case-sensitive on every backend (CaseSensitiveString).
func Inspect(ctx context.Context, q paneldb.Querier, srv paneldb.Server) (Schema, error) {
	s := Schema{Server: srv}
	rev, err := paneldb.Revision(ctx, q)
	if err != nil {
		return s, err
	}
	s.Revision = rev
	if rev != Revision {
		return s, &panel.VersionError{Kind: panel.PasarGuard, Revision: rev}
	}
	if gap, err := paneldb.MissingColumns(ctx, q, srv.Dialect, columns); err != nil || gap != "" {
		if err == nil {
			err = &panel.SchemaError{Kind: panel.PasarGuard, Detail: gap}
		}
		return s, err
	}
	if s.AdminsFoldCase, err = paneldb.CaseInsensitive(ctx, q, srv.Dialect, "admins", "username"); err != nil {
		return s, err
	}
	fks, err := paneldb.ForeignKeysTo(ctx, q, srv.Dialect, "users", "admins")
	if err != nil {
		return s, err
	}
	s.UnknownChildren = paneldb.UnknownKeys(fks, children)
	return s, nil
}
