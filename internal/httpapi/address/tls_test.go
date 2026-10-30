package address

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/PdYrust/karvan/internal/settings"
)

var (
	self   = settings.TLS{Mode: settings.TLSSelf}
	domain = settings.TLS{Mode: settings.TLSDomain, DomainID: 3}
)

func testCert(t *testing.T) *tls.Certificate {
	t.Helper()
	k, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	tpl := &x509.Certificate{SerialNumber: big.NewInt(1), IPAddresses: []net.IP{net.IPv4(127, 0, 0, 1)},
		NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().Add(time.Hour)}
	der, err := x509.CreateCertificate(rand.Reader, tpl, tpl, &k.PublicKey, k)
	if err != nil {
		t.Fatal(err)
	}
	return &tls.Certificate{Certificate: [][]byte{der}, PrivateKey: k}
}

// withCerts gives the fixture's HTTPS addresses a certificate.
func (f *fixture) withCerts() {
	c := testCert(f.t)
	f.srv.mu.Lock()
	defer f.srv.mu.Unlock()
	f.srv.Certificate = func(context.Context, settings.TLS) (*tls.Certificate, error) { return c, nil }
	for _, l := range f.srv.ports {
		l.certs = f.srv.Certificate
	}
}

// fetch is the body (or status) and the HSTS header at port and path over
// scheme.
func (f *fixture) fetch(scheme string, port int, path string) (string, string) {
	f.t.Helper()
	c := &http.Client{Timeout: 5 * time.Second, Transport: &http.Transport{DisableKeepAlives: true,
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true, MinVersion: tls.VersionTLS12}}}
	res, err := c.Get(fmt.Sprintf("%s://127.0.0.1:%d/%s/", scheme, port, path))
	if err != nil {
		return "refused", ""
	}
	defer res.Body.Close()
	b, _ := io.ReadAll(res.Body)
	if res.StatusCode != http.StatusOK {
		return fmt.Sprint(res.StatusCode), ""
	}
	return string(b), res.Header.Get("Strict-Transport-Security")
}

func noRedirects() *http.Client {
	return &http.Client{Timeout: 5 * time.Second, Transport: &http.Transport{DisableKeepAlives: true},
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
}

// plainGet is the status and Location of a plain HTTP GET of path.
func plainGet(t *testing.T, port int, path string) string {
	t.Helper()
	res, err := noRedirects().Get(fmt.Sprintf("http://127.0.0.1:%d%s", port, path))
	if err != nil {
		return "refused"
	}
	res.Body.Close()
	return fmt.Sprintf("%d %s", res.StatusCode, res.Header.Get("Location"))
}

// D7: a change to HTTPS on the same port serves both schemes there until
// it is confirmed from the HTTPS address; then plain HTTP stops.
func TestSchemeChangeOnTheSamePort(t *testing.T) {
	f := newFixture(t, 0)
	f.withCerts()
	p, err := f.srv.Start(Change{TLS: &self})
	if err != nil || p.Site != (Site{Port: f.port, BasePath: oldPath, TLS: self}) {
		t.Fatalf("Start = %+v, %v", p, err)
	}
	if got, _ := f.fetch("http", f.port, oldPath); !strings.HasPrefix(got, oldPath+" via") {
		t.Fatalf("plain HTTP while waiting: %s", got)
	}
	got, hsts := f.fetch("https", f.port, oldPath)
	if !strings.HasPrefix(got, oldPath+" via") || hsts != "" {
		t.Fatalf("HTTPS while waiting: %s, HSTS %q in self mode", got, hsts)
	}
	if _, err := f.srv.Confirm(context.Background(), Site{Port: f.port, BasePath: oldPath}); !errors.Is(err, ErrNotNew) {
		t.Fatalf("confirm over plain HTTP: %v", err)
	}
	if _, err := f.srv.Confirm(context.Background(), p.Site); err != nil {
		t.Fatal(err)
	}
	f.confirmed(p.Site)
	if got := plainGet(t, f.port, "/"+oldPath+"/"); got != fmt.Sprintf("307 https://127.0.0.1:%d/%s/", f.port, oldPath) {
		t.Fatalf("plain HTTP after the confirm: %s", got)
	}
	if got, _ := f.fetch("https", f.port, oldPath); !strings.HasPrefix(got, oldPath+" via") {
		t.Fatalf("HTTPS after the confirm: %s", got)
	}
	if _, err := f.srv.Start(Change{TLS: &self}); !errors.Is(err, ErrNoChange) {
		t.Fatalf("the same mode again: %v", err)
	}
}

// D8: HSTS only in domain mode, for a day.
func TestHSTSOnlyInDomainMode(t *testing.T) {
	f := newFixture(t, 0)
	f.withCerts()
	port := freePort(t)
	if _, err := f.srv.Start(Change{Port: port, TLS: &domain}); err != nil {
		t.Fatal(err)
	}
	if got, hsts := f.fetch("https", port, oldPath); !strings.HasPrefix(got, oldPath+" via") || hsts != "max-age=86400" {
		t.Fatalf("domain mode: %s, HSTS %q", got, hsts)
	}
	if _, hsts := f.fetch("http", f.port, oldPath); hsts != "" {
		t.Fatalf("plain HTTP sent HSTS %q", hsts)
	}
}

// Plain HTTP on an HTTPS-only port: a GET or HEAD under the hidden path is
// sent on to HTTPS, anything else gets the plain 404, so the redirect never
// shows the path to someone who does not know it.
func TestPlainHTTPOnAnHTTPSPort(t *testing.T) {
	f := newFixture(t, 0)
	f.withCerts()
	port := freePort(t)
	if _, err := f.srv.Start(Change{Port: port, TLS: &self}); err != nil {
		t.Fatal(err)
	}
	c := noRedirects()
	base := fmt.Sprintf("http://127.0.0.1:%d", port)
	for _, tc := range []struct{ method, path, want string }{
		{http.MethodGet, "/" + oldPath + "/", "307 https://127.0.0.1:" + fmt.Sprint(port) + "/" + oldPath + "/"},
		{http.MethodGet, "/" + oldPath + "/login?x=1", "307 https://127.0.0.1:" + fmt.Sprint(port) + "/" + oldPath + "/login?x=1"},
		{http.MethodHead, "/" + oldPath, "307 https://127.0.0.1:" + fmt.Sprint(port) + "/" + oldPath},
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
	if got, _ := f.fetch("https", port, oldPath); !strings.HasPrefix(got, oldPath+" via") {
		t.Fatalf("HTTPS next to plain HTTP: %s", got)
	}
}

// A change to HTTPS that is not confirmed goes away; plain HTTP stays.
func TestUnconfirmedSchemeChangeGoesAway(t *testing.T) {
	f := newFixture(t, 100*time.Millisecond)
	f.withCerts()
	if _, err := f.srv.Start(Change{TLS: &self}); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(5 * time.Second)
	for f.srv.Status().Pending != nil {
		if time.Now().After(deadline) {
			t.Fatal("the change did not expire")
		}
		time.Sleep(20 * time.Millisecond)
	}
	if got, _ := f.fetch("https", f.port, oldPath); got != "refused" {
		t.Fatalf("HTTPS after it expired: %s", got)
	}
	if got, _ := f.fetch("http", f.port, oldPath); !strings.HasPrefix(got, oldPath+" via") || len(f.savedSites()) != 0 {
		t.Fatalf("plain HTTP after it expired: %s, saved %v", got, f.savedSites())
	}
}

// A connection that never sends a byte does not hold up the others while
// both schemes share the port.
func TestSilentConnectionDoesNotBlockTheOthers(t *testing.T) {
	f := newFixture(t, 0)
	f.withCerts()
	if _, err := f.srv.Start(Change{TLS: &self}); err != nil {
		t.Fatal(err)
	}
	silent, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", f.port))
	if err != nil {
		t.Fatal(err)
	}
	defer silent.Close()
	if got, _ := f.fetch("http", f.port, oldPath); !strings.HasPrefix(got, oldPath+" via") {
		t.Fatalf("plain HTTP next to a silent connection: %s", got)
	}
}
