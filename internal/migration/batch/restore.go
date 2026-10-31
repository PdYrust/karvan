package batch

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

// Restore is a backup restore's hold on the batches it puts back:
// every batch on its destination made after the backup, the
// restored job's own included, that is not deleted or rolled back
// already. The restore wipes all their writes, so they stay inactive with
// reason restoring until it ends; the bridge then serves none of them.
type Restore struct {
	s      *Store
	jobID  string
	after  int64   // the backup's time, unix microseconds like ledger rows
	active []int64 // were active before the restore
	failed []int64 // were inactive after an earlier restore failed
}

// Restoring holds the batches a restore of jobID's backup, taken at
// backupAt, puts back on dest: one transaction, before anything is put
// back. The destination's lock must be held, so no run makes a batch
// there meanwhile.
func (s *Store) Restoring(ctx context.Context, dest, jobID string, backupAt time.Time) (*Restore, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback() //nolint:errcheck // a no-op after Commit
	r := &Restore{s: s, jobID: jobID, after: backupAt.UnixMicro()}
	rows, err := tx.QueryContext(ctx, `SELECT id, state FROM batches WHERE dest = ? AND (created_at > ? OR job_id = ?)
		AND (state = 'active' OR (state = 'inactive' AND inactive_reason = 'restore_failed')) ORDER BY id`,
		dest, backupAt.UnixMilli(), jobID)
	if err != nil {
		return nil, fmt.Errorf("read the batches a restore puts back: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var id int64
		var state string
		if err := rows.Scan(&id, &state); err != nil {
			return nil, err
		}
		if state == "active" {
			r.active = append(r.active, id)
		} else {
			r.failed = append(r.failed, id)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	now := s.now().UnixMilli()
	if err := setIn(ctx, tx, `UPDATE batches SET state = 'inactive', inactive_reason = 'restoring', inactive_at = ?, updated_at = ?
		WHERE id IN (:ids)`, r.active, now, now); err != nil {
		return nil, err
	}
	if err := setIn(ctx, tx, "UPDATE batches SET inactive_reason = 'restoring', updated_at = ? WHERE id IN (:ids)", r.failed, now); err != nil {
		return nil, err
	}
	return r, tx.Commit()
}

// Cancel ends a restore that failed before anything was put back: every
// batch is as it was.
func (r *Restore) Cancel(ctx context.Context) error {
	return r.end(ctx, func(tx *sql.Tx, now int64) error {
		if err := setIn(ctx, tx, `UPDATE batches SET state = 'active', inactive_reason = '', inactive_at = 0, updated_at = ?
			WHERE id IN (:ids)`, r.active, now); err != nil {
			return err
		}
		return setIn(ctx, tx, "UPDATE batches SET inactive_reason = 'restore_failed', updated_at = ? WHERE id IN (:ids)", r.failed, now)
	})
}

// Fail ends a restore that failed after it started putting the backup
// back: its batches are restore_failed, and the operator checks the
// destination before making one active again.
func (r *Restore) Fail(ctx context.Context) error {
	return r.end(ctx, func(tx *sql.Tx, now int64) error {
		return setIn(ctx, tx, "UPDATE batches SET inactive_reason = 'restore_failed', updated_at = ? WHERE id IN (:ids)", r.held(), now)
	})
}

// Done ends a restore that put the backup back: its batches are
// rolled back. The restored job's ledger rows in done, pending or
// unknown become compensated, and so do their rows written after the
// backup, since the backup holds none of them. Their other users were in
// the backup and are still on the destination: adopted ones, and ones an
// earlier failed run wrote that a batch claimed. Those leave the
// batch as they are, so manual rules serve them and a later batch can
// claim them.
func (r *Restore) Done(ctx context.Context) error {
	return r.end(ctx, func(tx *sql.Tx, now int64) error {
		held, micro := r.held(), now*1000 // ledger times are unix microseconds
		if _, err := tx.ExecContext(ctx, `UPDATE ledger SET state = 'compensated', updated_at = ?
			WHERE job_id = ? AND state IN ('done', 'pending', 'unknown')`, micro, r.jobID); err != nil {
			return fmt.Errorf("retire the restored job's rows: %w", err)
		}
		if err := setIn(ctx, tx, `UPDATE ledger SET state = 'compensated', updated_at = ?
			WHERE state IN ('done', 'pending', 'unknown') AND created_at > ? AND batch_id IN (:ids)`, held, micro, r.after); err != nil {
			return err
		}
		if err := setIn(ctx, tx, `UPDATE ledger SET batch_id = NULL, updated_at = ?
			WHERE state IN ('done', 'adopted') AND batch_id IN (:ids)`, held, micro); err != nil {
			return err
		}
		return setIn(ctx, tx, "UPDATE batches SET inactive_reason = 'rolled_back', updated_at = ? WHERE id IN (:ids)", held, now)
	})
}

// FailRestoring runs at start: a batch still restoring has no restore
// running (Karvan stopped during it), so its restore failed.
func (s *Store) FailRestoring(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, `UPDATE batches SET inactive_reason = 'restore_failed', updated_at = ?
		WHERE state = 'inactive' AND inactive_reason = 'restoring'`, s.now().UnixMilli())
	if err != nil {
		return fmt.Errorf("end restores that did not finish: %w", err)
	}
	return nil
}

func (r *Restore) held() []int64 { return append(append([]int64{}, r.active...), r.failed...) }

// end runs one of a restore's endings in a transaction.
func (r *Restore) end(ctx context.Context, f func(tx *sql.Tx, now int64) error) error {
	tx, err := r.s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck // a no-op after Commit
	if err := f(tx, r.s.now().UnixMilli()); err != nil {
		return err
	}
	return tx.Commit()
}

// setIn runs q on the rows of ids; q names them with "IN (:ids)" after
// its other arguments, args. No ids, nothing to run.
func setIn(ctx context.Context, tx *sql.Tx, q string, ids []int64, args ...any) error {
	if len(ids) == 0 {
		return nil
	}
	for _, id := range ids {
		args = append(args, id)
	}
	marks := strings.TrimPrefix(strings.Repeat(", ?", len(ids)), ", ")
	if _, err := tx.ExecContext(ctx, strings.Replace(q, ":ids", marks, 1), args...); err != nil {
		return fmt.Errorf("update the batches of a restore: %w", err)
	}
	return nil
}
