// Package paneltls gives the web panel its certificate (ADR
// 0010): Karvan's own self-signed one, or a registered domain's from the
// certificate store, with a fallback so a moved file never locks the
// operator out.
package paneltls

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"sync"
	"time"

	"github.com/PdYrust/karvan/internal/secret"
	"github.com/PdYrust/karvan/internal/settings"
)

// RenewEvery is how often Run checks the self-signed certificate.
const RenewEvery = time.Hour

// ErrPlain means the panel is served over plain HTTP: no certificate.
var ErrPlain = errors.New("the panel is served over plain HTTP")

// Manager picks the panel's certificate. Set the exported fields, then use
// it from any goroutine.
type Manager struct {
	Settings *settings.Store
	Key      *secret.Key
	// Domain is the certificate of a registered domain, read on each
	// handshake through the certificate store's cache.
	Domain func(ctx context.Context, id int64) (*tls.Certificate, error)
	Log    *slog.Logger
	// Now and IPs override the clock and this server's addresses, for tests.
	Now func() time.Time
	IPs func() []net.IP

	mu      sync.Mutex
	self    *tls.Certificate
	retry   time.Time                  // no new self-signed certificate is tried before this
	good    map[int64]*tls.Certificate // the last certificate each domain loaded
	failing map[int64]bool             // domains whose last load failed, logged once
}

// Certificate is the certificate the panel serves in mode t.
func (m *Manager) Certificate(ctx context.Context, t settings.TLS) (*tls.Certificate, error) {
	switch t.Mode {
	case settings.TLSSelf:
		return m.Self(ctx)
	case settings.TLSDomain:
		return m.domain(ctx, t.DomainID)
	}
	return nil, ErrPlain
}

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

// Run checks the self-signed certificate every RenewEvery until ctx ends.
func (m *Manager) Run(ctx context.Context) {
	t := time.NewTicker(RenewEvery)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			if _, err := m.Self(ctx); err != nil && ctx.Err() == nil {
				m.log().Error("check the self-signed certificate", "err", err)
			}
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
