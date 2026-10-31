// Package paneltls gives the web panel its certificate for each handshake
// (ADR 0010): the domain enabled for Karvan gets its own certificate for
// its name, and every other handshake, the server's IP included, gets
// Karvan's self-signed one. It also says who the panel opens for when only
// the domain is to open it (access.go).
package paneltls

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/PdYrust/karvan/internal/audit"
	"github.com/PdYrust/karvan/internal/secret"
	"github.com/PdYrust/karvan/internal/settings"
)

// RenewEvery is how often Run checks the self-signed certificate.
const RenewEvery = time.Hour

// Manager picks the panel's certificate. Set the exported fields, then use
// it from any goroutine.
type Manager struct {
	Settings *settings.Store
	Key      *secret.Key
	// Answers says whether registered domain id answers host (lower case,
	// no port) and whether it has a certificate; an error when it is gone.
	Answers func(ctx context.Context, id int64, host string) (answers, hasCert bool, err error)
	// Domain is the certificate of a registered domain, read on each
	// handshake through the certificate store's cache.
	Domain func(ctx context.Context, id int64) (*tls.Certificate, error)
	// Audit records when the IP opens again for a broken domain
	// certificate; nil records nothing.
	Audit *audit.Log
	Log   *slog.Logger
	// Now and IPs override the clock and this server's addresses, for tests.
	Now func() time.Time
	IPs func() []net.IP

	changeMu sync.Mutex // one Change at a time

	mu      sync.Mutex
	panel   settings.TLS // the domain enabled for Karvan and the plain HTTP choice
	self    *tls.Certificate
	retry   time.Time                  // no new self-signed certificate is tried before this
	good    map[int64]*tls.Certificate // the last certificate each domain loaded
	failing map[int64]bool             // domains whose last load failed, logged once
	access  accessState                // the last answer of Access
	gen     uint64                     // counts Set, so an answer from before one is not kept
}

// Set is the domain enabled for Karvan and the plain HTTP choice from now
// on, for the next handshake and request.
func (m *Manager) Set(t settings.TLS) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.panel = t
	m.access.until = time.Time{}
	m.gen++
}

// Panel is the domain enabled for Karvan and the plain HTTP choice.
func (m *Manager) Panel() settings.TLS {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.panel
}

// Change makes fn's answer current unless it fails. fn gets the stored
// value (the one in memory when none can be read) and stores the new one;
// changes run one at a time, so two of them never undo each other, nor a
// change made from the command line. Handshakes go on meanwhile.
func (m *Manager) Change(ctx context.Context, fn func(cur settings.TLS) (settings.TLS, error)) error {
	m.changeMu.Lock()
	defer m.changeMu.Unlock()
	cur := m.Panel()
	if m.Settings != nil {
		if h, err := m.Settings.HTTP(ctx); err == nil {
			cur = h.TLS
		}
	}
	next, err := fn(cur)
	if err != nil {
		return err
	}
	if next != m.Panel() {
		m.Set(next)
	}
	return nil
}

// Certificate is the certificate for a handshake that asks for name: the
// enabled domain's for a name it answers, while it has a certificate;
// Karvan's self-signed one for every other name and for none (a browser
// sends no name for an IP address).
func (m *Manager) Certificate(ctx context.Context, name string) (*tls.Certificate, error) {
	id := m.Panel().DomainID
	if answers, has := m.lookup(ctx, id, name); answers && has {
		return m.domain(ctx, id)
	}
	return m.Self(ctx)
}

// Plain reports whether a plain HTTP request for host is answered as is:
// host is the enabled domain's name, the domain has no certificate, and
// plain HTTP is chosen for it. Any other one is sent on to HTTPS.
func (m *Manager) Plain(ctx context.Context, host string) bool {
	p := m.Panel()
	if !p.Plain {
		return false
	}
	answers, has := m.lookup(ctx, p.DomainID, host)
	return answers && !has
}

// lookup says whether domain id answers name and whether it has a
// certificate; a domain that cannot be read answers nothing.
func (m *Manager) lookup(ctx context.Context, id int64, name string) (answers, hasCert bool) {
	host := hostOf(name)
	if id < 1 || m.Answers == nil || host == "" {
		return false, false
	}
	answers, hasCert, err := m.Answers(ctx, id, host)
	if err != nil {
		return false, false
	}
	return answers, hasCert
}

// hostOf is name in lower case without a trailing dot.
func hostOf(name string) string { return strings.ToLower(strings.TrimSuffix(name, ".")) }

// domain is the domain's certificate; when it cannot be loaded, the last
// one it loaded, else the self-signed one. A failure is logged once,
// not on every handshake.
func (m *Manager) domain(ctx context.Context, id int64) (*tls.Certificate, error) {
	var c *tls.Certificate
	err := errors.New("no certificate source")
	if m.Domain != nil {
		c, err = m.Domain(ctx, id)
	}
	m.mu.Lock()
	if m.good == nil {
		m.good, m.failing = map[int64]*tls.Certificate{}, map[int64]bool{}
	}
	if err == nil {
		if m.failing[id] {
			m.log().Info("the panel certificate loads again", "domain", id)
		}
		m.good[id], m.failing[id] = c, false
		m.mu.Unlock()
		return c, nil
	}
	last, logged := m.good[id], m.failing[id]
	m.failing[id] = true
	m.mu.Unlock()
	if !logged {
		m.log().Error("the panel certificate cannot be loaded", "domain", id, "err", err)
	}
	if last != nil {
		return last, nil
	}
	if !logged {
		m.log().Warn("the panel is served with its self-signed certificate until the domain's loads", "domain", id)
	}
	return m.Self(ctx)
}

// Self is the self-signed certificate. It is made on first use and again
// 30 days before its end; the key is stored sealed. When a new one
// cannot be made or stored, the old one stays in use while it is valid
// and the next try comes RenewEvery later.
func (m *Manager) Self(ctx context.Context) (*tls.Certificate, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	now := m.now()
	if m.self == nil {
		c, err := m.stored(ctx)
		if err != nil {
			m.log().Error("read the self-signed certificate", "err", err)
		}
		m.self = c
	}
	if m.self != nil && (!due(m.self.Leaf, now) || now.Before(m.retry)) {
		return m.self, nil
	}
	c, err := m.make(ctx, now)
	if err == nil {
		m.self = c
		return c, nil
	}
	m.retry = now.Add(RenewEvery)
	m.log().Error("make the self-signed certificate", "err", err)
	if m.self == nil || !now.Before(m.self.Leaf.NotAfter) {
		if c == nil {
			return nil, err
		}
		m.self = c // made but not stored: served until a restart makes another
	}
	return m.self, nil
}

// Fingerprint is the SHA-256 fingerprint of the self-signed certificate.
func (m *Manager) Fingerprint(ctx context.Context) (string, error) {
	c, err := m.Self(ctx)
	if err != nil {
		return "", err
	}
	return Fingerprint(c.Leaf.Raw), nil
}

// Run checks the self-signed certificate every RenewEvery, and the stored
// settings and the domain's certificate every few seconds, until ctx
// ends.
func (m *Manager) Run(ctx context.Context) {
	t := time.NewTicker(RenewEvery)
	defer t.Stop()
	check := time.NewTicker(accessFor)
	defer check.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			if _, err := m.Self(ctx); err != nil && ctx.Err() == nil {
				m.log().Error("check the self-signed certificate", "err", err)
			}
		case <-check.C:
			if err := m.Reload(ctx); err != nil && ctx.Err() == nil {
				m.log().Warn("read the panel settings", "err", err)
			}
			m.Access(ctx)
		}
	}
}

// stored reads the stored self-signed certificate; nil when none is.
func (m *Manager) stored(ctx context.Context) (*tls.Certificate, error) {
	sc, ok, err := m.Settings.SelfCert(ctx)
	if err != nil || !ok {
		return nil, err
	}
	sealed, err := base64.StdEncoding.DecodeString(sc.SealedKey)
	if err != nil {
		return nil, fmt.Errorf("decode the self-signed key: %w", err)
	}
	key, err := m.Key.Open(sealed, settings.SelfCertAAD)
	if err != nil {
		return nil, fmt.Errorf("open the self-signed key: %w", err)
	}
	c, err := tls.X509KeyPair([]byte(sc.CertPEM), key)
	if err != nil {
		return nil, fmt.Errorf("load the self-signed certificate: %w", err)
	}
	return &c, nil
}

// make creates a self-signed certificate and stores it. It returns the
// certificate also when only the store failed.
func (m *Manager) make(ctx context.Context, now time.Time) (*tls.Certificate, error) {
	ips := m.IPs
	if ips == nil {
		ips = serverIPs
	}
	certPEM, keyPEM, err := newSelf(ips(), now)
	if err != nil {
		return nil, err
	}
	c, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		return nil, fmt.Errorf("load the new self-signed certificate: %w", err)
	}
	sealed, err := m.Key.Seal(keyPEM, settings.SelfCertAAD)
	if err != nil {
		return &c, fmt.Errorf("seal the self-signed key: %w", err)
	}
	err = m.Settings.SetSelfCert(ctx, settings.SelfCert{CertPEM: string(certPEM), SealedKey: base64.StdEncoding.EncodeToString(sealed)})
	if err != nil {
		return &c, fmt.Errorf("store the self-signed certificate: %w", err)
	}
	m.log().Info("made a self-signed certificate for the panel", "until", c.Leaf.NotAfter.UTC().Format(time.DateOnly))
	return &c, nil
}

func (m *Manager) now() time.Time {
	if m.Now != nil {
		return m.Now()
	}
	return time.Now()
}

func (m *Manager) log() *slog.Logger {
	if m.Log == nil {
		return slog.Default()
	}
	return m.Log
}
