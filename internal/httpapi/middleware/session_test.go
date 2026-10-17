package middleware

import (
	"context"
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/PdYrust/karvan/internal/auth"
	"github.com/PdYrust/karvan/internal/store"
)

var cookies = Cookies{Path: "/" + base + "/"}

func newSessionStore(t *testing.T) (*auth.Sessions, string) {
	t.Helper()
	db, err := store.Open(context.Background(), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	if _, err := db.Exec(`INSERT INTO operators (id, username, password_hash, created_at, updated_at) VALUES (1, 'owner', 'x', 0, 0)`); err != nil {
		t.Fatal(err)
	}
	s := auth.NewSessions(db)
	token, _, err := s.Create(context.Background(), 1, "", "")
	if err != nil {
		t.Fatal(err)
	}
	return s, token
}

func withCookie(r *http.Request, name, value string) *http.Request {
	r.AddCookie(&http.Cookie{Name: name, Value: value})
	return r
}

func protected(s *auth.Sessions) http.Handler {
	return LoadSession(s, cookies, CSRF(cookies, RequireSession(http.HandlerFunc(ok))))
}

func TestRequireSession(t *testing.T) {
	s, token := newSessionStore(t)
	h := protected(s)

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("no cookie: status %d", rec.Code)
	}

	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, withCookie(httptest.NewRequest(http.MethodGet, "/", nil), "s", token))
	if rec.Code != http.StatusOK {
		t.Fatalf("valid cookie: status %d", rec.Code)
	}

	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, withCookie(httptest.NewRequest(http.MethodGet, "/", nil), "s", "stale"))
	if rec.Code != http.StatusUnauthorized || !strings.Contains(rec.Header().Get("Set-Cookie"), "Max-Age=0") {
		t.Fatalf("stale cookie: status %d, set-cookie %q", rec.Code, rec.Header().Get("Set-Cookie"))
	}
}

func post(token, csrf string, headers map[string]string) *http.Request {
	r := httptest.NewRequest(http.MethodPost, "http://panel.example/", nil)
	if token != "" {
		withCookie(r, "s", token)
	}
	if csrf != "" {
		r.Header.Set(CSRFHeader, csrf)
	}
	for k, v := range headers {
		r.Header.Set(k, v)
	}
	return r
}

func TestCSRFWithSession(t *testing.T) {
	s, token := newSessionStore(t)
	h := protected(s)
	same := map[string]string{"Sec-Fetch-Site": "same-origin"}
	cases := []struct {
		name string
		req  *http.Request
		want int
	}{
		{"token and same site", post(token, auth.CSRFToken(token), same), 200},
		{"origin fallback", post(token, auth.CSRFToken(token), map[string]string{"Origin": "http://panel.example"}), 200},
		{"cross site", post(token, auth.CSRFToken(token), map[string]string{"Sec-Fetch-Site": "cross-site"}), 403},
		{"foreign origin", post(token, auth.CSRFToken(token), map[string]string{"Origin": "http://evil.example"}), 403},
		{"no origin info", post(token, auth.CSRFToken(token), nil), 403},
		{"missing token", post(token, "", same), 403},
		{"token of another session", post(token, auth.CSRFToken("other"), same), 403},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, tc.req)
			if rec.Code != tc.want {
				t.Fatalf("status %d, want %d", rec.Code, tc.want)
			}
		})
	}
}

func TestCSRFBeforeSignIn(t *testing.T) {
	s, _ := newSessionStore(t)
	h := LoadSession(s, cookies, CSRF(cookies, http.HandlerFunc(ok)))
	same := map[string]string{"Sec-Fetch-Site": "same-origin"}

	good := withCookie(post("", "pre-token", same), "c", "pre-token")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, good)
	if rec.Code != 200 {
		t.Fatalf("matching pre-session token: %d", rec.Code)
	}
	for name, r := range map[string]*http.Request{
		"no cookie":      post("", "pre-token", same),
		"mismatch":       withCookie(post("", "other", same), "c", "pre-token"),
		"empty both":     withCookie(post("", "", same), "c", ""),
		"cross-site too": withCookie(post("", "pre-token", map[string]string{"Sec-Fetch-Site": "cross-site"}), "c", "pre-token"),
	} {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, r)
		if rec.Code != 403 {
			t.Errorf("%s: status %d, want 403", name, rec.Code)
		}
	}
}

func TestCookiesOverHTTPAndHTTPS(t *testing.T) {
	rec := httptest.NewRecorder()
	cookies.SetSession(rec, httptest.NewRequest(http.MethodGet, "/", nil), "tok")
	plain := rec.Header().Get("Set-Cookie")
	if !strings.HasPrefix(plain, "s=tok") || !strings.Contains(plain, "Path=/"+base+"/") ||
		!strings.Contains(plain, "HttpOnly") || !strings.Contains(plain, "SameSite=Strict") || strings.Contains(plain, "Secure") {
		t.Errorf("http cookie = %q", plain)
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.TLS = &tls.ConnectionState{}
	rec = httptest.NewRecorder()
	cookies.SetPreCSRF(rec, req, "pre")
	secure := rec.Header().Get("Set-Cookie")
	if !strings.HasPrefix(secure, "__Host-c=pre") || !strings.Contains(secure, "Path=/;") || !strings.Contains(secure, "Secure") {
		t.Errorf("https cookie = %q", secure)
	}
	if cookies.PreCSRF(withCookie(req, "__Host-c", "pre")) != "pre" {
		t.Error("PreCSRF did not read the __Host- cookie")
	}

	rec = httptest.NewRecorder()
	cookies.ClearPreCSRF(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if !strings.Contains(rec.Header().Get("Set-Cookie"), "Max-Age=0") {
		t.Errorf("clear cookie = %q", rec.Header().Get("Set-Cookie"))
	}
}

func TestBackgroundReadsDoNotKeepTheSessionAlive(t *testing.T) {
	db, err := store.Open(context.Background(), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	if _, err := db.Exec(`INSERT INTO operators (id, username, password_hash, created_at, updated_at) VALUES (1, 'owner', 'x', 0, 0)`); err != nil {
		t.Fatal(err)
	}
	s := auth.NewSessions(db)
	token, _, err := s.Create(context.Background(), 1, "", "")
	if err != nil {
		t.Fatal(err)
	}
	// Last seen ten minutes ago, so a real read would move it.
	if _, err := db.Exec("UPDATE sessions SET last_seen_at = last_seen_at - 600"); err != nil {
		t.Fatal(err)
	}
	seen := func() int64 {
		var at int64
		if err := db.QueryRow("SELECT last_seen_at FROM sessions").Scan(&at); err != nil {
			t.Fatal(err)
		}
		return at
	}
	before := seen()
	h := protected(s)
	r := withCookie(httptest.NewRequest(http.MethodGet, "/", nil), "s", token)
	r.Header.Set(BackgroundHeader, "1")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, r)
	if rec.Code != http.StatusOK || seen() != before {
		t.Fatalf("background read: status %d, last seen moved %d -> %d", rec.Code, before, seen())
	}
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, withCookie(httptest.NewRequest(http.MethodGet, "/", nil), "s", token))
	if rec.Code != http.StatusOK || seen() == before {
		t.Fatalf("operator read: status %d, last seen still %d", rec.Code, seen())
	}
}
