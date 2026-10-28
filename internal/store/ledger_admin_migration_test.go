package store

import (
	"context"
	"database/sql"
	"maps"
	"path/filepath"
	"testing"
)

// 0011 gives ledger rows a source admin, '' on every row from before,
// and keeps every row.
func TestLedgerRowsGetASourceAdmin(t *testing.T) {
	ctx := context.Background()
	db, err := sql.Open("sqlite", "file:"+filepath.Join(t.TempDir(), "k.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := migrateFrom(ctx, db, upTo(t, 11)); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO ledger (kind, job_id, state, source_origin, source_key, source_created, source_username,
		dest, dest_username, dest_created, created_at, updated_at) VALUES ('user', 'j', 'done', 'o', 'k', 1, 'u', 'd', 'u', 1, 1, 1)`); err != nil {
		t.Fatal(err)
	}
	_, countsBefore := schemaAndCounts(t, db)
	if err := migrateFrom(ctx, db, upTo(t, 12)); err != nil {
		t.Fatal(err)
	}
	if _, countsAfter := schemaAndCounts(t, db); !maps.Equal(countsBefore, countsAfter) {
		t.Errorf("row counts after 0011 = %v, want %v", countsAfter, countsBefore)
	}
	var admin string
	if err := db.QueryRowContext(ctx, "SELECT source_admin FROM ledger WHERE source_key = 'k'").Scan(&admin); err != nil || admin != "" {
		t.Fatalf("source_admin %q: %v", admin, err)
	}
}
