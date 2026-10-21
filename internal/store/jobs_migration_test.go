package store

import (
	"context"
	"database/sql"
	"io/fs"
	"path/filepath"
	"testing"
	"testing/fstest"
)

// upTo is the embedded migrations before version v.
func upTo(t *testing.T, v int) fstest.MapFS {
	t.Helper()
	all, err := loadMigrations(migrationFiles)
	if err != nil {
		t.Fatal(err)
	}
	out := fstest.MapFS{}
	for _, m := range all[:v-1] {
		out["migrations/"+m.name] = &fstest.MapFile{Data: []byte(m.sql)}
	}
	return out
}

// Jobs from before 0006 take their counts and end time from their report.
func TestJobListColumnsComeFromOldReports(t *testing.T) {
	ctx := context.Background()
	db, err := sql.Open("sqlite", "file:"+filepath.Join(t.TempDir(), "k.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := migrateFrom(ctx, db, upTo(t, 6)); err != nil {
		t.Fatal(err)
	}
	for _, q := range []string{
		"INSERT INTO panels (id, env_path, name, scanned_at) VALUES (1, '/p/.env', 'p', 0)",
		`INSERT INTO jobs (id, state, source_kind, source_origin, panel_id, safe_mode, report, created_at, updated_at) VALUES
			('ran', 'done', 'marzban', 'o', 1, 1, '{"endedAt":"2026-10-17T12:00:05.25+03:30","counts":{"read":3,"moved":2}}', 1, 1),
			('draft', 'checked', 'marzban', 'o', 1, 1, '', 2, 2),
			('broken', 'failed', 'marzban', 'o', 1, 1, 'not json', 3, 3),
			('unended', 'failed', 'marzban', 'o', 1, 1, '{"endedAt":"0001-01-01T00:00:00Z","counts":{"read":1}}', 4, 4)`,
	} {
		if _, err := db.ExecContext(ctx, q); err != nil {
			t.Fatal(err)
		}
	}
	if err := migrateFrom(ctx, db, fs.FS(migrationFiles)); err != nil {
		t.Fatal(err)
	}
	want := map[string][2]any{
		"ran":     {`{"read":3,"moved":2}`, int64(1792225805000)},
		"draft":   {"", int64(0)},
		"broken":  {"", int64(0)},
		"unended": {`{"read":1}`, int64(0)},
	}
	for id, w := range want {
		var counts string
		var ended int64
		if err := db.QueryRowContext(ctx, "SELECT counts, ended_at FROM jobs WHERE id = ?", id).Scan(&counts, &ended); err != nil {
			t.Fatal(err)
		}
		if counts != w[0] || ended != w[1] {
			t.Errorf("%s: counts %q ended %d, want %v", id, counts, ended, w)
		}
	}
}
