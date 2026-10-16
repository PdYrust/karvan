package paneldb

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
)

func TestLockIsExclusive(t *testing.T) {
	ctx := context.Background()
	tg := newSQLite(t)
	a, err := OpenWriter(ctx, tg)
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	release, err := a.Lock(ctx, tg)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := a.Lock(ctx, tg); !errors.Is(err, ErrLocked) {
		t.Fatalf("second lock: %v", err)
	}
	release()
	again, err := a.Lock(ctx, tg)
	if err != nil {
		t.Fatalf("lock after release: %v", err)
	}
	again()

	p := filepath.Join(t.TempDir(), "karvan.lock")
	r1, err := LockFile(p)
	if err != nil {
		t.Fatal(err)
	}
	defer r1()
	if _, err := LockFile(p); !errors.Is(err, ErrLocked) {
		t.Fatalf("data dir lock: %v", err)
	}
}
