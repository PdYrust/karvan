package store

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

//go:embed migrations/*.sql
var migrationFiles embed.FS

type migration struct {
	version int
	name    string
	sql     string
}

func loadMigrations(fsys fs.FS) ([]migration, error) {
	names, err := fs.Glob(fsys, "migrations/*.sql")
	if err != nil {
		return nil, fmt.Errorf("list migrations: %w", err)
	}
	sort.Strings(names)
	out := make([]migration, 0, len(names))
	for i, name := range names {
		base := strings.TrimPrefix(name, "migrations/")
		prefix, _, ok := strings.Cut(base, "_")
		v, err := strconv.Atoi(prefix)
		if !ok || err != nil || v != i+1 {
			return nil, fmt.Errorf("migration %s: want prefix %04d_", base, i+1)
		}
		body, err := fs.ReadFile(fsys, name)
		if err != nil {
			return nil, fmt.Errorf("read migration %s: %w", base, err)
		}
		out = append(out, migration{version: v, name: base, sql: string(body)})
	}
	return out, nil
}

// SchemaVersion is the store schema version this build migrates to: the
// number of its migrations. An older build refuses a store above its own.
func SchemaVersion() (int, error) {
	all, err := loadMigrations(migrationFiles)
	return len(all), err
}

func migrate(ctx context.Context, db *sql.DB) error {
	return migrateFrom(ctx, db, migrationFiles)
}

func migrateFrom(ctx context.Context, db *sql.DB, fsys fs.FS) error {
	all, err := loadMigrations(fsys)
	if err != nil {
		return err
	}
	if _, err := db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (
		version INTEGER PRIMARY KEY, name TEXT NOT NULL, applied_at INTEGER NOT NULL) STRICT`); err != nil {
		return fmt.Errorf("create schema_migrations: %w", err)
	}
	var current int
	if err := db.QueryRowContext(ctx, `SELECT COALESCE(MAX(version), 0) FROM schema_migrations`).Scan(&current); err != nil {
		return fmt.Errorf("read schema version: %w", err)
	}
	if current > len(all) {
		return fmt.Errorf("database schema version %d is newer than this Karvan build (%d); upgrade Karvan", current, len(all))
	}
	if current > 0 && current < len(all) {
		if err := copyBefore(ctx, db, current+1); err != nil {
			return fmt.Errorf("store copy before migration %d: %w", current+1, err)
		}
	}
	for _, m := range all[current:] {
		if err := apply(ctx, db, m); err != nil {
			return err
		}
	}
	return nil
}

// copyBefore writes <db>.before-<v>, a copy of the database as it is
// before migration v, and deletes the copies taken before older
// migrations (phase 4.1 D43). It is the way back for a migration that
// changes existing data. VACUUM INTO refuses an existing file and creates
// it with the umask, hence the tmp name and the chmod; the data folder is
// 0700, so no other user can open the file before the chmod.
func copyBefore(ctx context.Context, db *sql.DB, v int) error {
	var path string
	if err := db.QueryRowContext(ctx, "SELECT file FROM pragma_database_list WHERE name = 'main'").Scan(&path); err != nil {
		return fmt.Errorf("find database file: %w", err)
	}
	if path == "" {
		return errors.New("the database has no file")
	}
	final := path + ".before-" + strconv.Itoa(v)
	tmp := final + ".tmp"
	if err := os.Remove(tmp); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	// VACUUM cannot run inside a transaction; db.ExecContext runs it alone.
	if _, err := db.ExecContext(ctx, "VACUUM INTO ?", tmp); err != nil {
		return fmt.Errorf("vacuum into %s: %w", tmp, err)
	}
	if err := os.Chmod(tmp, 0o600); err != nil {
		return err
	}
	if err := syncPath(tmp); err != nil {
		return err
	}
	if err := os.Rename(tmp, final); err != nil {
		return err
	}
	if err := syncPath(filepath.Dir(final)); err != nil {
		return err
	}
	return removeOlderCopies(path, v)
}

// syncPath flushes a file or folder to disk.
func syncPath(path string) error {
	f, err := os.Open(path) //nolint:gosec // G304: path is Karvan's own database copy or its folder.
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()
	if err := f.Sync(); err != nil {
		return err
	}
	return f.Close()
}

// removeOlderCopies deletes <db>.before-<n> and <db>.before-<n>.tmp for
// every n below v. A copy for a later migration is not Karvan's to judge.
func removeOlderCopies(path string, v int) error {
	dir, base := filepath.Split(path)
	list, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, e := range list {
		rest, ok := strings.CutPrefix(e.Name(), base+".before-")
		if !ok {
			continue
		}
		n, err := strconv.Atoi(strings.TrimSuffix(rest, ".tmp"))
		if err != nil || n >= v {
			continue
		}
		if err := os.Remove(filepath.Join(dir, e.Name())); err != nil {
			return err
		}
	}
	return nil
}

func apply(ctx context.Context, db *sql.DB, m migration) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("migration %s: begin: %w", m.name, err)
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(ctx, m.sql); err != nil {
		return fmt.Errorf("migration %s: %w", m.name, err)
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO schema_migrations (version, name, applied_at) VALUES (?, ?, ?)`,
		m.version, m.name, time.Now().Unix()); err != nil {
		return fmt.Errorf("migration %s: record: %w", m.name, err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("migration %s: commit: %w", m.name, err)
	}
	return nil
}
