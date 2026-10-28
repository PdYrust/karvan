package bridge

import (
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"strings"
)

// ErrCertConflict means rules on one port that one name can reach, or
// rules with no domains on one port, would use different certificates.
var ErrCertConflict = errors.New("rules for one name on one port use different certificates")

// ErrPortTaken means the port is Karvan's own web panel.
var ErrPortTaken = errors.New("the port is Karvan's web panel")

// How well a rule's domains fit a request's host (D52).
const (
	hostNone     = iota // the rule does not answer the host
	hostAny             // a rule with no domains answers every host on its port
	hostWildcard        // a "*." pattern covers the host
	hostExact           // the host is one of the rule's names
)

// pathExact ranks an exact path above a prefix of any length.
const pathExact = 1 << 10

// route is a rule in the routing table.
type route struct {
	rule
	panel *Site // the panel rule it serves; nil for a redirect rule
}

// table is the routing table: the routes of each port, panel rules in id
// order, then redirect rules in id order.
type table map[int][]route

// newTable builds the routing table. A route points into sites.
func newTable(sites []Site, redirects []rule) table {
	t := table{}
	for i := range sites {
		s := &sites[i]
		t[s.Port] = append(t[s.Port], route{rule: ruleOf(*s), panel: s})
	}
	for _, r := range redirects {
		t[r.port] = append(t[r.port], route{rule: r})
	}
	return t
}

// match picks the route for host (lower case, no port) and the escaped
// path: the most specific host, then the most specific path. tail is the
// path after the rule's own path. Only rows stored before D52 can tie;
// the first of them wins, as before.
func (t table) match(port int, host, path string) (route, string, bool) {
	var best route
	var tail string
	bh, bp := hostNone, 0
	for _, r := range t[port] {
		h, p, rest, ok := r.rank(host, path)
		if ok && (h > bh || h == bh && p > bp) {
			best, tail, bh, bp = r, rest, h, p
		}
	}
	return best, tail, bh != hostNone
}

// rank is how well r fits a request; ok is false when it does not.
func (r rule) rank(host, path string) (h, p int, tail string, ok bool) {
	if h = r.hostRank(host); h == hostNone {
		return h, 0, "", false
	}
	own := "/" + r.path
	switch {
	case r.exact:
		return h, pathExact, "", path == own
	case path == own && r.ref.Kind == RuleRedirect: // a redirect prefix matches its own path too
		return h, r.segments(), "", true
	}
	if !strings.HasPrefix(path, own+"/") {
		return h, 0, "", false
	}
	return h, r.segments(), path[len(own):], true
}

func (r rule) hostRank(host string) int {
	if len(r.domains) == 0 {
		return hostAny
	}
	rank := hostNone
	for _, d := range r.domains {
		switch {
		case d == host:
			return hostExact
		case strings.HasPrefix(d, "*.") && matches(d, host):
			rank = hostWildcard
		}
	}
	return rank
}

func (r rule) segments() int { return strings.Count(r.path, "/") + 1 }

// sameRequests reports whether r and o share a host key (an identical
// name or wildcard, or both no domains) and a path key on one port. Any
// other pair of rules has a winner for every request (D52).
func (r rule) sameRequests(o rule) bool {
	if r.port != o.port || r.exact != o.exact || r.path != o.path {
		return false
	}
	if len(r.domains) == 0 || len(o.domains) == 0 {
		return len(r.domains) == len(o.domains)
	}
	return slices.ContainsFunc(r.domains, func(d string) bool { return slices.Contains(o.domains, d) })
}

// sameNames reports whether one TLS name on the port can pick r or o for
// its handshake: patterns that can meet, or both no domains (D53).
func (r rule) sameNames(o rule) bool {
	if r.port != o.port {
		return false
	}
	if len(r.domains) == 0 || len(o.domains) == 0 {
		return len(r.domains) == len(o.domains)
	}
	return slices.ContainsFunc(r.domains, func(a string) bool {
		return slices.ContainsFunc(o.domains, func(b string) bool { return canMeet(a, b) })
	})
}

// certClash reports whether r and o would serve one name with different
// certificates.
func (r rule) certClash(o rule) bool { return r.sameNames(o) && r.certKey() != o.certKey() }

// conflict is why r cannot be saved next to the stored rule o, or nil.
func (r rule) conflict(o rule) error {
	switch {
	case r.ref.Kind == o.ref.Kind && r.ref.Name == o.ref.Name:
		return ErrNameTaken
	case r.port != o.port:
		return nil
	case r.tls != o.tls:
		return fmt.Errorf("%w: %s", ErrPortMixed, o.ref.Name)
	case r.sameRequests(o):
		return fmt.Errorf("%w: %s", ErrOverlap, o.ref.Name)
	case r.certClash(o):
		return fmt.Errorf("%w: %s", ErrCertConflict, o.ref.Name)
	}
	return nil
}

// logConflicts warns about stored rules that break D52 or D53. They are
// served in today's order: the first rule wins a tie and a handshake.
func (t table) logConflicts(log *slog.Logger) {
	for port, routes := range t {
		for i, r := range routes {
			for _, o := range routes[i+1:] {
				switch {
				case r.sameRequests(o.rule):
					log.Warn("bridge: rules answer the same requests", "port", port, "rule", r.ref.Name, "other", o.ref.Name)
				case r.tls && o.tls && r.certClash(o.rule):
					log.Warn("bridge: rules on one port use different certificates", "port", port, "rule", r.ref.Name,
						"other", o.ref.Name)
				}
			}
		}
	}
}
