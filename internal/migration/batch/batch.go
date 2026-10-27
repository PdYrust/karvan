// Package batch makes the named batches of phase 4.1 (D1 to D7): a
// migration that ends done becomes a batch holding the users it moved.
// Membership lives on the ledger row (ledger.batch_id) and is set only
// here, when the batch is made; a claimed row never moves to another
// batch.
package batch

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

// Store makes and reads batches in Karvan's database.
type Store struct {
	db  *sql.DB
	now func() time.Time
}

// New returns a store over Karvan's database.
func New(db *sql.DB, now func() time.Time) *Store {
	if now == nil {
		now = time.Now
	}
	return &Store{db: db, now: now}
}

// ended is what Ensure reads of a done job.
type ended struct {
	panel               int64
	kind, origin, admin string
	at                  int64 // unix ms
}

// Ensure makes the batch of a done job and returns its id, or the id it
// has already (D5). dest is the job's destination; empty reads it from
// the job's ledger rows. In one transaction it claims (D4) the job's user
// rows in done or adopted, and user rows of the same old panel and
// destination in done or adopted that no batch holds and that were
// written before the job ended (left by an earlier failed or interrupted
// run), then joins every batch rule serving an older batch of the same old
// panel and destination (D26). With nothing to claim, or a job that is
// not done, there is no batch and it returns 0. f is stored only when the
// batch is made.
func (s *Store) Ensure(ctx context.Context, jobID, dest string, f Facts) (int64, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback() //nolint:errcheck // a no-op after Commit
	var id int64
	err = tx.QueryRowContext(ctx, "SELECT id FROM batches WHERE job_id = ?", jobID).Scan(&id)
	if err == nil || !errors.Is(err, sql.ErrNoRows) {
		return id, err
	}
	j, ok, err := doneJob(ctx, tx, jobID)
	if err != nil || !ok {
		return 0, err
	}
	if dest == "" {
		if err := tx.QueryRowContext(ctx, "SELECT COALESCE(min(dest), '') FROM ledger WHERE job_id = ?", jobID).Scan(&dest); err != nil {
			return 0, fmt.Errorf("read the job's destination: %w", err)
		}
	}
	if id, err = insert(ctx, tx, jobID, dest, j, f, s.now()); err != nil {
		return 0, err
	}
	res, err := tx.ExecContext(ctx, `UPDATE ledger SET batch_id = ? WHERE kind = 'user' AND batch_id IS NULL
		AND state IN ('done', 'adopted') AND source_origin = ? AND dest = ? AND (job_id = ? OR created_at <= ?)`,
		id, j.origin, dest, jobID, j.at*1000+999)
	if err != nil {
		return 0, fmt.Errorf("claim the batch's users: %w", err)
	}
	if n, err := res.RowsAffected(); err != nil || n == 0 {
		return 0, err // nothing to serve: no batch
	}
	if _, err := tx.ExecContext(ctx, `INSERT OR IGNORE INTO legacy_site_batches (site_id, batch_id)
		SELECT DISTINCT sb.site_id, ? FROM legacy_site_batches sb JOIN batches b ON b.id = sb.batch_id
		WHERE b.source_origin = ? AND b.dest = ? AND b.id <> ?`, id, j.origin, dest, id); err != nil {
		return 0, fmt.Errorf("join the batch rules: %w", err)
	}
	return id, tx.Commit()
}

// doneJob reads a job that ended done; false for any other state.
func doneJob(ctx context.Context, tx *sql.Tx, jobID string) (ended, bool, error) {
	var j ended
	var state string
	err := tx.QueryRowContext(ctx, `SELECT state, panel_id, source_kind, source_origin, source_admin,
		COALESCE(NULLIF(ended_at, 0), updated_at) FROM jobs WHERE id = ?`, jobID).
		Scan(&state, &j.panel, &j.kind, &j.origin, &j.admin, &j.at)
	if errors.Is(err, sql.ErrNoRows) {
		return j, false, nil
	}
	if err != nil {
		return j, false, fmt.Errorf("read job %s: %w", jobID, err)
	}
	return j, state == "done", nil
}

// insert adds the batch row with its default name (D6) and returns its id.
func insert(ctx context.Context, tx *sql.Tx, jobID, dest string, j ended, f Facts, now time.Time) (int64, error) {
	taken, err := names(ctx, tx, j.origin)
	if err != nil {
		return 0, err
	}
	hosts := f.OldHosts
	if hosts == nil {
		hosts = []string{}
	}
	raw, err := json.Marshal(hosts)
	if err != nil {
		return 0, err
	}
	res, err := tx.ExecContext(ctx, `INSERT INTO batches (job_id, panel_id, dest, source_kind, source_origin, source_ip, source_pin,
		old_path, old_hosts, old_port, name, state, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 'active', ?, ?)`,
		jobID, j.panel, dest, j.kind, j.origin, f.SourceIP, f.SourcePin, f.OldPath, string(raw), f.OldPort,
		numbered(Name(j.admin), taken), j.at, now.UnixMilli())
	if err != nil {
		return 0, fmt.Errorf("add the batch of %s: %w", jobID, err)
	}
	return res.LastInsertId()
}

// names lists the batch names one old panel uses, deleted ones included.
func names(ctx context.Context, tx *sql.Tx, origin string) ([]string, error) {
	return texts(ctx, tx, "SELECT name FROM batches WHERE source_origin = ?", origin)
}

// Missing lists done jobs that have no batch, oldest first, for Recover
// (D5): the batch of a run that stopped between saving the job and making
// it.
func (s *Store) Missing(ctx context.Context) ([]string, error) {
	return texts(ctx, s.db, `SELECT id FROM jobs j WHERE state = 'done'
		AND NOT EXISTS (SELECT 1 FROM batches b WHERE b.job_id = j.id) ORDER BY COALESCE(NULLIF(ended_at, 0), updated_at), id`)
}

// Foreign is the set of a job's ledger rows that another job's batch
// holds (claimed by D4): an undo or roll back of the job leaves them
// (D14).
func (s *Store) Foreign(ctx context.Context, jobID string) (map[int64]bool, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT l.id FROM ledger l JOIN batches b ON b.id = l.batch_id
		WHERE l.job_id = ? AND b.job_id <> ?`, jobID, jobID)
	if err != nil {
		return nil, fmt.Errorf("read the job's rows in other batches: %w", err)
	}
	defer rows.Close()
	out := map[int64]bool{}
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out[id] = true
	}
	return out, rows.Err()
}

type querier interface {
	QueryContext(ctx context.Context, q string, args ...any) (*sql.Rows, error)
}

// texts runs a query whose rows are one text column each.
func texts(ctx context.Context, db querier, q string, args ...any) ([]string, error) {
	rows, err := db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("read batches: %w", err)
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var v string
		if err := rows.Scan(&v); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}
