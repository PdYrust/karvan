package cli

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"strings"

	"github.com/PdYrust/karvan/internal/app"
	"github.com/PdYrust/karvan/internal/config"
)

func loadConfig(name string, streams IO, args []string, extra func(*flagSetter)) (config.Config, error) {
	fs := newFlagSet(name, streams)
	flags := config.BindFlags(fs)
	if extra != nil {
		extra(&flagSetter{fs})
	}
	if err := parseFlags(fs, args); err != nil {
		return config.Config{}, err
	}
	return flags.Load()
}

// newLogger also becomes the default, so code that logs through slog's
// package functions uses the configured level and output.
func newLogger(w io.Writer, cfg config.Config) *slog.Logger {
	l := slog.New(slog.NewTextHandler(w, &slog.HandlerOptions{Level: cfg.Level()}))
	slog.SetDefault(l)
	return l
}

func runInit(ctx context.Context, streams IO, args []string) error {
	cfg, err := loadConfig("init", streams, args, nil)
	if err != nil {
		return err
	}
	creds, err := app.Initialize(ctx, cfg, newLogger(streams.Stderr, cfg))
	if err != nil {
		return err
	}
	printCredentials(streams, "Karvan is set up.", creds)
	return nil
}

// printCredentials shows one-time secrets on the terminal. They are stored
// nowhere in plain text, so this is the only chance to copy them.
func printCredentials(streams IO, title string, c app.Credentials) {
	if !streams.StdoutIsTerminal {
		fmt.Fprintln(streams.Stderr, "warning: output is not a terminal; make sure it is not being saved to a log.")
	}
	var b strings.Builder
	fmt.Fprintf(&b, "\n%s Save these now; they are shown only once.\n\n", title)
	if c.BasePath != "" || c.Port != 0 {
		port := "<port>"
		if c.Port != 0 {
			port = fmt.Sprint(c.Port)
		}
		path := "<path>"
		if c.BasePath != "" {
			path = c.BasePath
		}
		scheme := "http"
		if c.HTTPS {
			scheme = "https"
		}
		fmt.Fprintf(&b, "  Address   %s://<server address>:%s/%s/\n", scheme, port, path)
		if !c.HTTPS {
			defer fmt.Fprintln(streams.Stdout, "This address is plain HTTP. Until HTTPS is set up, sign in only over a network you trust.")
		}
	}
	if c.Fingerprint != "" {
		defer fmt.Fprintf(streams.Stdout, "The browser warns about Karvan's own certificate. Go on only if it shows this SHA-256 fingerprint:\n  %s\n", c.Fingerprint)
	}
	if c.Username != "" {
		fmt.Fprintf(&b, "  Username  %s\n", c.Username)
	}
	if !c.Password.IsZero() {
		fmt.Fprintf(&b, "  Password  %s\n", c.Password.Reveal())
	}
	fmt.Fprint(streams.Stdout, b.String()+"\n")
}
