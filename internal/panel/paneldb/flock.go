package paneldb

import (
	"errors"
	"fmt"
	"os"
	"syscall"
)

// flock takes an exclusive, non-blocking flock on path.
func flock(path string) (func(), error) {
	f, err := os.Open(path) //nolint:gosec // the panel's own database file, from its ENV
	if err != nil {
		return nil, err
	}
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		_ = f.Close()
		if errors.Is(err, syscall.EWOULDBLOCK) {
			return nil, ErrLocked
		}
		return nil, fmt.Errorf("lock %s: %w", path, err)
	}
	return func() {
		_ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
		_ = f.Close()
	}, nil
}

// LockFile takes an exclusive flock on path, creating it 0600. Karvan uses
// it so only one process works on one data directory (D30).
func LockFile(path string) (func(), error) {
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0o600) //nolint:gosec // path is in Karvan's data dir
	if err != nil {
		return nil, err
	}
	_ = f.Close()
	return flock(path)
}
