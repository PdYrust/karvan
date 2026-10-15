package auth

import (
	"context"
	"testing"
	"time"
)

func newLimiter(f *fixture) *Limiter {
	l := NewLimiter(f.db, DefaultLockout)
	l.now = f.clock.Now
	return l
}

func failN(t *testing.T, l *Limiter, n int, ip, account string) time.Duration {
	t.Helper()
	var last time.Duration
	for range n {
		d, err := l.Fail(context.Background(), ip, account)
		if err != nil {
			t.Fatal(err)
		}
		last = d
	}
	return last
}

func wait(t *testing.T, l *Limiter, ip, account string) time.Duration {
	t.Helper()
	d, err := l.Check(context.Background(), ip, account)
	if err != nil {
		t.Fatal(err)
	}
	return d
}

func TestIPLocksAfterFiveFailures(t *testing.T) {
	f := newFixture(t)
	l := newLimiter(f)
	if d := failN(t, l, 4, "203.0.113.9", "a"); d != 0 {
		t.Fatalf("locked after 4 failures for %v", d)
	}
	if d := failN(t, l, 1, "203.0.113.9", "b"); d != 15*time.Minute {
		t.Fatalf("5th failure lock = %v, want 15m", d)
	}
	if d := wait(t, l, "203.0.113.9", "someone-else"); d != 15*time.Minute {
		t.Fatalf("IP wait = %v", d)
	}
	if d := wait(t, l, "198.51.100.1", "c"); d != 0 {
		t.Fatalf("other IP is locked for %v", d)
	}
	f.clock.Advance(15 * time.Minute)
	if d := wait(t, l, "203.0.113.9", "a"); d != 0 {
		t.Fatalf("still locked after the lock ended: %v", d)
	}
}

func TestFailuresOutsideTheWindowDoNotAddUp(t *testing.T) {
	f := newFixture(t)
	l := newLimiter(f)
	failN(t, l, 4, "203.0.113.9", "a")
	f.clock.Advance(16 * time.Minute)
	if d := failN(t, l, 4, "203.0.113.9", "a"); d != 0 {
		t.Fatalf("old failures counted: lock %v", d)
	}
}

func TestAccountLocksGrowAcrossIPs(t *testing.T) {
	f := newFixture(t)
	l := newLimiter(f)
	ips := func(round int) func(i int) string {
		return func(i int) string { return "10.0." + string(rune('0'+round)) + "." + string(rune('a'+i)) }
	}
	for round, want := range []time.Duration{time.Minute, 5 * time.Minute, 15 * time.Minute, time.Hour, time.Hour} {
		ip := ips(round)
		var lock time.Duration
		for i := range 10 {
			d, err := l.Fail(context.Background(), ip(i), "owner")
			if err != nil {
				t.Fatal(err)
			}
			lock = d
		}
		if lock != want {
			t.Fatalf("round %d lock = %v, want %v", round, lock, want)
		}
		f.clock.Advance(want)
	}
}

func TestAccountLevelIsForgivenAfterAQuietDay(t *testing.T) {
	f := newFixture(t)
	l := newLimiter(f)
	// One failure per IP, so only the account counter can lock.
	failFromManyIPs := func(prefix string) time.Duration {
		var last time.Duration
		for i := range 10 {
			last = failN(t, l, 1, prefix+string(rune('a'+i)), "owner")
		}
		return last
	}
	failFromManyIPs("first-")
	f.clock.Advance(25 * time.Hour)
	if d := failFromManyIPs("second-"); d != time.Minute {
		t.Fatalf("lock after a quiet day = %v, want the first level", d)
	}
}

func TestSucceedClearsCounters(t *testing.T) {
	f := newFixture(t)
	l := newLimiter(f)
	failN(t, l, 4, "203.0.113.9", "owner")
	if err := l.Succeed(context.Background(), "203.0.113.9", "owner"); err != nil {
		t.Fatal(err)
	}
	if d := failN(t, l, 4, "203.0.113.9", "owner"); d != 0 {
		t.Fatalf("counters not cleared: lock %v", d)
	}
}

func TestPruneKeepsActiveLocks(t *testing.T) {
	f := newFixture(t)
	l := newLimiter(f)
	failN(t, l, 1, "203.0.113.1", "old")
	f.clock.Advance(25 * time.Hour)
	failN(t, l, 5, "203.0.113.2", "fresh")
	if err := l.Prune(context.Background()); err != nil {
		t.Fatal(err)
	}
	var n int
	_ = f.db.QueryRow(`SELECT count(*) FROM login_failures`).Scan(&n)
	if n != 2 {
		t.Fatalf("rows after prune = %d, want 2 (fresh ip and account)", n)
	}
}
