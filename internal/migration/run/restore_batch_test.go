package run

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"testing"

	"github.com/PdYrust/karvan/internal/migration/batch"
	"github.com/PdYrust/karvan/internal/panel/fakepanel"
	"github.com/PdYrust/karvan/internal/panel/paneldb"
	"github.com/PdYrust/karvan/internal/panel/paneldb/paneldbtest"
)

// batchState is a batch's state and its reason, as "state reason".
func (h *harness) batchState(id int64) string {
	h.t.Helper()
	var state, reason string
	if err := h.svc.db.QueryRow("SELECT state, inactive_reason FROM batches WHERE id = ?", id).Scan(&state, &reason); err != nil {
		h.t.Fatal(err)
	}
	return state + " " + reason
}

// rowStates counts a batch's ledger rows by state.
func (h *harness) rowStates(batchID int64) map[string]int {
	h.t.Helper()
	rows, err := h.svc.db.Query("SELECT state, count(*) FROM ledger WHERE batch_id = ? GROUP BY state", batchID)
	if err != nil {
		h.t.Fatal(err)
	}
	defer rows.Close()
	out := map[string]int{}
	for rows.Next() {
		var st string
		var n int
		if err := rows.Scan(&st, &n); err != nil {
			h.t.Fatal(err)
		}
		out[st] = n
	}
	if err := rows.Err(); err != nil {
		h.t.Fatal(err)
	}
	return out
}

// addUsers gives the old panel n more users, so a re-run moves them.
func (h *harness) addUsers(n int) {
	h.src.Mutate(func(users *[]map[string]any) {
		for i := range n {
			n := 100 + i
			*users = append(*users, fakepanel.MarzbanUser(fmt.Sprintf("late%03d", i), n, map[string]any{
				"proxies": map[string]any{
					"vless":       map[string]any{"id": fmt.Sprintf("00000000-0000-4000-8000-%012d", n), "flow": ""},
					"vmess":       map[string]any{"id": fmt.Sprintf("00000000-0000-4000-9000-%012d", n)},
					"shadowsocks": map[string]any{"password": fmt.Sprintf("late-shadowsocks-password-%04d", n), "method": "aes-256-gcm"},
				},
				"inbounds": map[string]any{"vless": []string{"VLESS TCP"}, "vmess": []string{"VMess WS"}, "shadowsocks": []string{"Shadowsocks TCP"}},
			}))
		}
	})
}

// adopt turns one of a batch's rows into an adopted user, one that was on
// the destination before the batch and so is in every backup.
func (h *harness) adopt(batchID int64) int64 {
	h.t.Helper()
	var id int64
	if err := h.svc.db.QueryRow("SELECT min(id) FROM ledger WHERE batch_id = ?", batchID).Scan(&id); err != nil {
		h.t.Fatal(err)
	}
	if _, err := h.svc.db.Exec("UPDATE ledger SET state = 'adopted' WHERE id = ?", id); err != nil {
		h.t.Fatal(err)
	}
	return id
}

// checkAdoptedServed makes sure an adopted row left its batch and stays
// adopted, so manual rules serve it and a later batch can claim it (D18).
func (h *harness) checkAdoptedServed(id int64) {
	h.t.Helper()
	var st string
	var b sql.NullInt64
	if err := h.svc.db.QueryRow("SELECT state, batch_id FROM ledger WHERE id = ?", id).Scan(&st, &b); err != nil || st != "adopted" || b.Valid {
		h.t.Fatalf("adopted row %d: state %q batch %v %v", id, st, b, err)
	}
}

// D17, D18: restoring the latest job's backup takes its batch inactive
// with reason restoring before the panel stops, then rolled back; its
// written users are compensated and its adopted user leaves the batch.
func TestRestoreRollsBackTheJobsBatch(t *testing.T) {
	ctl := &fakeControl{}
	h, rep, _ := doneJob(t, ctl)
	b, ok := h.batchOf(rep.JobID)
	if !ok {
		t.Fatal("no batch")
	}
	adopted := h.adopt(b.id)
	ctl.onStop = func() { ctl.seen = h.batchState(b.id) }
	if _, err := h.svc.StartRestore(context.Background(), rep.BackupID); err != nil {
		t.Fatal(err)
	}
	h.svc.Wait()
	if ctl.seen != "inactive restoring" {
		t.Fatalf("while the panel stopped the batch was %q", ctl.seen)
	}
	if got := h.batchState(b.id); got != "inactive rolled_back" {
		t.Fatalf("batch %q", got)
	}
	if got := h.rowStates(b.id); len(got) != 1 || got["compensated"] != 11 {
		t.Fatalf("batch rows %v", got)
	}
	h.checkAdoptedServed(adopted)
}

// earlyBatch adds a batch made before any backup of the harness: a done
// job that ended at 1 ms with one user on the destination.
func (h *harness) earlyBatch() int64 {
	h.t.Helper()
	ctx := context.Background()
	var dest string
	if err := h.svc.db.QueryRow("SELECT min(dest) FROM ledger").Scan(&dest); err != nil {
		h.t.Fatal(err)
	}
	if _, err := h.svc.db.ExecContext(ctx, `INSERT INTO jobs (id, state, source_kind, source_origin, source_admin, panel_id,
		safe_mode, created_at, updated_at, ended_at) VALUES ('early', 'done', 'marzban', 'https://early.example', 'early', ?, 1, 1, 1, 1)`,
		h.panel); err != nil {
		h.t.Fatal(err)
	}
	if _, err := h.svc.db.ExecContext(ctx, `INSERT INTO ledger (kind, job_id, state, source_origin, source_key, source_created,
		source_username, dest, dest_username, dest_created, created_at, updated_at)
		VALUES ('user', 'early', 'done', 'https://early.example', 'e1', 1, 'e1', ?, 'e1', 1, 1, 1)`, dest); err != nil {
		h.t.Fatal(err)
	}
	id, err := h.svc.batches.Ensure(ctx, "early", "", batch.Facts{})
	if err != nil || id == 0 {
		h.t.Fatalf("early batch %d: %v", id, err)
	}
	return id
}

// D17, D18: restoring an earlier job's backup takes back every batch made
// after it on that destination, and only those: the batch made before
// the backup stays active with its user served.
func TestRestoreOfAnEarlierBackup(t *testing.T) {
	ctl := &fakeControl{}
	h, first, before := doneJob(t, ctl)
	h.addUsers(3)
	second, err := h.runWith(h.job(nil), RollbackUndo)
	if err != nil || second.State != StateDone || second.Counts.Moved != 3 {
		t.Fatalf("second run: %v %+v", err, second)
	}
	a, _ := h.batchOf(first.JobID)
	b, _ := h.batchOf(second.JobID)
	early := h.earlyBatch()
	adopted := h.adopt(b.id)
	if _, err := h.svc.StartRestore(context.Background(), first.BackupID); err != nil {
		t.Fatal(err)
	}
	h.svc.Wait()
	if paneldbtest.ChecksumExcept(t, h.tg) != before {
		t.Fatal("the destination is not as it was before the first run")
	}
	for _, id := range []int64{a.id, b.id} {
		if got := h.batchState(id); got != "inactive rolled_back" {
			t.Fatalf("batch %d %q", id, got)
		}
	}
	if got := h.rowStates(b.id); len(got) != 1 || got["compensated"] != 2 {
		t.Fatalf("second batch rows %v", got)
	}
	if got, rows := h.batchState(early), h.rowStates(early); got != "active " || rows["done"] != 1 || len(rows) != 1 {
		t.Fatalf("early batch %q rows %v", got, rows)
	}
	h.checkAdoptedServed(adopted)
}

// D19: a restore that fails before anything is put back (the panel would
// not stop) sets the batch back to active.
func TestRestoreThatFailsBeforePutBack(t *testing.T) {
	h, rep, _ := doneJob(t, &fakeControl{failStop: true})
	b, _ := h.batchOf(rep.JobID)
	if _, err := h.svc.StartRestore(context.Background(), rep.BackupID); err != nil {
		t.Fatal(err)
	}
	h.svc.Wait()
	if got := h.batchState(b.id); got != "active " {
		t.Fatalf("batch %q", got)
	}
	if got := h.rowStates(b.id); got["done"] != 12 {
		t.Fatalf("batch rows %v", got)
	}
}

// D19: a restore that fails once it started putting the backup back (the
// backup file is gone) leaves the batch restore_failed and its rows as
// they were; the operator can make it active again.
func TestRestoreThatFailsAfterPutBackStarted(t *testing.T) {
	ctl := &fakeControl{}
	h, rep, _ := doneJob(t, ctl)
	b, _ := h.batchOf(rep.JobID)
	rec, err := h.svc.backupRecord(context.Background(), rep.BackupID)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(rec.Path); err != nil {
		t.Fatal(err)
	}
	if _, err := h.svc.StartRestore(context.Background(), rep.BackupID); err != nil {
		t.Fatal(err)
	}
	h.svc.Wait()
	if got := h.batchState(b.id); got != "inactive restore_failed" || ctl.seq() != "stop,start" {
		t.Fatalf("batch %q control %q", got, ctl.seq())
	}
	if got := h.rowStates(b.id); got["done"] != 12 {
		t.Fatalf("batch rows %v", got)
	}
	if _, err := h.svc.Batches().Reactivate(context.Background(), b.id); err != nil {
		t.Fatal(err)
	}
}

// A restore whose batches cannot be held does not start: the caller gets
// the error, the lock is let go and nothing changed.
func TestRestoreThatCannotHoldItsBatches(t *testing.T) {
	ctl := &fakeControl{}
	h, rep, _ := doneJob(t, ctl)
	b, _ := h.batchOf(rep.JobID)
	if _, err := h.svc.db.Exec("CREATE TRIGGER no_hold BEFORE UPDATE ON batches BEGIN SELECT RAISE(ABORT, 'refused'); END"); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if _, err := h.svc.StartRestore(ctx, rep.BackupID); err == nil || errors.Is(err, ErrLocked) {
		t.Fatalf("err %v", err)
	}
	w, err := paneldb.OpenWriter(ctx, h.tg)
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	unlock, err := w.Lock(ctx, h.tg)
	if err != nil {
		t.Fatalf("the destination is still locked: %v", err)
	}
	unlock()
	if got, state := h.batchState(b.id), h.jobState(rep.JobID); got != "active " || state != StateDone || ctl.seq() != "" {
		t.Fatalf("batch %q job %s control %q", got, state, ctl.seq())
	}
}

// D19: Karvan stopped while a restore held its batches (a crash); at the
// next start no restore runs, so the batch becomes restore_failed.
func TestRecoverEndsARestoreThatCrashed(t *testing.T) {
	h, rep, _ := doneJob(t, &fakeControl{})
	b, _ := h.batchOf(rep.JobID)
	task, err := h.svc.prepareRestore(context.Background(), rep.BackupID, false)
	if err != nil {
		t.Fatal(err)
	}
	task.release() // the process ends here, the hold not ended
	if got := h.batchState(b.id); got != "inactive restoring" {
		t.Fatalf("batch during the restore %q", got)
	}
	if err := New(h.svc.Deps).Recover(context.Background()); err != nil {
		t.Fatal(err)
	}
	if got := h.batchState(b.id); got != "inactive restore_failed" {
		t.Fatalf("batch after the restart %q", got)
	}
}

// A roll back on request with the backup choice puts back a backup that
// later batches wrote over: a failed job's users that a later batch
// claimed (D4) go with that batch, which is rolled back (D17, D18).
func TestRollbackByBackupTakesALaterBatchDown(t *testing.T) {
	h, rep, before := doneJob(t, &fakeControl{})
	ctx := context.Background()
	for _, q := range []string{
		"UPDATE jobs SET state = 'failed', rollback = 'backup' WHERE id = ?",
		"UPDATE ledger SET batch_id = NULL WHERE job_id = ?",
		"DELETE FROM batches WHERE job_id = ?",
	} {
		if _, err := h.svc.db.ExecContext(ctx, q, rep.JobID); err != nil {
			t.Fatal(err)
		}
	}
	h.addUsers(3)
	later, err := h.runWith(h.job(nil), RollbackUndo)
	b, ok := h.batchOf(later.JobID)
	if err != nil || later.State != StateDone || !ok || b.members != 15 {
		t.Fatalf("later run: %v %+v batch %+v", err, later, b)
	}
	h.rollBackOnRequest(rep.JobID)
	if got := h.batchState(b.id); got != "inactive rolled_back" || h.jobState(rep.JobID) != StateRolledBack {
		t.Fatalf("batch %q job %s", got, h.jobState(rep.JobID))
	}
	if got := h.rowStates(b.id); got["compensated"] != 15 || paneldbtest.ChecksumExcept(t, h.tg) != before {
		t.Fatalf("batch rows %v; destination as before: %v", got, paneldbtest.ChecksumExcept(t, h.tg) == before)
	}
}

func (h *harness) jobState(id string) string {
	h.t.Helper()
	state, err := h.svc.JobState(context.Background(), id)
	if err != nil {
		h.t.Fatal(err)
	}
	return state
}
