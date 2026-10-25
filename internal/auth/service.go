package auth

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/PdYrust/karvan/internal/audit"
	"github.com/PdYrust/karvan/internal/secret"
)

// Sign-in outcomes the web panel shows to the operator.
var (
	ErrInvalidCredentials = errors.New("wrong username or password")
	ErrTOTPRequired       = errors.New("two-step code required")
	ErrTOTPInvalid        = errors.New("wrong two-step code")
	ErrBusy               = errors.New("too many sign-ins at once; try again shortly")
)

const (
	// Each argon2id run takes 64 MiB, so only a few may run at once.
	hashSlots = 2
	hashWait  = 10 * time.Second
)

// LockedError means sign-in is paused for RetryAfter.
type LockedError struct{ RetryAfter time.Duration }

func (e *LockedError) Error() string {
	return fmt.Sprintf("too many failed sign-ins; try again in %s", e.RetryAfter.Round(time.Second))
}

// Service runs sign-in, sign-out and two-step setup, and audits each step.
type Service struct {
	Operators *Operators
	Sessions  *Sessions
	Limiter   *Limiter
	Audit     *audit.Log
	now       func() time.Time
	hashes    chan struct{}
}

// NewService wires a Service.
func NewService(ops *Operators, sessions *Sessions, limiter *Limiter, log *audit.Log) *Service {
	return &Service{Operators: ops, Sessions: sessions, Limiter: limiter, Audit: log, now: time.Now,
		hashes: make(chan struct{}, hashSlots)}
}

// hashSlot waits for one of the few password-hash slots. The lockout check
// runs after it, so a burst of parallel guesses is checked one slot at a
// time instead of all passing the check before any failure is counted.
func (s *Service) hashSlot(ctx context.Context) (func(), error) {
	ctx, cancel := context.WithTimeout(ctx, hashWait)
	defer cancel()
	select {
	case s.hashes <- struct{}{}:
		return func() { <-s.hashes }, nil
	case <-ctx.Done():
		return nil, ErrBusy
	}
}

// LoginRequest is what the sign-in form sends, plus where it came from.
type LoginRequest struct {
	Username  string
	Password  secret.Value
	TOTPCode  string
	IP        string
	UserAgent string
}

// LoginResult is a started session.
type LoginResult struct {
	Token    string
	Operator Operator
	Session  Session
}

// dummyHash lets a sign-in for an unknown username cost the same as a real one.
var dummyHash = sync.OnceValue(func() string {
	h, err := HashPassword("placeholder used only to equalize timing")
	if err != nil {
		return ""
	}
	return h
})

// Login checks the credentials and starts a session.
func (s *Service) Login(ctx context.Context, req LoginRequest) (LoginResult, error) {
	account, err := NormalizeUsername(req.Username)
	if err != nil {
		account = "" // still counted against the IP
	}
	release, err := s.hashSlot(ctx)
	if err != nil {
		return LoginResult{}, err
	}
	defer release()
	if wait, err := s.Limiter.Check(ctx, req.IP, account); err != nil {
		return LoginResult{}, err
	} else if wait > 0 {
		return LoginResult{}, &LockedError{RetryAfter: wait}
	}

	op, err := s.checkPassword(ctx, account, req.Password)
	if err != nil {
		return LoginResult{}, s.fail(ctx, req, account, op, "bad_credentials", err)
	}
	if op.TOTPEnabled {
		if req.TOTPCode == "" {
			return LoginResult{}, ErrTOTPRequired
		}
		if err := s.checkTOTP(ctx, op.ID, req.TOTPCode); err != nil {
			return LoginResult{}, s.fail(ctx, req, account, op, "bad_totp", err)
		}
	}

	if err := s.Limiter.Succeed(ctx, req.IP, account); err != nil {
		return LoginResult{}, err
	}
	token, sess, err := s.Sessions.Create(ctx, op.ID, req.IP, req.UserAgent)
	if err != nil {
		return LoginResult{}, err
	}
	if err := s.record(ctx, op.Username, req.IP, audit.ActionLogin, audit.Success, nil); err != nil {
		return LoginResult{}, err
	}
	return LoginResult{Token: token, Operator: op, Session: sess}, nil
}

func (s *Service) checkPassword(ctx context.Context, account string, pw secret.Value) (Operator, error) {
	op, err := s.Operators.ByUsername(ctx, account)
	if errors.Is(err, ErrNotFound) {
		_, _ = VerifyPassword(dummyHash(), pw.Reveal())
		return Operator{}, ErrInvalidCredentials
	}
	if err != nil {
		return Operator{}, err
	}
	ok, err := VerifyPassword(op.PasswordHash, pw.Reveal())
	if err != nil {
		return op, fmt.Errorf("verify password of %s: %w", op.Username, err)
	}
	if !ok {
		return op, ErrInvalidCredentials
	}
	return op, nil
}

func (s *Service) checkTOTP(ctx context.Context, operatorID int64, code string) error {
	sec, last, err := s.Operators.TOTP(ctx, operatorID)
	if err != nil {
		return err
	}
	step, ok := VerifyTOTP(sec, code, s.now(), last)
	if !ok {
		return ErrTOTPInvalid
	}
	if used, err := s.Operators.UseTOTPStep(ctx, operatorID, step); err != nil {
		return err
	} else if !used {
		return ErrTOTPInvalid
	}
	return nil
}

// fail counts the failure, audits it and returns the error to show.
// Unknown usernames are audited as "unknown" so a password typed into the
// username field never lands in the log.
func (s *Service) fail(ctx context.Context, req LoginRequest, account string, op Operator, reason string, cause error) error {
	if !errors.Is(cause, ErrInvalidCredentials) && !errors.Is(cause, ErrTOTPInvalid) {
		return cause
	}
	actor := op.Username
	if actor == "" {
		actor = "unknown"
	}
	return s.failed(ctx, req.IP, account, actor, audit.ActionLogin, reason, cause)
}

// Logout ends the session for token.
func (s *Service) Logout(ctx context.Context, token, username, ip string) error {
	if err := s.Sessions.Delete(ctx, token); err != nil {
		return err
	}
	return s.record(ctx, username, ip, audit.ActionLogout, audit.Success, nil)
}

func (s *Service) record(ctx context.Context, actor, ip, action string, outcome audit.Outcome, detail map[string]string) error {
	return s.Audit.Record(ctx, audit.Event{Actor: actor, IP: ip, Action: action, Outcome: outcome, Detail: detail})
}
