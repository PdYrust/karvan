package store

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
)

func TestOpenCreatesPrivateDatabaseWithSchema(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "data")
	db, err := Open(context.Background(), dir)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer db.Close()

	for path, want := range map[string]os.FileMode{dir: 0o700, Path(dir): 0o600} {
		info, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode().Perm() != want {
			t.Errorf("%s mode = %o, want %o", path, info.Mode().Perm(), want)
		}
	}
	for _, table := range []string{"operators", "sessions", "login_failures", "settings", "audit_events"} {
		var n int
		if err := db.QueryRow(`SELECT count(*) FROM sqlite_master WHERE type='table' AND name=?`, table).Scan(&n); err != nil || n != 1 {
			t.Errorf("table %s missing (n=%d, err=%v)", table, n, err)
		}
	}
	var fk int
	if err := db.QueryRow(`PRAGMA foreign_keys`).Scan(&fk); err != nil || fk != 1 {
		t.Errorf("foreign_keys = %d, %v", fk, err)
	}
}

func TestExists(t *testing.T) {
	dir := t.TempDir()
	if ok, err := Exists(dir); ok || err != nil {
		t.Fatalf("Exists before open = %v, %v", ok, err)
	}
	db, err := Open(context.Background(), dir)
	if err != nil {
		t.Fatal(err)
	}
	db.Close()
	if ok, err := Exists(dir); !ok || err != nil {
		t.Fatalf("Exists after open = %v, %v", ok, err)
	}
}

func TestOpenIsIdempotent(t *testing.T) {
	dir := t.TempDir()
	for range 2 {
		db, err := Open(context.Background(), dir)
		if err != nil {
			t.Fatalf("Open: %v", err)
		}
		var n int
		if err := db.QueryRow(`SELECT count(*) FROM schema_migrations`).Scan(&n); err != nil || n != 1 {
			t.Fatalf("schema_migrations rows = %d, %v", n, err)
		}
		db.Close()
	}
}

func TestMigrateAppliesOnlyPendingSteps(t *testing.T) {
	db, err := Open(context.Background(), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	fsys := fstest.MapFS{
		"migrations/0001_init.sql":  {Data: []byte(`SELECT 1;`)},
		"migrations/0002_extra.sql": {Data: []byte(`CREATE TABLE extra (id INTEGER PRIMARY KEY) STRICT;`)},
	}
	if err := migrateFrom(context.Background(), db, fsys); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	var n int
	if err := db.QueryRow(`SELECT count(*) FROM schema_migrations`).Scan(&n); err != nil || n != 2 {
		t.Fatalf("rows = %d, %v", n, err)
	}
}

func TestMigrateRollsBackAFailedStep(t *testing.T) {
	db, err := Open(context.Background(), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	fsys := fstest.MapFS{
		"migrations/0001_init.sql": {Data: []byte(`SELECT 1;`)},
		"migrations/0002_bad.sql":  {Data: []byte(`CREATE TABLE half (id INTEGER); NOT SQL;`)},
	}
	if err := migrateFrom(context.Background(), db, fsys); err == nil || !strings.Contains(err.Error(), "0002_bad.sql") {
		t.Fatalf("err = %v, want failure naming 0002_bad.sql", err)
	}
	var n int
	_ = db.QueryRow(`SELECT count(*) FROM sqlite_master WHERE name='half'`).Scan(&n)
	if n != 0 {
		t.Fatal("failed migration left a table behind")
	}
}

func TestMigrateRejectsNewerDatabase(t *testing.T) {
	db, err := Open(context.Background(), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec(`INSERT INTO schema_migrations VALUES (99, 'future', 0)`); err != nil {
		t.Fatal(err)
	}
	if err := migrate(context.Background(), db); err == nil || !strings.Contains(err.Error(), "newer") {
		t.Fatalf("err = %v, want newer-schema error", err)
	}
}

func TestOpenFailsWhenDataDirIsAFile(t *testing.T) {
	blocker := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(blocker, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Open(context.Background(), filepath.Join(blocker, "data")); err == nil {
		t.Fatal("Open succeeded under a regular file")
	}
	if _, err := Exists(filepath.Join(blocker, "data")); err == nil {
		t.Fatal("Exists did not report the error")
	}
}

func TestLoadMigrationsRequiresContiguousNumbers(t *testing.T) {
	fsys := fstest.MapFS{
		"migrations/0001_a.sql": {Data: []byte(`SELECT 1;`)},
		"migrations/0003_c.sql": {Data: []byte(`SELECT 1;`)},
	}
	if _, err := loadMigrations(fsys); err == nil {
		t.Fatal("gap in migration numbers accepted")
	}
}
