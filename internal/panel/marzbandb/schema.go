// Package marzbandb is the Marzban destination: it reads and writes a
// Marzban 0.8.4 database directly, following the panel's own schema.
package marzbandb

import (
	"context"

	"github.com/PdYrust/karvan/internal/panel"
	"github.com/PdYrust/karvan/internal/panel/paneldb"
)

// Revision is the alembic head of Marzban 0.8.4, the only version Karvan
// writes to (owner decision 3).
const Revision = "2b231de97dc3"

// columns are the columns Karvan reads or writes, per table, at Revision.
var columns = map[string][]string{
	"admins": {"id", "username", "hashed_password", "created_at", "is_sudo", "password_reset_at",
		"telegram_id", "discord_webhook", "users_usage"},
	"users": {"id", "username", "status", "used_traffic", "data_limit", "expire", "created_at", "admin_id",
		"data_limit_reset_strategy", "sub_revoked_at", "note", "sub_updated_at", "sub_last_user_agent",
		"online_at", "edit_at", "on_hold_timeout", "on_hold_expire_duration", "auto_delete_in_days",
		"last_status_change"},
	"proxies":                      {"id", "user_id", "type", "settings"},
	"exclude_inbounds_association": {"proxy_id", "inbound_tag"},
	"inbounds":                     {"id", "tag"},
	"hosts": {"id", "remark", "address", "inbound_tag", "security", "alpn", "fingerprint", "is_disabled", "mux_enable",
		"random_user_agent", "use_sni_as_host"},
	"next_plans":      {"id", "user_id", "data_limit", "expire", "add_remaining_traffic", "fire_on_either"},
	"user_usage_logs": {"id", "user_id", "used_traffic_at_reset", "reset_at"},
	"jwt":             {"id", "secret_key"},
}

// children are the rows that hang off a user or an admin, in delete order
// (D35, D37). Anything else that references them stops Safe Mode.
var children = []paneldb.ForeignKey{
	{Table: "exclude_inbounds_association", Column: "proxy_id", RefTable: "proxies"},
	{Table: "proxies", Column: "user_id", RefTable: "users"},
	{Table: "user_usage_logs", Column: "user_id", RefTable: "users"},
	{Table: "node_user_usages", Column: "user_id", RefTable: "users"},
	{Table: "notification_reminders", Column: "user_id", RefTable: "users"},
	{Table: "next_plans", Column: "user_id", RefTable: "users"},
	{Table: "users", Column: "admin_id", RefTable: "admins"},
	{Table: "admin_usage_logs", Column: "admin_id", RefTable: "admins"},
}

// Schema is what Inspect learned about a destination database.
type Schema struct {
	Server         paneldb.Server
	Revision       string
	UsersFoldCase  bool // users.username compares without case
	AdminsFoldCase bool
	// UnknownChildren reference users, proxies or admins but are not in the
	// children list, so Karvan could not remove a moved user cleanly.
	UnknownChildren []paneldb.ForeignKey
}

// Inspect checks that the database is Marzban at Revision with the expected
// columns, and reads its collations and foreign keys.
func Inspect(ctx context.Context, q paneldb.Querier, srv paneldb.Server) (Schema, error) {
	s := Schema{Server: srv}
	rev, err := paneldb.Revision(ctx, q)
	if err != nil {
		return s, err
	}
	s.Revision = rev
	if rev != Revision {
		return s, &panel.VersionError{Kind: panel.Marzban, Revision: rev}
	}
	if gap, err := paneldb.MissingColumns(ctx, q, srv.Dialect, columns); err != nil || gap != "" {
		if err == nil {
			err = &panel.SchemaError{Kind: panel.Marzban, Detail: gap}
		}
		return s, err
	}
	if s.UsersFoldCase, err = paneldb.CaseInsensitive(ctx, q, srv.Dialect, "users", "username"); err != nil {
		return s, err
	}
	if s.AdminsFoldCase, err = paneldb.CaseInsensitive(ctx, q, srv.Dialect, "admins", "username"); err != nil {
		return s, err
	}
	fks, err := paneldb.ForeignKeysTo(ctx, q, srv.Dialect, "users", "proxies", "admins")
	if err != nil {
		return s, err
	}
	s.UnknownChildren = paneldb.UnknownKeys(fks, children)
	return s, nil
}
