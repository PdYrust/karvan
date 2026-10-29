package settings

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
)

const (
	keyTLSMode   = "http.tls"
	keyTLSDomain = "http.tls_domain"
	keyTLSShared = "http.tls_shared"
	keySelfCert  = "http.tls_self_cert"
	keySelfKey   = "http.tls_self_key"
)

// How the web panel is served (D4).
const (
	TLSOff    = "off"    // plain HTTP
	TLSSelf   = "self"   // HTTPS with Karvan's own self-signed certificate
	TLSDomain = "domain" // HTTPS with a registered domain's certificate
)

// TLS is how the web panel is served. The zero value is plain HTTP, which
// is what an installation made before HTTPS existed reads back.
type TLS struct {
	Mode     string // TLSOff, TLSSelf or TLSDomain; "" means off
	DomainID int64  // the domain whose certificate is served, in domain mode
	Shared   bool   // served on a bridge HTTPS port of that domain (D12)
}

// On reports whether the panel is served over HTTPS.
func (t TLS) On() bool { return t.Mode == TLSSelf || t.Mode == TLSDomain }

// Normal is t with the fields its mode does not use cleared, and plain
// HTTP as the zero value, so equal settings compare equal.
func (t TLS) Normal() TLS {
	switch t.Mode {
	case TLSDomain:
		return t
	case TLSSelf:
		return TLS{Mode: TLSSelf}
	}
	return TLS{}
}

// Validate checks the mode and that a domain goes with domain mode.
func (t TLS) Validate() error {
	switch t.Mode {
	case "", TLSOff, TLSSelf:
		if t.Shared {
			return errors.New("a shared port needs domain mode")
		}
	case TLSDomain:
		if t.DomainID < 1 {
			return errors.New("domain mode needs a domain")
		}
	default:
		return fmt.Errorf("tls mode %q must be off, self or domain", t.Mode)
	}
	return nil
}

// tlsRows are the stored values of t.
func (t TLS) rows() map[string]string {
	t = t.Normal()
	if t.Mode == "" {
		t.Mode = TLSOff
	}
	return map[string]string{keyTLSMode: t.Mode, keyTLSDomain: strconv.FormatInt(t.DomainID, 10),
		keyTLSShared: strconv.FormatBool(t.Shared)}
}

// tlsOf reads TLS from the stored values; missing ones mean plain HTTP.
func tlsOf(vals map[string]string) (TLS, error) {
	t := TLS{Mode: vals[keyTLSMode]}
	if v := vals[keyTLSDomain]; v != "" {
		id, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return TLS{}, fmt.Errorf("stored tls domain %q is not a number", v)
		}
		t.DomainID = id
	}
	t.Shared = vals[keyTLSShared] == "true"
	if err := t.Validate(); err != nil {
		return TLS{}, err
	}
	return t.Normal(), nil
}

// SelfCert is the panel's self-signed certificate (PEM) and its private
// key, sealed by the caller.
type SelfCert struct {
	CertPEM   string
	SealedKey string
}

// SelfCertAAD binds the sealed key to its setting.
const SelfCertAAD = "settings/" + keySelfKey

// SelfCert reads the stored self-signed certificate; ok is false when
// none was made yet.
func (s *Store) SelfCert(ctx context.Context) (SelfCert, bool, error) {
	var c SelfCert
	err := s.db.QueryRowContext(ctx, `SELECT c.value, k.value FROM settings c JOIN settings k ON k.key = ? WHERE c.key = ?`,
		keySelfKey, keySelfCert).Scan(&c.CertPEM, &c.SealedKey)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return SelfCert{}, false, nil
	case err != nil:
		return SelfCert{}, false, fmt.Errorf("read self-signed certificate: %w", err)
	}
	return c, true, nil
}

// SetSelfCert stores the self-signed certificate and its key together.
func (s *Store) SetSelfCert(ctx context.Context, c SelfCert) error {
	return s.setAll(ctx, map[string]string{keySelfCert: c.CertPEM, keySelfKey: c.SealedKey})
}

// setAll stores vals in one transaction.
func (s *Store) setAll(ctx context.Context, vals map[string]string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("save settings: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	for k, v := range vals {
		if _, err := tx.ExecContext(ctx, `INSERT INTO settings (key, value) VALUES (?, ?)
			ON CONFLICT (key) DO UPDATE SET value = excluded.value`, k, v); err != nil {
			return fmt.Errorf("save setting %s: %w", k, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("save settings: %w", err)
	}
	return nil
}
