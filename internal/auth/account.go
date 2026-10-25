package auth

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/PdYrust/karvan/internal/audit"
	"github.com/PdYrust/karvan/internal/secret"
)

// Reauth is a signed-in operator's password, and the two-step code when it
// is on, typed again to confirm an account change.
type Reauth struct {
	OperatorID int64
	Password   secret.Value
	Code       string
	IP         string
}

// Reauthenticate checks r before an account change, so a stolen session
// alone cannot make one. Wrong values count against the same limits as
// sign-in and are audited under action.
func (s *Service) Reauthenticate(ctx context.Context, r Reauth, action string) (Operator, error) {
	op, err := s.Operators.ByID(ctx, r.OperatorID)
	if err != nil {
		return Operator{}, err
	}
	release, err := s.hashSlot(ctx)
	if err != nil {
		return Operator{}, err
	}
	defer release()
	if wait, err := s.Limiter.Check(ctx, r.IP, op.Username); err != nil {
		return Operator{}, err
	} else if wait > 0 {
		return Operator{}, &LockedError{RetryAfter: wait}
	}
	ok, err := VerifyPassword(op.PasswordHash, r.Password.Reveal())
	if err != nil {
		return Operator{}, fmt.Errorf("verify password of %s: %w", op.Username, err)
	}
	if !ok {
		return Operator{}, s.failed(ctx, r.IP, op.Username, op.Username, action, "bad_password", ErrInvalidCredentials)
	}
	if op.TOTPEnabled {
		if r.Code == "" {
			return Operator{}, ErrTOTPRequired
		}
		if err := s.checkTOTP(ctx, op.ID, r.Code); err != nil {
			if !errors.Is(err, ErrTOTPInvalid) {
				return Operator{}, err
			}
			return Operator{}, s.failed(ctx, r.IP, op.Username, op.Username, action, "bad_totp", err)
		}
	}
	if err := s.Limiter.Succeed(ctx, r.IP, op.Username); err != nil {
		return Operator{}, err
	}
	return op, nil
}

// RenewedLogin is a new username and password, and the session that took
// the place of every old one.
type RenewedLogin struct {
	Username string
	Password secret.Value
	Token    string
	Session  Session
}

// RenewLogin gives the operator a new random username and password after r
// is checked. Every session ends and this browser gets a new one, in one
// transaction, so the values are either all in place or nothing changed.
// They are audited without the values.
func (s *Service) RenewLogin(ctx context.Context, r Reauth, userAgent string) (RenewedLogin, error) {
	op, err := s.Reauthenticate(ctx, r, audit.ActionLoginRenew)
	if err != nil {
		return RenewedLogin{}, err
	}
	user, err := GenerateUsername()
	if err != nil {
		return RenewedLogin{}, err
	}
	pw, err := GeneratePassword()
	if err != nil {
		return RenewedLogin{}, err
	}
	hash, err := s.hash(ctx, pw)
	if err != nil {
		return RenewedLogin{}, err
	}
	token, sess, err := s.Sessions.Replace(ctx, op.ID, r.IP, userAgent, func(tx *sql.Tx) error {
		return s.Operators.setLogin(ctx, tx, op.ID, user, hash)
	})
	if err != nil {
		return RenewedLogin{}, err
	}
	if err := s.record(ctx, op.Username, r.IP, audit.ActionLoginRenew, audit.Success, nil); err != nil {
		return RenewedLogin{}, err
	}
	return RenewedLogin{Username: user, Password: secret.NewValue(pw), Token: token, Session: sess}, nil
}

// hash runs HashPassword in one of the few hash slots.
func (s *Service) hash(ctx context.Context, pw string) (string, error) {
	release, err := s.hashSlot(ctx)
	if err != nil {
		return "", err
	}
	defer release()
	return HashPassword(pw)
}

// failed counts a wrong password or code, audits it under action and
// returns the error to show: a LockedError when this failure started a lock.
func (s *Service) failed(ctx context.Context, ip, account, actor, action, reason string, cause error) error {
	lock, err := s.Limiter.Fail(ctx, ip, account)
	if err != nil {
		return err
	}
	if err := s.record(ctx, actor, ip, action, audit.Failure, map[string]string{"reason": reason}); err != nil {
		return err
	}
	if lock > 0 {
		if err := s.record(ctx, actor, ip, audit.ActionLockout, audit.Success, map[string]string{"seconds": fmt.Sprint(int(lock.Seconds()))}); err != nil {
			return err
		}
		return &LockedError{RetryAfter: lock}
	}
	return cause
}
