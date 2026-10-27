package batch

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Errors of the operator's actions on a batch (D8, D20).
var (
	ErrInvalidName      = errors.New("batch name: 1 to 64 characters, no control characters")
	ErrActive           = errors.New("an active batch cannot be deleted")
	ErrBusy             = errors.New("a backup restore is putting this batch back")
	ErrNotReactivatable = errors.New("only a batch whose restore failed can be made active again")
)

// ConfirmError means the typed confirmation did not match the batch's
// name, which it carries.
type ConfirmError struct{ Name string }

func (e *ConfirmError) Error() string { return "type the batch's name to delete it" }

// InUseError means the batch is the only batch of these rules, so
// deleting it would leave them serving nothing.
type InUseError struct{ Rules []Rule }

func (e *InUseError) Error() string { return "the batch is the only batch of a rule" }

// CleanName trims a batch name and checks it: 1 to 64 characters, no
// control characters.
func CleanName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if n := utf8.RuneCountInString(name); n < 1 || n > 64 || !utf8.ValidString(name) || strings.ContainsFunc(name, unicode.IsControl) {
		return "", ErrInvalidName
	}
	return name, nil
}

// Rename gives a batch that is not deleted a new name.
func (s *Store) Rename(ctx context.Context, id int64, name string) (Batch, error) {
	name, err := CleanName(name)
	if err != nil {
		return Batch{}, err
	}
	res, err := s.db.ExecContext(ctx, "UPDATE batches SET name = ?, updated_at = ? WHERE id = ? AND state <> 'deleted'",
		name, s.now().UnixMilli(), id)
	if err != nil {
		return Batch{}, fmt.Errorf("rename batch %d: %w", id, err)
	}
	if n, err := res.RowsAffected(); err != nil {
		return Batch{}, err
	} else if n == 0 {
		return Batch{}, ErrNotFound
	}
	return s.Get(ctx, id)
}

// Delete marks an inactive batch deleted (D20) once confirm equals its
// name. Its row, name and job stay for history and its ledger rows keep
// batch_id, so its users stay out of every rule and out of later batches;
// its verified tokens and its links to rules go. The only batch of a rule
// stays (InUseError), and so does a batch a restore is putting back.
func (s *Store) Delete(ctx context.Context, id int64, confirm string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck // a no-op after Commit
	var name, state, reason string
	err = tx.QueryRowContext(ctx, "SELECT name, state, inactive_reason FROM batches WHERE id = ? AND state <> 'deleted'", id).
		Scan(&name, &state, &reason)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return ErrNotFound
	case err != nil:
		return fmt.Errorf("read batch %d: %w", id, err)
	case state == "active":
		return ErrActive
	case reason == "restoring":
		return ErrBusy
	}
	only, err := readRules(ctx, tx, `SELECT s.id, s.name FROM legacy_sites s JOIN legacy_site_batches sb ON sb.site_id = s.id
		WHERE sb.batch_id = ? AND NOT EXISTS (SELECT 1 FROM legacy_site_batches o WHERE o.site_id = s.id AND o.batch_id <> ?)
		ORDER BY s.name, s.id`, id, id)
	if err != nil {
		return err
	}
	if len(only) > 0 {
		return &InUseError{Rules: only}
	}
	if confirm != name {
		return &ConfirmError{Name: name}
	}
	now := s.now().UnixMilli()
	for _, q := range []string{
		"DELETE FROM legacy_tokens WHERE ledger_id IN (SELECT id FROM ledger WHERE batch_id = ?)",
		"DELETE FROM legacy_site_batches WHERE batch_id = ?",
	} {
		if _, err := tx.ExecContext(ctx, q, id); err != nil {
			return fmt.Errorf("delete batch %d: %w", id, err)
		}
	}
	if _, err := tx.ExecContext(ctx, "UPDATE batches SET state = 'deleted', deleted_at = ?, updated_at = ? WHERE id = ?", now, now, id); err != nil {
		return fmt.Errorf("delete batch %d: %w", id, err)
	}
	return tx.Commit()
}

// Reactivate makes a batch whose restore failed active again (D8): the
// operator checked the destination and its users are there.
func (s *Store) Reactivate(ctx context.Context, id int64) (Batch, error) {
	res, err := s.db.ExecContext(ctx, `UPDATE batches SET state = 'active', inactive_reason = '', inactive_at = 0, updated_at = ?
		WHERE id = ? AND state = 'inactive' AND inactive_reason = 'restore_failed'`, s.now().UnixMilli(), id)
	if err != nil {
		return Batch{}, fmt.Errorf("make batch %d active: %w", id, err)
	}
	if n, err := res.RowsAffected(); err != nil {
		return Batch{}, err
	} else if n == 0 {
		if _, err := s.Get(ctx, id); err != nil {
			return Batch{}, err
		}
		return Batch{}, ErrNotReactivatable
	}
	return s.Get(ctx, id)
}
