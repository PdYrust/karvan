// Package address serves the web panel on its address (a port and a
// hidden path) and, while a change of address waits for its confirmation,
// on the new one next to it. Nothing is saved until the operator confirms
// from the new address; otherwise the new one goes away (ADR 0009). Every
// port answers HTTPS, with the certificate the server name picks, and
// plain HTTP only where it is allowed (ADR 0010).
package address

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/netip"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/PdYrust/karvan/internal/httpapi/middleware"
	"github.com/PdYrust/karvan/internal/settings"
)

// TTL is how long a new address is served without a confirmation.
const TTL = 10 * time.Minute

const (
	shutdownTimeout = 10 * time.Second
	// A port that is let go gets this long to finish its requests.
	dropGrace   = 5 * time.Second
	randomTries = 8
)

// Errors of Start and Confirm.
var (
	ErrNoChange        = errors.New("the new address is the current one")
	ErrPortUnavailable = errors.New("the port cannot be opened")
	ErrNoPending       = errors.New("no address change is waiting")
	ErrNotNew          = errors.New("an address change is confirmed from the new address")
	errClosed          = errors.New("the server is shutting down")
)

// Site is an address the panel answers on.
type Site struct {
	Port     int
	BasePath string
	// Shared is a bridge HTTPS port of the Karvan domain (ADR 0011): the
	// bridge listens there, the panel has no listener of its own.
	Shared bool
}

// Change says what Start changes: a new random hidden path and the port
// (0 keeps it, -1 picks a random one).
type Change struct {
	NewPath bool
	Port    int
}

// Shared is an address on a bridge HTTPS port (Site.Shared): the bridge
// listens there and hands Handler the requests for the domain under the
// hidden path.
type Shared struct {
	Site
	Handler http.Handler
}

// Pending is a new address waiting for its confirmation.
type Pending struct {
	Site
	Expires time.Time
}

// Status is where the panel is served.
type Status struct {
	Current Site
	Pending *Pending
}

// Server owns the listeners. Set the exported fields, then call Serve.
type Server struct {
	Bind string
	// Handler builds the panel for a hidden path.
	Handler func(basePath string) (http.Handler, error)
	// Save stores a confirmed address.
	Save func(ctx context.Context, s Site) error
	// HTTP makes the server of each listener (timeouts, error log).
	HTTP func(h http.Handler) *http.Server
	// Certificate is the certificate for a TLS handshake that asks for
	// a server name ("" when it asks for none, as for an IP address).
	Certificate func(ctx context.Context, name string) (*tls.Certificate, error)
	// Plain reports whether a plain HTTP request for host (lower case, no
	// port) is answered as is; every other one is sent on to HTTPS.
	Plain func(ctx context.Context, host string) bool
	// PlainFrom are the peers whose plain HTTP requests are answered as
	// is: this server and the trusted proxies, where TLS ends in front of
	// the panel.
	PlainFrom []netip.Prefix
	// Allow reports whether the panel opens for name, a TLS server name
	// or a request's host (lower case, no port, "" for none); a handshake
	// for any other name is refused and a request gets the plain 404. Nil
	// opens it for every name.
	Allow func(ctx context.Context, name string) bool
	// Local are this server's own addresses, which always get the panel,
	// as through an SSH tunnel.
	Local []netip.Prefix
	// Share hands the addresses served on a bridge HTTPS port to the
	// bridge (ADR 0011), each time they change; nil means none.
	Share func(sites []Shared)
	// OnShutdown runs once when the final shutdown starts.
	OnShutdown func()
	Log        *slog.Logger
	// TTL overrides the package TTL, for tests.
	TTL time.Duration
	// Listen overrides net.Listen, for tests.
	Listen func(network, addr string) (net.Listener, error)

	mu      sync.Mutex
	current site
	pending *pending
	ports   map[int]*listener
	fatal   chan error
	closing bool
	wg      sync.WaitGroup
}

type site struct {
	Site
	h http.Handler
}

type pending struct {
	site
	expires time.Time
	timer   *time.Timer
}

// listener serves the sites of one port: the current one, or the new one,
// or both when only the path changes.
type listener struct {
	ln        net.Listener
	srv       *http.Server
	sites     atomic.Pointer[[]site]
	plain     func(r *http.Request) bool
	open      func(r *http.Request) bool
	tlsConfig *tls.Config
}

type ctxKey struct{}

// From returns the address a request came through.
func From(ctx context.Context) Site {
	s, _ := ctx.Value(ctxKey{}).(Site)
	return s
}

// ServeHTTP serves a request under a hidden path of the port, for a host
// the panel opens for. A plain HTTP one is served only when l.plain allows
// it; otherwise a GET or HEAD is sent on to HTTPS and anything else gets
// the plain 404.
func (l *listener) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !l.open(r) {
		middleware.NotFound(w, r)
		return
	}
	for _, s := range *l.sites.Load() {
		if !middleware.UnderBase(r.URL.Path, s.BasePath) {
			continue
		}
		switch {
		case r.TLS != nil || l.plain(r):
			s.ServeHTTP(w, r)
		case r.Method == http.MethodGet || r.Method == http.MethodHead:
			toHTTPS(w, r)
		default:
			middleware.NotFound(w, r)
		}
		return
	}
	middleware.NotFound(w, r)
}

// plainOK reports whether plain HTTP request r is answered as is: it comes
// from a peer in PlainFrom, or Plain allows its host.
func (s *Server) plainOK(r *http.Request) bool {
	if ip, err := netip.ParseAddrPort(r.RemoteAddr); err == nil {
		for _, p := range s.PlainFrom {
			if p.Contains(ip.Addr().Unmap()) {
				return true
			}
		}
	}
	return s.Plain != nil && s.Plain(r.Context(), RequestHost(r))
}

// opens reports whether the panel opens for name from peer: a local peer
// always, any other one when Allow says so.
func (s *Server) opens(ctx context.Context, peer, name string) bool {
	if s.Allow == nil {
		return true
	}
	if ip, err := netip.ParseAddrPort(peer); err == nil {
		for _, p := range s.Local {
			if p.Contains(ip.Addr().Unmap()) {
				return true
			}
		}
	}
	return s.Allow(ctx, name)
}

// openFor reports whether the panel opens for request r.
func (s *Server) openFor(r *http.Request) bool {
	return s.opens(r.Context(), r.RemoteAddr, RequestHost(r))
}

// RequestHost is the host a request names, in lower case, without the
// port.
func RequestHost(r *http.Request) string {
	host := r.Host
	if h, _, err := net.SplitHostPort(host); err == nil {
		host = h
	}
	return strings.ToLower(strings.TrimSuffix(strings.Trim(host, "[]"), "."))
}

// toHTTPS sends a plain HTTP request for an HTTPS address on to the same
// address over HTTPS. Only a request that already names the hidden path
// gets here, so the redirect reveals nothing; it is temporary, so a later
// change back to plain HTTP is not held up by a browser's cache.
func toHTTPS(w http.ResponseWriter, r *http.Request) {
	h := w.Header()
	h.Set("Location", "https://"+r.Host+r.URL.RequestURI())
	h.Set("Cache-Control", "no-store")
	h.Set("Content-Length", "0")
	w.WriteHeader(http.StatusTemporaryRedirect)
}

// ServeHTTP serves a request that came through s. No HSTS is sent: the
// Karvan domain's name falls back to the self-signed certificate when its
// own is gone or the domain is stopped, and other services may share the
// name over plain HTTP.
func (s site) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.h.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), ctxKey{}, s.Site)))
}

// share hands the shared addresses to the bridge. Callers hold mu.
func (s *Server) share() {
	if s.Share == nil {
		return
	}
	var out []Shared
	for _, st := range []*site{&s.current, s.pendingSite()} {
		if st != nil && st.Shared {
			out = append(out, Shared{Site: st.Site, Handler: *st})
		}
	}
	s.Share(out)
}

func (s *Server) pendingSite() *site {
	if s.pending == nil {
		return nil
	}
	return &s.pending.site
}

func (l *listener) serve(sites ...site) { l.sites.Store(&sites) }

// Serve serves current on ln until ctx ends or that listener fails, then
// shuts every listener down. A shared current address has no listener of
// its own: ln is nil and the bridge serves it.
func (s *Server) Serve(ctx context.Context, current Site, ln net.Listener) error {
	h, err := s.Handler(current.BasePath)
	if err != nil {
		if ln != nil {
			_ = ln.Close()
		}
		return err
	}
	s.mu.Lock()
	s.current, s.ports, s.fatal = site{Site: current, h: h}, map[int]*listener{}, make(chan error, 1)
	if ln != nil {
		s.start(current.Port, ln, s.current)
	}
	s.share()
	s.mu.Unlock()

	select {
	case err = <-s.fatal:
		err = fmt.Errorf("serve: %w", err)
	case <-ctx.Done():
	}
	if serr := s.shutdown(); serr != nil && err == nil {
		err = serr
	}
	return err
}

// start serves sites on ln. Callers hold mu.
func (s *Server) start(port int, ln net.Listener, sites ...site) {
	l := &listener{plain: s.plainOK, open: s.openFor}
	l.ln = newConnListener(ln, l)
	l.tlsConfig = &tls.Config{MinVersion: tls.VersionTLS12, NextProtos: []string{"http/1.1"}, GetCertificate: s.certificate}
	l.serve(sites...)
	l.srv = s.HTTP(l)
	s.ports[port] = l
	s.wg.Go(func() {
		if err := l.srv.Serve(l.ln); !errors.Is(err, http.ErrServerClosed) {
			s.failed(l, err)
		}
	})
}

// failed handles a listener that stopped by itself: the current one ends
// Serve, a new one is dropped with its change.
func (s *Server) failed(l *listener, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	switch {
	case s.closing:
	case s.ports[s.current.Port] == l:
		select {
		case s.fatal <- err:
		default:
		}
	case s.pending != nil && s.ports[s.pending.Port] == l:
		s.log().Warn("the new address stopped answering", "port", s.pending.Port, "err", err)
		s.dropPending()
	}
}

func (s *Server) shutdown() error {
	s.mu.Lock()
	s.closing = true
	if s.pending != nil {
		s.pending.timer.Stop()
		s.pending = nil
	}
	ls := make([]*listener, 0, len(s.ports))
	for _, l := range s.ports {
		ls = append(ls, l)
	}
	clear(s.ports)
	if s.Share != nil {
		s.Share(nil)
	}
	s.mu.Unlock()

	if s.OnShutdown != nil {
		s.OnShutdown()
	}
	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	var err error
	for _, l := range ls {
		if serr := l.srv.Shutdown(ctx); serr != nil {
			_ = l.srv.Close()
			err = fmt.Errorf("shut down: %w", serr)
		}
	}
	s.wg.Wait()
	return err
}

// drop stops serving port. The port is free when drop returns, so a new
// change may open it again; requests in flight get dropGrace to finish.
// Callers hold mu.
func (s *Server) drop(port int) {
	l := s.ports[port]
	if l == nil {
		return
	}
	delete(s.ports, port)
	_ = l.ln.Close()
	s.wg.Go(func() {
		ctx, cancel := context.WithTimeout(context.Background(), dropGrace)
		defer cancel()
		if err := l.srv.Shutdown(ctx); err != nil {
			_ = l.srv.Close()
		}
	})
}

// dropPending stops serving the new address. Callers hold mu.
func (s *Server) dropPending() {
	p := s.pending
	if p == nil {
		return
	}
	s.pending = nil
	p.timer.Stop()
	switch {
	case p.Shared:
	case p.Port == s.current.Port && !s.current.Shared:
		if l := s.ports[p.Port]; l != nil {
			l.serve(s.current)
		}
	default:
		s.drop(p.Port)
	}
	s.share()
}

func (s *Server) log() *slog.Logger {
	if s.Log == nil {
		return slog.Default()
	}
	return s.Log
}

func (s *Server) ttl() time.Duration {
	if s.TTL > 0 {
		return s.TTL
	}
	return TTL
}

func (s *Server) listen(port int) (net.Listener, error) {
	listen := s.Listen
	if listen == nil {
		listen = net.Listen
	}
	return listen("tcp", net.JoinHostPort(s.Bind, strconv.Itoa(port)))
}

// randomPort opens a random port other than the current one.
func (s *Server) randomPort() (int, net.Listener, error) {
	var last error
	for range randomTries {
		port, err := settings.RandomPort()
		if err != nil {
			return 0, nil, err
		}
		if port == s.current.Port {
			continue
		}
		ln, err := s.listen(port)
		if err == nil {
			return port, ln, nil
		}
		last = err
	}
	return 0, nil, fmt.Errorf("%w: %w", ErrPortUnavailable, last)
}
