package store

import (
	"context"
	"database/sql"
	"maps"
	"path/filepath"
	"testing"
)

// schemaAndCounts is every table, index and trigger with its SQL, and the
// row count of every table, apart from schema_migrations.
func schemaAndCounts(t *testing.T, db *sql.DB) (map[string]string, map[string]int) {
	t.Helper()
	rows, err := db.Query("SELECT type || ' ' || name, COALESCE(sql, ''), type = 'table' FROM sqlite_master WHERE name NOT LIKE 'sqlite_%' AND name != 'schema_migrations'")
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	schema := map[string]string{}
	var tables []string
	for rows.Next() {
		var key, text string
		var table bool
		if err := rows.Scan(&key, &text, &table); err != nil {
			t.Fatal(err)
		}
		schema[key] = text
		if table {
			tables = append(tables, key[len("table "):])
		}
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	counts := map[string]int{}
	for _, name := range tables {
		var n int
		if err := db.QueryRow(`SELECT COUNT(*) FROM "` + name + `"`).Scan(&n); err != nil {
			t.Fatal(err)
		}
		counts[name] = n
	}
	return schema, counts
}

// 0009 drops the templates table, saved templates and all, and touches
// nothing else.
func TestTemplatesTableIsDropped(t *testing.T) {
	ctx := context.Background()
	db, err := sql.Open("sqlite", "file:"+filepath.Join(t.TempDir(), "k.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := migrateFrom(ctx, db, upTo(t, 9)); err != nil {
		t.Fatal(err)
	}
	for _, q := range []string{
		"INSERT INTO templates (name, body, created_at, updated_at) VALUES ('monthly', '{}', 1, 1)",
		"INSERT INTO panels (id, env_path, name, scanned_at) VALUES (1, '/p/.env', 'p', 0)",
		"INSERT INTO jobs (id, state, source_kind, source_origin, panel_id, safe_mode, report, created_at, updated_at) VALUES ('j', 'done', 'marzban', 'o', 1, 1, '', 1, 1)",
	} {
		if _, err := db.ExecContext(ctx, q); err != nil {
			t.Fatal(err)
		}
	}
	schemaBefore, countsBefore := schemaAndCounts(t, db)
	if _, ok := schemaBefore["table templates"]; !ok {
		t.Fatal("no templates table at version 8")
	}
	if err := migrateFrom(ctx, db, upTo(t, 10)); err != nil {
		t.Fatal(err)
	}
	schemaAfter, countsAfter := schemaAndCounts(t, db)
	delete(schemaBefore, "table templates")
	delete(countsBefore, "templates")
	if !maps.Equal(schemaBefore, schemaAfter) {
		t.Errorf("schema after 0009 = %v, want %v", schemaAfter, schemaBefore)
	}
	if !maps.Equal(countsBefore, countsAfter) {
		t.Errorf("row counts after 0009 = %v, want %v", countsAfter, countsBefore)
	}
}
