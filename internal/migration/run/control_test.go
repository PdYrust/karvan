package run

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestPanelControlProbeIsReusedForAMinute(t *testing.T) {
	now := time.Unix(1_800_000_000, 0)
	calls := 0
	denied := errors.New("no docker")
	svc := New(Deps{Now: func() time.Time { return now }, Control: func(context.Context, string) (PanelControl, error) {
		calls++
		return nil, denied
	}})
	ctx := context.Background()
	for range 3 {
		if err := svc.PanelControlStatus(ctx, "/opt/p/.env", false); !errors.Is(err, ErrNoPanelControl) || !errors.Is(err, denied) {
			t.Fatalf("status = %v", err)
		}
	}
	if calls != 1 {
		t.Fatalf("probed %d times within the TTL, want 1", calls)
	}
	_ = svc.PanelControlStatus(ctx, "/opt/p/.env", true)
	now = now.Add(controlTTL)
	_ = svc.PanelControlStatus(ctx, "/opt/p/.env", false)
	_ = svc.PanelControlStatus(ctx, "/opt/other/.env", false)
	if calls != 4 {
		t.Fatalf("probed %d times, want 4 (fresh, expired, another panel)", calls)
	}
}

// A probe its request cut short is not kept: the next look probes again
// instead of showing "no Docker access" for a minute.
func TestPanelControlProbeCutShortIsNotKept(t *testing.T) {
	calls := 0
	svc := New(Deps{Control: func(ctx context.Context, _ string) (PanelControl, error) {
		calls++
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		return nil, nil
	}})
	gone, cancel := context.WithCancel(context.Background())
	cancel()
	if err := svc.PanelControlStatus(gone, "/opt/p/.env", false); err == nil {
		t.Fatal("a probe cut short reported the panel controllable")
	}
	if err := svc.PanelControlStatus(context.Background(), "/opt/p/.env", false); err != nil || calls != 2 {
		t.Fatalf("after a cut-short probe: %v, %d probes, want nil and 2", err, calls)
	}
}
