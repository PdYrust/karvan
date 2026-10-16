package bridge

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// A first request is checked once against the old panel; later ones are
// served from the verified set. The panel sees the client's headers and
// a token minted for the moved user.
func TestOldLinkIsVerifiedOnceThenServed(t *testing.T) {
	e := newEnv(t, "ali")
	tok := e.oldToken("ali", e.clock.now().Add(-time.Minute))
	for i := range 3 {
		code, body := e.get("/sub/"+tok+"/clash", "198.51.100.7")
		if code != http.StatusOK || body != "config /sub/dest-100-new_ali/clash" {
			t.Fatalf("request %d: %d %q", i, code, body)
		}
	}
	if n := e.oldCalls(); n != 1 {
		t.Fatalf("old panel asked %d times, want 1", n)
	}
	last := e.dest.last
	if last.Host != "old.example.com" || last.Header.Get("User-Agent") != "v2rayNG/1.9" || last.Header.Get("X-Hwid") != "device-1" ||
		last.Header.Get("X-Forwarded-For") != "198.51.100.7" {
		t.Fatalf("headers the panel saw: host %q %v", last.Host, last.Header)
	}
}

func TestResponseHeadersPassExceptHopByHop(t *testing.T) {
	e := newEnv(t, "ali")
	tok := e.oldToken("ali", e.clock.now())
	r := httpGet(e, "/sub/"+tok)
	if r.Header().Get("Subscription-Userinfo") == "" || r.Header().Get("X-Secret-Hop") != "" || r.Header().Get("Connection") != "" {
		t.Fatalf("response headers: %v", r.Header())
	}
}

// A token the new panel itself knows goes back as the panel answered,
// with no call to the old panel.
func TestNativeTokenPassesThrough(t *testing.T) {
	e := newEnv(t, "ali")
	code, body := e.get("/sub/native-token-0123456789", "198.51.100.7")
	if code != http.StatusOK || !strings.Contains(body, "native-token") || e.oldCalls() != 0 {
		t.Fatalf("%d %q, old calls %d", code, body, e.oldCalls())
	}
}

// Owner rule: a revoke on the new panel closes the user's old links,
// including fresh links from the old panel.
func TestRevokeOnNewPanelClosesForGood(t *testing.T) {
	e := newEnv(t, "ali")
	tok := e.oldToken("ali", e.clock.now())
	if code, _ := e.get("/sub/"+tok, "198.51.100.7"); code != http.StatusOK {
		t.Fatal(code)
	}
	revoked := time.Now().Add(time.Hour)
	e.dest.set(100, func(r *Row) { r.SubRevokedAt = &revoked })
	if code, _ := e.get("/sub/"+tok, "198.51.100.7"); code != http.StatusNotFound {
		t.Fatalf("after revoke: %d", code)
	}
	if e.tokens() != 0 {
		t.Fatal("tokens kept after close")
	}
	e.dest.set(100, func(r *Row) { r.SubRevokedAt = nil }) // even if the revoke is undone
	fresh := e.oldToken("ali", e.clock.now().Add(time.Minute))
	calls := e.oldCalls()
	if code, _ := e.get("/sub/"+fresh, "198.51.100.8"); code != http.StatusNotFound || e.oldCalls() != calls {
		t.Fatalf("fresh old link: %d, old calls %d -> %d", code, calls, e.oldCalls())
	}
}

func TestSweepClosesDeletedAndRevoked(t *testing.T) {
	e := newEnv(t, "ali", "bob", "cyrus")
	e.dest.mu.Lock()
	delete(e.dest.rows, 100)
	e.dest.mu.Unlock()
	revoked := time.Now().Add(time.Hour)
	e.dest.set(101, func(r *Row) { r.SubRevokedAt = &revoked })
	n, err := e.b.Sweep(context.Background())
	if err != nil || n != 2 {
		t.Fatalf("closed %d, %v", n, err)
	}
	if n, _ := e.b.Sweep(context.Background()); n != 0 {
		t.Fatalf("second sweep closed %d", n)
	}
	tok := e.oldToken("cyrus", e.clock.now())
	if code, _ := e.get("/sub/"+tok, "198.51.100.7"); code != http.StatusOK {
		t.Fatalf("an open user: %d", code)
	}
}

// A user deleted and re-created under the same id on the new panel is a
// different user: the old links close.
func TestReplacedRowCloses(t *testing.T) {
	e := newEnv(t, "ali")
	e.dest.set(100, func(r *Row) { r.CreatedAt = r.CreatedAt.Add(time.Hour) })
	tok := e.oldToken("ali", e.clock.now())
	if code, _ := e.get("/sub/"+tok, "198.51.100.7"); code != http.StatusNotFound || e.oldCalls() != 0 {
		t.Fatalf("%d, old calls %d", code, e.oldCalls())
	}
}

// A token older than the user on the new panel fails the panel's own
// rule, without closing anything.
func TestTokenOlderThanUserIsRefused(t *testing.T) {
	e := newEnv(t, "ali")
	tok := e.oldToken("ali", created.Add(-time.Hour))
	if code, _ := e.get("/sub/"+tok, "198.51.100.7"); code != http.StatusNotFound {
		t.Fatal(code)
	}
	if closed, _ := e.b.st.closed(context.Background(), e.users["ali"].ID); closed {
		t.Fatal("an old token closed the user")
	}
}

// A new user on the old panel reusing a moved user's name has another
// created_at and credentials: refused.
func TestReusedNameOnOldPanelIsRefused(t *testing.T) {
	e := newEnv(t, "ali")
	e.old.Mutate(func(users *[]map[string]any) { (*users)[0]["created_at"] = "2026-09-01T00:00:00" })
	tok := e.oldToken("ali", e.clock.now())
	if code, _ := e.get("/sub/"+tok, "198.51.100.7"); code != http.StatusNotFound || e.tokens() != 0 {
		t.Fatalf("%d, tokens %d", code, e.tokens())
	}
}

// The old panel names another user for the token: refused, not cached.
func TestInfoForAnotherUserIsRefused(t *testing.T) {
	e := newEnv(t, "ali", "bob")
	tok := e.oldToken("ali", e.clock.now())
	e.old.Subscribe(tok, "bob")
	if code, _ := e.get("/sub/"+tok, "198.51.100.7"); code != http.StatusNotFound || e.tokens() != 0 {
		t.Fatalf("%d, tokens %d", code, e.tokens())
	}
}

// A forged token (any change to the signature) is unknown to the old
// panel: refused, and the 404 is remembered for an hour.
func TestForgedTokenIsRefusedAndRemembered(t *testing.T) {
	e := newEnv(t, "ali")
	tok := e.oldToken("ali", e.clock.now())
	forged := tok[:len(tok)-1] + flip(tok[len(tok)-1])
	for range 3 {
		if code, _ := e.get("/sub/"+forged, "198.51.100.7"); code != http.StatusNotFound {
			t.Fatal(code)
		}
		e.clock.add(11 * time.Second) // stay under the per-address limit
	}
	if n := e.oldCalls(); n != 1 {
		t.Fatalf("old panel asked %d times, want 1", n)
	}
	e.clock.add(time.Hour)
	e.get("/sub/"+forged, "198.51.100.7")
	if n := e.oldCalls(); n != 2 {
		t.Fatalf("after the negative cache expired: %d calls", n)
	}
}

func flip(c byte) string {
	if c == 'A' {
		return "B"
	}
	return "A"
}

func httpGet(e *env, path string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(http.MethodGet, "http://old.example.com"+path, nil)
	r.RemoteAddr = "198.51.100.7:40000"
	w := httptest.NewRecorder()
	e.b.Handler(e.port).ServeHTTP(w, r)
	return w
}
