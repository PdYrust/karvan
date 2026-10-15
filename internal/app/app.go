// Package app wires Karvan's parts together and runs them.
package app

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"

	"github.com/PdYrust/karvan/internal/audit"
	"github.com/PdYrust/karvan/internal/auth"
	"github.com/PdYrust/karvan/internal/config"
	"github.com/PdYrust/karvan/internal/secret"
	"github.com/PdYrust/karvan/internal/settings"
	"github.com/PdYrust/karvan/internal/store"
)

// App is an opened, initialized Karvan installation.
type App struct {
	Config   config.Config
	DB       *sql.DB
	Key      *secret.Key
	Auth     *auth.Service
	Audit    *audit.Log
	Settings *settings.Store
	Logger   *slog.Logger
}

// Open loads the key and the database of an installation that `karvan init`
// has already set up.
func Open(ctx context.Context, cfg config.Config, logger *slog.Logger) (*App, error) {
	exists, err := store.Exists(cfg.DataDir)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, settings.ErrNotInitialized
	}
	key, err := secret.LoadKeyFile(cfg.KeyFile)
	if err != nil {
		return nil, err
	}
	db, err := store.Open(ctx, cfg.DataDir)
	if err != nil {
		return nil, err
	}
	return assemble(cfg, db, key, logger), nil
}

func assemble(cfg config.Config, db *sql.DB, key *secret.Key, logger *slog.Logger) *App {
	log := audit.New(db)
	ops := auth.NewOperators(db, key)
	svc := auth.NewService(ops, auth.NewSessions(db), auth.NewLimiter(db, auth.DefaultLockout), log)
	return &App{Config: cfg, DB: db, Key: key, Auth: svc, Audit: log, Settings: settings.New(db), Logger: logger}
}

// Close releases the database.
func (a *App) Close() error {
	if err := a.DB.Close(); err != nil {
		return fmt.Errorf("close database: %w", err)
	}
	return nil
}

// operator returns the single operator of a version 1 installation.
func (a *App) operator(ctx context.Context) (auth.Operator, error) {
	op, err := a.Auth.Operators.First(ctx)
	if errors.Is(err, auth.ErrNotFound) {
		return auth.Operator{}, settings.ErrNotInitialized
	}
	return op, err
}
