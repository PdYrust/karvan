package dest

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/pem"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/PdYrust/karvan/internal/discovery"
	"github.com/PdYrust/karvan/internal/panel"
)

// Local is where the destination panel listens on this server, the way
// its main.py binds.
type Local struct {
	BaseURL  string // http(s)://host:port
	Socket   string // a Unix socket path, when the panel listens on one
	CertFile string // the panel's certificate, pinned when BaseURL is https
}

// Local reads the panel's listener from its ENV.
func (p *Panel) Local() Local {
	s := p.Found.Settings
	l := Local{Socket: s.UDS, BaseURL: baseURL(p.Found.Kind, s)}
	if s.CertFile != "" && s.KeyFile != "" {
		l.CertFile = s.CertFile
	}
	return l
}

// baseURL: with no certificate Marzban binds 127.0.0.1 and PasarGuard a
// private host or localhost; with one, the configured host.
func baseURL(kind panel.Kind, s discovery.Settings) string {
	scheme, host := "http", "127.0.0.1"
	tls := s.CertFile != "" && s.KeyFile != ""
	if tls {
		scheme = "https"
	}
	ip := net.ParseIP(s.Host)
	switch {
	case s.UDS != "":
		host = "localhost" // the socket carries it; the name is only the Host header
	case tls:
		if ip != nil && !ip.IsUnspecified() {
			host = s.Host
		}
	case kind == panel.PasarGuard && ip != nil && (ip.IsPrivate() || ip.IsLoopback()):
		host = s.Host
	}
	return scheme + "://" + net.JoinHostPort(host, strconv.Itoa(s.Port))
}

// Transport dials the panel: through its socket when it has one, and over
// TLS pinned to the exact certificate its ENV names. The certificate is
// for the panel's public name, not for 127.0.0.1, so the pin replaces the
// name check.
func (l Local) Transport(maxIdle int) (*http.Transport, error) {
	t := &http.Transport{MaxIdleConnsPerHost: maxIdle, IdleConnTimeout: 30 * time.Second, Proxy: nil}
	if l.Socket != "" {
		var d net.Dialer
		t.DialContext = func(ctx context.Context, _, _ string) (net.Conn, error) {
			return d.DialContext(ctx, "unix", l.Socket)
		}
	}
	if !strings.HasPrefix(l.BaseURL, "https://") {
		return t, nil
	}
	pin, err := leaf(l.CertFile)
	if err != nil {
		return nil, err
	}
	t.TLSClientConfig = &tls.Config{
		MinVersion:         tls.VersionTLS12,
		InsecureSkipVerify: true, //nolint:gosec // G402: replaced by the pin in VerifyConnection
		VerifyConnection: func(cs tls.ConnectionState) error {
			if len(cs.PeerCertificates) == 0 || !bytes.Equal(cs.PeerCertificates[0].Raw, pin) {
				return errors.New("the panel serves a different certificate than its ENV names")
			}
			return nil
		},
	}
	return t, nil
}

// leaf reads the first certificate in a PEM file.
func leaf(path string) ([]byte, error) {
	b, err := os.ReadFile(path) //nolint:gosec // G304: the path comes from the panel's ENV
	if err != nil {
		return nil, fmt.Errorf("read the panel's certificate: %w", err)
	}
	for {
		var blk *pem.Block
		blk, b = pem.Decode(b)
		if blk == nil {
			return nil, errors.New("the panel's certificate file has no certificate")
		}
		if blk.Type == "CERTIFICATE" {
			return blk.Bytes, nil
		}
	}
}
