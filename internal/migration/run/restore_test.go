package run

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/PdYrust/karvan/internal/panel/paneldb"
	"github.com/PdYrust/karvan/internal/panel/paneldb/paneldbtest"
)

// eventsAfter lists a job's events after seq as "stage type word", the
// word being a stage's state, a log's code or done's state.
func eventsAfter(t *testing.T, s *Service, jobID string, after int64) []string {
	t.Helper()
	past, _, stop, err := s.Events(context.Background(), jobID, after)
	if err != nil {
		t.Fatal(err)
	}
	stop()
	out := make([]string, 0, len(past))
	for _, e := range past {
		word, _ := e.Data["state"].(string)
		if code, ok := e.Data["code"].(string); ok {
			word = code
		}
		out = append(out, strings.TrimSpace(e.Stage+" "+e.Type+" "+word))
	}
	return out
}

// doneJob runs a job with Safe Mode on that ends done.
func doneJob(t *testing.T, ctl *fakeControl) (*harness, *Report, string) {
	t.Helper()
	h, tg := backupHarness(t, ctl)
	before := paneldbtest.ChecksumExcept(t, tg)
	rep, err := h.runWith(h.job(nil), RollbackUndo)
	if err != nil || rep.State != StateDone {
		t.Fatalf("run: %v %+v", err, rep)
	}
	return h, rep, before
}

// A restore on request answers once the destination is locked, then the
// job's stream shows the panel going offline, the backup going back and
// the panel coming online, and the job's report is stored again.
func TestStartRestoreReportsOnTheStream(t *testing.T) {
	ctl := &fakeControl{}
	h, rep, before := doneJob(t, ctl)
	res, err := h.svc.StartRestore(context.Background(), rep.BackupID)
	if err != nil {
		t.Fatal(err)
	}
	h.svc.Wait()
	if res.JobID != rep.JobID || res.After == 0 {
		t.Fatalf("answer %+v", res)
	}
	want := []string{"restore stage start", "restore log panel_stopped", "restore log backup_put_back",
		"restore log panel_started", "restore stage done", "done rolled_back"}
	if got := eventsAfter(t, h.svc, rep.JobID, res.After); !slices.Equal(got, want) {
		t.Fatalf("events %q", got)
	}
	if ctl.seq() != "stop,start" || paneldbtest.ChecksumExcept(t, h.tg) != before {
		t.Fatalf("control %q, destination as before: %v", ctl.seq(), paneldbtest.ChecksumExcept(t, h.tg) == before)
	}
	checkRestoredReport(t, h.svc, rep)
}

// checkRestoredReport makes sure the job's stored report keeps the run's
// numbers and adds the restore.
func checkRestoredReport(t *testing.T, s *Service, rep *Report) {
	t.Helper()
	stored, err := s.storedReport(rep.JobID)
	if err != nil || stored == nil {
		t.Fatalf("stored report: %v", err)
	}
	last := stored.Stages[len(stored.Stages)-1]
	if stored.State != StateRolledBack || last.Stage != StageRestore || !last.OK || stored.Counts.Moved != rep.Counts.Moved ||
		!slices.Contains(stored.NotUndoable, UndoLostPanelWrites) || stored.EndedAt.Before(rep.EndedAt) {
		t.Fatalf("stored %s stages %+v counts %+v notUndoable %v", stored.State, stored.Stages, stored.Counts, stored.NotUndoable)
	}
	if state, _ := s.JobState(context.Background(), rep.JobID); state != StateRolledBack {
		t.Fatalf("job state %s", state)
	}
}

// A restore that could not stop the panel leaves the job as it was, says
// why on the stream before the failed stage, and stores the reason.
func TestStartRestoreThatFails(t *testing.T) {
	ctl := &fakeControl{failStop: true}
	h, rep, _ := doneJob(t, ctl)
	res, err := h.svc.StartRestore(context.Background(), rep.BackupID)
	if err != nil {
		t.Fatal(err)
	}
	h.svc.Wait()
	want := []string{"restore stage start", "restore log restore_failed", "restore stage failed", "done done"}
	if got := eventsAfter(t, h.svc, rep.JobID, res.After); !slices.Equal(got, want) {
		t.Fatalf("events %q", got)
	}
	stored, err := h.svc.storedReport(rep.JobID)
	if err != nil || stored == nil || stored.State != StateDone || stored.Error != "restore_failed" {
		t.Fatalf("stored %+v: %v", stored, err)
	}
	if state, _ := h.svc.JobState(context.Background(), rep.JobID); state != StateDone {
		t.Fatalf("job state %s", state)
	}
}

// A restore answers at once when it cannot start: the destination is held
// by someone else, or Karvan cannot stop the panel.
func TestStartRestoreRefusals(t *testing.T) {
	h, rep, _ := doneJob(t, &fakeControl{})
	ctx := context.Background()
	w, err := paneldb.OpenWriter(ctx, h.tg)
	if err != nil {
		t.Fatal(err)
	}
	unlock, err := w.Lock(ctx, h.tg)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := h.svc.StartRestore(ctx, rep.BackupID); !errors.Is(err, ErrLocked) {
		t.Fatalf("locked: %v", err)
	}
	unlock()
	_ = w.Close()
	h.svc.Control = func(context.Context, string) (PanelControl, error) { return nil, ErrNoPanelControl }
	if _, err := h.svc.StartRestore(ctx, rep.BackupID); !errors.Is(err, ErrNoPanelControl) {
		t.Fatalf("no control: %v", err)
	}
	if _, err := h.svc.StartRestore(ctx, "0123456789abcdef"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("unknown backup: %v", err)
	}
	if state, _ := h.svc.JobState(ctx, rep.JobID); state != StateDone {
		t.Fatalf("job state %s", state)
	}
}
