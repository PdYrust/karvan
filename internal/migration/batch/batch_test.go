package batch

import (
	"context"
	"database/sql"
	"slices"
	"testing"
	"time"

	"github.com/PdYrust/karvan/internal/store"
)

const (
	origin = "https://old.example"
	dest   = "sqlite:///d.db"
)

// open is Karvan's store at the latest version with one panel.
func open(t *testing.T) (*Store, *sql.DB) {
	t.Helper()
	db, err := store.Open(context.Background(), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	exec(t, db, "INSERT INTO panels (id, env_path, name, scanned_at) VALUES (1, '/p/.env', 'p', 0)")
	return New(db, func() time.Time { return time.UnixMilli(99_000) }), db
}

func exec(t *testing.T, db *sql.DB, q string, args ...any) {
	t.Helper()
	if _, err := db.Exec(q, args...); err != nil {
		t.Fatalf("%s: %v", q, err)
	}
}

// job adds a job that ended at ended (unix ms).
func job(t *testing.T, db *sql.DB, id, state, from, admin string, ended int64) {
	t.Helper()
	exec(t, db, `INSERT INTO jobs (id, state, source_kind, source_origin, source_admin, panel_id, safe_mode, created_at, updated_at, ended_at)
		VALUES (?, ?, 'marzban', ?, ?, 1, 1, 1, ?, ?)`, id, state, from, admin, ended, ended)
}

// row adds a ledger row written at written (unix ms) and returns its id.
func row(t *testing.T, db *sql.DB, kind, jobID, state, from, to string, written int64) int64 {
	t.Helper()
	var n int64
	if err := db.QueryRow("SELECT count(*) FROM ledger").Scan(&n); err != nil {
		t.Fatal(err)
	}
	key := "k" + string(rune('a'+n))
	res, err := db.Exec(`INSERT INTO ledger (kind, job_id, state, source_origin, source_key, source_created, source_username,
		dest, dest_username, dest_created, created_at, updated_at) VALUES (?, ?, ?, ?, ?, 1, ?, ?, ?, 1, ?, ?)`,
		kind, jobID, state, from, key, key, to, key, written*1000, written*1000)
	if err != nil {
		t.Fatal(err)
	}
	id, _ := res.LastInsertId()
	return id
}

// members lists the ledger ids in batch b.
func members(t *testing.T, db *sql.DB, b int64) []int64 {
	t.Helper()
	return ids(t, db, "SELECT id FROM ledger WHERE batch_id = ? ORDER BY id", b)
}

// ids runs a query whose rows are one integer column each.
func ids(t *testing.T, db *sql.DB, q string, args ...any) []int64 {
	t.Helper()
	rows, err := db.Query(q, args...)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var out []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			t.Fatal(err)
		}
		out = append(out, id)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return out
}

func ensure(t *testing.T, s *Store, jobID, to string, f Facts) int64 {
	t.Helper()
	id, err := s.Ensure(context.Background(), jobID, to, f)
	if err != nil {
		t.Fatal(err)
	}
	return id
}

// A batch claims the job's written and adopted users, and users of
// the same old panel and destination an earlier run left without a batch.
// Compensated, void, unknown, pending and claimed rows stay out, and so do
// admins, other old panels and other destinations.
func TestEnsureClaims(t *testing.T) {
	s, db := open(t)
	job(t, db, "old", "done", origin, "alice", 1000)
	job(t, db, "failed", "failed", origin, "alice", 2000)
	job(t, db, "j", "done", origin, "alice", 5000)
	claimedBefore := row(t, db, "user", "old", "done", origin, dest, 900)
	oldBatch := ensure(t, s, "old", dest, Facts{})
	want := []int64{
		row(t, db, "user", "failed", "done", origin, dest, 1500),
		row(t, db, "user", "failed", "adopted", origin, dest, 1500),
		row(t, db, "user", "j", "done", origin, dest, 4000),
		row(t, db, "user", "j", "adopted", origin, dest, 4000),
	}
	for _, st := range []string{"compensated", "superseded", "unknown", "pending", "merged"} {
		row(t, db, "user", "failed", st, origin, dest, 1500)
	}
	row(t, db, "admin", "j", "done", origin, dest, 4000)
	row(t, db, "user", "failed", "done", "https://other.example", dest, 1500)
	row(t, db, "user", "failed", "done", origin, "sqlite:///other.db", 1500)
	b := ensure(t, s, "j", dest, Facts{})
	if b == 0 || b == oldBatch {
		t.Fatalf("batch %d (older %d)", b, oldBatch)
	}
	if got := members(t, db, b); !slices.Equal(got, want) {
		t.Fatalf("members %v, want %v", got, want)
	}
	if got := members(t, db, oldBatch); !slices.Equal(got, []int64{claimedBefore}) {
		t.Fatalf("the older batch lost or gained rows: %v", got)
	}
}

// A batch made at start for a job that ended long ago does not
// claim what later runs left behind.
func TestEnsureClaimsOnlyEarlierRows(t *testing.T) {
	s, db := open(t)
	job(t, db, "j", "done", origin, "alice", 5000)
	job(t, db, "later", "failed", origin, "alice", 7000)
	mine := row(t, db, "user", "j", "done", origin, dest, 4000)
	row(t, db, "user", "later", "done", origin, dest, 6000)
	b := ensure(t, s, "j", "", Facts{})
	if got := members(t, db, b); !slices.Equal(got, []int64{mine}) {
		t.Fatalf("members %v", got)
	}
}

// Ensure is idempotent and does nothing for a job that is not done or
// has nothing to hold.
func TestEnsureOnceAndOnlyWithUsers(t *testing.T) {
	s, db := open(t)
	job(t, db, "j", "done", origin, "alice", 5000)
	job(t, db, "adminonly", "done", origin, "bob", 6000)
	job(t, db, "failed", "failed", origin, "carol", 6000)
	job(t, db, "empty", "done", origin, "dave", 6000)
	row(t, db, "user", "j", "done", origin, dest, 4000)
	row(t, db, "admin", "adminonly", "done", origin, dest, 5500)
	row(t, db, "user", "failed", "pending", origin, dest, 5500)
	first := ensure(t, s, "j", dest, Facts{OldPath: "sub"})
	if again := ensure(t, s, "j", dest, Facts{OldPath: "other"}); again != first {
		t.Fatalf("second Ensure gave %d, first %d", again, first)
	}
	for _, id := range []string{"adminonly", "failed", "empty"} {
		if b := ensure(t, s, id, dest, Facts{}); b != 0 {
			t.Errorf("%s: batch %d", id, b)
		}
	}
	var n int
	var path string
	if err := db.QueryRow("SELECT count(*), max(old_path) FROM batches").Scan(&n, &path); err != nil || n != 1 || path != "sub" {
		t.Fatalf("batches %d path %q err %v", n, path, err)
	}
}

// The batch row: its default name, the old link facts, the job's
// source, panel and end time.
func TestEnsureStoresTheBatch(t *testing.T) {
	s, db := open(t)
	job(t, db, "a", "done", origin, " alice ", 5000)
	job(t, db, "b", "done", origin, "alice", 6000)
	job(t, db, "c", "done", "https://other.example", "alice", 7000)
	job(t, db, "d", "done", origin, "", 8000)
	// Each written after the batch before it, so each job keeps its own.
	row(t, db, "user", "a", "done", origin, dest, 4000)
	row(t, db, "user", "b", "done", origin, dest, 5500)
	row(t, db, "user", "d", "done", origin, dest, 7500)
	row(t, db, "user", "c", "done", "https://other.example", dest, 4000)
	f := Facts{OldPath: "sub", OldHosts: []string{"s.example"}, OldPort: 8443, SourceIP: "10.0.0.5", SourcePin: "ab"}
	ensure(t, s, "a", dest, f)
	for _, j := range []string{"b", "c", "d"} {
		ensure(t, s, j, dest, Facts{})
	}
	type got struct {
		name, from, to, path, hosts, ip, pin, state string
		port, created, panel                        int64
	}
	var a got
	err := db.QueryRow(`SELECT name, source_origin, dest, old_path, old_hosts, source_ip, source_pin, state, old_port, created_at, panel_id
		FROM batches WHERE job_id = 'a'`).Scan(&a.name, &a.from, &a.to, &a.path, &a.hosts, &a.ip, &a.pin, &a.state, &a.port, &a.created, &a.panel)
	if err != nil {
		t.Fatal(err)
	}
	if a != (got{"alice", origin, dest, "sub", `["s.example"]`, "10.0.0.5", "ab", "active", 8443, 5000, 1}) {
		t.Fatalf("batch a: %+v", a)
	}
	names := map[string]string{"a": "alice", "b": "alice 2", "c": "alice", "d": "batch"}
	for j, want := range names {
		var name, hosts string
		if err := db.QueryRow("SELECT name, old_hosts FROM batches WHERE job_id = ?", j).Scan(&name, &hosts); err != nil {
			t.Fatal(err)
		}
		if name != want || j != "a" && hosts != "[]" {
			t.Errorf("%s: name %q hosts %s", j, name, hosts)
		}
	}
}

// A new batch joins every batch rule that serves an older batch of
// the same old panel and destination, in the same transaction.
func TestEnsureJoinsBatchRules(t *testing.T) {
	s, db := open(t)
	job(t, db, "old", "done", origin, "alice", 1000)
	job(t, db, "other", "done", "https://other.example", "alice", 1000)
	job(t, db, "new", "done", origin, "alice", 5000)
	row(t, db, "user", "old", "done", origin, dest, 900)
	row(t, db, "user", "other", "done", "https://other.example", dest, 900)
	row(t, db, "user", "new", "done", origin, dest, 4000)
	older, otherB := ensure(t, s, "old", dest, Facts{}), ensure(t, s, "other", dest, Facts{})
	exec(t, db, `INSERT INTO legacy_sites (id, name, panel_id, source_kind, source_url, old_path, listen_port, mode, created_at, updated_at) VALUES
		(1, 'rule', 1, 'marzban', ?, 'sub', 443, 'batches', 1, 1), (2, 'rule2', 1, 'marzban', ?, 'sub', 443, 'batches', 1, 1),
		(3, 'manual', 1, 'marzban', ?, 'sub', 8443, 'manual', 1, 1)`, origin, origin, origin)
	exec(t, db, "INSERT INTO legacy_site_batches (site_id, batch_id) VALUES (1, ?), (2, ?)", older, otherB)
	b := ensure(t, s, "new", dest, Facts{})
	sites := ids(t, db, "SELECT site_id FROM legacy_site_batches WHERE batch_id = ? ORDER BY site_id", b)
	if !slices.Equal(sites, []int64{1}) {
		t.Fatalf("the new batch joined rules %v", sites)
	}
}

// Missing lists done jobs without a batch; Foreign lists a job's rows in
// another job's batch.
func TestMissingAndForeign(t *testing.T) {
	s, db := open(t)
	ctx := context.Background()
	job(t, db, "failed", "failed", origin, "alice", 1000)
	job(t, db, "j", "done", origin, "alice", 5000)
	job(t, db, "k", "done", origin, "alice", 6000)
	claimed := row(t, db, "user", "failed", "done", origin, dest, 900)
	row(t, db, "user", "failed", "pending", origin, dest, 900)
	row(t, db, "user", "j", "done", origin, dest, 4000)
	missing, err := s.Missing(ctx)
	if err != nil || !slices.Equal(missing, []string{"j", "k"}) {
		t.Fatalf("missing %v %v", missing, err)
	}
	ensure(t, s, "j", dest, Facts{})
	if missing, _ = s.Missing(ctx); !slices.Equal(missing, []string{"k"}) {
		t.Fatalf("missing after Ensure %v", missing)
	}
	foreign, err := s.Foreign(ctx, "failed")
	if err != nil || len(foreign) != 1 || !foreign[claimed] {
		t.Fatalf("foreign %v %v", foreign, err)
	}
	if own, _ := s.Foreign(ctx, "j"); len(own) != 0 {
		t.Fatalf("a job's own batch is not foreign: %v", own)
	}
}
