package batch

import (
	"context"
	"database/sql"
	"testing"
	"time"
)

// restoreSet is a destination with batches on both sides of a backup
// taken at 2000: e before it, j (the restored job) and k after it, and
// batches the restore must not touch.
type restoreSet struct {
	s                                       *Store
	db                                      *sql.DB
	e, j, k, failed, rolled, deleted, other int64
}

func newRestoreSet(t *testing.T) *restoreSet {
	t.Helper()
	s, db := open(t)
	r := &restoreSet{s: s, db: db}
	for _, j := range []struct {
		id    string
		ended int64
	}{{"e", 1000}, {"j", 3000}, {"k", 5000}, {"failed", 6000}, {"rolled", 7000}, {"deleted", 8000}} {
		job(t, db, j.id, "done", origin, "alice", j.ended)
		row(t, db, "user", j.id, "done", origin, dest, j.ended-100)
	}
	job(t, db, "other", "done", origin, "alice", 5000)
	row(t, db, "user", "other", "done", origin, "sqlite:///other.db", 4900)
	for _, p := range []struct {
		id string
		to *int64
	}{{"e", &r.e}, {"j", &r.j}, {"k", &r.k}, {"failed", &r.failed}, {"rolled", &r.rolled}, {"deleted", &r.deleted}} {
		*p.to = ensure(t, s, p.id, dest, Facts{})
	}
	r.other = ensure(t, s, "other", "sqlite:///other.db", Facts{})
	exec(t, db, "UPDATE batches SET state = 'inactive', inactive_reason = 'restore_failed', inactive_at = 6500 WHERE id = ?", r.failed)
	exec(t, db, "UPDATE batches SET state = 'inactive', inactive_reason = 'rolled_back', inactive_at = 7500 WHERE id = ?", r.rolled)
	exec(t, db, "UPDATE batches SET state = 'deleted', inactive_reason = 'rolled_back', deleted_at = 8500 WHERE id = ?", r.deleted)
	return r
}

// state reads a batch's state, reason and inactive time.
func (r *restoreSet) state(t *testing.T, id int64) string {
	t.Helper()
	var state, reason string
	var at int64
	if err := r.db.QueryRow("SELECT state, inactive_reason, inactive_at FROM batches WHERE id = ?", id).Scan(&state, &reason, &at); err != nil {
		t.Fatal(err)
	}
	return state + " " + reason + " " + time.UnixMilli(at).UTC().Format("05.000")
}

// check compares the states of every batch of the set.
func (r *restoreSet) check(t *testing.T, want map[*int64]string) {
	t.Helper()
	names := map[*int64]string{&r.e: "e", &r.j: "j", &r.k: "k", &r.failed: "failed", &r.rolled: "rolled", &r.deleted: "deleted", &r.other: "other"}
	for p, name := range names {
		w, ok := want[p]
		if !ok {
			w = map[string]string{"e": "active  00.000", "rolled": "inactive rolled_back 07.500",
				"deleted": "deleted rolled_back 00.000", "other": "active  00.000"}[name]
		}
		if got := r.state(t, *p); got != w {
			t.Errorf("batch %s: %q, want %q", name, got, w)
		}
	}
}

func (r *restoreSet) hold(t *testing.T) *Restore {
	t.Helper()
	h, err := r.s.Restoring(context.Background(), dest, "j", time.UnixMilli(2000))
	if err != nil {
		t.Fatal(err)
	}
	return h
}

// D17: before anything is put back, every batch on the destination made
// after the backup is inactive with reason restoring, the restored job's
// own and every later one, a batch whose earlier restore failed included.
// Batches made before the backup, on another destination, deleted or
// rolled back already stay as they are.
func TestRestoringHoldsLaterBatches(t *testing.T) {
	r := newRestoreSet(t)
	r.hold(t)
	r.check(t, map[*int64]string{&r.j: "inactive restoring 39.000", &r.k: "inactive restoring 39.000",
		&r.failed: "inactive restoring 06.500"})
}

// The restored job's own batch is held even when the clocks say it was
// made before the backup.
func TestRestoringHoldsTheJobsOwnBatch(t *testing.T) {
	r := newRestoreSet(t)
	if _, err := r.s.Restoring(context.Background(), dest, "j", time.UnixMilli(9000)); err != nil {
		t.Fatal(err)
	}
	r.check(t, map[*int64]string{&r.j: "inactive restoring 39.000", &r.k: "active  00.000", &r.failed: "inactive restore_failed 06.500"})
}

// D19: a restore that fails before the put-back step sets every batch
// back as it was.
func TestRestoreCancelPutsBatchesBack(t *testing.T) {
	r := newRestoreSet(t)
	if err := r.hold(t).Cancel(context.Background()); err != nil {
		t.Fatal(err)
	}
	r.check(t, map[*int64]string{&r.j: "active  00.000", &r.k: "active  00.000", &r.failed: "inactive restore_failed 06.500"})
}

// D19: a restore that fails after the put-back started leaves its
// batches restore_failed, and changes no ledger row.
func TestRestoreFailLeavesRestoreFailed(t *testing.T) {
	r := newRestoreSet(t)
	if err := r.hold(t).Fail(context.Background()); err != nil {
		t.Fatal(err)
	}
	r.check(t, map[*int64]string{&r.j: "inactive restore_failed 39.000", &r.k: "inactive restore_failed 39.000",
		&r.failed: "inactive restore_failed 06.500"})
	if n := len(ids(t, r.db, "SELECT id FROM ledger WHERE state <> 'done'")); n != 0 {
		t.Fatalf("%d ledger rows changed", n)
	}
}

// D18: after a successful restore the held batches are rolled back.
// Their rows in done, pending or unknown, and the restored job's own,
// become compensated; their adopted rows, and done rows written before
// the backup, leave the batch as they are. Rows of other batches are not
// touched.
func TestRestoreDoneSettlesTheLedger(t *testing.T) {
	r := newRestoreSet(t)
	jAdopted := row(t, r.db, "user", "j", "adopted", origin, dest, 2900)
	kPending := row(t, r.db, "user", "k", "pending", origin, dest, 4900)
	kUnknown := row(t, r.db, "user", "k", "unknown", origin, dest, 4900)
	jAdmin := row(t, r.db, "admin", "j", "done", origin, dest, 2900)
	// A user an earlier failed run wrote before the backup, which k
	// claimed (D4): the backup holds it, so it stays done and leaves k.
	job(t, r.db, "f", "failed", origin, "alice", 1500)
	kClaimed := row(t, r.db, "user", "f", "done", origin, dest, 1400)
	exec(t, r.db, "UPDATE ledger SET batch_id = ? WHERE id = ?", r.j, jAdopted)
	exec(t, r.db, "UPDATE ledger SET batch_id = ? WHERE id IN (?, ?, ?)", r.k, kPending, kUnknown, kClaimed)
	if err := r.hold(t).Done(context.Background()); err != nil {
		t.Fatal(err)
	}
	r.check(t, map[*int64]string{&r.j: "inactive rolled_back 39.000", &r.k: "inactive rolled_back 39.000",
		&r.failed: "inactive rolled_back 06.500"})
	want := map[string]string{
		"j": "compensated", "k": "compensated", "failed": "compensated",
		"e": "done", "rolled": "done", "deleted": "done", "other": "done",
	}
	for jobID, st := range want {
		var got string
		err := r.db.QueryRow("SELECT state FROM ledger WHERE job_id = ? AND kind = 'user' AND state <> 'adopted' ORDER BY id LIMIT 1", jobID).Scan(&got)
		if err != nil || got != st {
			t.Errorf("job %s: %q (%v), want %q", jobID, got, err, st)
		}
	}
	for _, id := range []int64{kPending, kUnknown, jAdmin} {
		var st string
		if err := r.db.QueryRow("SELECT state FROM ledger WHERE id = ?", id).Scan(&st); err != nil || st != "compensated" {
			t.Errorf("row %d: %q %v", id, st, err)
		}
	}
	for id, want := range map[int64]string{jAdopted: "adopted", kClaimed: "done"} {
		var st string
		var b sql.NullInt64
		if err := r.db.QueryRow("SELECT state, batch_id FROM ledger WHERE id = ?", id).Scan(&st, &b); err != nil || st != want || b.Valid {
			t.Errorf("row %d: %q batch %v %v, want %q and no batch", id, st, b, err, want)
		}
	}
}

// D19: at start, a batch still restoring has no restore running (Karvan
// stopped during it), so it becomes restore_failed.
func TestFailRestoring(t *testing.T) {
	r := newRestoreSet(t)
	r.hold(t)
	if err := r.s.FailRestoring(context.Background()); err != nil {
		t.Fatal(err)
	}
	r.check(t, map[*int64]string{&r.j: "inactive restore_failed 39.000", &r.k: "inactive restore_failed 39.000",
		&r.failed: "inactive restore_failed 06.500"})
}
