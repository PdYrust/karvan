package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"regexp"

	"github.com/PdYrust/karvan/internal/app"
	"github.com/PdYrust/karvan/internal/config"
)

var backupIDRe = regexp.MustCompile(`^[0-9a-f]{16}$`)

// runSafeMode is "karvan safe-mode restore [flags] <backup id>": the
// last resort after a failed migration whose undo left rows behind (D36).
func runSafeMode(ctx context.Context, streams IO, args []string) error {
	if len(args) == 0 || args[0] != "restore" {
		return fmt.Errorf("%w: usage: karvan safe-mode restore -panel-stopped <backup id>", errUsage)
	}
	fs := newFlagSet("safe-mode restore", streams)
	flags := config.BindFlags(fs)
	stopped := fs.Bool("panel-stopped", false, "confirm the panel is stopped (for example: marzban down, pasarguard down)")
	if err := fs.Parse(args[1:]); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return err
		}
		return fmt.Errorf("%w: %w", errUsage, err)
	}
	if fs.NArg() != 1 || !backupIDRe.MatchString(fs.Arg(0)) {
		return fmt.Errorf("%w: give one backup id (see Safe Mode backups in the panel)", errUsage)
	}
	if !*stopped {
		return fmt.Errorf("%w: stop the panel first, then run again with -panel-stopped; "+
			"restoring under a running panel mixes its new writes into the old data", errUsage)
	}
	cfg, err := flags.Load()
	if err != nil {
		return err
	}
	a, err := app.Open(ctx, cfg, newLogger(streams.Stderr, cfg))
	if err != nil {
		return err
	}
	defer func() { _ = a.Close() }()
	if err := a.RestoreBackup(ctx, fs.Arg(0)); err != nil {
		return err
	}
	fmt.Fprintln(streams.Stdout, "Backup restored. Start the panel again.")
	return nil
}
