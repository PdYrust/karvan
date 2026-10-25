package auth

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestSessionListShowsLiveSessionsWithOpaqueIDs(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	s := newSessions(f)
	o, other := f.createOperator(t, "owner"), f.createOperator(t, "second")
	old, oldSess, _ := s.Create(ctx, o.ID, "198.51.100.1", "first")
	f.clock.Advance(time.Minute)
	_, newSess, _ := s.Create(ctx, o.ID, "198.51.100.2", "second")
	if _, _, err := s.Create(ctx, other.ID, "198.51.100.3", "not theirs"); err != nil {
		t.Fatal(err)
	}

	list, err := s.List(ctx, o.ID)
	if err != nil || len(list) != 2 {
		t.Fatalf("List = %+v, %v", list, err)
	}
	if list[0].ID != newSess.ID || list[1].ID != oldSess.ID || list[1].IP != "198.51.100.1" || list[0].UserAgent != "second" {
		t.Fatalf("order or fields: %+v", list)
	}
	looked, _ := s.Peek(ctx, old)
	if looked.ID != oldSess.ID || strings.Contains(old, oldSess.ID) || len(oldSess.ID) != 16 {
		t.Fatalf("id %q of token %q, lookup id %q", oldSess.ID, old, looked.ID)
	}

	f.clock.Advance(IdleTimeout - 30*time.Second)
	if list, _ := s.List(ctx, o.ID); len(list) != 1 || list[0].ID != newSess.ID {
		t.Fatalf("idle session still listed: %+v", list)
	}
}

func TestSessionDeleteOthersKeepsThisOne(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	s := newSessions(f)
	o, other := f.createOperator(t, "owner"), f.createOperator(t, "second")
	keep, _, _ := s.Create(ctx, o.ID, "", "")
	gone, _, _ := s.Create(ctx, o.ID, "", "")
	theirs, _, _ := s.Create(ctx, other.ID, "", "")

	n, err := s.DeleteOthers(ctx, o.ID, keep)
	if err != nil || n != 1 {
		t.Fatalf("DeleteOthers = %d, %v", n, err)
	}
	if _, err := s.Lookup(ctx, gone); !errors.Is(err, ErrNoSession) {
		t.Fatal("other session survived")
	}
	for _, tok := range []string{keep, theirs} {
		if _, err := s.Lookup(ctx, tok); err != nil {
			t.Fatalf("session ended that should stay: %v", err)
		}
	}
}

func TestSessionDeleteByIDOnlyTouchesTheOperatorsOwn(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	s := newSessions(f)
	o, other := f.createOperator(t, "owner"), f.createOperator(t, "second")
	mine, mineSess, _ := s.Create(ctx, o.ID, "", "")
	theirs, theirSess, _ := s.Create(ctx, other.ID, "", "")

	if ok, err := s.DeleteByID(ctx, o.ID, theirSess.ID); ok || err != nil {
		t.Fatalf("ended another operator's session: %v, %v", ok, err)
	}
	if ok, err := s.DeleteByID(ctx, o.ID, "unknown-id-here0"); ok || err != nil {
		t.Fatalf("unknown id: %v, %v", ok, err)
	}
	if ok, err := s.DeleteByID(ctx, o.ID, mineSess.ID); !ok || err != nil {
		t.Fatalf("own session: %v, %v", ok, err)
	}
	if _, err := s.Lookup(ctx, mine); !errors.Is(err, ErrNoSession) {
		t.Fatal("deleted session still valid")
	}
	if _, err := s.Lookup(ctx, theirs); err != nil {
		t.Fatalf("other operator's session ended: %v", err)
	}
}
