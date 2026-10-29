package bridge

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"regexp"
	"strings"
	"syscall"
)

// Why the challenge port could not be opened (D67).
var (
	ErrPortDenied = errors.New("karvan may not listen on the challenge port")
	ErrPortBusy   = errors.New("another program listens on the challenge port")
	ErrPortTLS    = errors.New("an HTTPS rule uses the challenge port")
)

// ReasonReserved means a redirect's fromPath is under the HTTP-01
// challenge path, which Let's Encrypt validation needs (D67).
const ReasonReserved = "reserved"

// challengePrefix is where HTTP-01 tokens are asked for.
const challengePrefix = "/.well-known/acme-challenge/"

var challengeTokenRe = regexp.MustCompile(`^[A-Za-z0-9_-]{1,128}$`)

// challenge is the answer to one live HTTP-01 token.
type challenge struct{ host, keyAuth string }

// challenges are the holders of the challenge port and the live tokens,
// in memory only. holders change only under the store's save lock.
type challenges struct {
	holders map[string]bool
	tokens  map[string]challenge
}

// WantChallengePort opens (on) or lets go of (off) the challenge port for
// holder: an order's key, or "keep" for acme.keepPort80. The port is
// served as plain HTTP while any holder wants it. It takes the save lock,
// so no rule save can put an HTTPS rule on the port meanwhile (D63).
func (b *Bridge) WantChallengePort(ctx context.Context, holder string, on bool) error {
	b.st.saveMu.Lock()
	defer b.st.saveMu.Unlock()
	port := b.ChallengePort
	if !on {
		b.mu.Lock()
		_, had := b.chal.holders[holder]
		delete(b.chal.holders, holder)
		b.mu.Unlock()
		if !had {
			return nil
		}
		return b.Reload(ctx)
	}
	rules, err := b.st.rules(ctx)
	if err != nil {
		return err
	}
	for _, r := range rules {
		if r.port == port && r.tls {
			return ErrPortTLS
		}
	}
	b.mu.Lock()
	b.chal.holders[holder] = true
	b.mu.Unlock()
	if err := b.Reload(ctx); err != nil {
		return err
	}
	b.mu.Lock()
	_, up := b.servers[port]
	lerr := b.errs[port]
	if !up {
		delete(b.chal.holders, holder)
	}
	b.mu.Unlock()
	if up {
		return nil
	}
	if err := b.Reload(ctx); err != nil {
		return err
	}
	return portError(port, lerr)
}

// portError is the D67 reason a listen on the challenge port failed.
func portError(port int, err error) error {
	switch {
	case errors.Is(err, syscall.EACCES):
		return fmt.Errorf("%w: port %d: %w", ErrPortDenied, port, err)
	case errors.Is(err, syscall.EADDRINUSE):
		return fmt.Errorf("%w: port %d: %w", ErrPortBusy, port, err)
	}
	return fmt.Errorf("listen on challenge port %d: %w", port, err)
}

// challengeHeld reports whether any holder keeps the challenge port open.
func (b *Bridge) challengeHeld() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	return len(b.chal.holders) > 0
}

// ChallengeHeld reports whether Karvan keeps the challenge port open now.
func (b *Bridge) ChallengeHeld() bool { return b.challengeHeld() }

// challengeOpen reports whether the challenge port is port and held, for
// the save checks, which run under the save lock.
func (b *Bridge) challengeOpen(port int) bool { return port == b.ChallengePort && b.challengeHeld() }

// AddChallenge makes token answer keyAuth for host on the challenge port
// while its order validates.
func (b *Bridge) AddChallenge(token, host, keyAuth string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.chal.tokens[token] = challenge{host: strings.ToLower(host), keyAuth: keyAuth}
}

// RemoveChallenge stops answering token.
func (b *Bridge) RemoveChallenge(token string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	delete(b.chal.tokens, token)
}

// serveChallenge runs before the router on the challenge port. It answers
// a GET or HEAD of the exact path of a live token on that token's host,
// and on a port no rule uses it answers everything else with the plain
// 404. It reports whether it answered; it never redirects.
func (b *Bridge) serveChallenge(w http.ResponseWriter, r *http.Request) bool {
	b.mu.Lock()
	c, live := b.liveToken(r)
	ruled := len(b.routes[b.ChallengePort]) > 0
	b.mu.Unlock()
	switch {
	case live:
		w.Header().Set("Content-Type", "text/plain")
		w.Header().Set("Cache-Control", "no-store")
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, c.keyAuth)
		return true
	case !ruled:
		answer(w, http.StatusNotFound)
		return true
	}
	return false
}

// liveToken is the challenge r asks for, if it is live on r's host. The
// caller holds mu.
func (b *Bridge) liveToken(r *http.Request) (challenge, bool) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		return challenge{}, false
	}
	token, ok := strings.CutPrefix(r.URL.EscapedPath(), challengePrefix)
	if !ok || !challengeTokenRe.MatchString(token) {
		return challenge{}, false
	}
	c, ok := b.chal.tokens[token]
	return c, ok && c.host == requestHost(r)
}

// requestHost is r's Host in lower case without its port.
func requestHost(r *http.Request) string {
	host := strings.ToLower(r.Host)
	if h, _, err := net.SplitHostPort(host); err == nil {
		host = h
	}
	return host
}

// reservedPath reports whether a redirect's stored fromPath is under the
// challenge path.
func reservedPath(from string) bool {
	return from+"/" == challengePrefix || strings.HasPrefix(from, challengePrefix)
}
