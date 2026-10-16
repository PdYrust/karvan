package cli

import (
	"context"
	"flag"
	"fmt"

	"github.com/PdYrust/karvan/internal/app"
)

type flagSetter struct{ *flag.FlagSet }

func runReset(ctx context.Context, streams IO, args []string) error {
	var o app.ResetOptions
	var port bool
	var portNumber int
	cfg, err := loadConfig("reset", streams, args, func(fs *flagSetter) {
		fs.BoolVar(&o.Path, "path", false, "generate a new hidden path")
		fs.BoolVar(&o.Username, "user", false, "generate a new username")
		fs.BoolVar(&o.Password, "password", false, "generate a new password")
		fs.BoolVar(&o.TOTP, "totp", false, "turn two-step sign-in off")
		fs.BoolVar(&port, "port", false, "pick a new random port")
		fs.IntVar(&portNumber, "port-number", 0, "use this port instead of a random one")
	})
	if err != nil {
		return err
	}
	switch {
	case portNumber != 0:
		o.Port = portNumber
	case port:
		o.Port = -1
	}
	if !o.Any() {
		return fmt.Errorf("%w: choose at least one of -path, -user, -password, -totp, -port, -port-number", errUsage)
	}

	a, err := app.Open(ctx, cfg, newLogger(streams.Stderr, cfg))
	if err != nil {
		return err
	}
	defer func() { _ = a.Close() }()
	creds, err := a.Reset(ctx, o)
	if err != nil {
		if creds != (app.Credentials{}) {
			printCredentials(streams, "Reset stopped part way; these values were already changed.", creds)
		}
		return err
	}
	printCredentials(streams, "Reset done.", creds)
	if o.Path || o.Port != 0 {
		fmt.Fprintln(streams.Stdout, "Restart Karvan to use the new address. If the port changed, open it in your firewall first.")
	}
	if o.TOTP {
		fmt.Fprintln(streams.Stdout, "Two-step sign-in is off. Turn it on again from the panel.")
	}
	return nil
}
