// Package pasarguarddb is the PasarGuard destination: it reads and writes a
// PasarGuard database directly, following the panel's own schema.
package pasarguarddb

import (
	"context"
	_ "embed"
	"slices"

	"github.com/PdYrust/karvan/internal/panel"
	"github.com/PdYrust/karvan/internal/panel/paneldb"
	"github.com/PdYrust/karvan/internal/panel/panelstats"
)

// Revision is the alembic head of PasarGuard 5.4.1.
const Revision = "48a6bcb8bba1"

// Revision551 is the alembic head of PasarGuard 5.5.1, which adds admin
// passkeys (admin_passkeys, passkey_challenges).
const Revision551 = "c8d2e4f6a1b3"

// Revisions are the known revisions. Each has fixtures the write, check
// and undo tests run on, and the shape in schema.json. Other revisions are
// used when their schema matches that shape (CheckShape).
var Revisions = []string{Revision, Revision551}

// columns are the columns Karvan reads or writes, per table, at each of Revisions.
var columns = map[string][]string{
	"admins": {"id", "username", "hashed_password", "created_at", "password_reset_at", "telegram_id",
		"discord_webhook", "used_traffic", "note", "role_id", "status", "data_limit", "last_status_change", "permission_overrides"},
	"admin_roles": {"id", "name", "is_owner", "limits", "features", "access"},
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
	"hosts":                       {"id", "inbound_tag", "is_disabled"},
}

// inserts are the tables Karvan adds rows to (write.go).
var inserts = []string{"admins", "users", "users_groups_association", "next_plans", "user_usage_logs"}

// knownJSON is the shape of columns at Revisions on each server flavor,
// read from the fixtures (go test -tags dbtest -run TestKnownSchema -update).
//
//go:embed schema.json
var knownJSON []byte

var known = mustSpecs(knownJSON)

func mustSpecs(b []byte) map[string]paneldb.Spec {
	s, err := paneldb.ParseSpecs(b)
	if err != nil {
		panic("pasarguarddb: schema.json: " + err.Error())
	}
	return s
}

// children are the rows that hang off a user or an admin, in delete order.
// Some cascade in the database; Karvan deletes them anyway, so
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
	// The passkey tables come with PasarGuard 5.5 (revision 9a5c2e7f1b4d):
	// a row per passkey an admin registers, and a short-lived row per
	// passkey sign-in or registration challenge (admin_id may be null).
	// Both keys are ON DELETE CASCADE, so deleting an admin Karvan created
	// also deletes its rows (RemoveAdmin), on SQLite too: the writer turns
	// foreign keys on. Older revisions do not have these tables.
	{Table: "admin_passkeys", Column: "admin_id", RefTable: "admins"},
	{Table: "passkey_challenges", Column: "admin_id", RefTable: "admins"},
}

// Schema is what Inspect learned about a destination database.
type Schema struct {
	Server   paneldb.Server
	Revision string
	// Checked means Revision is not a known one but the live schema
	// matched the known one (paneldb.CheckShape), so Karvan uses it.
	Checked         bool
	AdminsFoldCase  bool
	UnknownChildren []paneldb.ForeignKey
}

// Inspect checks that the database is PasarGuard at one of Revisions, or at
// another revision with the same shape for what Karvan uses, with the
// expected columns, and reads the admin collation and foreign keys. User
// names are case-sensitive on every backend (CaseSensitiveString).
func Inspect(ctx context.Context, q paneldb.Querier, srv paneldb.Server) (Schema, error) {
	s := Schema{Server: srv}
	rev, err := paneldb.Revision(ctx, q)
	if err != nil {
		return s, err
	}
	s.Revision = rev
	if !slices.Contains(Revisions, rev) {
		diffs, err := paneldb.CheckShape(ctx, q, srv, known, inserts)
		if err != nil {
			return s, err
		}
		if len(diffs) > 0 {
			return s, &panel.VersionError{Kind: panel.PasarGuard, Revision: rev, Diffs: diffs}
		}
		s.Checked = true
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

// Stats tells panelstats how PasarGuard differs: subscription apps come
// from its subscription updates table.
var Stats = panelstats.Schema{Updates: true}
