package auth

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/PdYrust/karvan/internal/audit"
	"github.com/PdYrust/karvan/internal/secret"
)

const reauthIP = "203.0.113.9"

func (s *serviceFixture) reauth(password, code string) (Operator, error) {
	r := Reauth{OperatorID: s.op.ID, Password: secret.NewValue(password), Code: code, IP: reauthIP}
	return s.svc.Reauthenticate(context.Background(), r, "account.test")
}

// count returns how many audit events match.
func (s *serviceFixture) count(t *testing.T, match func(audit.Event) bool) int {
	t.Helper()
	n := 0
	for _, e := range s.events(t) {
		if match(e) {
			n++
		}
	}
	return n
}

func TestReauthenticateLocksLikeSignIn(t *testing.T) {
	s := newService(t)
	for i := range DefaultLockout.IPMaxFailures - 1 {
		if _, err := s.reauth("wrong password!", ""); !errors.Is(err, ErrInvalidCredentials) {
			t.Fatalf("guess %d: %v", i+1, err)
		}
	}
	var locked *LockedError
	if _, err := s.reauth("wrong password!", ""); !errors.As(err, &locked) || locked.RetryAfter != DefaultLockout.IPLock {
		t.Fatalf("last guess = %v, want a %s lock", err, DefaultLockout.IPLock)
	}
	if _, err := s.reauth(goodPassword, ""); !errors.As(err, &locked) {
		t.Fatalf("the right password during the lock = %v, want locked", err)
	}
	if _, err := s.login("owner", goodPassword, ""); err != nil {
		t.Fatalf("sign-in from another IP: %v", err)
	}
	s.clock.Advance(DefaultLockout.IPLock + time.Second)
	if op, err := s.reauth(goodPassword, ""); err != nil || op.ID != s.op.ID {
		t.Fatalf("after the lock: %+v, %v", op, err)
	}
}

func TestReauthenticateAuditsEachGuessAndTheLock(t *testing.T) {
	s := newService(t)
	for range DefaultLockout.IPMaxFailures {
		_, _ = s.reauth("wrong password!", "")
	}
	failures := s.count(t, func(e audit.Event) bool {
		return e.Action == "account.test" && e.Outcome == audit.Failure && e.Detail["reason"] == "bad_password"
	})
	lockouts := s.count(t, func(e audit.Event) bool { return e.Action == audit.ActionLockout && e.IP == reauthIP })
	if failures != DefaultLockout.IPMaxFailures || lockouts != 1 {
		t.Fatalf("audited %d failures and %d lockouts", failures, lockouts)
	}
}

func TestReauthenticateNeedsTheCodeWhenTwoStepIsOn(t *testing.T) {
	s := newService(t)
	sec := enableTOTP(t, s)
	if _, err := s.reauth(goodPassword, ""); !errors.Is(err, ErrTOTPRequired) {
		t.Fatalf("no code: %v", err)
	}
	if _, err := s.reauth(goodPassword, "000000"); !errors.Is(err, ErrTOTPInvalid) {
		t.Fatalf("wrong code: %v", err)
	}
	code := totpCode(sec, totpStep(s.clock.Now()))
	if _, err := s.reauth(goodPassword, code); err != nil {
		t.Fatalf("right code: %v", err)
	}
	if _, err := s.reauth(goodPassword, code); !errors.Is(err, ErrTOTPInvalid) {
		t.Fatalf("replayed code: %v", err)
	}
	// A missing code is not a guess; the wrong and the replayed ones are.
	if n := s.count(t, func(e audit.Event) bool { return e.Action == "account.test" && e.Detail["reason"] == "bad_totp" }); n != 2 {
		t.Fatalf("audited %d bad codes, want 2", n)
	}
}

func TestDisableTOTPLocksLikeSignIn(t *testing.T) {
	s := newService(t)
	sec := enableTOTP(t, s)
	ctx := context.Background()
	var err error
	for range DefaultLockout.IPMaxFailures {
		err = s.svc.DisableTOTP(ctx, s.op.ID, secret.NewValue("wrong password!"), "000000", reauthIP)
	}
	var locked *LockedError
	if !errors.As(err, &locked) {
		t.Fatalf("last guess = %v, want locked", err)
	}
	code := totpCode(sec, totpStep(s.clock.Now()))
	if err := s.svc.DisableTOTP(ctx, s.op.ID, secret.NewValue(goodPassword), code, reauthIP); !errors.As(err, &locked) {
		t.Fatalf("during the lock = %v, want locked", err)
	}
}
