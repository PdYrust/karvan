package bridge

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"io"
	"math/big"
	"net"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/PdYrust/karvan/internal/panel"
)

func TestSiteValidation(t *testing.T) {
	ok := Site{Name: "s", PanelID: 1, SourceKind: panel.Marzban, SourceURL: "https://old.example.com:8443", OldPath: "/sub/", Port: 443,
		Domains: []string{"Old.Example.com.", "*.cdn.example.com", "old.example.com"}}
	if err := ok.normalize(); err != nil || ok.OldPath != "sub" || len(ok.Domains) != 2 {
		t.Fatalf("%v %+v", err, ok)
	}
	for name, edit := range map[string]func(s *Site){
		"kind":      func(s *Site) { s.SourceKind = "x-ui" },
		"path":      func(s *Site) { s.OldPath = "a/b" },
		"port":      func(s *Site) { s.Port = 0 },
		"url":       func(s *Site) { s.SourceURL = "ftp://old" },
		"ip":        func(s *Site) { s.SourceIP = "old.example.com" },
		"metadata":  func(s *Site) { s.SourceIP = "169.254.169.254" },
		"pin":       func(s *Site) { s.SourcePin = "abc" },
		"domain":    func(s *Site) { s.Domains = []string{"*.*.example.com"} },
		"key file":  func(s *Site) { s.CertFile = "/c.pem" },
		"two certs": func(s *Site) { s.CertFile, s.KeyFile, s.CertPEM = "/c", "/k", "pem" },
	} {
		s := ok
		s.Domains = append([]string(nil), ok.Domains...)
		edit(&s)
		if err := s.normalize(); !errors.Is(err, ErrInvalidSite) {
			t.Errorf("%s: %v", name, err)
		}
	}
}

func TestDomainMatching(t *testing.T) {
	for _, c := range []struct {
		pattern, host string
		want          bool
	}{
		{"old.example.com", "old.example.com", true},
		{"*.example.com", "a.example.com", true},
		{"*.example.com", "example.com", false},
		{"*.example.com", "a.b.example.com", false},
		{"old.example.com", "xold.example.com", false},
	} {
		if got := matches(c.pattern, c.host); got != c.want {
			t.Errorf("%s ~ %s = %v", c.pattern, c.host, got)
		}
	}
}

func TestOverlappingSitesAreRefused(t *testing.T) {
	e := newEnv(t)
	ctx := context.Background()
	s := e.site
	s.ID, s.Name, s.Domains = 0, "second", []string{"*.example.com"}
	if _, err := e.b.st.Save(ctx, s); !errors.Is(err, ErrOverlap) {
		t.Fatalf("overlap: %v", err)
	}
	s.OldPath = "other"
	if _, err := e.b.st.Save(ctx, s); err != nil {
		t.Fatalf("another path: %v", err)
	}
}

// A pasted private key is stored sealed and never read back out.
func TestPastedKeyIsSealed(t *testing.T) {
	e := newEnv(t)
	ctx := context.Background()
	certPEM, keyPEM := selfSigned(t, "old.example.com")
	s := e.site
	s.CertPEM, s.KeyPEM = certPEM, keyPEM
	saved, err := e.b.st.Save(ctx, s)
	if err != nil || !saved.HasKey || saved.KeyPEM != "" {
		t.Fatalf("%v %+v", err, saved)
	}
	var raw []byte
	_ = e.b.DB.QueryRow("SELECT key_sealed FROM legacy_sites WHERE id = ?", s.ID).Scan(&raw)
	if len(raw) == 0 || string(raw) == keyPEM {
		t.Fatal("key stored in the clear")
	}
	if got, err := e.b.st.siteKey(ctx, s.ID); err != nil || string(got) != keyPEM {
		t.Fatalf("key round trip: %v", err)
	}
	saved.KeyPEM = "" // an update without a key keeps the stored one
	if again, err := e.b.st.Save(ctx, saved); err != nil || !again.HasKey {
		t.Fatalf("update: %v %+v", err, again)
	}
}

// The bridge listens on the site's port and serves HTTPS with the site's
// certificate picked by SNI.
func TestServesOverTLS(t *testing.T) {
	e := newEnv(t, "ali")
	ctx := context.Background()
	certPEM, keyPEM := selfSigned(t, "old.example.com")
	s := e.site
	s.Port, s.CertPEM, s.KeyPEM = freePort(t), certPEM, keyPEM
	if _, err := e.b.SaveSite(ctx, s); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(e.b.stopAll)
	pool := x509.NewCertPool()
	pool.AppendCertsFromPEM([]byte(certPEM))
	c := &http.Client{Transport: &http.Transport{TLSClientConfig: &tls.Config{RootCAs: pool, ServerName: "old.example.com", MinVersion: tls.VersionTLS12},
		DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
			return (&net.Dialer{}).DialContext(ctx, "tcp", "127.0.0.1:"+strconv.Itoa(s.Port))
		}}}
	resp, err := c.Get("https://old.example.com/sub/" + e.oldToken("ali", e.clock.now()))
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK || string(body) != "config /sub/dest-100-new_ali" {
		t.Fatalf("%d %q", resp.StatusCode, body)
	}
	if len(e.b.Status()) != 0 {
		t.Fatalf("status %v", e.b.Status())
	}
}

func freePort(t *testing.T) int {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	return ln.Addr().(*net.TCPAddr).Port
}

func selfSigned(t *testing.T, name string) (string, string) {
	t.Helper()
	return selfSignedFor(t, name, time.Hour)
}

// selfSignedFor is selfSigned valid for d from an hour ago.
func selfSignedFor(t *testing.T, name string, d time.Duration) (string, string) {
	t.Helper()
	k, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	tpl := &x509.Certificate{SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: name}, DNSNames: []string{name},
		NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().Add(d)}
	der, err := x509.CreateCertificate(rand.Reader, tpl, tpl, &k.PublicKey, k)
	if err != nil {
		t.Fatal(err)
	}
	kb, err := x509.MarshalPKCS8PrivateKey(k)
	if err != nil {
		t.Fatal(err)
	}
	return string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})),
		string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: kb}))
}
