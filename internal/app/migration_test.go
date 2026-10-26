package app

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/PdYrust/karvan/internal/audit"
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
	if err := a.RestoreBackup(ctx, "0123456789abcdef", true); !errors.Is(err, ErrDataDirInUse) {
		t.Fatalf("restore while running: %v", err)
	}
	sctx, cancel := context.WithCancel(ctx)
	s := a.startServices(sctx)
	cancel()
	s.stop()
	unlock()
	if err := a.RestoreBackup(ctx, "0123456789abcdef", true); err == nil {
		t.Fatal("an unknown backup restored")
	}
}

// The services trim audit events past audit.Kept as they start.
func TestServicesTrimOldAuditEvents(t *testing.T) {
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
	old := audit.Event{At: time.Now().Add(-audit.Kept - time.Hour), Actor: "owner", Action: audit.ActionLogin, Outcome: audit.Success}
	if err := a.Audit.Record(ctx, old); err != nil {
		t.Fatal(err)
	}
	count := func() (n int) {
		_ = a.DB.QueryRow("SELECT count(*) FROM audit_events WHERE at < ?", time.Now().Add(-audit.Kept).UnixMilli()).Scan(&n)
		return n
	}
	sctx, cancel := context.WithCancel(ctx)
	s := a.startServices(sctx)
	deadline := time.Now().Add(5 * time.Second)
	for count() > 0 && time.Now().Before(deadline) {
		time.Sleep(20 * time.Millisecond)
	}
	cancel()
	s.stop()
	if n := count(); n != 0 {
		t.Fatalf("%d old audit events left", n)
	}
}
