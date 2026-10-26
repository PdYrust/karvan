package run

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

// ErrNotDeletable means the job ran or runs, so its record stays: its
// ledger rows, report and backup point at it.
var ErrNotDeletable = errors.New("only a draft that never ran can be deleted")

func isDraft(state string) bool { return state == StateCreated || state == StateChecked }

// DeleteDraft deletes a job that never ran (created or checked, also one
// whose login Karvan dropped) with its events, and wipes the login it
// holds. A draft whose panels are being read is busy.
func (s *Service) DeleteDraft(ctx context.Context, id string) error {
	j, err := s.job(id)
	if errors.Is(err, ErrNoJob) {
		return s.deleteDraftRow(ctx, id)
	}
	if !j.lockIdle() {
		return ErrNotDeletable
	}
	defer j.mu.Unlock()
	if j.checking.Load() {
		return ErrBusy
	}
	if !isDraft(j.state) {
		return ErrNotDeletable
	}
	if err := s.deleteDraftRow(ctx, id); err != nil {
		return err
	}
	// The job's lock is held, so a request that found the job before it
	// left the map finds no login and no reading once it gets the lock.
	j.wipe()
	s.mu.Lock()
	delete(s.jobs, id)
	s.mu.Unlock()
	return nil
}

// deleteDraftRow deletes the row of a draft and its events in one
// transaction, and says why when the row is not a draft's.
func (s *Service) deleteDraftRow(ctx context.Context, id string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("delete draft %s: %w", id, err)
	}
	defer func() { _ = tx.Rollback() }()
	var state string
	err = tx.QueryRowContext(ctx, "SELECT state FROM jobs WHERE id = ?", id).Scan(&state)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNoJob
	}
	if err != nil {
		return fmt.Errorf("delete draft %s: %w", id, err)
	}
	if !isDraft(state) {
		return ErrNotDeletable
	}
	if _, err := tx.ExecContext(ctx, "DELETE FROM job_events WHERE job_id = ?", id); err != nil {
		return fmt.Errorf("delete events of draft %s: %w", id, err)
	}
	if _, err := tx.ExecContext(ctx, "DELETE FROM jobs WHERE id = ?", id); err != nil {
		return fmt.Errorf("delete draft %s: %w", id, err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("delete draft %s: %w", id, err)
	}
	return nil
}
