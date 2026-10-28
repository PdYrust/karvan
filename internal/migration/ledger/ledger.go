// Package ledger records, in Karvan's own database, every admin and user
// Karvan moved, so a re-run never creates anything twice and a rollback
// removes exactly what one job wrote (D30-D35).
package ledger

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/PdYrust/karvan/internal/panel"
)

// States of a ledger row.
const (
	Pending     = "pending"     // written in a destination transaction not yet confirmed
	Done        = "done"        // committed on the destination
	Adopted     = "adopted"     // already there (C28); never written, never removed
	Merged      = "merged"      // an existing destination admin the job used
	Unknown     = "unknown"     // pending, and the row is not on the destination any more
	Compensated = "compensated" // removed again by a rollback
	Superseded  = "superseded"  // the operator chose "move again"
)

// Kinds of rows.
const (
	KindAdmin = "admin"
	KindUser  = "user"
)

// ErrClaimed means another live row already holds this source identity
// for this destination: the user was moved before.
var ErrClaimed = errors.New("already in the ledger")

// Entry is one ledger row.
type Entry struct {
	ID             int64
	Kind           string
	JobID          string
	State          string
	Chunk          int
	SourceOrigin   string
	SourceKey      string
	SourceCreated  time.Time
	SourceUsername string
	CredPrint      string // Fingerprint of the source credentials
	SourceAdmin    string // a user's source admin (D34); '' on admin rows and rows from before
	Dest           string
	panel.Row                // destination id, username, created_at
	Written        time.Time // when the row was added
}

// Ledger reads and writes ledger rows.
type Ledger struct {
	db  *sql.DB
	now func() time.Time
}

// New returns a ledger over Karvan's database.
func New(db *sql.DB) *Ledger { return &Ledger{db: db, now: time.Now} }

func micro(t time.Time) int64 { return t.UTC().UnixMicro() }

// Columns is the column list ScanEntry reads, over the ledger as table
// l, so a query can join it with other tables.
const Columns = "l.id, l.kind, l.job_id, l.state, l.chunk, l.source_origin, l.source_key, l.source_created, " +
	"l.source_username, l.cred_fingerprint, l.dest, l.dest_id, l.dest_username, l.dest_created, l.created_at, l.source_admin"

// ScanEntry reads one row of the ledger's columns, then extra columns
// that follow them in the row.
func ScanEntry(row interface{ Scan(...any) error }, extra ...any) (Entry, error) {
	var e Entry
	var src, dst, written int64
	dest := []any{&e.ID, &e.Kind, &e.JobID, &e.State, &e.Chunk, &e.SourceOrigin, &e.SourceKey, &src,
		&e.SourceUsername, &e.CredPrint, &e.Dest, &e.Row.ID, &e.Username, &dst, &written, &e.SourceAdmin}
	if err := row.Scan(append(dest, extra...)...); err != nil {
		return e, fmt.Errorf("read ledger: %w", err)
	}
	e.SourceCreated, e.CreatedAt = time.UnixMicro(src).UTC(), time.UnixMicro(dst).UTC()
	e.Written = time.UnixMicro(written).UTC()
	return e, nil
}

func scan(rows *sql.Rows) ([]Entry, error) {
	defer rows.Close()
	var out []Entry
	for rows.Next() {
		e, err := ScanEntry(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// Live returns the live rows (pending, done, adopted, unknown) for one
// source panel and destination: what a new check must know about.
func (l *Ledger) Live(ctx context.Context, origin, dest string) ([]Entry, error) {
	rows, err := l.db.QueryContext(ctx, "SELECT "+Columns+" FROM ledger l WHERE source_origin = ? AND dest = ? "+
		"AND state IN ('pending', 'done', 'adopted', 'unknown') ORDER BY id", origin, dest)
	if err != nil {
		return nil, fmt.Errorf("read ledger: %w", err)
	}
	return scan(rows)
}

// Origin is the ledger's name for a source panel: scheme, host, port and
// base path, lowercased, without a default port (D31).
func Origin(u *url.URL) string {
	host := strings.ToLower(u.Hostname())
	port := u.Port()
	if port == "" || u.Scheme == "https" && port == "443" || u.Scheme == "http" && port == "80" {
		return u.Scheme + "://" + host + strings.ToLower(u.Path)
	}
	return u.Scheme + "://" + host + ":" + port + strings.ToLower(u.Path)
}

// ErrNotFound means no row matched.
var ErrNotFound = errors.New("not in the ledger")

// Get returns one row.
func (l *Ledger) Get(ctx context.Context, id int64) (Entry, error) {
	rows, err := l.db.QueryContext(ctx, "SELECT "+Columns+" FROM ledger l WHERE id = ?", id)
	if err != nil {
		return Entry{}, fmt.Errorf("read ledger: %w", err)
	}
	return one(rows)
}

func one(rows *sql.Rows) (Entry, error) {
	got, err := scan(rows)
	if err != nil {
		return Entry{}, err
	}
	if len(got) == 0 {
		return Entry{}, ErrNotFound
	}
	return got[0], nil
}

// Fingerprint names a credential set without holding it: the first 12
// bytes of SHA-256 over the sorted credential keys.
func Fingerprint(c panel.Credentials) string {
	keys := c.Keys()
	slices.Sort(keys)
	h := sha256.New()
	for _, k := range keys {
		h.Write([]byte(k))
		h.Write([]byte{0})
	}
	return hex.EncodeToString(h.Sum(nil)[:12])
}

// Job returns a job's rows in the given states, newest chunk first.
func (l *Ledger) Job(ctx context.Context, jobID string, states ...string) ([]Entry, error) {
	q := "SELECT " + Columns + " FROM ledger l WHERE job_id = ?"
	args := []any{jobID}
	if len(states) > 0 {
		q += " AND state IN (" + strings.TrimSuffix(strings.Repeat("?, ", len(states)), ", ") + ")"
		for _, s := range states {
			args = append(args, s)
		}
	}
	rows, err := l.db.QueryContext(ctx, q+" ORDER BY chunk DESC, id", args...)
	if err != nil {
		return nil, fmt.Errorf("read ledger: %w", err)
	}
	return scan(rows)
}

// Add inserts entries in one transaction. A live duplicate fails the whole
// call with ErrClaimed, so a source identity is claimed at most once.
func (l *Ledger) Add(ctx context.Context, entries []Entry) ([]int64, error) {
	tx, err := l.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback() //nolint:errcheck // a no-op after Commit
	now := micro(l.now())
	ids := make([]int64, 0, len(entries))
	for _, e := range entries {
		res, err := tx.ExecContext(ctx, "INSERT INTO ledger (kind, job_id, state, chunk, source_origin, source_key, "+
			"source_created, source_username, cred_fingerprint, dest, dest_id, dest_username, dest_created, created_at, updated_at, source_admin) "+
			"VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
			e.Kind, e.JobID, e.State, e.Chunk, e.SourceOrigin, e.SourceKey, micro(e.SourceCreated), e.SourceUsername,
			e.CredPrint, e.Dest, e.Row.ID, e.Username, micro(e.CreatedAt), now, now, e.SourceAdmin)
		if err != nil {
			if strings.Contains(err.Error(), "UNIQUE") {
				return nil, fmt.Errorf("%w: %s %s", ErrClaimed, e.Kind, e.SourceKey)
			}
			return nil, fmt.Errorf("add to ledger: %w", err)
		}
		id, err := res.LastInsertId()
		if err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, tx.Commit()
}

// Set moves rows to state, all or none.
func (l *Ledger) Set(ctx context.Context, state string, ids []int64) error {
	if len(ids) == 0 {
		return nil
	}
	tx, err := l.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck // a no-op after Commit
	now := micro(l.now())
	for _, id := range ids {
		if _, err := tx.ExecContext(ctx, "UPDATE ledger SET state = ?, updated_at = ? WHERE id = ?", state, now, id); err != nil {
			return fmt.Errorf("update ledger: %w", err)
		}
	}
	return tx.Commit()
}
