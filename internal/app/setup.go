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
}

// Initialize creates the key, the database and the operator account of a
// new installation. It refuses to touch an existing one.
func Initialize(ctx context.Context, cfg config.Config, logger *slog.Logger) (Credentials, error) {
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

	creds, err := newCredentials()
	if err != nil {
		return Credentials{}, err
	}
	hash, err := auth.HashPassword(creds.Password.Reveal())
	if err != nil {
		return Credentials{}, err
	}
	if _, err := a.Auth.Operators.Create(ctx, creds.Username, hash); err != nil {
		return Credentials{}, err
	}
	if err := a.Settings.SetHTTP(ctx, settings.HTTP{Port: creds.Port, BasePath: creds.BasePath}); err != nil {
		return Credentials{}, err
	}
	err = a.Audit.Record(ctx, audit.Event{Actor: audit.ActorCLI, Action: audit.ActionInit, Outcome: audit.Success})
	return creds, err
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
			return Credentials{}, err
		}
		out.Username, changed = gen.Username, append(changed, "username")
	}
	if o.Password {
		hash, err := auth.HashPassword(gen.Password.Reveal())
		if err != nil {
			return Credentials{}, err
		}
		if err := a.Auth.Operators.SetPasswordHash(ctx, op.ID, hash); err != nil {
			return Credentials{}, err
		}
		out.Password, changed = gen.Password, append(changed, "password")
	}
	if o.TOTP {
		if err := a.Auth.Operators.DisableTOTP(ctx, op.ID); err != nil {
			return Credentials{}, err
		}
		changed = append(changed, "totp")
	}
	if err := a.resetHTTP(ctx, o, gen, &out, &changed); err != nil {
		return Credentials{}, err
	}
	if o.Username || o.Password || o.TOTP {
		if err := a.Auth.Sessions.DeleteForOperator(ctx, op.ID); err != nil {
			return Credentials{}, err
		}
	}
	err = a.Audit.Record(ctx, audit.Event{Actor: audit.ActorCLI, Action: audit.ActionReset, Outcome: audit.Success,
		Detail: map[string]string{"changed": strings.Join(changed, ",")}})
	return out, err
}

func (a *App) resetHTTP(ctx context.Context, o ResetOptions, gen Credentials, out *Credentials, changed *[]string) error {
	if !o.Path && o.Port == 0 {
		return nil
	}
	h, err := a.Settings.HTTP(ctx)
	if err != nil {
		return err
	}
	if o.Path {
		h.BasePath, out.BasePath = gen.BasePath, gen.BasePath
		*changed = append(*changed, "path")
	}
	if o.Port != 0 {
		h.Port = o.Port
		if o.Port < 0 {
			h.Port = gen.Port
		}
		out.Port = h.Port
		*changed = append(*changed, "port")
	}
	return a.Settings.SetHTTP(ctx, h)
}
