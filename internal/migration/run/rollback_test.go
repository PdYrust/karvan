package run

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"

	"github.com/PdYrust/karvan/internal/migration/ledger"
	"github.com/PdYrust/karvan/internal/panel"
	"github.com/PdYrust/karvan/internal/panel/paneldb"
	"github.com/PdYrust/karvan/internal/panel/paneldb/paneldbtest"
	"github.com/PdYrust/karvan/internal/panel/pasarguarddb"
)

// fakeControl records stop and start; failStop makes Stop fail, and
// onStop runs as the panel stops (seen keeps what it read).
type fakeControl struct {
	mu       sync.Mutex
	calls    []string
	failStop bool
	onStop   func()
	seen     string
}

func (f *fakeControl) Stop(context.Context) error {
	if f.onStop != nil {
		f.onStop()
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, "stop")
	if f.failStop {
		return errors.New("docker: permission denied")
	}
	return nil
}

func (f *fakeControl) Start(context.Context) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, "start")
	return nil
}

func (f *fakeControl) seq() string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return strings.Join(f.calls, ",")
}

func backupHarness(t *testing.T, ctl *fakeControl) (*harness, paneldb.Target) {
	t.Helper()
	tg := paneldbtest.Load(t, paneldbtest.Fixture{Flavor: "sqlite", Path: "../../panel/pasarguarddb/testdata/" + pasarguarddb.Revision + ".sqlite.sql"})
	h := newHarness(t, panel.Marzban, panel.PasarGuard, tg, 12)
	if ctl != nil {
		h.svc.Control = func(context.Context, string) (PanelControl, error) { return ctl, nil }
	}
	return h, tg
}

func (h *harness) runWith(j *Job, rollback string) (*Report, error) {
	h.t.Helper()
	_, _, v, _ := h.svc.View(j.ID)
	if err := h.svc.Run(context.Background(), j.ID, Options{Version: v, Reload: ReloadManual, SafeMode: true, Rollback: rollback}); err != nil {
		return nil, err
	}
	h.svc.Wait()
	return h.svc.Report(j.ID)
}

// On failure, put the whole backup back, with the
// panel stopped and started again, and nothing of the job left.
func TestBackupRollbackPutsTheWholeBackupBack(t *testing.T) {
	ctl := &fakeControl{}
	h, tg := backupHarness(t, ctl)
	before := paneldbtest.ChecksumExcept(t, tg)
	h.fault = func(step string, n int) error {
		if step == "ledger" && n == 2 {
			return errors.New("injected")
		}
		return nil
	}
	rep, err := h.runWith(h.job(nil), RollbackBackup)
	if err != nil {
		t.Fatal(err)
	}
	if rep.State != StateRolledBack || rep.Rollback != RollbackBackup || ctl.seq() != "stop,start" || rep.Counts.Undone != 0 {
		t.Fatalf("state %s rollback %s control %q undone %d", rep.State, rep.Rollback, ctl.seq(), rep.Counts.Undone)
	}
	if after := paneldbtest.ChecksumExcept(t, tg); after != before {
		t.Fatal("the destination is not as it was")
	}
	if live, _ := h.svc.ledger.Job(context.Background(), rep.JobID, ledger.Done, ledger.Pending); len(live) != 0 {
		t.Fatalf("%d live ledger rows after the restore", len(live))
	}
}

// A backup that cannot go back (the panel did not stop) falls back to
// removing the job's own rows, and the panel is not left stopped.
func TestBackupRollbackFallsBackToUndo(t *testing.T) {
	ctl := &fakeControl{failStop: true}
	h, tg := backupHarness(t, ctl)
	before := paneldbtest.ChecksumExcept(t, tg, "inbounds")
	h.fault = func(step string, n int) error {
		if step == "chunk" && n == 2 {
			return errors.New("injected")
		}
		return nil
	}
	rep, err := h.runWith(h.job(nil), RollbackBackup)
	if err != nil {
		t.Fatal(err)
	}
	if rep.State != StateRolledBack || rep.Counts.Undone == 0 || ctl.seq() != "stop" {
		t.Fatalf("state %s undone %d control %q", rep.State, rep.Counts.Undone, ctl.seq())
	}
	if after := paneldbtest.ChecksumExcept(t, tg, "inbounds"); after != before {
		t.Fatal("the destination is not as it was")
	}
}

// With the backup choice, a run that could not put its backup back does
// not start.
func TestBackupRollbackNeedsPanelControl(t *testing.T) {
	h, _ := backupHarness(t, nil)
	h.svc.Control = func(context.Context, string) (PanelControl, error) { return nil, errors.New("no compose file") }
	j := h.job(nil)
	if _, err := h.runWith(j, RollbackBackup); !errors.Is(err, ErrNoPanelControl) {
		t.Fatalf("err %v", err)
	}
	if n := h.count("users"); n != 0 {
		t.Fatalf("%d users written", n)
	}
	if rep, err := h.runWith(j, RollbackUndo); err != nil || rep.State != StateDone {
		t.Fatalf("undo choice: %v %+v", err, rep)
	}
}

// A backup restored on request takes back a finished job and stops and
// starts the panel around it.
func TestRestoreBackupOnRequest(t *testing.T) {
	ctl := &fakeControl{}
	h, tg := backupHarness(t, ctl)
	before := paneldbtest.ChecksumExcept(t, tg)
	rep, err := h.runWith(h.job(nil), RollbackUndo)
	if err != nil || rep.State != StateDone {
		t.Fatalf("run: %v %+v", err, rep)
	}
	if err := h.svc.RestoreBackup(context.Background(), rep.BackupID, false); err != nil {
		t.Fatal(err)
	}
	if ctl.seq() != "stop,start" || paneldbtest.ChecksumExcept(t, tg) != before {
		t.Fatalf("control %q; destination as before: %v", ctl.seq(), paneldbtest.ChecksumExcept(t, tg) == before)
	}
	if state, _ := h.svc.JobState(context.Background(), rep.JobID); state != StateRolledBack {
		t.Fatalf("job state %s", state)
	}
}
