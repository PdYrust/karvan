package app

import (
	"context"
	"fmt"
	"io/fs"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/PdYrust/karvan/internal/buildinfo"
	"github.com/PdYrust/karvan/internal/hoststats"
	"github.com/PdYrust/karvan/internal/httpapi"
	"github.com/PdYrust/karvan/internal/httpapi/address"
	"github.com/PdYrust/karvan/internal/settings"
)

const janitorInterval = 10 * time.Minute

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
	deps := httpapi.Deps{
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
	}
	srv := a.addressServer(h.Bind, func() { close(closing) })
	deps.Address = srv
	srv.Handler = func(base string) (http.Handler, error) {
		d := deps
		d.BasePath = base
		return httpapi.New(d)
	}
	ln, err := net.Listen("tcp", h.Addr())
	if err != nil {
		return fmt.Errorf("listen on %s: %w", h.Addr(), err)
	}
	return a.serveOn(ctx, srv, address.Site{Port: h.Port, BasePath: h.BasePath}, ln)
}

// addressServer serves the panel on bind and saves a confirmed change of
// address in the settings.
func (a *App) addressServer(bind string, onShutdown func()) *address.Server {
	return &address.Server{
		Bind: bind,
		Save: func(ctx context.Context, s address.Site) error {
			return a.Settings.SetHTTP(ctx, settings.HTTP{Bind: bind, Port: s.Port, BasePath: s.BasePath})
		},
		HTTP:       a.httpServer,
		OnShutdown: onShutdown,
		Log:        a.Logger,
	}
}

// httpServer is the server of each listener.
func (a *App) httpServer(h http.Handler) *http.Server {
	return &http.Server{
		Handler:           h,
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
}

// serveOn serves site on ln through srv until ctx ends.
func (a *App) serveOn(ctx context.Context, srv *address.Server, site address.Site, ln net.Listener) error {
	go a.janitor(ctx)
	// The address is logged; the hidden path never is.
	a.Logger.Info("karvan is listening", "addr", ln.Addr().String(), "version", buildinfo.Version)
	if err := srv.Serve(ctx, site, ln); err != nil {
		return err
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
