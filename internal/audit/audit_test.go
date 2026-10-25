package audit

import (
	"context"
	"testing"
	"time"

	"github.com/PdYrust/karvan/internal/store"
)

func newLog(t *testing.T) *Log {
	t.Helper()
	db, err := store.Open(context.Background(), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	return New(db)
}

func TestRecordAndListNewestFirst(t *testing.T) {
	l := newLog(t)
	ctx := context.Background()
	base := time.Date(2026, 10, 15, 9, 0, 0, 0, time.UTC)
	events := []Event{
		{At: base, Actor: "owner", IP: "198.51.100.7", Action: ActionLogin, Outcome: Success},
		{At: base.Add(time.Minute), Actor: "owner", IP: "198.51.100.7", Action: ActionLogin, Outcome: Failure, Detail: map[string]string{"reason": "bad_password"}},
		{At: base.Add(2 * time.Minute), Actor: ActorCLI, Action: ActionReset, Outcome: Success},
	}
	for _, e := range events {
		if err := l.Record(ctx, e); err != nil {
			t.Fatal(err)
		}
	}

	all, err := l.List(ctx, Filter{})
	if err != nil || len(all) != 3 {
		t.Fatalf("List = %d events, %v", len(all), err)
	}
	if all[0].Action != ActionReset || all[2].At.UnixMilli() != base.UnixMilli() {
		t.Errorf("order wrong: %+v", all)
	}
	if all[1].Detail["reason"] != "bad_password" {
		t.Errorf("detail = %v", all[1].Detail)
	}
}

func TestListFilters(t *testing.T) {
	l := newLog(t)
	ctx := context.Background()
	now := time.Now()
	for i, o := range []Outcome{Success, Failure, Success} {
		if err := l.Record(ctx, Event{At: now.Add(time.Duration(i) * time.Second), Actor: "owner", Action: ActionLogin, Outcome: o, IP: "203.0.113.1"}); err != nil {
			t.Fatal(err)
		}
	}
	if err := l.Record(ctx, Event{Actor: "other", Action: ActionLogout, Outcome: Success}); err != nil {
		t.Fatal(err)
	}

	got, err := l.List(ctx, Filter{Actor: "owner", Action: ActionLogin, Outcome: Success, SkipNewest: 1})
	if err != nil || len(got) != 1 || !got[0].At.Equal(now.Truncate(time.Millisecond)) {
		t.Fatalf("got %+v, %v", got, err)
	}
	limited, _ := l.List(ctx, Filter{Limit: 2})
	if len(limited) != 2 {
		t.Fatalf("limit ignored: %d", len(limited))
	}
}

func TestListPagesByID(t *testing.T) {
	l := newLog(t)
	ctx := context.Background()
	now := time.Now()
	for i := range 5 {
		// The clock jumps back for the last two; the order stays the ids'.
		at := now.Add(time.Duration(i) * time.Minute)
		if i >= 3 {
			at = now.Add(-time.Hour)
		}
		if err := l.Record(ctx, Event{At: at, Actor: "owner", Action: ActionLogin, Outcome: Success}); err != nil {
			t.Fatal(err)
		}
	}
	first, err := l.List(ctx, Filter{Limit: 2})
	if err != nil || len(first) != 2 || first[0].ID != 5 || first[1].ID != 4 {
		t.Fatalf("first page = %+v, %v", first, err)
	}
	next, _ := l.List(ctx, Filter{Limit: 2, Before: first[1].ID})
	last, _ := l.List(ctx, Filter{Limit: 2, Before: next[1].ID})
	if len(next) != 2 || next[0].ID != 3 || next[1].ID != 2 || len(last) != 1 || last[0].ID != 1 {
		t.Fatalf("next = %+v, last = %+v", next, last)
	}
}

func TestRecordDefaultsTimeToNow(t *testing.T) {
	l := newLog(t)
	fixed := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	l.now = func() time.Time { return fixed }
	if err := l.Record(context.Background(), Event{Actor: "a", Action: ActionLogout, Outcome: Success}); err != nil {
		t.Fatal(err)
	}
	got, _ := l.List(context.Background(), Filter{})
	if !got[0].At.Equal(fixed) {
		t.Fatalf("At = %v, want %v", got[0].At, fixed)
	}
}

func TestRecordRejectsUnknownOutcome(t *testing.T) {
	if err := newLog(t).Record(context.Background(), Event{Actor: "a", Action: "x", Outcome: "maybe"}); err == nil {
		t.Fatal("Record accepted an unknown outcome")
	}
}
