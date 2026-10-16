package app

import (
	"context"
	"errors"
	"testing"
)

// One process per data directory (D30); the services start and stop with
// the context, and a restore refuses while Karvan runs.
func TestServicesAndTheDataDirLock(t *testing.T) {
	ctx := context.Background()
	cfg := testConfig(t)
	if _, err := Initialize(ctx, cfg, quiet()); err != nil {
		t.Fatal(err)
	}
	a, err := Open(ctx, cfg, quiet())
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	unlock, err := a.lockDataDir()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := a.lockDataDir(); !errors.Is(err, ErrDataDirInUse) {
		t.Fatalf("second lock: %v", err)
	}
	if err := a.RestoreBackup(ctx, "0123456789abcdef"); !errors.Is(err, ErrDataDirInUse) {
		t.Fatalf("restore while running: %v", err)
	}
	sctx, cancel := context.WithCancel(ctx)
	s := a.startServices(sctx)
	cancel()
	s.stop()
	unlock()
	if err := a.RestoreBackup(ctx, "0123456789abcdef"); err == nil {
		t.Fatal("an unknown backup restored")
	}
}
