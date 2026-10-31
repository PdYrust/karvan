package store

import (
	"context"
	"database/sql"
	"fmt"
	"io/fs"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// v6Seed is a database at version 6 with one job of every kind 0008 has
// to sort out. Origins: o is https://old.example, p is https://other.example.
var v6Seed = []string{
	"INSERT INTO panels (id, env_path, name, scanned_at) VALUES (1, '/p/.env', 'p', 0)",
	`INSERT INTO jobs (id, state, source_kind, source_origin, source_admin, panel_id, safe_mode, created_at, updated_at, ended_at) VALUES
		('done', 'done', 'marzban', 'https://old.example', 'alice', 1, 1, 1, 5000, 5000),
		('rbtok', 'rolled_back', 'marzban', 'https://old.example', 'alice', 1, 1, 2, 6100, 6000),
		('noadmin', 'done', 'marzban', 'https://old.example', '  ', 1, 1, 3, 7000, 0),
		('other', 'done', 'pasarguard', 'https://other.example', 'alice', 1, 1, 4, 8000, 8000),
		('done2', 'done', 'marzban', 'https://old.example', 'alice', 1, 1, 5, 9000, 9000),
		('adminonly', 'done', 'marzban', 'https://old.example', 'bob', 1, 1, 6, 9500, 9500),
		('failed', 'failed', 'marzban', 'https://old.example', 'carol', 1, 1, 7, 9600, 9600),
		('rbnone', 'rolled_back', 'marzban', 'https://old.example', 'dave', 1, 1, 8, 9700, 9700),
		('rbadopt', 'rolled_back', 'marzban', 'https://old.example', 'erin', 1, 1, 9, 9800, 9800),
		('interrupted', 'interrupted', 'marzban', 'https://old.example', 'frank', 1, 1, 10, 9900, 0)`,
	"INSERT INTO job_events (job_id, seq, at, stage, type) VALUES ('done', 1, 1, 'write', 'chunk'), ('rbtok', 1, 2, 'undo', 'done')",
	`INSERT INTO ledger (id, kind, job_id, state, batch, source_origin, source_key, source_created, source_username, dest, dest_username, dest_created, created_at, updated_at) VALUES
		(1, 'user', 'done', 'done', 1, 'https://old.example', 'u1', 1, 'u1', 'd', 'u1', 1, 1, 1),
		(2, 'user', 'done', 'adopted', 1, 'https://old.example', 'u2', 1, 'u2', 'd', 'u2', 1, 1, 1),
		(3, 'user', 'done', 'compensated', 1, 'https://old.example', 'u3', 1, 'u3', 'd', 'u3', 1, 1, 1),
		(4, 'admin', 'done', 'done', 0, 'https://old.example', 'a1', 1, 'a1', 'd', 'a1', 1, 1, 1),
		(5, 'user', 'done2', 'done', 1, 'https://old.example', 'u4', 1, 'u4', 'd', 'u4', 1, 1, 1),
		(6, 'user', 'noadmin', 'done', 1, 'https://old.example', 'u5', 1, 'u5', 'd', 'u5', 1, 1, 1),
		(7, 'admin', 'adminonly', 'done', 0, 'https://old.example', 'a2', 1, 'a2', 'd', 'a2', 1, 1, 1),
		(8, 'user', 'failed', 'done', 1, 'https://old.example', 'u6', 1, 'u6', 'd', 'u6', 1, 1, 1),
		(9, 'user', 'failed', 'pending', 2, 'https://old.example', 'u7', 1, 'u7', 'd', 'u7', 1, 1, 1),
		(10, 'user', 'rbtok', 'compensated', 1, 'https://old.example', 'u8', 1, 'u8', 'd', 'u8', 1, 1, 1),
		(11, 'user', 'rbtok', 'adopted', 1, 'https://old.example', 'u9', 1, 'u9', 'd', 'u9', 1, 1, 1),
		(12, 'user', 'rbnone', 'compensated', 1, 'https://old.example', 'u10', 1, 'u10', 'd', 'u10', 1, 1, 1),
		(13, 'user', 'rbadopt', 'adopted', 1, 'https://old.example', 'u11', 1, 'u11', 'd', 'u11', 1, 1, 1),
		(14, 'user', 'interrupted', 'done', 1, 'https://old.example', 'u12', 1, 'u12', 'd', 'u12', 1, 1, 1),
		(15, 'user', 'other', 'done', 1, 'https://other.example', 'u13', 1, 'u13', 'd', 'u13', 1, 1, 1),
		(16, 'user', 'done', 'superseded', 1, 'https://old.example', 'u14', 1, 'u14', 'd', 'u14', 1, 1, 1),
		(17, 'user', 'rbtok', 'unknown', 1, 'https://old.example', 'u15', 1, 'u15', 'd', 'u15', 1, 1, 1)`,
	`INSERT INTO legacy_sites (id, name, panel_id, source_kind, source_url, old_path, listen_port, created_at, updated_at) VALUES
		(1, 'old', 1, 'marzban', 'https://old.example', 'sub', 443, 1, 1),
		(2, 'other', 1, 'pasarguard', 'https://other.example', 'sub', 8443, 1, 1)`,
	`INSERT INTO legacy_tokens (mac, site_id, ledger_id, issued_at, verified_at, checked_at) VALUES
		(X'01', 1, 1, 1, 1, 1), (X'02', 1, 2, 1, 1, 1), (X'03', 1, 3, 1, 1, 1), (X'05', 1, 5, 1, 1, 1),
		(X'08', 1, 8, 1, 1, 1), (X'0a', 1, 10, 1, 1, 1), (X'0b', 1, 11, 1, 1, 1), (X'0d', 1, 13, 1, 1, 1),
		(X'0e', 1, 14, 1, 1, 1), (X'0f', 2, 15, 1, 1, 1)`,
	"INSERT INTO legacy_closed (ledger_id, reason, closed_at) VALUES (6, 'revoked', 1)",
}

// keptTables are the tables 0008 must not lose a row of.
var keptTables = []string{"jobs", "job_events", "ledger", "legacy_tokens", "legacy_closed", "legacy_sites"}

// snapshot reads every row of the kept tables, column by column, in rowid
// order. 0007 renamed ledger.batch to chunk, so after it the old name
// reads the new one.
func snapshot(t *testing.T, db *sql.DB, cols map[string][]string, renamed bool) map[string][]string {
	t.Helper()
	out := map[string][]string{}
	for _, table := range keptTables {
		names := slices.Clone(cols[table])
		for i, c := range names {
			if renamed && table == "ledger" && c == "batch" {
				names[i] = "chunk"
			}
		}
		out[table] = strs(t, db, "SELECT quote("+strings.Join(names, ") || ',' || quote(")+") FROM "+table+" ORDER BY rowid")
	}
	return out
}

// columns lists each kept table's columns as they are now.
func columns(t *testing.T, db *sql.DB) map[string][]string {
	t.Helper()
	out := map[string][]string{}
	for _, table := range keptTables {
		out[table] = strs(t, db, "SELECT name FROM pragma_table_info('"+table+"') ORDER BY cid")
	}
	return out
}

// strs runs a query whose rows are one text column each.
func strs(t *testing.T, db *sql.DB, q string) []string {
	t.Helper()
	rows, err := db.Query(q)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			t.Fatal(err)
		}
		out = append(out, s)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return out
}

// servedBefore is what a site serves today: tokens of rows in done or
// adopted whose user is not closed.
const servedBefore = `SELECT t.site_id || ':' || hex(t.mac) FROM legacy_tokens t JOIN ledger l ON l.id = t.ledger_id
	WHERE l.state IN ('done', 'adopted') AND NOT EXISTS (SELECT 1 FROM legacy_closed c WHERE c.ledger_id = l.id)
	ORDER BY 1`

// servedAfter adds the manual rule scope: the row's batch is active
// or the row has none.
const servedAfter = `SELECT t.site_id || ':' || hex(t.mac) FROM legacy_tokens t JOIN ledger l ON l.id = t.ledger_id
	LEFT JOIN batches b ON b.id = l.batch_id
	WHERE l.state IN ('done', 'adopted') AND NOT EXISTS (SELECT 1 FROM legacy_closed c WHERE c.ledger_id = l.id)
	AND (l.batch_id IS NULL OR b.state = 'active')
	ORDER BY 1`

func migrateSeededV6(t *testing.T) (*sql.DB, map[string][]string, []string) {
	t.Helper()
	ctx := context.Background()
	db, err := sql.Open("sqlite", "file:"+filepath.Join(t.TempDir(), "k.db")+"?_pragma=foreign_keys(1)")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := migrateFrom(ctx, db, upTo(t, 7)); err != nil {
		t.Fatal(err)
	}
	for _, q := range v6Seed {
		if _, err := db.ExecContext(ctx, q); err != nil {
			t.Fatalf("%s: %v", q, err)
		}
	}
	cols := columns(t, db)
	before := snapshot(t, db, cols, false)
	served := strs(t, db, servedBefore)
	if err := migrateFrom(ctx, db, fs.FS(migrationFiles)); err != nil {
		t.Fatal(err)
	}
	if after := snapshot(t, db, cols, true); fmt.Sprint(after) != fmt.Sprint(before) {
		t.Fatalf("rows changed:\nbefore %v\nafter  %v", before, after)
	}
	return db, before, served
}

// 0008 makes a batch of every done job with users and of every rolled-back
// job whose done or compensated users still have tokens, and loses nothing.
func TestBatchesBackfill(t *testing.T) {
	db, before, _ := migrateSeededV6(t)
	if len(before["ledger"]) != 17 || len(before["legacy_tokens"]) != 10 || len(before["job_events"]) != 2 {
		t.Fatalf("seed rows = %d ledger, %d tokens, %d events", len(before["ledger"]), len(before["legacy_tokens"]), len(before["job_events"]))
	}
	got := strs(t, db, `SELECT id || '|' || job_id || '|' || panel_id || '|' || dest || '|' || source_kind || '|' || source_origin || '|' ||
		name || '|' || state || '|' || inactive_reason || '|' || inactive_at || '|' || created_at || '|' || updated_at || '|' ||
		source_ip || source_pin || old_path || '|' || old_hosts || '|' || old_port || '|' || deleted_at FROM batches ORDER BY id`)
	want := []string{
		"1|done|1|d|marzban|https://old.example|alice|active||0|5000|5000||[]|0|0",
		"2|rbtok|1|d|marzban|https://old.example|alice 2|inactive|rolled_back|6000|6000|6000||[]|0|0",
		"3|noadmin|1|d|marzban|https://old.example|batch|active||0|7000|7000||[]|0|0",
		"4|other|1|d|pasarguard|https://other.example|alice|active||0|8000|8000||[]|0|0",
		"5|done2|1|d|marzban|https://old.example|alice 3|active||0|9000|9000||[]|0|0",
	}
	if !slices.Equal(got, want) {
		t.Errorf("batches:\n got %q\nwant %q", got, want)
	}
}

// Only users join a batch; adopted rows of a rolled-back job stay out so
// manual rules keep serving them.
func TestBatchesMembership(t *testing.T) {
	db, _, _ := migrateSeededV6(t)
	got := strs(t, db, "SELECT id || ':' || COALESCE(batch_id, '-') FROM ledger ORDER BY id")
	want := []string{"1:1", "2:1", "3:1", "4:-", "5:5", "6:3", "7:-", "8:-", "9:-", "10:2", "11:-", "12:-", "13:-", "14:-", "15:4", "16:-", "17:-"}
	if !slices.Equal(got, want) {
		t.Errorf("ledger batch_id:\n got %q\nwant %q", got, want)
	}
	if got := strs(t, db, "SELECT DISTINCT mode FROM legacy_sites"); !slices.Equal(got, []string{"manual"}) {
		t.Errorf("site modes = %q, want manual", got)
	}
	if got := strs(t, db, "SELECT site_id FROM legacy_site_batches"); len(got) != 0 {
		t.Errorf("legacy_site_batches = %q, want empty", got)
	}
	if got := strs(t, db, "SELECT name FROM sqlite_master WHERE type = 'index' AND name IN ('batches_dest', 'ledger_batch', 'legacy_site_batches_batch') ORDER BY name"); len(got) != 3 {
		t.Errorf("indexes = %q", got)
	}
}

// Every token served before the migration is served after it, by the same
// site, and nothing more.
func TestBatchesKeepServedTokens(t *testing.T) {
	db, _, before := migrateSeededV6(t)
	want := []string{"1:01", "1:02", "1:05", "1:08", "1:0B", "1:0D", "1:0E", "2:0F"}
	if !slices.Equal(before, want) {
		t.Fatalf("served before = %q, want %q", before, want)
	}
	if after := strs(t, db, servedAfter); !slices.Equal(after, before) {
		t.Errorf("served after = %q, want %q", after, before)
	}
}

// The batch rules of 0008 hold for new rows too.
func TestBatchesChecks(t *testing.T) {
	db, _, _ := migrateSeededV6(t)
	for _, q := range []string{
		"UPDATE batches SET name = '' WHERE id = 1",
		"UPDATE batches SET name = printf('%065d', 0) WHERE id = 1",
		"UPDATE batches SET state = 'gone' WHERE id = 1",
		"UPDATE batches SET inactive_reason = 'later' WHERE id = 1",
		"INSERT INTO batches (job_id, panel_id, dest, source_kind, source_origin, name, state, created_at, updated_at) VALUES ('done', 1, 'd', 'marzban', 'o', 'x', 'active', 1, 1)",
		"UPDATE ledger SET batch_id = 99 WHERE id = 4",
		"UPDATE legacy_sites SET mode = 'auto' WHERE id = 1",
	} {
		if _, err := db.Exec(q); err == nil {
			t.Errorf("%s: accepted", q)
		}
	}
}
