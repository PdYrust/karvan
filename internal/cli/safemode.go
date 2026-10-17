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

// runSafeMode is "karvan safe-mode restore [flags] <backup id>". The web
// panel does the same; this is for when Karvan's server is not running.
// Karvan stops and starts the panel itself unless -panel-stopped says the
// operator already stopped it.
func runSafeMode(ctx context.Context, streams IO, args []string) error {
	if len(args) == 0 || args[0] != "restore" {
		return fmt.Errorf("%w: usage: karvan safe-mode restore [-panel-stopped] <backup id>", errUsage)
	}
	fs := newFlagSet("safe-mode restore", streams)
	flags := config.BindFlags(fs)
	stopped := fs.Bool("panel-stopped", false, "the panel is already stopped; do not stop or start it")
	if err := fs.Parse(args[1:]); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return err
		}
		return fmt.Errorf("%w: %w", errUsage, err)
	}
	if fs.NArg() != 1 || !backupIDRe.MatchString(fs.Arg(0)) {
		return fmt.Errorf("%w: give one backup id (see Safe Mode backups in the panel)", errUsage)
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
	if err := a.RestoreBackup(ctx, fs.Arg(0), *stopped); err != nil {
		return err
	}
	if *stopped {
		fmt.Fprintln(streams.Stdout, "Backup restored. Start the panel again.")
	} else {
		fmt.Fprintln(streams.Stdout, "Backup restored. The panel is running again.")
	}
	return nil
}
