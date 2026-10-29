package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/PdYrust/karvan/internal/audit"
	"github.com/PdYrust/karvan/internal/auth"
	"github.com/PdYrust/karvan/internal/config"
	"github.com/PdYrust/karvan/internal/secret"
	"github.com/PdYrust/karvan/internal/settings"
	"github.com/PdYrust/karvan/internal/store"
)

// Credentials are the one-time values printed to the terminal. Fields that
// did not change in a reset are empty.
type Credentials struct {
	Username string
	Password secret.Value
	BasePath string
	Port     int
	// HTTPS and Fingerprint say how the panel is served: the SHA-256 of
	// the self-signed certificate, in self mode, to compare with what the
	// browser shows.
	HTTPS       bool
	Fingerprint string
}

// Initialize creates the key, the database and the operator account of a
// new installation on a random port. It refuses to touch an existing one,
// and removes what it created if a later step fails, so init can simply be
// run again.
func Initialize(ctx context.Context, cfg config.Config, logger *slog.Logger) (Credentials, error) {
	return InitializeOnPort(ctx, cfg, 0, logger)
}

// InitializeOnPort is Initialize on the given port; 0 picks a random one.
func InitializeOnPort(ctx context.Context, cfg config.Config, port int, logger *slog.Logger) (creds Credentials, err error) {
	exists, err := store.Exists(cfg.DataDir)
	if err != nil {
		return Credentials{}, err
	}
	if exists {
		return Credentials{}, errors.New("karvan is already initialized here; use: karvan reset")
	}
	if _, err := os.Stat(cfg.KeyFile); err == nil {
		return Credentials{}, fmt.Errorf("key file %s already exists but there is no database; if nothing was set up yet, delete it and run init again", cfg.KeyFile)
	}
	if err := secret.CreateKeyFile(cfg.KeyFile); err != nil {
		return Credentials{}, err
	}
	defer func() {
		if err != nil {
			removeInstallation(cfg)
		}
	}()
	key, err := secret.LoadKeyFile(cfg.KeyFile)
	if err != nil {
		return Credentials{}, err
	}
	db, err := store.Open(ctx, cfg.DataDir)
	if err != nil {
		return Credentials{}, err
	}
	a := assemble(cfg, db, key, logger)
	defer func() { _ = a.Close() }()

	creds, err = newCredentials()
	if err != nil {
		return Credentials{}, err
	}
	if port != 0 {
		creds.Port = port
	}
	hash, err := auth.HashPassword(creds.Password.Reveal())
	if err != nil {
		return Credentials{}, err
	}
	if _, err := a.Auth.Operators.Create(ctx, creds.Username, hash); err != nil {
		return Credentials{}, err
	}
	// A new installation starts on HTTPS with its own certificate (D5).
	h := settings.HTTP{Port: creds.Port, BasePath: creds.BasePath, TLS: settings.TLS{Mode: settings.TLSSelf}}
	if err := a.Settings.SetHTTP(ctx, h); err != nil {
		return Credentials{}, err
	}
	if err := a.describeTLS(ctx, h.TLS, &creds); err != nil {
		return Credentials{}, err
	}
	if err = a.Audit.Record(ctx, audit.Event{Actor: audit.ActorCLI, Action: audit.ActionInit, Outcome: audit.Success}); err != nil {
		return Credentials{}, err
	}
	return creds, nil
}

// removeInstallation deletes the files a failed Initialize created. Only
// these exact paths are touched, never the directories around them.
func removeInstallation(cfg config.Config) {
	db := store.Path(cfg.DataDir)
	for _, p := range []string{db, db + "-wal", db + "-shm", cfg.KeyFile} {
		_ = os.Remove(p)
	}
}

func newCredentials() (Credentials, error) {
	user, err := auth.GenerateUsername()
	if err != nil {
		return Credentials{}, err
	}
	pw, err := auth.GeneratePassword()
	if err != nil {
		return Credentials{}, err
	}
	path, err := settings.NewBasePath()
	if err != nil {
		return Credentials{}, err
	}
	port, err := settings.RandomPort()
	if err != nil {
		return Credentials{}, err
	}
	return Credentials{Username: user, Password: secret.NewValue(pw), BasePath: path, Port: port}, nil
}

// ResetOptions choose what Reset regenerates.
type ResetOptions struct {
	Path, Username, Password, TOTP bool
	// Port: 0 keeps it, -1 picks a random one, anything else is used as is.
	Port int
}

// Any reports whether anything was chosen.
func (o ResetOptions) Any() bool { return o.Path || o.Username || o.Password || o.TOTP || o.Port != 0 }

// Reset regenerates the chosen values for an operator who lost access.
// A new username or password, or turning two-step off, ends every session.
// If a step fails, the values already changed are still returned with the
// error, so they can be shown and are never lost.
func (a *App) Reset(ctx context.Context, o ResetOptions) (Credentials, error) {
	op, err := a.operator(ctx)
	if err != nil {
		return Credentials{}, err
	}
	gen, err := newCredentials()
	if err != nil {
		return Credentials{}, err
	}
	var out Credentials
	var changed []string
	if o.Username {
		if err := a.Auth.Operators.SetUsername(ctx, op.ID, gen.Username); err != nil {
			return out, err
		}
		out.Username, changed = gen.Username, append(changed, "username")
	}
	if o.Password {
		hash, err := auth.HashPassword(gen.Password.Reveal())
		if err != nil {
			return out, err
		}
		if err := a.Auth.Operators.SetPasswordHash(ctx, op.ID, hash); err != nil {
			return out, err
		}
		out.Password, changed = gen.Password, append(changed, "password")
	}
	if o.TOTP {
		if err := a.Auth.Operators.DisableTOTP(ctx, op.ID); err != nil {
			return out, err
		}
		changed = append(changed, "totp")
	}
	if err := a.resetHTTP(ctx, o, gen, &out, &changed); err != nil {
		return out, err
	}
	if o.Username || o.Password || o.TOTP {
		if err := a.Auth.Sessions.DeleteForOperator(ctx, op.ID); err != nil {
			return out, err
		}
	}
	err = a.Audit.Record(ctx, audit.Event{Actor: audit.ActorCLI, Action: audit.ActionReset, Outcome: audit.Success,
		Detail: map[string]string{"changed": strings.Join(changed, ",")}})
	return out, err
}

func (a *App) resetHTTP(ctx context.Context, o ResetOptions, gen Credentials, out *Credentials, changed *[]string) error {
	h, err := a.Settings.HTTP(ctx)
	if err != nil {
		return err
	}
	if err := a.describeTLS(ctx, h.TLS, out); err != nil {
		return err
	}
	if !o.Path && o.Port == 0 {
		return nil
	}
	if o.Path {
		h.BasePath, out.BasePath = gen.BasePath, gen.BasePath
		*changed = append(*changed, "path")
	}
	if o.Port != 0 {
		h.Port, h.TLS.Shared = o.Port, false // the new port is the panel's own
		if o.Port < 0 {
			h.Port = gen.Port
		}
		out.Port = h.Port
		*changed = append(*changed, "port")
	}
	return a.Settings.SetHTTP(ctx, h)
}

// describeTLS fills in how the panel is served, making the self-signed
// certificate when it does not exist yet.
func (a *App) describeTLS(ctx context.Context, t settings.TLS, c *Credentials) error {
	c.HTTPS = t.On()
	if t.Mode != settings.TLSSelf {
		return nil
	}
	fp, err := a.panelTLS().Fingerprint(ctx)
	if err != nil {
		return err
	}
	c.Fingerprint = fp
	return nil
}
