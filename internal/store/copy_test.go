package store

import (
	"context"
	"database/sql"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// atVersion opens dir/k.db with the embedded migrations applied up to v.
func atVersion(t *testing.T, dir string, v int) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", "file:"+filepath.Join(dir, "k.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := migrateFrom(context.Background(), db, upTo(t, v+1)); err != nil {
		t.Fatal(err)
	}
	return db
}

func version(t *testing.T, db *sql.DB) int {
	t.Helper()
	var v int
	if err := db.QueryRow("SELECT COALESCE(MAX(version), 0) FROM schema_migrations").Scan(&v); err != nil {
		t.Fatal(err)
	}
	return v
}

func entries(t *testing.T, dir string) []string {
	t.Helper()
	list, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, e := range list {
		if strings.Contains(e.Name(), ".before-") {
			out = append(out, e.Name())
		}
	}
	return out
}

// Before the first pending migration, Karvan copies the database to
// karvan.db.before-<v>, private, replacing a stale tmp and older copies.
func TestMigrateCopiesTheDatabaseFirst(t *testing.T) {
	dir := t.TempDir()
	db := atVersion(t, dir, 7)
	for name, body := range map[string]string{"k.db.before-8.tmp": "stale", "k.db.before-3": "old", "k.db.before-7": "old", "k.db.before-6.tmp": "old"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if err := migrateFrom(context.Background(), db, fs.FS(migrationFiles)); err != nil {
		t.Fatal(err)
	}
	if got := entries(t, dir); len(got) != 1 || got[0] != "k.db.before-8" {
		t.Fatalf("copies = %q, want only k.db.before-8", got)
	}
	copyPath := filepath.Join(dir, "k.db.before-8")
	info, err := os.Stat(copyPath)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Errorf("copy mode = %o, want 600", info.Mode().Perm())
	}
	old, err := sql.Open("sqlite", "file:"+copyPath+"?mode=ro")
	if err != nil {
		t.Fatal(err)
	}
	defer old.Close()
	if v := version(t, old); v != 7 {
		t.Errorf("copy schema version = %d, want 7", v)
	}
}

// A new database and a current one need no copy.
func TestMigrateCopiesOnlyAnExistingOldDatabase(t *testing.T) {
	dir := t.TempDir()
	for range 2 {
		db, err := Open(context.Background(), dir)
		if err != nil {
			t.Fatal(err)
		}
		_ = db.Close()
	}
	if got := entries(t, dir); len(got) != 0 {
		t.Errorf("copies = %q, want none", got)
	}
}

// When the copy cannot be written, no migration runs.
func TestMigrateAppliesNothingWithoutTheCopy(t *testing.T) {
	dir := t.TempDir()
	db := atVersion(t, dir, 7)
	if err := os.Chmod(dir, 0o500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(dir, 0o700) })
	err := migrateFrom(context.Background(), db, fs.FS(migrationFiles))
	if err == nil || !strings.HasPrefix(err.Error(), "store copy before migration 8: ") {
		t.Fatalf("err = %v, want store copy before migration 8", err)
	}
	if v := version(t, db); v != 7 {
		t.Errorf("schema version = %d, want 7", v)
	}
	var n int
	if err := db.QueryRow("SELECT count(*) FROM sqlite_master WHERE name = 'batches'").Scan(&n); err != nil || n != 0 {
		t.Errorf("batches table = %d, %v; want none", n, err)
	}
}

// A folder where the tmp file or the copy goes stops the copy too.
func TestMigrateStopsWhenACopyNameIsTaken(t *testing.T) {
	for _, name := range []string{"k.db.before-8.tmp", "k.db.before-8"} {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			db := atVersion(t, dir, 7)
			if err := os.MkdirAll(filepath.Join(dir, name, "inside"), 0o700); err != nil {
				t.Fatal(err)
			}
			err := migrateFrom(context.Background(), db, fs.FS(migrationFiles))
			if err == nil || !strings.HasPrefix(err.Error(), "store copy before migration 8: ") {
				t.Fatalf("err = %v, want store copy before migration 8", err)
			}
			if v := version(t, db); v != 7 {
				t.Errorf("schema version = %d, want 7", v)
			}
		})
	}
}

// A database with no file has nowhere to put its copy.
func TestMigrateNeedsADatabaseFileToCopy(t *testing.T) {
	db, err := sql.Open("sqlite", "file::memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	if err := migrateFrom(context.Background(), db, upTo(t, 8)); err != nil {
		t.Fatal(err)
	}
	err = migrateFrom(context.Background(), db, fs.FS(migrationFiles))
	if err == nil || !strings.Contains(err.Error(), "has no file") {
		t.Fatalf("err = %v, want the no file error", err)
	}
}
