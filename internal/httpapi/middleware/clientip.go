package middleware

import (
	"context"
	"net"
	"net/http"
	"net/netip"
	"strings"
)

type ctxKey int

const (
	ipKey ctxKey = iota
	httpsKey
	sessionKey
)

// ClientIP records the caller's address and whether the browser reached us
// over HTTPS. X-Forwarded-For and X-Forwarded-Proto are only believed when
// the direct peer is a trusted proxy. Of the forwarded addresses, the
// rightmost one that is not itself a trusted proxy wins, since entries left
// of it can be forged by the client.
func ClientIP(trusted []netip.Prefix, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := peerIP(r.RemoteAddr)
		https := r.TLS != nil
		if ip.IsValid() && isTrusted(ip, trusted) {
			if fwd, ok := forwardedIP(r.Header.Values("X-Forwarded-For"), trusted); ok {
				ip = fwd
			}
			https = https || strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https")
		}
		s := ""
		if ip.IsValid() {
			s = ip.Unmap().String()
		}
		ctx := context.WithValue(context.WithValue(r.Context(), ipKey, s), httpsKey, https)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// IsHTTPS reports whether the browser's connection is HTTPS, directly or
// through a trusted proxy.
func IsHTTPS(r *http.Request) bool {
	if v, ok := r.Context().Value(httpsKey).(bool); ok {
		return v
	}
	return r.TLS != nil
}

// IP returns the address recorded by ClientIP.
func IP(ctx context.Context) string {
	s, _ := ctx.Value(ipKey).(string)
	return s
}

func peerIP(remote string) netip.Addr {
	host, _, err := net.SplitHostPort(remote)
	if err != nil {
		host = remote
	}
	a, _ := netip.ParseAddr(host)
	return a
}

func isTrusted(a netip.Addr, trusted []netip.Prefix) bool {
	a = a.Unmap()
	for _, p := range trusted {
		if p.Contains(a) {
			return true
		}
	}
	return false
}

func forwardedIP(values []string, trusted []netip.Prefix) (netip.Addr, bool) {
	var hops []string
	for _, v := range values {
		hops = append(hops, strings.Split(v, ",")...)
	}
	for i := len(hops) - 1; i >= 0; i-- {
		a, err := netip.ParseAddr(strings.TrimSpace(hops[i]))
		if err != nil {
			return netip.Addr{}, false
		}
		if !isTrusted(a, trusted) {
			return a, true
		}
	}
	return netip.Addr{}, false
}
