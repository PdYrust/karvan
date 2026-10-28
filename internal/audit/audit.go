// Package audit records sign-ins and sensitive actions: who, when, from
// where, and whether it worked. Details never hold secrets.
package audit

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

// Outcome is the result of an audited action.
type Outcome string

// Outcomes.
const (
	Success Outcome = "success"
	Failure Outcome = "failure"
)

// Actions.
const (
	ActionLogin       = "login"
	ActionLogout      = "logout"
	ActionLockout     = "lockout"
	ActionTOTPEnable  = "totp.enable"
	ActionTOTPDisable = "totp.disable"
	ActionInit        = "cli.init"
	ActionReset       = "cli.reset"

	ActionPanelsScan       = "panels.scan"
	ActionMigrationRun     = "migration.run"
	ActionMigrationUndo    = "migration.rollback"
	ActionMigrationStop    = "migration.cancel"
	ActionMigrationDelete  = "migration.delete"
	ActionSafeModeOn       = "safe_mode.on"
	ActionSafeModeOff      = "safe_mode.off"
	ActionBackupDelete     = "safe_mode.backup_delete"
	ActionBackupRestore    = "cli.safe_mode_restore"
	ActionBackupRestoreWeb = "safe_mode.backup_restore"
	ActionSafeModeRollback = "safe_mode.rollback"
	ActionSiteSave         = "legacy_site.save"
	ActionSiteDelete       = "legacy_site.delete"
	ActionSiteTokens       = "legacy_site.tokens"
	ActionPasswordsTaken   = "migration.passwords_download"
	ActionBatchRename      = "batch.rename"
	ActionBatchDelete      = "batch.delete"
	ActionBatchReactivate  = "batch.reactivate"
	ActionBatchShowLink    = "batch.show_link"
	ActionDomainAdd        = "domain.add"
	ActionDomainDelete     = "domain.delete"
	ActionDomainCert       = "domain.certificate"
	ActionCertAdd          = "certificate.add"
	ActionCertSave         = "certificate.save"
	ActionCertDelete       = "certificate.delete"
	ActionCertScan         = "certificate.scan"
	ActionRedirectSave     = "redirect.save"
	ActionRedirectDelete   = "redirect.delete"

	ActionSessionsEnd   = "sessions.end"
	ActionLoginRenew    = "account.credentials"
	ActionAddressStart  = "settings.address_start"
	ActionAddressChange = "settings.address"
)

// ActorCLI is the actor for actions taken from the command line.
const ActorCLI = "cli"

// Event is one audited action.
type Event struct {
	ID      int64
	At      time.Time
	Actor   string
	IP      string
	Action  string
	Outcome Outcome
	Detail  map[string]string
}

// Log writes and reads audit events.
type Log struct {
	db  *sql.DB
	now func() time.Time
}

// New returns a Log over db.
func New(db *sql.DB) *Log { return &Log{db: db, now: time.Now} }

// Record stores e. At is set to now when zero.
func (l *Log) Record(ctx context.Context, e Event) error {
	if e.At.IsZero() {
		e.At = l.now()
	}
	detail := []byte("{}")
	if len(e.Detail) > 0 {
		var err error
		if detail, err = json.Marshal(e.Detail); err != nil {
			return fmt.Errorf("encode audit detail: %w", err)
		}
	}
	_, err := l.db.ExecContext(ctx, `INSERT INTO audit_events (at, actor, ip, action, outcome, detail)
		VALUES (?, ?, ?, ?, ?, ?)`, e.At.UnixMilli(), e.Actor, e.IP, e.Action, string(e.Outcome), string(detail))
	if err != nil {
		return fmt.Errorf("record audit event %s: %w", e.Action, err)
	}
	return nil
}

// Kept is how long audit events are kept; older ones are deleted daily,
// so the log cannot fill the disk over the years.
const Kept = 180 * 24 * time.Hour

// Trim deletes the events older than Kept and returns how many it deleted.
func (l *Log) Trim(ctx context.Context) (int64, error) {
	res, err := l.db.ExecContext(ctx, `DELETE FROM audit_events WHERE at < ?`, l.now().Add(-Kept).UnixMilli())
	if err != nil {
		return 0, fmt.Errorf("trim audit events: %w", err)
	}
	return res.RowsAffected()
}

// Filter narrows List. Zero fields match everything.
type Filter struct {
	Actor   string
	Action  string
	Outcome Outcome
	Limit   int
	// SkipNewest drops the newest matches, for example the sign-in that
	// started the current session.
	SkipNewest int
	// Before keeps events with a smaller id, to read the page after an
	// event (keyset paging). 0 starts at the newest.
	Before int64
}

// List returns matching events, newest first. Ids grow with time, so the
// order is the id's; it stays stable under paging even if the clock jumps.
func (l *Log) List(ctx context.Context, f Filter) ([]Event, error) {
	if f.Limit <= 0 || f.Limit > 500 {
		f.Limit = 50
	}
	rows, err := l.db.QueryContext(ctx, `SELECT id, at, actor, ip, action, outcome, detail FROM audit_events
		WHERE (? = '' OR actor = ?) AND (? = '' OR action = ?) AND (? = '' OR outcome = ?) AND (? = 0 OR id < ?)
		ORDER BY id DESC LIMIT ? OFFSET ?`,
		f.Actor, f.Actor, f.Action, f.Action, string(f.Outcome), string(f.Outcome), f.Before, f.Before, f.Limit, max(f.SkipNewest, 0))
	if err != nil {
		return nil, fmt.Errorf("list audit events: %w", err)
	}
	defer rows.Close()
	var out []Event
	for rows.Next() {
		var e Event
		var at int64
		var outcome, detail string
		if err := rows.Scan(&e.ID, &at, &e.Actor, &e.IP, &e.Action, &outcome, &detail); err != nil {
			return nil, fmt.Errorf("scan audit event: %w", err)
		}
		e.At, e.Outcome = time.UnixMilli(at), Outcome(outcome)
		if err := json.Unmarshal([]byte(detail), &e.Detail); err != nil {
			return nil, fmt.Errorf("decode audit detail %d: %w", e.ID, err)
		}
		out = append(out, e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list audit events: %w", err)
	}
	return out, nil
}
