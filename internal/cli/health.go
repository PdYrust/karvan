package cli

import (
	"context"
	"fmt"
	"time"

	"github.com/PdYrust/karvan/internal/health"
)

// runHealth exits 0 when serve wrote its heartbeat in the last 30 seconds
// (D15). Docker's HEALTHCHECK runs it, so it needs no network.
func runHealth(_ context.Context, streams IO, args []string) error {
	cfg, err := loadConfig("health", streams, args, nil)
	if err != nil {
		return err
	}
	if err := health.Check(health.HeartbeatPath(cfg.DataDir), time.Now()); err != nil {
		return err
	}
	_, err = fmt.Fprintln(streams.Stdout, "Karvan is running.")
	return err
}
