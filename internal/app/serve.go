package app

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/PdYrust/karvan/internal/buildinfo"
	"github.com/PdYrust/karvan/internal/hoststats"
	"github.com/PdYrust/karvan/internal/httpapi"
)

const (
	shutdownTimeout = 10 * time.Second
	janitorInterval = 10 * time.Minute
)

// Serve runs the web panel until ctx ends, then shuts down gracefully.
func (a *App) Serve(ctx context.Context, web fs.FS) error {
	h, err := a.Settings.HTTP(ctx)
	if err != nil {
		return err
	}
	unlock, err := a.lockDataDir()
	if err != nil {
		return err
	}
	defer unlock()
	ctx, cancel := context.WithCancel(ctx)
	closing := make(chan struct{})
	svc := a.startServices(ctx)
	defer func() { cancel(); svc.stop() }()
	handler, err := httpapi.New(httpapi.Deps{
		BasePath:       h.BasePath,
		TrustedProxies: a.Config.Proxies(),
		Auth:           a.Auth,
		Audit:          a.Audit,
		Web:            web,
		Version:        buildinfo.Version,
		Logger:         a.Logger,
		Runs:           svc.runs,
		Bridge:         svc.bridge,
		Settings:       a.Settings,
		ScanRoots:      scanRoots(),
		Closing:        closing,
		Host:           svc.host,
		HostStore:      svc.hostStore,
		HostReader:     hoststats.System,
		DBStats:        svc.dbstats,
	})
	if err != nil {
		return err
	}
	ln, err := net.Listen("tcp", h.Addr())
	if err != nil {
		return fmt.Errorf("listen on %s: %w", h.Addr(), err)
	}
	return a.serveOn(ctx, ln, handler, func() { close(closing) })
}

// serveOn serves handler on ln until ctx ends. onShutdown, if set, runs
// when the shutdown starts.
func (a *App) serveOn(ctx context.Context, ln net.Listener, handler http.Handler, onShutdown func()) error {
	srv := &http.Server{
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       120 * time.Second,
		MaxHeaderBytes:    16 << 10,
		ErrorLog:          slog.NewLogLogger(a.Logger.Handler(), slog.LevelDebug),
		// Otherwise net/http answers "OPTIONS *" with 200 itself, which tells a
		// scanner something is here.
		DisableGeneralOptionsHandler: true,
	}
	if onShutdown != nil {
		srv.RegisterOnShutdown(onShutdown)
	}
	go a.janitor(ctx)

	errc := make(chan error, 1)
	go func() { errc <- srv.Serve(ln) }()
	// The address is logged; the hidden path never is.
	a.Logger.Info("karvan is listening", "addr", ln.Addr().String(), "version", buildinfo.Version)

	select {
	case err := <-errc:
		return fmt.Errorf("serve: %w", err)
	case <-ctx.Done():
	}
	shutdownCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), shutdownTimeout)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		_ = srv.Close()
		return fmt.Errorf("shut down: %w", err)
	}
	if err := <-errc; err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("serve: %w", err)
	}
	a.Logger.Info("karvan stopped")
	return nil
}

// janitor removes expired sessions and stale lockout rows.
func (a *App) janitor(ctx context.Context) {
	t := time.NewTicker(janitorInterval)
	defer t.Stop()
	for {
		if err := a.Auth.Sessions.PurgeExpired(ctx); err != nil && ctx.Err() == nil {
			a.Logger.Warn("purge sessions", "err", err)
		}
		if err := a.Auth.Limiter.Prune(ctx); err != nil && ctx.Err() == nil {
			a.Logger.Warn("prune sign-in failures", "err", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
	}
}
