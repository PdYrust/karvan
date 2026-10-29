package run

import (
	"context"
	"testing"
	"time"

	"github.com/PdYrust/karvan/internal/store/storetest"
)

// trimService holds one job per case, each with two events, at a fixed now.
func trimService(t *testing.T, now time.Time) *Service {
	t.Helper()
	db, err := storetest.Open(context.Background(), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if _, err := db.Exec("INSERT INTO panels (id, env_path, name, kind, scanned_at) VALUES (1, '/a/.env', 'alpha', 'pasarguard', 0)"); err != nil {
		t.Fatal(err)
	}
	day := int64(24 * time.Hour / time.Millisecond)
	old, recent := now.UnixMilli()-91*day, now.UnixMilli()-89*day
	for _, j := range []struct {
		id, state      string
		ended, updated int64
	}{
		{"old-done", StateDone, old, old},
		{"old-failed", StateFailed, old, old},
		{"old-rolled", StateRolledBack, old, old},
		{"recent-done", StateDone, recent, recent},
		{"old-but-touched", StateDone, old, recent},
		{"old-interrupted", StateInterrupted, old, old},
		{"running", StateRunning, 0, old},
	} {
		_, err := db.Exec(`INSERT INTO jobs (id, state, source_kind, source_origin, panel_id, safe_mode, report, ended_at,
			created_at, updated_at) VALUES (?, ?, 'marzban', 'https://old.example', 1, 1, '{"counts":{}}', ?, ?, ?)`,
			j.id, j.state, j.ended, old, j.updated)
		if err != nil {
			t.Fatal(err)
		}
		for seq := 1; seq <= 2; seq++ {
			if _, err := db.Exec("INSERT INTO job_events (job_id, seq, at, stage, type) VALUES (?, ?, ?, 'run', 'log')", j.id, seq, old); err != nil {
				t.Fatal(err)
			}
		}
	}
	return New(Deps{DB: db, Now: func() time.Time { return now }})
}

func eventCount(t *testing.T, s *Service, job string) int {
	t.Helper()
	var n int
	if err := s.db.QueryRow("SELECT COUNT(*) FROM job_events WHERE job_id = ?", job).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

// Each trimmed job keeps its last event, the end a reader replays.
func TestTrimEventsDropsOnlyLongEndedJobs(t *testing.T) {
	s := trimService(t, time.Date(2026, 10, 25, 12, 0, 0, 0, time.UTC))
	n, err := s.TrimEvents(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if n != 3 {
		t.Fatalf("trimmed %d events, want 3", n)
	}
	want := map[string]int{"old-done": 1, "old-failed": 1, "old-rolled": 1, "recent-done": 2, "old-but-touched": 2, "old-interrupted": 2, "running": 2}
	for job, w := range want {
		if got := eventCount(t, s, job); got != w {
			t.Errorf("%s: %d events, want %d", job, got, w)
		}
	}
}

func TestTrimEventsKeepsReports(t *testing.T) {
	s := trimService(t, time.Date(2026, 10, 25, 12, 0, 0, 0, time.UTC))
	if _, err := s.TrimEvents(context.Background()); err != nil {
		t.Fatal(err)
	}
	var jobs, reports int
	if err := s.db.QueryRow("SELECT COUNT(*), COUNT(NULLIF(report, '')) FROM jobs").Scan(&jobs, &reports); err != nil {
		t.Fatal(err)
	}
	if jobs != 7 || reports != 7 {
		t.Fatalf("jobs %d, reports %d, want 7 and 7", jobs, reports)
	}
	// A second trim finds nothing left to drop.
	if n, err := s.TrimEvents(context.Background()); err != nil || n != 0 {
		t.Fatalf("second trim: %d, %v", n, err)
	}
}

// A job whose events were trimmed numbers its next event after the last
// one it had, so a reader's Last-Event-ID never repeats.
func TestTrimEventsKeepsSequence(t *testing.T) {
	s := trimService(t, time.Date(2026, 10, 25, 12, 0, 0, 0, time.UTC))
	ctx := context.Background()
	if _, err := s.TrimEvents(ctx); err != nil {
		t.Fatal(err)
	}
	if err := s.events.emit(ctx, "old-done", "restore", "log", nil); err != nil {
		t.Fatal(err)
	}
	last, err := s.events.last(ctx, "old-done")
	if err != nil {
		t.Fatal(err)
	}
	if last != 3 {
		t.Fatalf("next seq %d, want 3", last)
	}
}
