package store

import (
	"context"
	"database/sql"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"
	"testing"
)

func needSQLite325(t *testing.T, db *sql.DB) {
	t.Helper()
	var version string
	if err := db.QueryRow("SELECT sqlite_version()").Scan(&version); err != nil {
		t.Fatal(err)
	}
	var major, minor int
	if _, err := fmt.Sscanf(version, "%d.%d", &major, &minor); err != nil || major < 3 || (major == 3 && minor < 25) {
		t.Fatalf("sqlite_version() = %s, want at least 3.25 (%v)", version, err)
	}
}

// 0007 renames ledger.batch to chunk; RENAME COLUMN needs SQLite 3.25, and
// SQLite rewrites the ledger_job index with it.
func TestLedgerBatchBecomesChunk(t *testing.T) {
	ctx := context.Background()
	db, err := sql.Open("sqlite", "file:"+filepath.Join(t.TempDir(), "k.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	needSQLite325(t, db)
	if err := migrateFrom(ctx, db, upTo(t, 7)); err != nil {
		t.Fatal(err)
	}
	for _, q := range []string{
		"INSERT INTO panels (id, env_path, name, scanned_at) VALUES (1, '/p/.env', 'p', 0)",
		"INSERT INTO jobs (id, state, source_kind, source_origin, panel_id, safe_mode, report, created_at, updated_at) VALUES ('j', 'done', 'marzban', 'o', 1, 1, '', 1, 1)",
		"INSERT INTO ledger (kind, job_id, state, batch, source_origin, source_key, source_created, source_username, dest, dest_id, dest_username, dest_created, created_at, updated_at) VALUES ('user', 'j', 'done', 4, 'o', 'k', 1, 'u', 'd', 9, 'u', 1, 1, 1)",
	} {
		if _, err := db.ExecContext(ctx, q); err != nil {
			t.Fatal(err)
		}
	}
	if err := migrateFrom(ctx, db, fs.FS(migrationFiles)); err != nil {
		t.Fatal(err)
	}
	var chunk int
	if err := db.QueryRowContext(ctx, "SELECT chunk FROM ledger WHERE job_id = 'j'").Scan(&chunk); err != nil || chunk != 4 {
		t.Fatalf("chunk = %d, %v; want 4", chunk, err)
	}
	var index string
	if err := db.QueryRowContext(ctx, "SELECT sql FROM sqlite_master WHERE type = 'index' AND name = 'ledger_job'").Scan(&index); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(index, "chunk") || strings.Contains(index, "batch") {
		t.Errorf("ledger_job = %s, want it on chunk", index)
	}
}
