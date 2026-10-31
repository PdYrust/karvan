package store

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"slices"
	"testing"
)

// v14Rules is a store at version 14 with two runs of one old panel, a
// rule of batches saved with the first run only, a manual rule and a
// token on each rule.
var v14Rules = []string{
	"INSERT INTO panels (id, env_path, name, kind, scanned_at) VALUES (1, '/opt/p/.env', 'p', 'pasarguard', 0)",
	`INSERT INTO jobs (id, state, source_kind, source_origin, panel_id, safe_mode, created_at, updated_at, ended_at) VALUES
		('r1', 'done', 'marzban', 'https://old.example', 1, 1, 1, 1, 1000), ('r2', 'done', 'marzban', 'https://old.example', 1, 1, 1, 1, 2000)`,
	`INSERT INTO ledger (id, kind, job_id, state, source_origin, source_key, source_created, source_username, dest, dest_id,
		dest_username, dest_created, created_at, updated_at) VALUES
		(1, 'user', 'r1', 'done', 'https://old.example', 'ali', 1, 'ali', 'd', 10, 'ali', 1, 900000, 900000),
		(2, 'user', 'r2', 'done', 'https://old.example', 'bob', 1, 'bob', 'd', 11, 'bob', 1, 1900000, 1900000)`,
	`INSERT INTO batches (id, job_id, panel_id, dest, source_kind, source_origin, name, state, created_at, updated_at) VALUES
		(1, 'r1', 1, 'd', 'marzban', 'https://old.example', 'root', 'active', 1000, 1000),
		(2, 'r2', 1, 'd', 'marzban', 'https://old.example', 'root 2', 'active', 2000, 2000)`,
	"UPDATE ledger SET batch_id = id",
	`INSERT INTO legacy_sites (id, name, panel_id, source_kind, source_url, old_path, listen_port, mode, created_at, updated_at) VALUES
		(1, 'from batches', 1, 'marzban', 'https://old.example', 'sub', 443, 'batches', 1, 1),
		(2, 'typed', 1, 'marzban', 'https://Old.example/', 'sub', 8443, 'manual', 1, 1)`,
	"INSERT INTO legacy_site_batches (site_id, batch_id) VALUES (1, 1)",
	`INSERT INTO legacy_tokens (mac, site_id, ledger_id, issued_at, verified_at, checked_at) VALUES
		(x'01', 1, 1, 1, 1, 1), (x'02', 2, 2, 1, 1, 1)`,
}

// 0015 drops the batches a rule of batches was saved with, so the rule
// serves its old panel's every run; every other row stays as it was.
func TestBatchGroupsKeepEveryRow(t *testing.T) {
	ctx := context.Background()
	db, err := sql.Open("sqlite", "file:"+filepath.Join(t.TempDir(), "k.db")+"?_pragma=foreign_keys(1)")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := migrateFrom(ctx, db, upTo(t, 15)); err != nil {
		t.Fatal(err)
	}
	for _, q := range v14Rules {
		if _, err := db.ExecContext(ctx, q); err != nil {
			t.Fatalf("%s: %v", q, err)
		}
	}
	tables := []string{"jobs", "ledger", "batches", "legacy_sites", "legacy_tokens"}
	read := func() string {
		out := ""
		for _, table := range tables {
			cols := strs(t, db, "SELECT name FROM pragma_table_info('"+table+"') ORDER BY cid")
			q := "SELECT quote(" + cols[0] + ")"
			for _, c := range cols[1:] {
				q += " || ',' || quote(" + c + ")"
			}
			out += fmt.Sprint(table, strs(t, db, q+" FROM "+table+" ORDER BY rowid"))
		}
		return out
	}
	before := read()
	if err := migrateFrom(ctx, db, migrationFiles); err != nil {
		t.Fatal(err)
	}
	if after := read(); after != before {
		t.Fatalf("rows changed:\nbefore %s\nafter  %s", before, after)
	}
	if got := strs(t, db, "SELECT name FROM sqlite_master WHERE name LIKE 'legacy_site_batches%'"); len(got) != 0 {
		t.Fatalf("left %q", got)
	}
	if got := strs(t, db, "SELECT name FROM sqlite_master WHERE type = 'table' AND name LIKE 'legacy_%' ORDER BY name"); !slices.Equal(got,
		[]string{"legacy_closed", "legacy_sites", "legacy_tokens"}) {
		t.Fatalf("tables %q", got)
	}
}
