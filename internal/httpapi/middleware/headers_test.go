package middleware

import (
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"testing"
)

func TestSecurityHeaders(t *testing.T) {
	h := SecurityHeaders(http.HandlerFunc(ok))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	for k, want := range map[string]string{
		"Content-Security-Policy": ContentSecurityPolicy,
		"Referrer-Policy":         "no-referrer",
		"X-Frame-Options":         "DENY",
		"X-Content-Type-Options":  "nosniff",
	} {
		if got := rec.Header().Get(k); got != want {
			t.Errorf("%s = %q, want %q", k, got, want)
		}
	}
	if rec.Header().Get("Strict-Transport-Security") != "" {
		t.Error("HSTS sent over plain HTTP")
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.TLS = &tls.ConnectionState{}
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Header().Get("Strict-Transport-Security") == "" {
		t.Error("HSTS missing over HTTPS")
	}
}

func TestClientIP(t *testing.T) {
	trusted := []netip.Prefix{netip.MustParsePrefix("10.0.0.0/8")}
	cases := []struct {
		name, remote string
		xff          []string
		want         string
	}{
		{"direct", "203.0.113.5:4000", nil, "203.0.113.5"},
		{"untrusted peer ignores header", "203.0.113.5:4000", []string{"1.2.3.4"}, "203.0.113.5"},
		{"trusted proxy", "10.0.0.2:4000", []string{"198.51.100.9"}, "198.51.100.9"},
		{"rightmost untrusted wins", "10.0.0.2:4000", []string{"6.6.6.6, 198.51.100.9", "10.0.0.3"}, "198.51.100.9"},
		{"garbage header falls back", "10.0.0.2:4000", []string{"not-an-ip"}, "10.0.0.2"},
		{"only proxies falls back", "10.0.0.2:4000", []string{"10.0.0.7"}, "10.0.0.2"},
		{"ipv4-mapped ipv6", "[::ffff:203.0.113.5]:4000", nil, "203.0.113.5"},
		{"unparseable peer", "pipe", nil, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var got string
			h := ClientIP(trusted, http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) { got = IP(r.Context()) }))
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.RemoteAddr = tc.remote
			for _, v := range tc.xff {
				req.Header.Add("X-Forwarded-For", v)
			}
			h.ServeHTTP(httptest.NewRecorder(), req)
			if got != tc.want {
				t.Errorf("IP = %q, want %q", got, tc.want)
			}
		})
	}
}
