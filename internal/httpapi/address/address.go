// Package address serves the web panel on its address (a port, a hidden
// path and HTTP or HTTPS) and, while a change of address waits for its confirmation, on the
// new one next to it. Nothing is saved until the operator confirms from
// the new address; otherwise the new one goes away (ADR 0009).
package address

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"strconv"
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
	TLS      settings.TLS // normal form (settings.TLS.Normal)
}

// Change says what Start changes: a new random hidden path, the port (0
// keeps it, -1 picks a random one) and how the panel is served (nil keeps
// it).
type Change struct {
	NewPath bool
	Port    int
	TLS     *settings.TLS
}

// hsts is sent in domain mode only: a self-signed certificate under
// HSTS cannot be clicked through, and a day is short enough that going
// back to plain HTTP does not lock a browser out.
const hsts = "max-age=86400"

// Shared is an address on a bridge HTTPS port (settings.TLS.Shared): the
// bridge listens there and hands Handler the requests for the domain
// under the hidden path.
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
	// Certificate is the certificate of an HTTPS address.
	Certificate func(ctx context.Context, t settings.TLS) (*tls.Certificate, error)
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
	certs     func(ctx context.Context, t settings.TLS) (*tls.Certificate, error)
	tlsConfig *tls.Config
}

type ctxKey struct{}

// From returns the address a request came through.
func From(ctx context.Context) Site {
	s, _ := ctx.Value(ctxKey{}).(Site)
	return s
}

func (l *listener) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	secure := r.TLS != nil
	for _, s := range *l.sites.Load() {
		if s.TLS.On() == secure && middleware.UnderBase(r.URL.Path, s.BasePath) {
			s.ServeHTTP(w, r)
			return
		}
	}
	if !secure && (r.Method == http.MethodGet || r.Method == http.MethodHead) {
		for _, s := range *l.sites.Load() {
			if s.TLS.On() && middleware.UnderBase(r.URL.Path, s.BasePath) {
				toHTTPS(w, r)
				return
			}
		}
	}
	middleware.NotFound(w, r)
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

// ServeHTTP serves a request that came through s.
func (s site) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.TLS != nil && s.TLS.Mode == settings.TLSDomain {
		w.Header().Set("Strict-Transport-Security", hsts)
	}
	s.h.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), ctxKey{}, s.Site)))
}

// share hands the shared addresses to the bridge. Callers hold mu.
func (s *Server) share() {
	if s.Share == nil {
		return
	}
	var out []Shared
	for _, st := range []*site{&s.current, s.pendingSite()} {
		if st != nil && st.TLS.Shared {
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
	l := &listener{certs: s.Certificate}
	l.ln = newConnListener(ln, l)
	l.tlsConfig = &tls.Config{MinVersion: tls.VersionTLS12, NextProtos: []string{"http/1.1"}, GetCertificate: l.certificate}
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
	case p.TLS.Shared:
	case p.Port == s.current.Port && !s.current.TLS.Shared:
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
