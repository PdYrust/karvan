package bridge

import (
	"net/netip"
	"sync"
	"time"
)

// Lazy check limits (D61).
const (
	siteRate     = 10 // checks per second per site
	siteInFlight = 4
	ipChecks     = 3 // per client address (IPv6: /64) per ipWindow
	ipWindow     = 10 * time.Second
	negativeTTL  = time.Hour
	negativeMax  = 100_000
	retryAfter   = "10"
)

// limits gate lazy checks so a flood can delay first-time checks but
// never drain the old panel or this server.
type limits struct {
	mu       sync.Mutex
	now      func() time.Time
	sites    map[int64]*bucket
	inFlight map[int64]int
	ips      map[string]*window
	swept    time.Time
}

type bucket struct {
	tokens float64
	at     time.Time
}

type window struct {
	start time.Time
	n     int
}

func newLimits(now func() time.Time) *limits {
	return &limits{now: now, sites: map[int64]*bucket{}, inFlight: map[int64]int{}, ips: map[string]*window{}}
}

// acquire takes one check for site from ip. done must be called when the
// check ends. ok is false when the request is over a limit.
func (l *limits) acquire(site int64, ip string) (done func(), ok bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now()
	l.sweep(now)
	key := clientKey(ip)
	w := l.ips[key]
	if w == nil || now.Sub(w.start) >= ipWindow {
		w = &window{start: now}
		l.ips[key] = w
	}
	b := l.sites[site]
	if b == nil {
		b = &bucket{tokens: siteRate, at: now}
		l.sites[site] = b
	}
	b.tokens = min(siteRate, b.tokens+now.Sub(b.at).Seconds()*siteRate)
	b.at = now
	if w.n >= ipChecks || b.tokens < 1 || l.inFlight[site] >= siteInFlight {
		return nil, false
	}
	w.n++
	b.tokens--
	l.inFlight[site]++
	return func() {
		l.mu.Lock()
		l.inFlight[site]--
		l.mu.Unlock()
	}, true
}

// sweep forgets client windows that ended, once a minute.
func (l *limits) sweep(now time.Time) {
	if now.Sub(l.swept) < time.Minute {
		return
	}
	l.swept = now
	for k, w := range l.ips {
		if now.Sub(w.start) >= ipWindow {
			delete(l.ips, k)
		}
	}
}

// clientKey groups IPv6 clients by /64, the block one subscriber usually
// gets, so rotating addresses inside it does not reset the count.
func clientKey(ip string) string {
	a, err := netip.ParseAddr(ip)
	if err != nil || !a.Unmap().Is6() {
		return ip
	}
	p, _ := a.Prefix(64)
	return p.String()
}

// negative remembers tokens the old panel said it does not know (its
// real 404), so repeating one does not reach the old panel again.
type negative struct {
	mu  sync.Mutex
	now func() time.Time
	m   map[string]time.Time
}

func newNegative(now func() time.Time) *negative {
	return &negative{now: now, m: map[string]time.Time{}}
}

func (n *negative) has(mac []byte) bool {
	n.mu.Lock()
	defer n.mu.Unlock()
	until, ok := n.m[string(mac)]
	if ok && n.now().After(until) {
		delete(n.m, string(mac))
		return false
	}
	return ok
}

func (n *negative) add(mac []byte) {
	n.mu.Lock()
	defer n.mu.Unlock()
	now := n.now()
	if len(n.m) >= negativeMax {
		for k, until := range n.m {
			if now.After(until) {
				delete(n.m, k)
			}
		}
		for k := range n.m {
			if len(n.m) < negativeMax {
				break
			}
			delete(n.m, k)
		}
	}
	n.m[string(mac)] = now.Add(negativeTTL)
}
