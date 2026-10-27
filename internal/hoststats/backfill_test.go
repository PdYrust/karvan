package hoststats

import (
	"context"
	"reflect"
	"testing"
	"time"
)

var restart = time.Date(2026, 10, 27, 12, 20, 10, 0, time.UTC)

// restarted is a sampler that started at 12:20:10 and has two samples, the
// oldest at 12:20:15; now is 12:20:20.
func restarted(t *testing.T, st *Store) (*Sampler, time.Time) {
	t.Helper()
	f := newFixture(t)
	now := restart
	s := &Sampler{Reader: f.r, Store: st, Now: func() time.Time { return now }}
	for i := range 3 {
		f.counters(uint64(i))
		s.Tick(context.Background())
		if i < 2 {
			now = now.Add(5 * time.Second)
		}
	}
	return s, now
}

// putMinutes saves one minute at each given time, with its unix minute as CPU.
func putMinutes(t *testing.T, st *Store, at ...time.Time) {
	t.Helper()
	for _, m := range at {
		if err := st.Put(context.Background(), Minute{TS: m.Unix(), CPU: float64(m.Minute()), MemTotal: 1}); err != nil {
			t.Fatal(err)
		}
	}
}

func minutesFrom(from time.Time, n int, skip ...int) []time.Time {
	out := []time.Time{}
	for i := range n {
		if !contains(skip, i) {
			out = append(out, from.Add(time.Duration(i)*time.Minute))
		}
	}
	return out
}

func contains(xs []int, x int) bool {
	for _, v := range xs {
		if v == x {
			return true
		}
	}
	return false
}

func stamps(ms []Minute) []int64 {
	out := []int64{}
	for _, m := range ms {
		out = append(out, m.TS)
	}
	return out
}

func unix(ts []time.Time) []int64 {
	out := []int64{}
	for _, t := range ts {
		out = append(out, t.Unix())
	}
	return out
}

func TestBackfillBounds(t *testing.T) {
	now := time.Date(2026, 10, 27, 12, 20, 20, 0, time.UTC)
	oldest := time.Date(2026, 10, 27, 12, 18, 45, 0, time.UTC)
	from, to := backfillBounds(&oldest, now)
	// The window starts at 12:05:20, inside the 12:05 minute; the oldest
	// sample is in the 12:18 minute, which is left to the samples.
	if wantFrom, wantTo := time.Date(2026, 10, 27, 12, 5, 0, 0, time.UTC).Unix(), time.Date(2026, 10, 27, 12, 18, 0, 0, time.UTC).Unix(); from != wantFrom || to != wantTo {
		t.Fatalf("bounds %d..%d, want %d..%d", from, to, wantFrom, wantTo)
	}
	if _, to := backfillBounds(nil, now); to != time.Date(2026, 10, 27, 12, 20, 0, 0, time.UTC).Unix() {
		t.Fatalf("no samples: up to %d, want the minute of now", to)
	}
	if LiveWindow != 15*time.Minute {
		t.Fatalf("live window %v", LiveWindow)
	}
}

func TestBackfillStopsBeforeTheSamples(t *testing.T) {
	st := openStore(t)
	// 12:00 to 12:21: before the window, the whole window, and the minutes
	// the samples hold (12:20 left half done by the last run, and a stray later one).
	putMinutes(t, st, minutesFrom(restart.Truncate(time.Hour), 22)...)
	s, now := restarted(t, st)
	got, err := s.Backfill(context.Background(), now)
	if err != nil {
		t.Fatal(err)
	}
	want := unix(minutesFrom(time.Date(2026, 10, 27, 12, 5, 0, 0, time.UTC), 15))
	if !reflect.DeepEqual(stamps(got), want) {
		t.Fatalf("minutes %v, want %v", stamps(got), want)
	}
	oldest := s.Since(0)[0].At
	if last := got[len(got)-1]; last.TS+60 > oldest.Unix() || last.Minutes != 1 || last.CPU != 19 {
		t.Fatalf("last minute %+v overlaps the oldest sample at %v", last, oldest)
	}
}

func TestBackfillKeepsGaps(t *testing.T) {
	st := openStore(t)
	// The window's 15 minutes, without 12:10 and 12:11 (Karvan was down).
	putMinutes(t, st, minutesFrom(time.Date(2026, 10, 27, 12, 5, 0, 0, time.UTC), 15, 5, 6)...)
	s, now := restarted(t, st)
	got, err := s.Backfill(context.Background(), now)
	if err != nil {
		t.Fatal(err)
	}
	want := unix(minutesFrom(time.Date(2026, 10, 27, 12, 5, 0, 0, time.UTC), 15, 5, 6))
	if !reflect.DeepEqual(stamps(got), want) {
		t.Fatalf("minutes %v, want %v (no minute made up)", stamps(got), want)
	}
}

func TestBackfillWithNothingSaved(t *testing.T) {
	s, now := restarted(t, openStore(t))
	got, err := s.Backfill(context.Background(), now)
	if err != nil || got == nil || len(got) != 0 {
		t.Fatalf("empty database: %v %v", got, err)
	}
	got, err = (&Sampler{}).Backfill(context.Background(), now)
	if err != nil || got == nil || len(got) != 0 {
		t.Fatalf("no store: %v %v", got, err)
	}
}

func TestBackfillWithoutSamples(t *testing.T) {
	st := openStore(t)
	putMinutes(t, st, minutesFrom(time.Date(2026, 10, 27, 12, 18, 0, 0, time.UTC), 3)...)
	s := &Sampler{Store: st}
	got, err := s.Backfill(context.Background(), time.Date(2026, 10, 27, 12, 20, 30, 0, time.UTC))
	// 12:20 is the minute now is in: the first samples will cover it.
	if want := unix(minutesFrom(time.Date(2026, 10, 27, 12, 18, 0, 0, time.UTC), 2)); err != nil || !reflect.DeepEqual(stamps(got), want) {
		t.Fatalf("minutes %v %v, want %v", stamps(got), err, want)
	}
}
