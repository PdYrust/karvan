package store

import (
	"context"
	"database/sql"
	"maps"
	"path/filepath"
	"slices"
	"testing"
)

// migrateSeededV12 is a database at version 12 with the sites of v9Seed,
// two certificates and a domain, migrated to the latest version.
func migrateSeededV12(t *testing.T) *sql.DB {
	t.Helper()
	ctx := context.Background()
	db, err := sql.Open("sqlite", "file:"+filepath.Join(t.TempDir(), "k.db")+"?_pragma=foreign_keys(1)")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := migrateFrom(ctx, db, upTo(t, 13)); err != nil {
		t.Fatal(err)
	}
	seed := append(slices.Clone(v9Seed),
		"INSERT INTO certificates (id, name, cert_file, key_file, created_at, updated_at) VALUES (1, 'files', '/c/a.crt', '/c/a.key', 1, 1)",
		"INSERT INTO certificates (id, name, cert_pem, origin, created_at, updated_at) VALUES (2, 'le', 'pem', 'acme', 2, 2)",
		"INSERT INTO domains (name, certificate_id, created_at, updated_at) VALUES ('a.example.com', 2, 1, 1)",
		"INSERT INTO panels (id, env_path, name, scanned_at) VALUES (2, '/opt/new/.env', 'new', 0)")
	for _, q := range seed {
		if _, err := db.ExecContext(ctx, q); err != nil {
			t.Fatalf("%s: %v", q, err)
		}
	}
	_, before := schemaAndCounts(t, db)
	if err := migrateFrom(ctx, db, migrationFiles); err != nil {
		t.Fatal(err)
	}
	_, after := schemaAndCounts(t, db)
	maps.DeleteFunc(after, func(table string, _ int) bool { _, old := before[table]; return !old })
	if !maps.Equal(before, after) {
		t.Fatalf("row counts after 0013 = %v, want %v", after, before)
	}
	return db
}

// 0013 keeps every row and starts with no copies.
func TestCertificateCopiesKeepEveryRow(t *testing.T) {
	db := migrateSeededV12(t)
	got := strs(t, db, "SELECT id || '|' || name || '|' || origin FROM certificates ORDER BY id")
	if want := []string{"1|files|", "2|le|acme"}; !slices.Equal(got, want) {
		t.Fatalf("certificates = %v, want %v", got, want)
	}
	got = strs(t, db, "SELECT count(*) FROM certificate_copies UNION ALL SELECT count(*) FROM certificate_copy_files")
	if want := []string{"0", "0"}; !slices.Equal(got, want) {
		t.Fatalf("new tables = %v, want empty", got)
	}
}

// One certificate per folder; a copy goes with its certificate or panel,
// and the record of written files stays.
func TestCertificateCopiesChecks(t *testing.T) {
	db := migrateSeededV12(t)
	const add = "INSERT INTO certificate_copies (certificate_id, panel_id, base, rel, created_at) VALUES "
	for _, q := range []string{
		add + "(1, NULL, '/etc/letsencrypt', 'live/a.example.com', 1)",
		add + "(1, 2, '/var/lib/new', 'certs/a.example.com', 1)",
		add + "(2, NULL, '/etc/letsencrypt', 'live/b.example.com', 1)",
		"INSERT INTO certificate_copy_files (path, sum, written_at) VALUES ('/etc/letsencrypt/live/a.example.com/fullchain.pem', 's', 1)",
	} {
		if _, err := db.Exec(q); err != nil {
			t.Fatalf("%s: %v", q, err)
		}
	}
	for _, q := range []string{
		add + "(2, NULL, '/etc/letsencrypt', 'live/a.example.com', 1)", // the folder is taken
		add + "(99, NULL, '/etc/letsencrypt', 'live/c.example.com', 1)",
		add + "(1, 99, '/var/lib/x', 'certs/c.example.com', 1)",
		"INSERT INTO certificate_copy_files (path, sum, written_at) VALUES ('/etc/letsencrypt/live/a.example.com/fullchain.pem', 't', 2)",
	} {
		if _, err := db.Exec(q); err == nil {
			t.Errorf("accepted: %s", q)
		}
	}
	for _, q := range []string{"DELETE FROM certificates WHERE id = 1", "DELETE FROM panels WHERE id = 2"} {
		if _, err := db.Exec(q); err != nil {
			t.Fatalf("%s: %v", q, err)
		}
	}
	got := strs(t, db, "SELECT rel FROM certificate_copies UNION ALL SELECT count(*) FROM certificate_copy_files")
	if want := []string{"live/b.example.com", "1"}; !slices.Equal(got, want) {
		t.Fatalf("after the deletes = %v, want %v", got, want)
	}
}
