package cli

import (
	"context"
	"fmt"

	"github.com/PdYrust/karvan/internal/app"
)

// runAccess is "karvan access open-ip": the way back in when only the
// Karvan domain opens the panel and the domain does not work.
func runAccess(ctx context.Context, streams IO, args []string) error {
	if len(args) == 0 || args[0] != "open-ip" {
		return fmt.Errorf("%w: usage: karvan access open-ip", errUsage)
	}
	cfg, err := loadConfig("access open-ip", streams, args[1:], nil)
	if err != nil {
		return err
	}
	a, err := app.Open(ctx, cfg, newLogger(streams.Stderr, cfg))
	if err != nil {
		return err
	}
	defer func() { _ = a.Close() }()
	changed, err := a.OpenIP(ctx)
	if err != nil {
		return err
	}
	if changed {
		fmt.Fprintln(streams.Stdout, "The server's IP opens the panel again, with Karvan's own certificate. A running Karvan takes this within a few seconds.")
		fmt.Fprintln(streams.Stdout, "Once the domain works, turn on Only open Karvan on its domain again in Domain settings.")
	} else {
		fmt.Fprintln(streams.Stdout, "The server's IP already opens the panel.")
	}
	return nil
}
