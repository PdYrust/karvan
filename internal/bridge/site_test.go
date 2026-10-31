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
	mrand "math/rand/v2"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
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
		"kind":     func(s *Site) { s.SourceKind = "x-ui" },
		"path":     func(s *Site) { s.OldPath = "a/b" },
		"port":     func(s *Site) { s.Port = 0 },
		"url":      func(s *Site) { s.SourceURL = "ftp://old" },
		"ip":       func(s *Site) { s.SourceIP = "old.example.com" },
		"metadata": func(s *Site) { s.SourceIP = "169.254.169.254" },
		"pin":      func(s *Site) { s.SourcePin = "abc" },
		"domain":   func(s *Site) { s.Domains = []string{"*.*.example.com"} },
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
	s.ID, s.Name = 0, "second"
	if _, err := e.b.st.Save(ctx, s); !errors.Is(err, ErrOverlap) {
		t.Fatalf("overlap: %v", err)
	}
	s.OldPath = "other"
	if _, err := e.b.st.Save(ctx, s); err != nil {
		t.Fatalf("another path: %v", err)
	}
}

// The bridge listens on the site's port and serves HTTPS with the site's
// certificate picked by SNI.
func TestServesOverTLS(t *testing.T) {
	e := newEnv(t, "ali")
	ctx := context.Background()
	certPEM, keyPEM := selfSigned(t, "old.example.com")
	e.certify("old.example.com", certPEM, keyPEM)
	s := e.site
	on := true
	s.Port, s.HTTPS = freePort(t), &on
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

// handedOut is every port freePort gave out in this run. A test seeds
// several rules from freePort, and a port the kernel handed out for :0 and
// got back could come again, so a plain HTTP rule would share a port with
// a TLS rule and the bridge would refuse the port as mixed.
var handedOut = struct {
	sync.Mutex
	ports map[int]bool
}{ports: map[int]bool{}}

// freePort is a port no other freePort call in the run gave out, free on
// every address that the bridge opens later. It comes from below the
// kernel's ephemeral range, so an outgoing connection cannot take it as
// its source port before the bridge binds it.
func freePort(t *testing.T) int {
	t.Helper()
	handedOut.Lock()
	defer handedOut.Unlock()
	low, high := 10000, ephemeralStart()
	for range 200 {
		p := low + mrand.IntN(high-low)
		if handedOut.ports[p] {
			continue
		}
		if ln, err := net.Listen("tcp", ":"+strconv.Itoa(p)); err == nil {
			_ = ln.Close()
			handedOut.ports[p] = true
			return p
		}
	}
	t.Fatal("no free port below the ephemeral range")
	return 0
}

// ephemeralStart is the first port of the kernel's ephemeral range.
func ephemeralStart() int {
	b, err := os.ReadFile("/proc/sys/net/ipv4/ip_local_port_range")
	if f := strings.Fields(string(b)); err == nil && len(f) == 2 {
		if n, err := strconv.Atoi(f[0]); err == nil && n > 10100 {
			return n
		}
	}
	return 32768
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
