// Package store opens Karvan's own SQLite database and keeps its schema
// current. Feature packages own their queries; this package owns the file.
package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite" // registers the "sqlite" driver
)

// FileName is the database file inside the data directory.
const FileName = "karvan.db"

// Path returns the database path for a data directory.
func Path(dataDir string) string { return filepath.Join(dataDir, FileName) }

// Exists reports whether a database file is present in dataDir.
func Exists(dataDir string) (bool, error) {
	_, err := os.Stat(Path(dataDir))
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("check database: %w", err)
	}
	return true, nil
}

// Open opens (creating if needed) the database in dataDir and applies any
// pending migrations.
func Open(ctx context.Context, dataDir string) (*sql.DB, error) {
	if err := os.MkdirAll(dataDir, 0o700); err != nil {
		return nil, fmt.Errorf("create data directory: %w", err)
	}
	path := Path(dataDir)
	// SQLite creates the -wal and -shm files with the database file's mode,
	// so creating it 0600 first keeps all three private.
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0o600) //nolint:gosec // G304: path is derived from the operator's data dir.
	if err != nil {
		return nil, fmt.Errorf("create database file: %w", err)
	}
	if err := f.Close(); err != nil {
		return nil, fmt.Errorf("create database file: %w", err)
	}

	q := url.Values{}
	for _, p := range []string{"foreign_keys(1)", "journal_mode(WAL)", "synchronous(NORMAL)", "busy_timeout(5000)"} {
		q.Add("_pragma", p)
	}
	q.Set("_txlock", "immediate")
	db, err := sql.Open("sqlite", "file:"+path+"?"+q.Encode())
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	if err := migrate(ctx, db); err != nil {
		_ = db.Close()
		return nil, err
	}
	return db, nil
}
