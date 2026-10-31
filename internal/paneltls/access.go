package paneltls

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"strconv"
	"time"

	"github.com/PdYrust/karvan/internal/audit"
	"github.com/PdYrust/karvan/internal/settings"
)

// Access is who the panel opens for, by "Only open Karvan on its domain"
// (ADR 0010).
type Access string

const (
	// AccessAll is no domain enabled, or the switch off: every name and
	// the server's IP open the panel.
	AccessAll Access = "all"
	// AccessWaiting is the switch on, with the IP open until the operator
	// first signs in through the domain.
	AccessWaiting Access = "waiting"
	// AccessDomain is only the domain's names opening the panel.
	AccessDomain Access = "domain"
	// AccessFallback is the switch on and armed while the domain's
	// certificate has ended or cannot be loaded, so the IP opens again
	// with Karvan's own certificate.
	AccessFallback Access = "fallback"
)

// Why the IP opened again.
const (
	FallbackExpired = "expired" // the domain's certificate has ended
	FallbackMissing = "missing" // its files are gone or cannot be read
)

// accessFor is how long an answer of Access is used again, so requests
// and handshakes do not each read the domain's certificate.
const accessFor = 5 * time.Second

// accessState is the last answer of Access.
type accessState struct {
	access Access
	reason string
	until  time.Time
}

// Access is who the panel opens for now, and for AccessFallback why. The
// first fallback after the domain worked is logged and audited, and so is
// the return to the domain alone.
func (m *Manager) Access(ctx context.Context) (Access, string) {
	now := m.now()
	m.mu.Lock()
	if now.Before(m.access.until) {
		a := m.access
		m.mu.Unlock()
		return a.access, a.reason
	}
	p, gen := m.panel, m.gen
	m.mu.Unlock()
	a := accessState{access: AccessAll, until: now.Add(accessFor)}
	switch {
	case p.DomainID < 1 || p.OpenIP:
	case !p.Armed:
		a.access = AccessWaiting
	default:
		a.access = AccessDomain
		if a.reason = m.broken(ctx, p.DomainID, now); a.reason != "" {
			a.access = AccessFallback
		}
	}
	m.mu.Lock()
	was := m.access.access
	if gen == m.gen {
		m.access = a
	}
	m.mu.Unlock()
	m.note(ctx, p.DomainID, was, a)
	return a.access, a.reason
}

// Allows reports whether the panel opens for name, a TLS server name or a
// request's host ("" for none, as for an IP address).
func (m *Manager) Allows(ctx context.Context, name string) bool {
	if a, _ := m.Access(ctx); a != AccessDomain {
		return true
	}
	answers, _ := m.lookup(ctx, m.Panel().DomainID, name)
	return answers
}

// Arm makes the domain the only way in once the operator loads the panel
// through it: every one of names (the request's host, and its TLS server
// name) must be the domain's. It reports whether this request armed it.
func (m *Manager) Arm(ctx context.Context, names ...string) (bool, error) {
	if p := m.Panel(); p.DomainID < 1 || p.OpenIP || p.Armed || m.Settings == nil {
		return false, nil
	}
	armed := false
	err := m.Change(ctx, func(cur settings.TLS) (settings.TLS, error) {
		if cur.DomainID < 1 || cur.OpenIP || cur.Armed || len(names) == 0 {
			return cur, nil
		}
		for _, n := range names {
			if answers, _ := m.lookup(ctx, cur.DomainID, n); !answers {
				return cur, nil
			}
		}
		next := cur
		next.Armed = true
		if err := m.Settings.SetKarvan(ctx, next); err != nil {
			return cur, err
		}
		armed = true
		m.log().Info("only the Karvan domain opens the panel now", "domain", cur.DomainID)
		return next, nil
	})
	return armed, err
}

// broken is why the domain's certificate cannot serve at now: it has
// ended, or it cannot be loaded; "" while it serves or the domain has no
// certificate (its name then gets Karvan's own).
func (m *Manager) broken(ctx context.Context, id int64, now time.Time) string {
	if m.Answers == nil {
		return FallbackMissing
	}
	// The host is left out: only whether the domain has a certificate counts here.
	_, has, err := m.Answers(ctx, id, "")
	switch {
	case err != nil:
		return FallbackMissing
	case !has:
		return ""
	}
	c := (*tls.Certificate)(nil)
	err = errors.New("no certificate source")
	if m.Domain != nil {
		c, err = m.Domain(ctx, id)
	}
	if err != nil {
		return FallbackMissing
	}
	leaf, err := leafOf(c)
	if err != nil {
		return FallbackMissing
	}
	if !now.Before(leaf.NotAfter) {
		return FallbackExpired
	}
	return ""
}

func leafOf(c *tls.Certificate) (*x509.Certificate, error) {
	if c.Leaf != nil {
		return c.Leaf, nil
	}
	if len(c.Certificate) == 0 {
		return nil, errors.New("an empty certificate")
	}
	return x509.ParseCertificate(c.Certificate[0])
}

// note logs and audits a change between the domain alone and the
// fallback: the fallback as a failure, the return as the domain alone.
func (m *Manager) note(ctx context.Context, id int64, was Access, now accessState) {
	domain := strconv.FormatInt(id, 10)
	switch {
	case now.access == AccessFallback && was != AccessFallback:
		m.log().Warn("the Karvan domain's certificate cannot serve, so the server's IP opens the panel again",
			"domain", id, "reason", now.reason)
		m.record(ctx, audit.ActionKarvanFallback, audit.Failure, map[string]string{"domain": domain, "reason": now.reason})
	case now.access == AccessDomain && was == AccessFallback:
		m.log().Info("the Karvan domain's certificate serves again, so only the domain opens the panel", "domain", id)
		m.record(ctx, audit.ActionKarvanOnly, audit.Success, map[string]string{"domain": domain})
	}
}

func (m *Manager) record(ctx context.Context, action string, outcome audit.Outcome, detail map[string]string) {
	if m.Audit == nil {
		return
	}
	err := m.Audit.Record(ctx, audit.Event{Actor: audit.ActorSystem, Action: action, Outcome: outcome, Detail: detail})
	if err != nil {
		m.log().Warn("audit", "action", action, "err", err)
	}
}

// Reload takes the stored settings, so a change from the command line
// (karvan access open-ip) reaches the running server.
func (m *Manager) Reload(ctx context.Context) error {
	if m.Settings == nil {
		return nil
	}
	h, err := m.Settings.HTTP(ctx)
	if err != nil {
		return err
	}
	return m.Change(ctx, func(settings.TLS) (settings.TLS, error) { return h.TLS, nil })
}
