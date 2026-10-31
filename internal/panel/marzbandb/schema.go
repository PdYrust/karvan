// Package marzbandb is the Marzban destination: it reads and writes a
// Marzban 0.8.4 database directly, following the panel's own schema.
package marzbandb

import (
	"context"
	_ "embed"

	"github.com/PdYrust/karvan/internal/panel"
	"github.com/PdYrust/karvan/internal/panel/paneldb"
	"github.com/PdYrust/karvan/internal/panel/panelstats"
)

// Revision is the alembic head of Marzban 0.8.4, the known version. Other
// revisions are used when their schema matches this one's (CheckShape).
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

// inserts are the tables Karvan adds rows to (write.go).
var inserts = []string{"admins", "inbounds", "hosts", "users", "proxies", "exclude_inbounds_association", "next_plans", "user_usage_logs"}

// knownJSON is the shape of columns at Revision on each server flavor,
// read from the fixtures (go test -tags dbtest -run TestKnownSchema -update).
//
//go:embed schema.json
var knownJSON []byte

var known = mustSpecs(knownJSON)

func mustSpecs(b []byte) map[string]paneldb.Spec {
	s, err := paneldb.ParseSpecs(b)
	if err != nil {
		panic("marzbandb: schema.json: " + err.Error())
	}
	return s
}

// children are the rows that hang off a user or an admin, in delete order.
// Anything else that references them stops Safe Mode.
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
	Server   paneldb.Server
	Revision string
	// Checked means Revision is not a known one but the live schema
	// matched the known one (paneldb.CheckShape), so Karvan uses it.
	Checked        bool
	UsersFoldCase  bool // users.username compares without case
	AdminsFoldCase bool
	// UnknownChildren reference users, proxies or admins but are not in the
	// children list, so Karvan could not remove a moved user cleanly.
	UnknownChildren []paneldb.ForeignKey
}

// Inspect checks that the database is Marzban at Revision, or at another
// revision with the same shape for what Karvan uses, with the expected
// columns, and reads its collations and foreign keys.
func Inspect(ctx context.Context, q paneldb.Querier, srv paneldb.Server) (Schema, error) {
	s := Schema{Server: srv}
	rev, err := paneldb.Revision(ctx, q)
	if err != nil {
		return s, err
	}
	s.Revision = rev
	if rev != Revision {
		diffs, err := paneldb.CheckShape(ctx, q, srv, known, inserts)
		if err != nil {
			return s, err
		}
		if len(diffs) > 0 {
			return s, &panel.VersionError{Kind: panel.Marzban, Revision: rev, Diffs: diffs}
		}
		s.Checked = true
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

// Stats tells panelstats how Marzban differs: expire is unix seconds, and
// subscription apps are each user's last user agent.
var Stats = panelstats.Schema{ExpireUnix: true}
