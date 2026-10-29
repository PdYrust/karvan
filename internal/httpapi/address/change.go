package address

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/PdYrust/karvan/internal/settings"
)

// Status returns the current address and the one waiting, if any.
func (s *Server) Status() Status {
	s.mu.Lock()
	defer s.mu.Unlock()
	st := Status{Current: s.current.Site}
	if p := s.pending; p != nil {
		st.Pending = &Pending{Site: p.Site, Expires: p.expires}
	}
	return st
}

// Start serves the new address next to the current one for the TTL. A
// change already waiting is dropped first: the newest one wins. If the
// port cannot be opened, nothing new is served.
func (s *Server) Start(c Change) (Pending, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closing || s.ports == nil {
		return Pending{}, errClosed
	}
	next := s.current.next(c)
	if !c.NewPath && c.Port >= 0 && next == s.current.Site {
		return Pending{}, ErrNoChange
	}
	if c.NewPath {
		p, err := settings.NewBasePath()
		if err != nil {
			return Pending{}, err
		}
		next.BasePath = p
	}
	s.dropPending()

	var ln net.Listener
	var err error
	switch {
	case c.Port < 0:
		next.Port, ln, err = s.randomPort()
	case !next.TLS.Shared && s.ports[next.Port] == nil:
		if ln, err = s.listen(next.Port); err != nil {
			err = fmt.Errorf("%w: %w", ErrPortUnavailable, err)
		}
	}
	if err != nil {
		return Pending{}, err
	}
	return s.serveNew(next, ln)
}

// next is s with the port and the mode of c; the path is changed later.
func (s Site) next(c Change) Site {
	if c.Port > 0 {
		s.Port = c.Port
	}
	switch {
	case c.TLS != nil:
		s.TLS = c.TLS.Normal()
	case c.Port != 0:
		s.TLS.Shared = false // a new port is the panel's own
	}
	return s
}

// serveNew serves next: through the bridge when it is shared, on ln when
// the port changes, else next to the current path on the current port.
// Callers hold mu.
func (s *Server) serveNew(next Site, ln net.Listener) (Pending, error) {
	h := s.current.h
	if next.BasePath != s.current.BasePath {
		var err error
		if h, err = s.Handler(next.BasePath); err != nil {
			if ln != nil {
				_ = ln.Close()
			}
			return Pending{}, err
		}
	}
	p := &pending{site: site{Site: next, h: h}, expires: time.Now().Add(s.ttl())}
	switch {
	case next.TLS.Shared:
	case ln != nil:
		s.start(next.Port, ln, p.site)
	default:
		s.ports[next.Port].serve(s.current, p.site)
	}
	p.timer = time.AfterFunc(s.ttl(), func() { s.expire(p) })
	s.pending = p
	s.share()
	s.log().Info("serving a new address until it is confirmed", "port", next.Port, "tls", next.TLS.On(), "minutes", int(s.ttl().Minutes()))
	return Pending{Site: next, Expires: p.expires}, nil
}

func (s *Server) expire(p *pending) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.pending != p {
		return
	}
	s.dropPending()
	s.log().Info("the new address was not confirmed in time; the address stays", "port", s.current.Port)
}

// Confirm saves the waiting address and stops serving the old one. It must
// come through the new address (via), which proves the operator reached
// it. It returns the address before the change.
func (s *Server) Confirm(ctx context.Context, via Site) (Site, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	p := s.pending
	if p == nil {
		return Site{}, ErrNoPending
	}
	if via != p.Site {
		return Site{}, ErrNotNew
	}
	if err := s.Save(ctx, p.Site); err != nil {
		return Site{}, fmt.Errorf("save the new address: %w", err)
	}
	p.timer.Stop()
	old := s.current
	s.current, s.pending = p.site, nil
	switch {
	case old.TLS.Shared:
	case p.TLS.Shared || old.Port != p.Port:
		s.drop(old.Port)
	default:
		s.ports[p.Port].serve(p.site)
	}
	s.share()
	s.log().Info("the new address is confirmed", "port", p.Port, "tls", p.TLS.On())
	return old.Site, nil
}
