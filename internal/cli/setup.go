package cli

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/netip"
	"regexp"
	"strings"

	"github.com/PdYrust/karvan/internal/app"
	"github.com/PdYrust/karvan/internal/config"
	"github.com/PdYrust/karvan/internal/settings"
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

// hostName is what -host takes: an IP address or a host name.
var hostName = regexp.MustCompile(`^[A-Za-z0-9]([A-Za-z0-9.-]{0,251}[A-Za-z0-9])?$`)

func runInit(ctx context.Context, streams IO, args []string) error {
	var o app.InitOptions
	cfg, err := loadConfig("init", streams, args, func(fs *flagSetter) {
		fs.IntVar(&o.Port, "port", 0, "panel port, 1 to 65535 (default 0: a random one)")
		fs.StringVar(&o.TLS, "tls", settings.TLSSelf, "how the panel is served: off (plain HTTP), self (Karvan's own certificate) or domain")
		fs.StringVar(&o.Domain, "domain", "", "with -tls domain: the domain to get a Let's Encrypt certificate for, by HTTP-01 on port 80")
		fs.StringVar(&o.Email, "email", "", "with -tls domain: an optional contact address for Let's Encrypt")
		fs.StringVar(&o.Host, "host", "", "this server's address, shown in the panel's address")
	})
	if err != nil {
		return err
	}
	if o.Port < 0 || o.Port > 65535 {
		return fmt.Errorf("%w: -port must be 1 to 65535", errUsage)
	}
	if _, ipErr := netip.ParseAddr(o.Host); o.Host != "" && ipErr != nil && !hostName.MatchString(o.Host) {
		return fmt.Errorf("%w: -host must be an IP address or a host name", errUsage)
	}
	if err := o.Validate(); err != nil {
		return fmt.Errorf("%w: %w", errUsage, err)
	}
	creds, err := app.InitializeWith(ctx, cfg, o, newLogger(streams.Stderr, cfg))
	if err != nil {
		return err
	}
	if creds.Notice != "" {
		fmt.Fprintf(streams.Stdout, "\n%s\n", creds.Notice)
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
		host := "<server address>"
		if c.Host != "" {
			host = c.Host
			if ip, err := netip.ParseAddr(host); err == nil && ip.Is6() {
				host = "[" + host + "]"
			}
		}
		fmt.Fprintf(&b, "  Address   %s://%s:%s/%s/\n", scheme, host, port, path)
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
