package run

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/PdYrust/karvan/internal/panel"
)

func jobRows(t *testing.T, s *Service, id string) (jobs, events int) {
	t.Helper()
	if err := s.db.QueryRow("SELECT COUNT(*) FROM jobs WHERE id = ?", id).Scan(&jobs); err != nil {
		t.Fatal(err)
	}
	return jobs, eventCount(t, s, id)
}

func TestDeleteDraftDropsItsLoginAndEvents(t *testing.T) {
	h := newHarness(t, panel.Marzban, panel.PasarGuard, pasarGuardDB(t), 3)
	j := h.job(nil)
	if _, events := jobRows(t, h.svc, j.ID); events == 0 {
		t.Fatal("a checked job should have events from its read")
	}
	if err := h.svc.DeleteDraft(context.Background(), j.ID); err != nil {
		t.Fatal(err)
	}
	j.mu.Lock()
	held := j.source != nil || j.snapshot != nil || !j.login.Password.IsZero() || j.login.Username != ""
	j.mu.Unlock()
	if held {
		t.Error("the deleted draft still holds the old panel's login")
	}
	if jobs, events := jobRows(t, h.svc, j.ID); jobs != 0 || events != 0 {
		t.Errorf("left %d job rows and %d events", jobs, events)
	}
	if _, err := h.svc.JobState(context.Background(), j.ID); !errors.Is(err, ErrNoJob) {
		t.Errorf("JobState after delete: %v", err)
	}
	if err := h.svc.DeleteDraft(context.Background(), j.ID); !errors.Is(err, ErrNoJob) {
		t.Errorf("a second delete: %v", err)
	}
}

func TestDeleteDraftKeepsAJobThatRan(t *testing.T) {
	h := newHarness(t, panel.Marzban, panel.PasarGuard, pasarGuardDB(t), 3)
	j := h.job(nil)
	h.run(j, false)
	if err := h.svc.DeleteDraft(context.Background(), j.ID); !errors.Is(err, ErrNotDeletable) {
		t.Fatalf("delete a done job: %v", err)
	}
	if jobs, _ := jobRows(t, h.svc, j.ID); jobs != 1 {
		t.Error("the done job's record went")
	}
}

func TestDeleteDraftWaitsForARead(t *testing.T) {
	h := newHarness(t, panel.Marzban, panel.PasarGuard, pasarGuardDB(t), 3)
	j := h.created()
	release := h.hold()
	defer release()
	if err := h.svc.Compare(j.ID); err != nil {
		t.Fatal(err)
	}
	if err := h.svc.DeleteDraft(context.Background(), j.ID); !errors.Is(err, ErrBusy) {
		t.Errorf("delete while reading: %v", err)
	}
	release()
	h.svc.Wait()
}

// Jobs from before a restart are only rows: drafts go, the others stay.
func TestDeleteDraftFromBeforeARestart(t *testing.T) {
	now := time.Date(2026, 10, 26, 12, 0, 0, 0, time.UTC)
	s := trimService(t, now)
	_, err := s.db.Exec(`INSERT INTO jobs (id, state, source_kind, source_origin, panel_id, safe_mode, created_at, updated_at)
		VALUES ('draft', 'checked', 'marzban', 'https://old.example', 1, 1, 0, 0)`)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if err := s.DeleteDraft(ctx, "draft"); err != nil {
		t.Fatal(err)
	}
	for id, want := range map[string]error{"draft": ErrNoJob, "old-done": ErrNotDeletable, "old-interrupted": ErrNotDeletable,
		"running": ErrNotDeletable, "unknown": ErrNoJob} {
		if err := s.DeleteDraft(ctx, id); !errors.Is(err, want) {
			t.Errorf("delete %s: %v, want %v", id, err, want)
		}
	}
	if jobs, events := jobRows(t, s, "old-done"); jobs != 1 || events != 2 {
		t.Errorf("old-done kept %d rows and %d events", jobs, events)
	}
}
