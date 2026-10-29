// Package storetest gives tests a migrated Karvan store without running
// every migration again for each test.
package storetest

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"sync"

	"github.com/PdYrust/karvan/internal/store"
)

// migrated is the database file of a store migrated once per test binary.
var migrated = sync.OnceValues(func() ([]byte, error) {
	dir, err := os.MkdirTemp("", "karvan-storetest-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(dir)
	db, err := store.Open(context.Background(), dir)
	if err != nil {
		return nil, err
	}
	// Closing the last connection checkpoints the WAL into the file.
	if err := db.Close(); err != nil {
		return nil, err
	}
	if _, err := os.Stat(store.Path(dir) + "-wal"); err == nil {
		return nil, errors.New("migrated store kept its WAL")
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("migrated store: %w", err)
	}
	return os.ReadFile(store.Path(dir))
})

// Open is store.Open on a new data directory. The database file starts as
// a copy of one migrated store, so store.Open finds nothing to migrate.
func Open(ctx context.Context, dataDir string) (*sql.DB, error) {
	b, err := migrated()
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(dataDir, 0o700); err != nil {
		return nil, err
	}
	if err := os.WriteFile(store.Path(dataDir), b, 0o600); err != nil {
		return nil, err
	}
	return store.Open(ctx, dataDir)
}
