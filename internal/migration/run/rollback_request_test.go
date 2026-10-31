package run

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/PdYrust/karvan/internal/panel"
	"github.com/PdYrust/karvan/internal/panel/marzbandb"
	"github.com/PdYrust/karvan/internal/panel/paneldb"
	"github.com/PdYrust/karvan/internal/panel/paneldb/paneldbtest"
)

// rollBackOnRequest asks for a rollback and waits for it to end.
func (h *harness) rollBackOnRequest(jobID string) {
	h.t.Helper()
	if err := h.svc.Rollback(context.Background(), jobID); err != nil {
		h.t.Fatal(err)
	}
	h.svc.Wait()
}

// failedJob runs a job that fails after writing 10 users, Safe Mode off.
func failedJob(t *testing.T) (*harness, *Report) {
	t.Helper()
	tg := paneldbtest.Load(t, paneldbtest.Fixture{Flavor: "sqlite", Path: "../../panel/marzbandb/testdata/" + marzbandb.Revision + ".sqlite.sql"})
	h := newHarness(t, panel.PasarGuard, panel.Marzban, tg, 12)
	h.fault = func(s string, n int) error {
		if s == "chunk" && n == 3 {
			return errors.New("injected")
		}
		return nil
	}
	rep := h.run(h.job(nil), false)
	if rep.State != StateFailed || h.count("users") != 10 {
		t.Fatalf("state %s users %d", rep.State, h.count("users"))
	}
	h.fault = nil
	return h, rep
}

// checkStoredRollback makes sure the jobs row says rolled_back, with the
// undo in its counts and its report and an end time.
func checkStoredRollback(t *testing.T, s *Service, jobID string, undone int) {
	t.Helper()
	var state, counts string
	var ended int64
	if err := s.db.QueryRow("SELECT state, counts, ended_at FROM jobs WHERE id = ?", jobID).Scan(&state, &counts, &ended); err != nil {
		t.Fatal(err)
	}
	var c Counts
	if err := json.Unmarshal([]byte(counts), &c); err != nil {
		t.Fatal(err)
	}
	rep, err := s.storedReport(jobID)
	if err != nil || rep == nil {
		t.Fatalf("stored report: %v", err)
	}
	if state != StateRolledBack || c.Undone != undone || rep.Counts.Undone != undone || rep.State != StateRolledBack ||
		ended != rep.EndedAt.UnixMilli() || ended == 0 {
		t.Fatalf("state %s counts %+v report %s %+v ended %d", state, c, rep.State, rep.Counts, ended)
	}
}

// A rollback that cannot lock the destination answers at once and
// changes nothing.
func TestRollbackOnRequestNeedsTheDestination(t *testing.T) {
	h, rep := failedJob(t)
	ctx := context.Background()
	w, err := paneldb.OpenWriter(ctx, h.tg)
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	unlock, err := w.Lock(ctx, h.tg)
	if err != nil {
		t.Fatal(err)
	}
	if err := h.svc.Rollback(ctx, rep.JobID); !errors.Is(err, ErrLocked) {
		t.Fatalf("err %v", err)
	}
	unlock()
	h.svc.Wait()
	if state, _ := h.svc.JobState(ctx, rep.JobID); state != StateFailed || h.count("users") != 10 {
		t.Fatalf("state %s users %d", state, h.count("users"))
	}
}

// After a restart an interrupted job has no report; its rollback makes
// one with the job's source and destination and stores it.
func TestRollbackOfAnInterruptedJobAfterARestart(t *testing.T) {
	h, rep := failedJob(t)
	ctx := context.Background()
	if _, err := h.svc.db.ExecContext(ctx, "UPDATE jobs SET state = ?, report = '', counts = '' WHERE id = ?", StateInterrupted, rep.JobID); err != nil {
		t.Fatal(err)
	}
	restarted := New(h.svc.Deps)
	if err := restarted.Rollback(ctx, rep.JobID); err != nil {
		t.Fatal(err)
	}
	restarted.Wait()
	checkStoredRollback(t, restarted, rep.JobID, 10)
	back, _ := restarted.Report(rep.JobID)
	if back.Source.Kind != string(panel.PasarGuard) || back.Source.Origin == "" || back.Destination.DB == "" || h.count("users") != 0 {
		t.Fatalf("report %+v users %d", back, h.count("users"))
	}
}

// A roll back of a failed job leaves the users a later batch claimed
// and lists them as left with in_batch; the job still counts as rolled
// back, and nothing else is undone.
func TestRollbackLeavesUsersOfALaterBatch(t *testing.T) {
	h, failed := failedJob(t)
	later := h.run(h.job(nil), false)
	b, ok := h.batchOf(later.JobID)
	if later.State != StateDone || later.Counts.Moved != 2 || !ok || b.members != 12 {
		t.Fatalf("later run: state %s counts %+v batch %+v", later.State, later.Counts, b)
	}
	h.rollBackOnRequest(failed.JobID)
	rep, err := h.svc.Report(failed.JobID)
	if err != nil {
		t.Fatal(err)
	}
	if rep.State != StateRolledBack || rep.Counts.Undone != 0 || len(rep.Left) != 10 || len(rep.LeftReasons) != 10 {
		t.Fatalf("state %s counts %+v left %v reasons %v", rep.State, rep.Counts, rep.Left, rep.LeftReasons)
	}
	for _, id := range rep.Left {
		if rep.LeftReasons[id] != LeftInBatch {
			t.Fatalf("left %d: reason %q", id, rep.LeftReasons[id])
		}
	}
	if n := h.count("users"); n != 12 {
		t.Fatalf("%d users on the destination", n)
	}
	if b, _ := h.batchOf(later.JobID); b.members != 12 {
		t.Fatalf("the batch lost users: %+v", b)
	}
}
