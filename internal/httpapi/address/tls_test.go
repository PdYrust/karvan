package address

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"fmt"
	"io"
	"math/big"
	"net"
	"net/http"
	"net/netip"
	"strings"
	"testing"
	"time"
)

func testCert(t *testing.T) *tls.Certificate { return namedCert(t, "test") }

// namedCert is a certificate whose subject is name, for 127.0.0.1.
func namedCert(t *testing.T, name string) *tls.Certificate {
	t.Helper()
	k, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	tpl := &x509.Certificate{SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: name},
		IPAddresses: []net.IP{net.IPv4(127, 0, 0, 1)}, NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().Add(time.Hour)}
	der, err := x509.CreateCertificate(rand.Reader, tpl, tpl, &k.PublicKey, k)
	if err != nil {
		t.Fatal(err)
	}
	return &tls.Certificate{Certificate: [][]byte{der}, PrivateKey: k}
}

// fetch is the body (or status) and the HSTS header of a GET of path on
// port over scheme, asking for host.
func (f *fixture) fetch(scheme string, port int, host, path string) (string, string) {
	f.t.Helper()
	c := &http.Client{Timeout: 5 * time.Second, Transport: &http.Transport{DisableKeepAlives: true,
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true, MinVersion: tls.VersionTLS12, ServerName: host}},
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s://127.0.0.1:%d/%s/", scheme, port, path), nil)
	if err != nil {
		f.t.Fatal(err)
	}
	if host != "" {
		req.Host = fmt.Sprintf("%s:%d", host, port)
	}
	res, err := c.Do(req)
	if err != nil {
		return "refused", ""
	}
	defer res.Body.Close()
	b, _ := io.ReadAll(res.Body)
	if res.StatusCode != http.StatusOK {
		return fmt.Sprint(res.StatusCode, " ", res.Header.Get("Location")), ""
	}
	return string(b), res.Header.Get("Strict-Transport-Security")
}

func noRedirects() *http.Client {
	return &http.Client{Timeout: 5 * time.Second, Transport: &http.Transport{DisableKeepAlives: true},
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
}

// Plain HTTP: a GET or HEAD under the hidden path is sent on to HTTPS on
// the same port, anything else gets the plain 404, so the redirect never
// shows the path to someone who does not know it.
func TestPlainHTTPIsSentOnToHTTPS(t *testing.T) {
	f := newFixture(t, 0)
	c := noRedirects()
	base := fmt.Sprintf("http://127.0.0.1:%d", f.port)
	to := fmt.Sprintf("307 https://127.0.0.1:%d/%s", f.port, oldPath)
	for _, tc := range []struct{ method, path, want string }{
		{http.MethodGet, "/" + oldPath + "/", to + "/"},
		{http.MethodGet, "/" + oldPath + "/login?x=1", to + "/login?x=1"},
		{http.MethodHead, "/" + oldPath, to},
		{http.MethodGet, "/", "404 "},
		{http.MethodGet, "/wrong/", "404 "},
		{http.MethodGet, "/" + oldPath + "x/", "404 "},
		{http.MethodPost, "/" + oldPath + "/", "404 "},
	} {
		req, err := http.NewRequest(tc.method, base+tc.path, nil)
		if err != nil {
			t.Fatal(err)
		}
		res, err := c.Do(req)
		if err != nil {
			t.Fatalf("%s %s: %v", tc.method, tc.path, err)
		}
		body, _ := io.ReadAll(res.Body)
		res.Body.Close()
		if got := fmt.Sprintf("%d %s", res.StatusCode, res.Header.Get("Location")); got != tc.want {
			t.Errorf("%s %s = %s, want %s", tc.method, tc.path, got, tc.want)
		}
		if res.StatusCode == http.StatusNotFound && strings.Contains(string(body), oldPath) {
			t.Errorf("%s %s: the 404 shows the hidden path", tc.method, tc.path)
		}
	}
	if got, hsts := f.fetch("https", f.port, "", oldPath); !strings.HasPrefix(got, oldPath+" via") || hsts != "" {
		t.Fatalf("HTTPS next to plain HTTP: %s, HSTS %q", got, hsts)
	}
}

// Plain HTTP is answered as is for a host Plain allows (the Karvan domain
// without a certificate) and from a peer in PlainFrom (this server, a
// trusted proxy); never with HSTS.
func TestPlainHTTPWhereAllowed(t *testing.T) {
	f := newFixtureWith(t, 0, func(s *Server) {
		s.Plain = func(_ context.Context, host string) bool { return host == "panel.example.com" }
	})
	trusted := newFixtureWith(t, 0, func(s *Server) { s.PlainFrom = []netip.Prefix{netip.MustParsePrefix("127.0.0.0/8")} })
	via := fmt.Sprintf("%s via %d/%s", oldPath, f.port, oldPath)
	if got, hsts := f.fetch("http", f.port, "PANEL.example.com", oldPath); got != via || hsts != "" {
		t.Fatalf("the allowed host: %s, HSTS %q", got, hsts)
	}
	if got, _ := f.fetch("http", f.port, "other.example.com", oldPath); !strings.HasPrefix(got, "307 https://other.example.com:") {
		t.Fatalf("another host: %s", got)
	}
	via = fmt.Sprintf("%s via %d/%s", oldPath, trusted.port, oldPath)
	if got, _ := trusted.fetch("http", trusted.port, "other.example.com", oldPath); got != via {
		t.Fatalf("from a trusted peer: %s", got)
	}
}

// leafCN is the subject of the certificate a handshake for name gets on
// port; "" sends no server name, as a browser does for an IP address.
func leafCN(t *testing.T, port int, name string) string {
	t.Helper()
	conn, err := tls.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", port), &tls.Config{InsecureSkipVerify: true,
		MinVersion: tls.VersionTLS12, ServerName: name})
	if err != nil {
		t.Fatalf("handshake for %q: %v", name, err)
	}
	defer conn.Close()
	return conn.ConnectionState().PeerCertificates[0].Subject.CommonName
}

// Each handshake gets the certificate its server name picks, on the
// current port and on a new one waiting for its confirm.
func TestHandshakePicksByServerName(t *testing.T) {
	own, domain := namedCert(t, "own"), namedCert(t, "domain")
	f := newFixtureWith(t, 0, func(s *Server) {
		s.Certificate = func(_ context.Context, name string) (*tls.Certificate, error) {
			if name == "panel.example.com" {
				return domain, nil
			}
			return own, nil
		}
	})
	port := freePort(t)
	if _, err := f.srv.Start(Change{Port: port}); err != nil {
		t.Fatal(err)
	}
	for _, p := range []int{f.port, port} {
		for name, want := range map[string]string{"panel.example.com": "domain", "other.example.com": "own", "": "own"} {
			if got := leafCN(t, p, name); got != want {
				t.Errorf("port %d, name %q: %s, want %s", p, name, got, want)
			}
		}
	}
}

// A connection that never sends a byte does not hold up the others.
func TestSilentConnectionDoesNotBlockTheOthers(t *testing.T) {
	f := newFixture(t, 0)
	silent, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", f.port))
	if err != nil {
		t.Fatal(err)
	}
	defer silent.Close()
	if got := f.get(f.port, oldPath); !strings.HasPrefix(got, oldPath+" via") {
		t.Fatalf("HTTPS next to a silent connection: %s", got)
	}
	if got, _ := f.fetch("http", f.port, "", oldPath); !strings.HasPrefix(got, "307 ") {
		t.Fatalf("plain HTTP next to a silent connection: %s", got)
	}
}
