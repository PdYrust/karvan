package safemode

import (
	"context"
	"fmt"
	"os"
	"syscall"

	"github.com/PdYrust/karvan/internal/panel/paneldb"
)

// headroom is kept free on top of the estimate (D44).
const headroom = 100 << 20

// Size estimates how big the database is.
func Size(ctx context.Context, q paneldb.Querier, t paneldb.Target) (int64, error) {
	return paneldb.Size(ctx, q, t)
}

// Free is the space available to Karvan's user in dir.
func Free(dir string) (int64, error) {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return 0, err
	}
	var st syscall.Statfs_t
	if err := syscall.Statfs(dir, &st); err != nil {
		return 0, fmt.Errorf("read free space of %s: %w", dir, err)
	}
	return int64(st.Bavail) * st.Bsize, nil //nolint:gosec // block counts fit in int64 on every real disk
}

// SpaceError means the backup would not fit.
type SpaceError struct{ Need, Free int64 }

func (e *SpaceError) Error() string {
	return fmt.Sprintf("backup needs about %d bytes, %d are free", e.Need, e.Free)
}

// CheckSpace refuses a backup that needs more than 1.5 times the database
// plus headroom.
func CheckSpace(size, free int64) error {
	need := size + size/2 + headroom
	if free < need {
		return &SpaceError{Need: need, Free: free}
	}
	return nil
}
