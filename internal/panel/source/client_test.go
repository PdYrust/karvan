package source

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

type recorder struct{ waits []time.Duration }

func (r *recorder) sleep(_ context.Context, d time.Duration) error {
	r.waits = append(r.waits, d)
	return nil
}

func client(t *testing.T, base string, o Options) (*Client, *recorder) {
	t.Helper()
	rec := &recorder{}
	o.BaseURL, o.Sleep = base, rec.sleep
	c, err := New(o)
	if err != nil {
		t.Fatal(err)
	}
	return c, rec
}

func TestRetriesRateLimitWithRetryAfter(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if calls.Add(1) < 3 {
			w.Header().Set("Retry-After", "7")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()
	c, rec := client(t, srv.URL, Options{})
	resp, err := c.Get(context.Background(), "/api/users", nil, nil)
	if err != nil || string(resp.Body) != `{"ok":true}` {
		t.Fatalf("%v %v", resp, err)
	}
	if len(rec.waits) != 2 || rec.waits[0] != 7*time.Second {
		t.Fatalf("waits = %v", rec.waits)
	}
}

func TestGivesUpOnServerErrors(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer srv.Close()
	c, _ := client(t, srv.URL, Options{Attempts: 4})
	_, err := c.Get(context.Background(), "/x", nil, nil)
	if CodeOf(err) != CodeServerError || calls.Load() != 4 {
		t.Fatalf("err = %v after %d calls", err, calls.Load())
	}
}

func TestDoesNotRetryOrFollow(t *testing.T) {
	cases := map[int]string{
		http.StatusUnauthorized: CodeUnauthorized,
		http.StatusForbidden:    CodeForbidden,
		http.StatusNotFound:     CodeNotFound,
		http.StatusFound:        CodeRedirect,
		422:                     CodeBadResponse,
	}
	for status, code := range cases {
		var calls atomic.Int32
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			calls.Add(1)
			if status == http.StatusFound {
				w.Header().Set("Location", "http://169.254.169.254/")
			}
			w.WriteHeader(status)
		}))
		c, _ := client(t, srv.URL, Options{})
		_, err := c.Get(context.Background(), "/x", nil, nil)
		if CodeOf(err) != code || calls.Load() != 1 {
			t.Errorf("%d: err = %v, calls = %d", status, err, calls.Load())
		}
		srv.Close()
	}
}

func TestCapsTheBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(strings.Repeat("x", 2048)))
	}))
	defer srv.Close()
	c, _ := client(t, srv.URL, Options{MaxBody: 1024})
	if _, err := c.Get(context.Background(), "/", nil, nil); CodeOf(err) != CodeTooLarge {
		t.Fatalf("err = %v", err)
	}
}

type fakeResolver map[string][]netip.Addr

func (f fakeResolver) LookupNetIP(_ context.Context, _, host string) ([]netip.Addr, error) {
	return f[host], nil
}

func TestRefusesMetadataAndLinkLocal(t *testing.T) {
	r := fakeResolver{
		"meta.test":  {netip.MustParseAddr("169.254.169.254")},
		"mixed.test": {netip.MustParseAddr("127.0.0.1"), netip.MustParseAddr("fe80::1")},
		"aws6.test":  {netip.MustParseAddr("fd00:ec2::254")},
	}
	for host := range r {
		c, rec := client(t, "http://"+host+":8000", Options{Resolver: r})
		_, err := c.Get(context.Background(), "/api/admin", nil, nil)
		if CodeOf(err) != CodeForbiddenAddress || len(rec.waits) != 0 {
			t.Errorf("%s: err = %v, retries = %d", host, err, len(rec.waits))
		}
	}
	// Literal addresses skip DNS and are checked the same way.
	for _, base := range []string{"http://169.254.169.254", "http://[::ffff:169.254.169.254]:80", "http://0.0.0.0:8000",
		"http://[fe80::1]:8000", "http://224.0.0.1"} {
		c, _ := client(t, base, Options{Resolver: r})
		if _, err := c.Get(context.Background(), "/api/admin", nil, nil); CodeOf(err) != CodeForbiddenAddress {
			t.Errorf("%s: err = %v", base, err)
		}
	}
	for _, a := range []string{"127.0.0.1", "10.0.0.5", "192.168.1.10", "::1"} {
		if !allowed(netip.MustParseAddr(a)) {
			t.Errorf("%s must be allowed: the old panel can be local", a)
		}
	}
}

func TestTLSPinning(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("{}"))
	}))
	defer srv.Close()
	sum := sha256.Sum256(srv.Certificate().Raw)
	pin := hex.EncodeToString(sum[:])

	c, _ := client(t, srv.URL, Options{})
	if _, err := c.Get(context.Background(), "/", nil, nil); CodeOf(err) != CodeTLSUntrusted {
		t.Fatalf("self-signed without a pin: %v", err)
	}
	c, _ = client(t, srv.URL, Options{PinSHA256: Fingerprint(pin)})
	if _, err := c.Get(context.Background(), "/", nil, nil); err != nil {
		t.Fatalf("pinned: %v", err)
	}
	c, _ = client(t, srv.URL, Options{PinSHA256: strings.Repeat("ab", 32)})
	if _, err := c.Get(context.Background(), "/", nil, nil); CodeOf(err) != CodeTLS {
		t.Fatalf("wrong pin: %v", err)
	}
	cert, err := FetchCertificate(context.Background(), srv.URL, nil)
	if err != nil || cert.SHA256 != pin || cert.Trusted {
		t.Fatalf("FetchCertificate = %+v, %v", cert, err)
	}
}

func TestClockSkew(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Date", time.Now().Add(3*time.Minute).UTC().Format(http.TimeFormat))
		_, _ = w.Write([]byte("{}"))
	}))
	defer srv.Close()
	c, _ := client(t, srv.URL, Options{})
	if _, err := c.Get(context.Background(), "/", nil, nil); err != nil {
		t.Fatal(err)
	}
	if s := c.Skew(); s < 2*time.Minute || s > 4*time.Minute {
		t.Fatalf("skew = %v", s)
	}
}

func TestParseBase(t *testing.T) {
	for _, bad := range []string{"ftp://x", "http://", "https://u:p@x", "http://x/?a=1", "x.com"} {
		if _, err := ParseBase(bad); err == nil {
			t.Errorf("%q accepted", bad)
		}
	}
	u, err := ParseBase(" https://panel.example:8443/prefix/ ")
	if err != nil || u.String() != "https://panel.example:8443/prefix" {
		t.Fatalf("%v %v", u, err)
	}
}

func TestBackoffStaysInItsWindow(t *testing.T) {
	for attempt := 0; attempt < 10; attempt++ {
		if d := backoff(attempt, 0); d <= 0 || d > backoffCap {
			t.Fatalf("attempt %d: %v", attempt, d)
		}
	}
	if backoff(0, time.Hour) != maxWait {
		t.Fatal("Retry-After must be capped")
	}
	if retryAfter("Wed, 21 Oct 2015 07:28:00 GMT") != 0 || retryAfter("3") != 3*time.Second {
		t.Fatal("Retry-After parsing")
	}
}
