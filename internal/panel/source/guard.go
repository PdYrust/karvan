package source

import (
	"context"
	"fmt"
	"net"
	"net/netip"
	"time"
)

// Resolver looks up host addresses; net.DefaultResolver satisfies it.
type Resolver interface {
	LookupNetIP(ctx context.Context, network, host string) ([]netip.Addr, error)
}

// metadata are cloud metadata endpoints outside the link-local ranges.
var metadata = []netip.Addr{
	netip.MustParseAddr("fd00:ec2::254"),
}

// allowed reports whether Karvan may connect to a. Loopback and private
// ranges are allowed, because the old panel can sit on the same server or
// LAN; link-local (cloud metadata), multicast and unspecified are not.
func allowed(a netip.Addr) bool {
	a = a.Unmap()
	switch {
	case !a.IsValid(), a.IsUnspecified(), a.IsMulticast(), a.IsInterfaceLocalMulticast(),
		a.IsLinkLocalUnicast(), a.IsLinkLocalMulticast():
		return false
	}
	for _, m := range metadata {
		if a == m {
			return false
		}
	}
	return true
}

// guardedDialer resolves the host itself, refuses the whole host if any
// address is forbidden, and dials the checked address, so a DNS answer
// cannot change between the check and the connection.
type guardedDialer struct {
	resolver Resolver
	dialer   net.Dialer
}

func newGuardedDialer(r Resolver) *guardedDialer {
	if r == nil {
		r = net.DefaultResolver
	}
	return &guardedDialer{resolver: r, dialer: net.Dialer{Timeout: 10 * time.Second, KeepAlive: 30 * time.Second}}
}

func (g *guardedDialer) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return nil, err
	}
	addrs, err := g.resolve(ctx, host)
	if err != nil {
		return nil, &Error{Code: CodeUnreachable, Err: err}
	}
	for _, a := range addrs {
		if !allowed(a) {
			return nil, &Error{Code: CodeForbiddenAddress, Err: fmt.Errorf("%s resolves to %s", host, a)}
		}
	}
	var last error
	for _, a := range addrs {
		c, err := g.dialer.DialContext(ctx, network, net.JoinHostPort(a.String(), port))
		if err == nil {
			return c, nil
		}
		last = err
	}
	return nil, last
}

func (g *guardedDialer) resolve(ctx context.Context, host string) ([]netip.Addr, error) {
	if a, err := netip.ParseAddr(host); err == nil {
		return []netip.Addr{a}, nil
	}
	addrs, err := g.resolver.LookupNetIP(ctx, "ip", host)
	if err == nil && len(addrs) == 0 {
		err = fmt.Errorf("%s has no address", host)
	}
	return addrs, err
}
