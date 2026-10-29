package app

import (
	"context"
	"fmt"
	"io/fs"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/PdYrust/karvan/internal/bridge"
	"github.com/PdYrust/karvan/internal/buildinfo"
	"github.com/PdYrust/karvan/internal/hoststats"
	"github.com/PdYrust/karvan/internal/httpapi"
	"github.com/PdYrust/karvan/internal/httpapi/address"
	"github.com/PdYrust/karvan/internal/paneltls"
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
	srv := a.addressServer(h.Bind, func() { close(closing) })
	svc := a.startServices(ctx, panelPorts(srv))
	defer func() { cancel(); svc.stop() }()
	tlsm := a.panelTLS()
	tlsm.Domain = svc.bridge.PanelCertificate
	srv.Certificate = tlsm.Certificate
	srv.Share = shareWith(ctx, svc.bridge)
	go tlsm.Run(ctx)
	if h.TLS.On() {
		// A certificate that cannot be loaded is logged now, not at the first visit (D11).
		if _, err := tlsm.Certificate(ctx, h.TLS); err != nil {
			a.Logger.Error("the panel certificate cannot be loaded", "err", err)
		}
	}
	deps := httpapi.Deps{
		TrustedProxies: a.Config.Proxies(),
		Auth:           a.Auth,
		Audit:          a.Audit,
		Web:            web,
		Version:        buildinfo.Version,
		Logger:         a.Logger,
		Runs:           svc.runs,
		Bridge:         svc.bridge,
		Orders:         svc.orders,
		Settings:       a.Settings,
		ScanRoots:      scanRoots(),
		Closing:        closing,
		Host:           svc.host,
		HostStore:      svc.hostStore,
		HostReader:     hoststats.System,
		DBStats:        svc.dbstats,
		PanelTLS:       tlsm,
	}
	deps.Address = srv
	srv.Handler = func(base string) (http.Handler, error) {
		d := deps
		d.BasePath = base
		return httpapi.New(d)
	}
	var ln net.Listener
	if !h.TLS.Shared { // a shared port is the bridge's (D12)
		if ln, err = net.Listen("tcp", h.Addr()); err != nil {
			return fmt.Errorf("listen on %s: %w", h.Addr(), err)
		}
	}
	return a.serveOn(ctx, srv, address.Site{Port: h.Port, BasePath: h.BasePath, TLS: h.TLS}, ln)
}

// shareWith hands the panel addresses on shared ports to br.
func shareWith(ctx context.Context, br *bridge.Bridge) func([]address.Shared) {
	return func(sites []address.Shared) {
		out := make([]bridge.PanelSite, len(sites))
		for i, s := range sites {
			out[i] = bridge.PanelSite{Port: s.Port, DomainID: s.TLS.DomainID, BasePath: s.BasePath, Handler: s.Handler}
		}
		br.SharePanel(context.WithoutCancel(ctx), out)
	}
}

// addressServer serves the panel on bind and saves a confirmed change of
// address in the settings.
func (a *App) addressServer(bind string, onShutdown func()) *address.Server {
	return &address.Server{
		Bind: bind,
		Save: func(ctx context.Context, s address.Site) error {
			return a.Settings.SetHTTP(ctx, settings.HTTP{Bind: bind, Port: s.Port, BasePath: s.BasePath, TLS: s.TLS})
		},
		HTTP:       a.httpServer,
		OnShutdown: onShutdown,
		Log:        a.Logger,
	}
}

// panelTLS gives the panel its self-signed certificate; serve adds the
// domains' certificates.
func (a *App) panelTLS() *paneltls.Manager {
	return &paneltls.Manager{Settings: a.Settings, Key: a.Key, Log: a.Logger}
}

// panelPorts lists the ports srv listens on for the panel: the current
// one and a new one waiting for its confirmation. A shared port is the
// bridge's, and rules stay on it.
func panelPorts(srv *address.Server) func() []int {
	return func() []int {
		st := srv.Status()
		var ports []int
		if !st.Current.TLS.Shared {
			ports = append(ports, st.Current.Port)
		}
		if p := st.Pending; p != nil && !p.TLS.Shared {
			ports = append(ports, p.Port)
		}
		return ports
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
	if ln != nil {
		a.Logger.Info("karvan is listening", "addr", ln.Addr().String(), "version", buildinfo.Version)
	} else {
		a.Logger.Info("karvan is served on a shared bridge port", "port", site.Port, "version", buildinfo.Version)
	}
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
