package auth

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/PdYrust/karvan/internal/secret"
)

// ErrNoPendingTOTP means TOTP setup was not started, or already finished.
var ErrNoPendingTOTP = errors.New("no two-step setup in progress")

const (
	colTOTPSecret  = "totp_secret"
	colTOTPPending = "totp_pending"
)

// SetPendingTOTP stores a secret that is not active until ConfirmTOTP.
func (s *Operators) SetPendingTOTP(ctx context.Context, id int64, totpSecret []byte) error {
	sealed, err := s.key.Seal(totpSecret, secret.AAD("operators", colTOTPPending, id))
	if err != nil {
		return fmt.Errorf("seal TOTP secret: %w", err)
	}
	return s.update(ctx, id, colTOTPPending+` = ?`, sealed)
}

// PendingTOTP returns the secret stored by SetPendingTOTP.
func (s *Operators) PendingTOTP(ctx context.Context, id int64) ([]byte, error) {
	return s.readSealed(ctx, id, colTOTPPending, ErrNoPendingTOTP)
}

// ActivateTOTP moves the pending secret into place and records step as used.
func (s *Operators) ActivateTOTP(ctx context.Context, id int64, step int64) error {
	pending, err := s.PendingTOTP(ctx, id)
	if err != nil {
		return err
	}
	sealed, err := s.key.Seal(pending, secret.AAD("operators", colTOTPSecret, id))
	if err != nil {
		return fmt.Errorf("seal TOTP secret: %w", err)
	}
	return s.update(ctx, id, colTOTPSecret+` = ?, `+colTOTPPending+` = NULL, totp_last_step = ?`, sealed, step)
}

// DisableTOTP removes the active and any pending secret.
func (s *Operators) DisableTOTP(ctx context.Context, id int64) error {
	return s.update(ctx, id, colTOTPSecret+` = NULL, `+colTOTPPending+` = NULL, totp_last_step = 0`)
}

// TOTP returns the active secret and the last step that was used.
func (s *Operators) TOTP(ctx context.Context, id int64) ([]byte, int64, error) {
	var lastStep int64
	if err := s.db.QueryRowContext(ctx, `SELECT totp_last_step FROM operators WHERE id = ?`, id).Scan(&lastStep); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, 0, ErrNotFound
		}
		return nil, 0, fmt.Errorf("read TOTP step: %w", err)
	}
	sec, err := s.readSealed(ctx, id, colTOTPSecret, errors.New("two-step sign-in is not enabled"))
	return sec, lastStep, err
}

// UseTOTPStep records step as used. It returns false when a later or equal
// step was already recorded, which is how a replayed code is caught even
// when two requests race.
func (s *Operators) UseTOTPStep(ctx context.Context, id int64, step int64) (bool, error) {
	res, err := s.db.ExecContext(ctx, `UPDATE operators SET totp_last_step = ? WHERE id = ? AND totp_last_step < ?`, step, id, step)
	if err != nil {
		return false, fmt.Errorf("record TOTP step: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("record TOTP step: %w", err)
	}
	return n == 1, nil
}

func (s *Operators) readSealed(ctx context.Context, id int64, col string, errEmpty error) ([]byte, error) {
	var sealed []byte
	err := s.db.QueryRowContext(ctx, `SELECT `+col+` FROM operators WHERE id = ?`, id).Scan(&sealed)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", col, err)
	}
	if sealed == nil {
		return nil, errEmpty
	}
	pt, err := s.key.Open(sealed, secret.AAD("operators", col, id))
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", col, err)
	}
	return pt, nil
}
