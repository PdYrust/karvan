// Package address serves the web panel on its address (a port and a hidden
// path) and, while a change of address waits for its confirmation, on the
// new one next to it. Nothing is saved until the operator confirms from
// the new address; otherwise the new one goes away (ADR 0009).
package address

import (
	"context"
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
}

// Change says what Start changes: a new random hidden path, and the port
// (0 keeps it, -1 picks a random one).
type Change struct {
	NewPath bool
	Port    int
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
	ln    net.Listener
	srv   *http.Server
	sites atomic.Pointer[[]site]
}

type ctxKey struct{}

// From returns the address a request came through.
func From(ctx context.Context) Site {
	s, _ := ctx.Value(ctxKey{}).(Site)
	return s
}

func (l *listener) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	for _, s := range *l.sites.Load() {
		if middleware.UnderBase(r.URL.Path, s.BasePath) {
			s.h.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), ctxKey{}, s.Site)))
			return
		}
	}
	middleware.NotFound(w, r)
}

func (l *listener) serve(sites ...site) { l.sites.Store(&sites) }

// Serve serves current on ln until ctx ends or that listener fails, then
// shuts every listener down.
func (s *Server) Serve(ctx context.Context, current Site, ln net.Listener) error {
	h, err := s.Handler(current.BasePath)
	if err != nil {
		_ = ln.Close()
		return err
	}
	s.mu.Lock()
	s.current, s.ports, s.fatal = site{Site: current, h: h}, map[int]*listener{}, make(chan error, 1)
	s.start(current.Port, ln, s.current)
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
	l := &listener{ln: ln}
	l.serve(sites...)
	l.srv = s.HTTP(l)
	s.ports[port] = l
	s.wg.Go(func() {
		if err := l.srv.Serve(ln); !errors.Is(err, http.ErrServerClosed) {
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
	if p.Port != s.current.Port {
		s.drop(p.Port)
	} else if l := s.ports[s.current.Port]; l != nil {
		l.serve(s.current)
	}
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
