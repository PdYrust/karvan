// Package marzbandb is the Marzban destination: it reads and writes a
// Marzban 0.8.4 database directly, following the panel's own schema.
package marzbandb

import (
	"context"
	"fmt"
	"slices"
	"strings"

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
	"next_plans":                   {"id", "user_id", "data_limit", "expire", "add_remaining_traffic", "fire_on_either"},
	"user_usage_logs":              {"id", "user_id", "used_traffic_at_reset", "reset_at"},
	"jwt":                          {"id", "secret_key"},
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
	if err := checkColumns(ctx, q, srv.Dialect); err != nil {
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
	for _, fk := range fks {
		if !known(fk) {
			s.UnknownChildren = append(s.UnknownChildren, fk)
		}
	}
	return s, nil
}

func checkColumns(ctx context.Context, q paneldb.Querier, d paneldb.Dialect) error {
	tables := make([]string, 0, len(columns))
	for t := range columns {
		tables = append(tables, t)
	}
	slices.Sort(tables)
	for _, t := range tables {
		have, err := paneldb.Columns(ctx, q, d, t)
		if err != nil {
			return fmt.Errorf("read columns of %s: %w", t, err)
		}
		var missing []string
		for _, c := range columns[t] {
			if !slices.Contains(have, c) {
				missing = append(missing, c)
			}
		}
		if len(missing) > 0 {
			return &panel.SchemaError{Kind: panel.Marzban, Detail: t + " lacks " + strings.Join(missing, ", ")}
		}
	}
	return nil
}

func known(fk paneldb.ForeignKey) bool {
	for _, c := range children {
		if c.Table == fk.Table && c.Column == fk.Column && c.RefTable == fk.RefTable {
			return true
		}
	}
	return false
}
