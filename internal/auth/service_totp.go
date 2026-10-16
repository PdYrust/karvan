package auth

import (
	"context"
	"errors"

	"github.com/PdYrust/karvan/internal/audit"
	"github.com/PdYrust/karvan/internal/secret"
)

// TOTPSetup is what the operator needs to add Karvan to an authenticator app.
type TOTPSetup struct {
	Secret string // base32, for typing by hand
	URI    string // otpauth://, for the QR code
}

// BeginTOTP creates a pending secret. It is not used for sign-in until
// ConfirmTOTP proves the operator's app produces matching codes. While
// two-step is on it is refused: otherwise a stolen session could swap the
// second factor for one the thief controls. Turn it off first.
func (s *Service) BeginTOTP(ctx context.Context, operatorID int64) (TOTPSetup, error) {
	op, err := s.Operators.ByID(ctx, operatorID)
	if err != nil {
		return TOTPSetup{}, err
	}
	if op.TOTPEnabled {
		return TOTPSetup{}, ErrTOTPAlreadyActive
	}
	sec, err := NewTOTPSecret()
	if err != nil {
		return TOTPSetup{}, err
	}
	if err := s.Operators.SetPendingTOTP(ctx, operatorID, sec); err != nil {
		return TOTPSetup{}, err
	}
	return TOTPSetup{Secret: TOTPSecretText(sec), URI: TOTPURI(sec, op.Username)}, nil
}

// ConfirmTOTP turns two-step sign-in on if code matches the pending secret.
func (s *Service) ConfirmTOTP(ctx context.Context, operatorID int64, code, ip string) error {
	op, err := s.Operators.ByID(ctx, operatorID)
	if err != nil {
		return err
	}
	if op.TOTPEnabled {
		return ErrTOTPAlreadyActive
	}
	sec, err := s.Operators.PendingTOTP(ctx, operatorID)
	if err != nil {
		return err
	}
	step, ok := VerifyTOTP(sec, code, s.now(), 0)
	if !ok {
		return ErrTOTPInvalid
	}
	if err := s.Operators.ActivateTOTP(ctx, operatorID, sec, step); err != nil {
		return err
	}
	return s.record(ctx, op.Username, ip, audit.ActionTOTPEnable, audit.Success, nil)
}

// DisableTOTP turns two-step sign-in off. It asks for the password and a
// current code, so a stolen session alone cannot remove the second factor.
func (s *Service) DisableTOTP(ctx context.Context, operatorID int64, password secret.Value, code, ip string) error {
	op, err := s.Operators.ByID(ctx, operatorID)
	if err != nil {
		return err
	}
	if !op.TOTPEnabled {
		return ErrTOTPNotEnabled
	}
	if wait, err := s.Limiter.Check(ctx, ip, op.Username); err != nil {
		return err
	} else if wait > 0 {
		return &LockedError{RetryAfter: wait}
	}
	ok, err := VerifyPassword(op.PasswordHash, password.Reveal())
	if err != nil {
		return err
	}
	if !ok {
		return s.failDisable(ctx, op.Username, ip, "bad_password", ErrInvalidCredentials)
	}
	if err := s.checkTOTP(ctx, operatorID, code); err != nil {
		if !errors.Is(err, ErrTOTPInvalid) {
			return err
		}
		return s.failDisable(ctx, op.Username, ip, "bad_totp", err)
	}
	if err := s.Operators.DisableTOTP(ctx, operatorID); err != nil {
		return err
	}
	return s.record(ctx, op.Username, ip, audit.ActionTOTPDisable, audit.Success, nil)
}

// failDisable counts a failed attempt against the same limits as sign-in, so
// a stolen session cannot be used to guess the password.
func (s *Service) failDisable(ctx context.Context, username, ip, reason string, cause error) error {
	if _, err := s.Limiter.Fail(ctx, ip, username); err != nil {
		return err
	}
	if err := s.record(ctx, username, ip, audit.ActionTOTPDisable, audit.Failure, map[string]string{"reason": reason}); err != nil {
		return err
	}
	return cause
}
