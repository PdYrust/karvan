package settings

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
)

const (
	keyTLSDomain = "http.tls_domain"
	keyTLSPlain  = "http.tls_plain"
	keyTLSShared = "http.tls_shared"
	keySelfCert  = "http.tls_self_cert"
	keySelfKey   = "http.tls_self_key"
)

// TLS is how the web panel answers on its port (ADR 0010). Every TLS
// handshake gets Karvan's own self-signed certificate, except one for the
// name of the domain enabled for Karvan while that domain has a
// certificate: it gets the domain's. The zero value is no domain.
type TLS struct {
	DomainID int64 // the domain enabled for Karvan; 0 for none
	// Plain answers a plain HTTP request for the enabled domain's name as
	// is, while the domain has no certificate, instead of sending it on to
	// HTTPS.
	Plain  bool
	Shared bool // served on a bridge HTTPS port of that domain (ADR 0011)
}

// Validate checks that a shared port goes with a domain.
func (t TLS) Validate() error {
	switch {
	case t.DomainID < 0:
		return fmt.Errorf("tls domain %d is not a domain", t.DomainID)
	case t.Shared && t.DomainID == 0:
		return errors.New("a shared port needs a domain")
	}
	return nil
}

// rows are the stored values of t.
func (t TLS) rows() map[string]string {
	return map[string]string{keyTLSDomain: strconv.FormatInt(t.DomainID, 10), keyTLSPlain: strconv.FormatBool(t.Plain),
		keyTLSShared: strconv.FormatBool(t.Shared)}
}

// tlsOf reads TLS from the stored values; missing ones mean no domain.
func tlsOf(vals map[string]string) (TLS, error) {
	var t TLS
	if v := vals[keyTLSDomain]; v != "" {
		id, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return TLS{}, fmt.Errorf("stored tls domain %q is not a number", v)
		}
		t.DomainID = id
	}
	t.Plain = vals[keyTLSPlain] == "true"
	t.Shared = vals[keyTLSShared] == "true"
	if err := t.Validate(); err != nil {
		return TLS{}, err
	}
	return t, nil
}

// SetKarvan stores the domain enabled for Karvan (0 for none) and whether
// it answers plain HTTP for its name while it has no certificate, in one
// transaction. The address stays as it is.
func (s *Store) SetKarvan(ctx context.Context, domainID int64, plain bool) error {
	if domainID < 0 {
		return fmt.Errorf("tls domain %d is not a domain", domainID)
	}
	return s.setAll(ctx, map[string]string{keyTLSDomain: strconv.FormatInt(domainID, 10), keyTLSPlain: strconv.FormatBool(plain)})
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
