package bridge

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"database/sql"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"
)

// Certificate warnings, as codes the web panel puts in words.
const (
	WarnUnreadable   = "unreadable"      // no certificate could be read or parsed
	WarnNoKey        = "no_key"          // no readable private key for it
	WarnKeyMismatch  = "key_mismatch"    // the key does not belong to the certificate
	WarnDomain       = "domain_mismatch" // a site domain the certificate does not cover
	WarnExpiresSoon  = "expires_soon"    // ends within expiresSoon
	WarnExpired      = "expired"
	WarnNotYetValid  = "not_yet_valid"
	expiresSoon      = 14 * 24 * time.Hour
	maxCertFileBytes = 1 << 20
)

// CertFacts is what the bridge can tell about a site's certificate.
type CertFacts struct {
	Domains   []string  `json:"domains"`
	Issuer    string    `json:"issuer"`
	NotBefore time.Time `json:"notBefore,omitzero"`
	NotAfter  time.Time `json:"notAfter,omitzero"`
	Warnings  []string  `json:"warnings"`
}

// CertInput is a certificate to look at before saving a rule: files on
// this server, a pasted certificate with its key, or a stored certificate
// (CertificateID). With SiteID set and no KeyPEM, the site's stored key is
// used.
type CertInput struct {
	SiteID        int64    `json:"id"`
	CertificateID int64    `json:"certificateId"`
	CertFile      string   `json:"certFile"`
	KeyFile       string   `json:"keyFile"`
	CertPEM       string   `json:"certPem"`
	KeyPEM        string   `json:"keyPem"`
	Domains       []string `json:"domains"`
}

// ErrNoCert means a certificate inspection was given no certificate.
var ErrNoCert = errors.New("no certificate to inspect")

// InspectCert reads the certificate in in and says what it covers and what
// is wrong with it. Nothing read from a file is returned except these facts.
func (b *Bridge) InspectCert(ctx context.Context, in CertInput) (CertFacts, error) {
	if in.CertificateID != 0 {
		return b.inspectStored(ctx, in)
	}
	s := Site{ID: in.SiteID, CertFile: in.CertFile, KeyFile: in.KeyFile, CertPEM: in.CertPEM, KeyPEM: in.KeyPEM,
		Domains: in.Domains}
	if err := s.checkCert(); err != nil {
		return CertFacts{}, err
	}
	if !s.TLS() {
		return CertFacts{}, ErrNoCert
	}
	if err := s.normalizeDomains(); err != nil {
		return CertFacts{}, err
	}
	if s.CertPEM != "" && s.KeyPEM == "" && s.ID != 0 {
		key, err := b.st.siteKey(ctx, s.ID)
		if err != nil && !errors.Is(err, ErrNotFound) {
			return CertFacts{}, err
		}
		s.KeyPEM = string(key)
	}
	return b.certFacts(s), nil
}

// inspectStored checks stored certificate in.CertificateID against
// in.Domains; it takes no other certificate.
func (b *Bridge) inspectStored(ctx context.Context, in CertInput) (CertFacts, error) {
	if in.SiteID != 0 || in.CertFile != "" || in.KeyFile != "" || in.CertPEM != "" || in.KeyPEM != "" {
		return CertFacts{}, invalid("certificateId")
	}
	s := Site{Domains: in.Domains}
	if err := s.normalizeDomains(); err != nil {
		return CertFacts{}, err
	}
	r, err := scanCertRow(b.DB.QueryRowContext(ctx, certRowColumns+" WHERE id = ?", in.CertificateID))
	if errors.Is(err, sql.ErrNoRows) {
		return CertFacts{}, invalid("certificateId")
	}
	if err != nil {
		return CertFacts{}, err
	}
	certPEM, keyPEM := b.material(r)
	return inspect(certPEM, keyPEM, s.Domains, b.Now()), nil
}

// siteCert reads the facts of a saved site's certificate, or nil without TLS.
func (b *Bridge) siteCert(ctx context.Context, s Site) *CertFacts {
	if !s.TLS() {
		return nil
	}
	if s.CertPEM != "" {
		key, err := b.st.siteKey(ctx, s.ID)
		if err != nil {
			b.Log.Warn("bridge: read site key", "site", s.Name, "err", err)
		}
		s.KeyPEM = string(key)
	}
	f := b.certFacts(s)
	return &f
}

// certFacts reads s's certificate and key (files or pasted) and checks them.
func (b *Bridge) certFacts(s Site) CertFacts {
	certPEM, keyPEM := []byte(s.CertPEM), []byte(s.KeyPEM)
	if s.CertFile != "" {
		var err error
		if certPEM, err = readSmallFile(s.CertFile); err != nil {
			return CertFacts{Domains: []string{}, Warnings: []string{WarnUnreadable}}
		}
		if keyPEM, err = readSmallFile(s.KeyFile); err != nil {
			keyPEM = nil
		}
	}
	return inspect(certPEM, keyPEM, s.Domains, b.Now())
}

// readSmallFile reads a regular file of at most maxCertFileBytes. A FIFO or
// device is refused before it is opened, so a wrong path cannot hang a request.
func readSmallFile(path string) ([]byte, error) {
	if !filepath.IsAbs(path) {
		return nil, fmt.Errorf("%s: not an absolute path", path)
	}
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() || info.Size() > maxCertFileBytes {
		return nil, fmt.Errorf("%s: not a certificate file", path)
	}
	f, err := os.Open(path) //nolint:gosec // the operator names this file; only parsed facts leave it
	if err != nil {
		return nil, err
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, maxCertFileBytes+1))
	if err == nil && len(data) > maxCertFileBytes {
		err = fmt.Errorf("%s: too large", path)
	}
	return data, err
}

// inspect parses the first certificate in certPEM and checks it against
// its key (nil: none), the site's domains and the time now.
func inspect(certPEM, keyPEM []byte, domains []string, now time.Time) CertFacts {
	leaf := firstCert(certPEM)
	if leaf == nil {
		return CertFacts{Domains: []string{}, Warnings: []string{WarnUnreadable}}
	}
	f := CertFacts{Domains: certNames(leaf), Issuer: issuerName(leaf), NotBefore: leaf.NotBefore.UTC(),
		NotAfter: leaf.NotAfter.UTC(), Warnings: []string{}}
	switch {
	case len(keyPEM) == 0:
		f.Warnings = append(f.Warnings, WarnNoKey)
	default:
		if _, err := tls.X509KeyPair(certPEM, keyPEM); err != nil {
			f.Warnings = append(f.Warnings, WarnKeyMismatch)
		}
	}
	if slices.ContainsFunc(domains, func(d string) bool { return !certCovers(leaf, d) }) {
		f.Warnings = append(f.Warnings, WarnDomain)
	}
	switch {
	case now.Before(leaf.NotBefore):
		f.Warnings = append(f.Warnings, WarnNotYetValid)
	case !now.Before(leaf.NotAfter):
		f.Warnings = append(f.Warnings, WarnExpired)
	case leaf.NotAfter.Sub(now) <= expiresSoon:
		f.Warnings = append(f.Warnings, WarnExpiresSoon)
	}
	return f
}

func firstCert(data []byte) *x509.Certificate {
	for len(data) > 0 {
		var block *pem.Block
		block, data = pem.Decode(data)
		if block == nil {
			return nil
		}
		if block.Type != "CERTIFICATE" {
			continue
		}
		c, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			return nil
		}
		return c
	}
	return nil
}

// certNames are the names a certificate covers: its DNS and IP names, or
// the subject's common name when it has none.
func certNames(c *x509.Certificate) []string {
	out := make([]string, 0, len(c.DNSNames)+len(c.IPAddresses))
	for _, d := range c.DNSNames {
		out = append(out, strings.ToLower(d))
	}
	for _, ip := range c.IPAddresses {
		out = append(out, ip.String())
	}
	if len(out) == 0 && c.Subject.CommonName != "" {
		out = append(out, strings.ToLower(c.Subject.CommonName))
	}
	return out
}

func issuerName(c *x509.Certificate) string {
	switch {
	case c.Issuer.CommonName != "":
		return c.Issuer.CommonName
	case len(c.Issuer.Organization) > 0:
		return c.Issuer.Organization[0]
	}
	return c.Issuer.String()
}

// certCovers reports whether c serves the site pattern d. A wildcard
// pattern needs the same wildcard in the certificate.
func certCovers(c *x509.Certificate, d string) bool {
	if strings.HasPrefix(d, "*.") {
		return slices.ContainsFunc(c.DNSNames, func(n string) bool { return strings.EqualFold(n, d) })
	}
	return c.VerifyHostname(d) == nil
}
