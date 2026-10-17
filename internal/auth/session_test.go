package auth

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

func newSessions(f *fixture) *Sessions {
	s := NewSessions(f.db)
	s.now = f.clock.Now
	return s
}

func TestSessionCreateAndLookup(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	o := f.createOperator(t, "owner")
	s := newSessions(f)

	token, created, err := s.Create(ctx, o.ID, "198.51.100.7", strings.Repeat("u", 400))
	if err != nil {
		t.Fatal(err)
	}
	if len(token) < 43 || len(created.UserAgent) != maxUserAgent {
		t.Fatalf("token %q, ua len %d", token, len(created.UserAgent))
	}
	got, err := s.Lookup(ctx, token)
	if err != nil || got.OperatorID != o.ID || got.IP != "198.51.100.7" {
		t.Fatalf("Lookup = %+v, %v", got, err)
	}
	if _, err := s.Lookup(ctx, token+"x"); !errors.Is(err, ErrNoSession) {
		t.Fatalf("wrong token err = %v", err)
	}
	if _, err := s.Lookup(ctx, ""); !errors.Is(err, ErrNoSession) {
		t.Fatalf("empty token err = %v", err)
	}
}

func TestSessionTokenIsNotStored(t *testing.T) {
	f := newFixture(t)
	s := newSessions(f)
	token, _, err := s.Create(context.Background(), f.createOperator(t, "owner").ID, "", "")
	if err != nil {
		t.Fatal(err)
	}
	var stored []byte
	if err := f.db.QueryRow(`SELECT token_hash FROM sessions`).Scan(&stored); err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(stored, []byte(token)) || len(stored) != 32 {
		t.Fatal("session table holds the raw token")
	}
}

func TestSessionIdleTimeout(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	s := newSessions(f)
	token, _, _ := s.Create(ctx, f.createOperator(t, "owner").ID, "", "")

	f.clock.Advance(IdleTimeout - time.Minute)
	if _, err := s.Lookup(ctx, token); err != nil {
		t.Fatalf("active session ended early: %v", err)
	}
	f.clock.Advance(IdleTimeout - time.Minute)
	if _, err := s.Lookup(ctx, token); err != nil {
		t.Fatalf("use did not extend the idle timer: %v", err)
	}
	f.clock.Advance(IdleTimeout)
	if _, err := s.Lookup(ctx, token); !errors.Is(err, ErrNoSession) {
		t.Fatalf("idle session err = %v, want ErrNoSession", err)
	}
}

func TestSessionPeekDoesNotKeepItAlive(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	s := newSessions(f)
	token, _, err := s.Create(ctx, f.createOperator(t, "owner").ID, "", "")
	if err != nil {
		t.Fatal(err)
	}
	// A dashboard polling every few minutes, with nobody at the keyboard.
	for i := 1; i <= 5; i++ {
		f.clock.Advance(5 * time.Minute)
		if _, err := s.Peek(ctx, token); err != nil {
			t.Fatalf("peek after %d minutes: %v", 5*i, err)
		}
	}
	f.clock.Advance(4 * time.Minute)
	if _, err := s.Peek(ctx, token); err != nil {
		t.Fatalf("peek just before the idle timeout: %v", err)
	}
	f.clock.Advance(2 * time.Minute)
	if _, err := s.Peek(ctx, token); !errors.Is(err, ErrNoSession) {
		t.Fatalf("polling kept the session alive: %v", err)
	}
}

func TestSessionAbsoluteTimeout(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	s := newSessions(f)
	token, _, _ := s.Create(ctx, f.createOperator(t, "owner").ID, "", "")
	for elapsed := time.Duration(0); elapsed < AbsoluteTimeout-10*time.Minute; elapsed += 10 * time.Minute {
		f.clock.Advance(10 * time.Minute)
		if _, err := s.Lookup(ctx, token); err != nil {
			t.Fatalf("session ended at %v: %v", elapsed, err)
		}
	}
	f.clock.Advance(10 * time.Minute)
	if _, err := s.Lookup(ctx, token); !errors.Is(err, ErrNoSession) {
		t.Fatalf("err = %v after the absolute timeout", err)
	}
}

func TestSessionDeleteAndPurge(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	s := newSessions(f)
	o := f.createOperator(t, "owner")
	a, _, _ := s.Create(ctx, o.ID, "", "")
	b, _, _ := s.Create(ctx, o.ID, "", "")
	c, _, _ := s.Create(ctx, o.ID, "", "")

	if err := s.Delete(ctx, a); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Lookup(ctx, a); !errors.Is(err, ErrNoSession) {
		t.Fatal("deleted session still valid")
	}
	if err := s.DeleteForOperator(ctx, o.ID); err != nil {
		t.Fatal(err)
	}
	for _, tok := range []string{b, c} {
		if _, err := s.Lookup(ctx, tok); !errors.Is(err, ErrNoSession) {
			t.Fatal("session survived DeleteForOperator")
		}
	}

	d, _, _ := s.Create(ctx, o.ID, "", "")
	f.clock.Advance(IdleTimeout)
	if err := s.PurgeExpired(ctx); err != nil {
		t.Fatal(err)
	}
	var n int
	_ = f.db.QueryRow(`SELECT count(*) FROM sessions`).Scan(&n)
	if n != 0 {
		t.Fatalf("%d sessions left after purge (token %s)", n, d[:4])
	}
}

func TestCSRFTokenIsBoundToSession(t *testing.T) {
	a, b := "session-token-a", "session-token-b"
	first, again := CSRFToken(a), CSRFToken(a)
	if first == CSRFToken(b) || first != again {
		t.Fatal("CSRF token not a stable function of the session")
	}
	if !ValidCSRF(a, CSRFToken(a)) {
		t.Fatal("own token refused")
	}
	for _, bad := range []string{"", CSRFToken(b), CSRFToken(a) + "x"} {
		if ValidCSRF(a, bad) {
			t.Errorf("token %q accepted", bad)
		}
	}
	if ValidCSRF("", CSRFToken("")) {
		t.Error("empty session accepted")
	}
}
