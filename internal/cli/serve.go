package cli

import (
	"context"
	"os/signal"
	"syscall"

	"github.com/PdYrust/karvan/internal/app"
	"github.com/PdYrust/karvan/web"
)

func runServe(ctx context.Context, streams IO, args []string) error {
	cfg, err := loadConfig("serve", streams, args, nil)
	if err != nil {
		return err
	}
	dist, err := web.Dist()
	if err != nil {
		return err
	}
	a, err := app.Open(ctx, cfg, newLogger(streams.Stderr, cfg))
	if err != nil {
		return err
	}
	defer func() { _ = a.Close() }()

	ctx, stop := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	return a.Serve(ctx, dist)
}
