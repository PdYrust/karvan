package paneltls

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"strings"
	"time"
)

// Self-signed certificate lifetimes: 397 days is the longest a browser
// accepts, and the certificate is made again 30 days before its end.
const (
	selfLife    = 397 * 24 * time.Hour
	renewBefore = 30 * 24 * time.Hour
)

// newSelf makes an ECDSA P-256 certificate for ips and localhost, valid
// from now for selfLife. It returns the certificate and key in PEM.
func newSelf(ips []net.IP, now time.Time) (certPEM, keyPEM []byte, err error) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, nil, fmt.Errorf("generate key: %w", err)
	}
	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 127))
	if err != nil {
		return nil, nil, fmt.Errorf("generate serial: %w", err)
	}
	tmpl := &x509.Certificate{
		SerialNumber: serial,
		Subject:      pkix.Name{CommonName: "Karvan"},
		// An hour back, so a client clock a little behind still accepts it.
		NotBefore:   now.Add(-time.Hour),
		NotAfter:    now.Add(selfLife),
		KeyUsage:    x509.KeyUsageDigitalSignature,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		DNSNames:    []string{"localhost"},
		IPAddresses: ips,
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		return nil, nil, fmt.Errorf("create certificate: %w", err)
	}
	keyDER, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		return nil, nil, fmt.Errorf("encode key: %w", err)
	}
	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}),
		pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyDER}), nil
}

// due reports whether c is to be made again at now.
func due(c *x509.Certificate, now time.Time) bool { return !now.Before(c.NotAfter.Add(-renewBefore)) }

// Fingerprint is the SHA-256 of a certificate as browsers show it: upper
// case hex pairs joined by colons.
func Fingerprint(der []byte) string {
	sum := sha256.Sum256(der)
	h := strings.ToUpper(hex.EncodeToString(sum[:]))
	pairs := make([]string, 0, len(sum))
	for i := 0; i < len(h); i += 2 {
		pairs = append(pairs, h[i:i+2])
	}
	return strings.Join(pairs, ":")
}

// serverIPs are this server's addresses a browser can reach it on: every
// interface address but link-local and multicast ones, loopback included.
func serverIPs() []net.IP {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return []net.IP{net.IPv4(127, 0, 0, 1), net.IPv6loopback}
	}
	var out []net.IP
	for _, a := range addrs {
		n, ok := a.(*net.IPNet)
		if !ok || n.IP.IsLinkLocalUnicast() || n.IP.IsMulticast() || n.IP.IsUnspecified() {
			continue
		}
		out = append(out, n.IP)
	}
	return out
}
