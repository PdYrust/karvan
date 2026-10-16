package auth

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/PdYrust/karvan/internal/audit"
	"github.com/PdYrust/karvan/internal/secret"
)

const goodPassword = "correct horse battery"

var goodHash = sync.OnceValue(func() string {
	h, err := HashPassword(goodPassword)
	if err != nil {
		panic(err)
	}
	return h
})

type serviceFixture struct {
	*fixture
	svc *Service
	op  Operator
}

func newService(t *testing.T) *serviceFixture {
	t.Helper()
	f := newFixture(t)
	op, err := f.operators.Create(context.Background(), "owner", goodHash())
	if err != nil {
		t.Fatal(err)
	}
	limiter := NewLimiter(f.db, DefaultLockout)
	limiter.now = f.clock.Now
	sessions := NewSessions(f.db)
	sessions.now = f.clock.Now
	svc := NewService(f.operators, sessions, limiter, audit.New(f.db))
	svc.now = f.clock.Now
	return &serviceFixture{fixture: f, svc: svc, op: op}
}

func (s *serviceFixture) login(username, password, code string) (LoginResult, error) {
	return s.svc.Login(context.Background(), LoginRequest{
		Username: username, Password: secret.NewValue(password), TOTPCode: code, IP: "198.51.100.7", UserAgent: "test",
	})
}

func (s *serviceFixture) events(t *testing.T) []audit.Event {
	t.Helper()
	ev, err := s.svc.Audit.List(context.Background(), audit.Filter{})
	if err != nil {
		t.Fatal(err)
	}
	return ev
}

func TestLoginSucceedsAndAudits(t *testing.T) {
	s := newService(t)
	res, err := s.login(" Owner ", goodPassword, "")
	if err != nil {
		t.Fatalf("Login: %v", err)
	}
	if res.Token == "" || res.Operator.ID != s.op.ID {
		t.Fatalf("result = %+v", res)
	}
	if _, err := s.svc.Sessions.Lookup(context.Background(), res.Token); err != nil {
		t.Fatalf("session not usable: %v", err)
	}
	ev := s.events(t)
	if len(ev) != 1 || ev[0].Action != audit.ActionLogin || ev[0].Outcome != audit.Success || ev[0].IP != "198.51.100.7" {
		t.Fatalf("audit = %+v", ev)
	}
}

func TestLoginFailuresLookTheSame(t *testing.T) {
	s := newService(t)
	_, wrongPw := s.login("owner", "wrong password!", "")
	_, unknown := s.login("nobody", goodPassword, "")
	_, invalid := s.login("x", goodPassword, "")
	for _, err := range []error{wrongPw, unknown, invalid} {
		if !errors.Is(err, ErrInvalidCredentials) {
			t.Errorf("err = %v, want ErrInvalidCredentials", err)
		}
	}
	for _, e := range s.events(t) {
		if e.Outcome != audit.Failure {
			t.Errorf("event %+v, want failure", e)
		}
		if e.Actor != "owner" && e.Actor != "unknown" {
			t.Errorf("actor %q leaks the typed username", e.Actor)
		}
	}
}

func TestLoginLocksAfterRepeatedFailures(t *testing.T) {
	s := newService(t)
	for range DefaultLockout.IPMaxFailures {
		if _, err := s.login("owner", "wrong password!", ""); !errors.Is(err, ErrInvalidCredentials) {
			t.Fatalf("err = %v", err)
		}
	}
	_, err := s.login("owner", goodPassword, "")
	var locked *LockedError
	if !errors.As(err, &locked) || locked.RetryAfter != DefaultLockout.IPLock {
		t.Fatalf("err = %v, want LockedError for %v", err, DefaultLockout.IPLock)
	}
	if locked.Error() == "" {
		t.Fatal("empty lock message")
	}
	var sawLockout bool
	for _, e := range s.events(t) {
		sawLockout = sawLockout || e.Action == audit.ActionLockout
	}
	if !sawLockout {
		t.Fatal("lockout not audited")
	}
	s.clock.Advance(DefaultLockout.IPLock)
	if _, err := s.login("owner", goodPassword, ""); err != nil {
		t.Fatalf("login after lock ended: %v", err)
	}
}

func enableTOTP(t *testing.T, s *serviceFixture) []byte {
	t.Helper()
	ctx := context.Background()
	if _, err := s.svc.BeginTOTP(ctx, s.op.ID); err != nil {
		t.Fatal(err)
	}
	sec, err := s.operators.PendingTOTP(ctx, s.op.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.svc.ConfirmTOTP(ctx, s.op.ID, "000000", ""); !errors.Is(err, ErrTOTPInvalid) {
		t.Fatalf("wrong confirm code err = %v", err)
	}
	if err := s.svc.ConfirmTOTP(ctx, s.op.ID, totpCode(sec, totpStep(s.clock.Now())), ""); err != nil {
		t.Fatal(err)
	}
	s.clock.Advance(totpPeriod * time.Second)
	return sec
}

func TestLoginWithTOTP(t *testing.T) {
	s := newService(t)
	sec := enableTOTP(t, s)

	if _, err := s.login("owner", goodPassword, ""); !errors.Is(err, ErrTOTPRequired) {
		t.Fatalf("no code err = %v", err)
	}
	if _, err := s.login("owner", goodPassword, "123456"); !errors.Is(err, ErrTOTPInvalid) {
		t.Fatalf("bad code err = %v", err)
	}
	code := totpCode(sec, totpStep(s.clock.Now()))
	if _, err := s.login("owner", goodPassword, code); err != nil {
		t.Fatalf("good code: %v", err)
	}
	if _, err := s.login("owner", goodPassword, code); !errors.Is(err, ErrTOTPInvalid) {
		t.Fatalf("replayed code err = %v", err)
	}
}

func TestTOTPCannotBeReplacedWhileOn(t *testing.T) {
	s := newService(t)
	enableTOTP(t, s)
	ctx := context.Background()
	if _, err := s.svc.BeginTOTP(ctx, s.op.ID); !errors.Is(err, ErrTOTPAlreadyActive) {
		t.Fatalf("BeginTOTP while on: %v", err)
	}
	if err := s.svc.ConfirmTOTP(ctx, s.op.ID, "123456", ""); !errors.Is(err, ErrTOTPAlreadyActive) {
		t.Fatalf("ConfirmTOTP while on: %v", err)
	}
}

func TestDisableTOTPWhenOff(t *testing.T) {
	s := newService(t)
	err := s.svc.DisableTOTP(context.Background(), s.op.ID, secret.NewValue(goodPassword), "123456", "")
	if !errors.Is(err, ErrTOTPNotEnabled) {
		t.Fatalf("err = %v, want ErrTOTPNotEnabled", err)
	}
}

func TestDisableTOTPNeedsPasswordAndCode(t *testing.T) {
	s := newService(t)
	sec := enableTOTP(t, s)
	ctx := context.Background()
	code := totpCode(sec, totpStep(s.clock.Now()))

	if err := s.svc.DisableTOTP(ctx, s.op.ID, secret.NewValue("wrong password!"), code, ""); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("wrong password err = %v", err)
	}
	if err := s.svc.DisableTOTP(ctx, s.op.ID, secret.NewValue(goodPassword), "000000", ""); !errors.Is(err, ErrTOTPInvalid) {
		t.Fatalf("wrong code err = %v", err)
	}
	if err := s.svc.DisableTOTP(ctx, s.op.ID, secret.NewValue(goodPassword), code, ""); err != nil {
		t.Fatal(err)
	}
	var failures int
	for _, e := range s.events(t) {
		if e.Action == audit.ActionTOTPDisable && e.Outcome == audit.Failure {
			failures++
		}
	}
	if failures != 2 {
		t.Fatalf("audited %d failed disables, want 2", failures)
	}
	if _, err := s.login("owner", goodPassword, ""); err != nil {
		t.Fatalf("login after disabling TOTP: %v", err)
	}
}

func TestLogoutEndsSession(t *testing.T) {
	s := newService(t)
	res, err := s.login("owner", goodPassword, "")
	if err != nil {
		t.Fatal(err)
	}
	if err := s.svc.Logout(context.Background(), res.Token, "owner", "198.51.100.7"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.svc.Sessions.Lookup(context.Background(), res.Token); !errors.Is(err, ErrNoSession) {
		t.Fatalf("session still valid: %v", err)
	}
	if ev := s.events(t); ev[0].Action != audit.ActionLogout {
		t.Fatalf("newest event = %+v", ev[0])
	}
}
