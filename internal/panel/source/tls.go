package source

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"errors"
	"net"
	"strings"
	"time"
)

var errPinMismatch = errors.New("certificate does not match the pinned fingerprint")

// tlsConfig verifies certificates normally. With a pin, the chain is not
// checked against system roots; instead the leaf must be exactly the
// certificate the operator confirmed. There is no skip-verification mode.
func tlsConfig(pinHex string) (*tls.Config, error) {
	c := &tls.Config{MinVersion: tls.VersionTLS12}
	if pinHex == "" {
		return c, nil
	}
	pin, err := hex.DecodeString(strings.ReplaceAll(strings.ToLower(pinHex), ":", ""))
	if err != nil || len(pin) != sha256.Size {
		return nil, errors.New("certificate fingerprint must be a SHA-256 in hex")
	}
	// VerifyConnection also runs on resumed sessions, unlike
	// VerifyPeerCertificate.
	c.InsecureSkipVerify = true
	c.VerifyConnection = func(cs tls.ConnectionState) error {
		if len(cs.PeerCertificates) == 0 {
			return errPinMismatch
		}
		sum := sha256.Sum256(cs.PeerCertificates[0].Raw)
		if subtle.ConstantTimeCompare(sum[:], pin) != 1 {
			return errPinMismatch
		}
		return nil
	}
	return c, nil
}

// Certificate is what a panel presents, shown to the operator before they
// decide to pin it.
type Certificate struct {
	SHA256   string
	Subject  string
	Issuer   string
	DNSNames []string
	NotAfter time.Time
	Trusted  bool // valid for the host against the system roots
}

// FetchCertificate connects to base (an https panel address) through the
// same address guard and reports the leaf certificate.
func FetchCertificate(ctx context.Context, base string, r Resolver) (Certificate, error) {
	u, err := ParseBase(base)
	if err != nil {
		return Certificate{}, err
	}
	if u.Scheme != "https" {
		return Certificate{}, errors.New("panel address is not https")
	}
	port := u.Port()
	if port == "" {
		port = "443"
	}
	raw, err := newGuardedDialer(r).DialContext(ctx, "tcp", net.JoinHostPort(u.Hostname(), port))
	if err != nil {
		return Certificate{}, transportError(err)
	}
	conn := tls.Client(raw, &tls.Config{ServerName: u.Hostname(), InsecureSkipVerify: true}) //nolint:gosec // only reads the certificate to show it
	defer conn.Close()
	if err := conn.HandshakeContext(ctx); err != nil {
		return Certificate{}, &Error{Code: CodeTLS, Err: err}
	}
	certs := conn.ConnectionState().PeerCertificates
	if len(certs) == 0 {
		return Certificate{}, &Error{Code: CodeTLS, Err: errors.New("no certificate")}
	}
	leaf := certs[0]
	sum := sha256.Sum256(leaf.Raw)
	pool := x509.NewCertPool()
	for _, c := range certs[1:] {
		pool.AddCert(c)
	}
	_, verr := leaf.Verify(x509.VerifyOptions{DNSName: u.Hostname(), Intermediates: pool})
	return Certificate{
		SHA256:   hex.EncodeToString(sum[:]),
		Subject:  leaf.Subject.String(),
		Issuer:   leaf.Issuer.String(),
		DNSNames: leaf.DNSNames,
		NotAfter: leaf.NotAfter,
		Trusted:  verr == nil,
	}, nil
}

// Fingerprint formats a SHA-256 as colon-separated hex pairs for display.
func Fingerprint(hexSum string) string {
	var b strings.Builder
	for i := 0; i+1 < len(hexSum); i += 2 {
		if i > 0 {
			b.WriteByte(':')
		}
		b.WriteString(strings.ToUpper(hexSum[i : i+2]))
	}
	return b.String()
}
