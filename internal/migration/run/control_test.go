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
