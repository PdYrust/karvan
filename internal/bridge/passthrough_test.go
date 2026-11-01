package bridge

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// sendWith requests path on the env's old host with the client headers a
// subscription app and a browser send.
func sendWith(e *env, path string) *httptest.ResponseRecorder {
	e.t.Helper()
	r := httptest.NewRequest(http.MethodGet, "http://old.example.com"+path, nil)
	r.RemoteAddr = "198.51.100.7:40000"
	r.Header.Set("User-Agent", "Hiddify/2.5")
	r.Header.Set("Accept", "*/*")
	r.Header.Set("Cookie", "session=karvan")
	r.Header.Set("Authorization", "Bearer karvan")
	w := httptest.NewRecorder()
	e.b.Handler(e.port).ServeHTTP(w, r)
	return w
}

// A token Karvan does not know goes to the destination panel, which
// answers with its own body and headers; the client's credentials stay
// behind and its app headers, tail and query go along.
func TestUnknownTokenGoesToTheDestination(t *testing.T) {
	e := newEnv(t, "ali")
	w := sendWith(e, "/sub/native-token-0123456789/clash?flag=1")
	if w.Code != http.StatusOK || w.Body.String() != "config /sub/native-token-0123456789/clash" || e.oldCalls() != 0 {
		t.Fatalf("%d %q, old calls %d", w.Code, w.Body.String(), e.oldCalls())
	}
	for _, h := range []string{"Subscription-Userinfo", "Profile-Title", "Content-Disposition"} {
		if w.Header().Get(h) == "" {
			t.Errorf("no %s in %v", h, w.Header())
		}
	}
	last := e.dest.last
	if last.URL.RawQuery != "flag=1" || last.Header.Get("User-Agent") != "Hiddify/2.5" || last.Header.Get("Accept") != "*/*" {
		t.Fatalf("the panel saw %s %v", last.URL, last.Header)
	}
	if last.Header.Get("Cookie") != "" || last.Header.Get("Authorization") != "" {
		t.Fatalf("credentials forwarded: %v", last.Header)
	}
}

// The destination's 404 for a token it does not know stays a 404.
func TestDestinationNotFoundStaysNotFound(t *testing.T) {
	e := newEnv(t, "ali")
	hits := e.hits()
	if w := sendWith(e, "/sub/unknown-token-0123456789"); w.Code != http.StatusNotFound || e.hits() != hits+1 {
		t.Fatalf("%d, panel hits %d -> %d", w.Code, hits, e.hits())
	}
}

// Only the subscription path goes to the panel.
func TestOtherPathsAreNotForwarded(t *testing.T) {
	e := newEnv(t, "ali")
	e.dest.path = "pg"
	hits := e.hits()
	for _, p := range []string{"/api/admin", "/dashboard/", "/native-token-0123456789", "/subx/native-token-0123456789",
		"/sub/native-token-0123456789/a/b", "/pg/", "/pg/native-token-0123456789/a/b", "/pg/..%2Fapi%2Fadmin1234567"} {
		if w := sendWith(e, p); w.Code != http.StatusNotFound {
			t.Errorf("%s: %d", p, w.Code)
		}
	}
	if e.hits() != hits {
		t.Fatalf("panel hits %d -> %d", hits, e.hits())
	}
}

// A link on the destination's own subscription path, when it differs
// from the rule's old path, works on the same host and port, with its
// tail and query.
func TestDestinationsOwnPathIsServed(t *testing.T) {
	e := newEnv(t, "ali")
	e.dest.path = "pg"
	w := sendWith(e, "/pg/native-token-0123456789/info?x=1")
	if w.Code != http.StatusOK || w.Body.String() != "config /pg/native-token-0123456789/info" || e.oldCalls() != 0 {
		t.Fatalf("%d %q, old calls %d", w.Code, w.Body.String(), e.oldCalls())
	}
	if last := e.dest.last; last.URL.RawQuery != "x=1" || last.Header.Get("Cookie") != "" {
		t.Fatalf("the panel saw %s %v", last.URL, last.Header)
	}
	if w := sendWith(e, "/pg/unknown-token-0123456789"); w.Code != http.StatusNotFound || e.oldCalls() != 0 {
		t.Fatalf("unknown on the panel's path: %d, old calls %d", w.Code, e.oldCalls())
	}
}

// A token naming a user in a held batch is never forwarded, on the old
// path or the destination's own.
func TestHeldTokenIsNotForwarded(t *testing.T) {
	e := newEnv(t, "ali")
	e.batch("inactive", "restoring", "ali")
	e.dest.path = "pg"
	hits := e.hits()
	tok := e.oldToken("ali", e.clock.now())
	for _, p := range []string{"/sub/" + tok, "/pg/" + tok} {
		if w := sendWith(e, p); w.Code != http.StatusNotFound {
			t.Errorf("%s: %d", p, w.Code)
		}
	}
	if e.hits() != hits || e.oldCalls() != 0 {
		t.Fatalf("panel hits %d -> %d, old calls %d", hits, e.hits(), e.oldCalls())
	}
}

// A token whose payload names no moved user goes to the destination and
// never to the old panel, even when the old panel knows it.
func TestNoCandidateNeverAsksTheOldPanel(t *testing.T) {
	e := newEnv(t, "ali")
	tok := e.oldToken("zed", e.clock.now()) // a user Karvan did not move
	hits := e.hits()
	for range 3 {
		if w := sendWith(e, "/sub/"+tok); w.Code != http.StatusNotFound {
			t.Fatal(w.Code)
		}
	}
	if e.oldCalls() != 0 || e.hits() != hits+3 {
		t.Fatalf("old calls %d, panel hits %d -> %d", e.oldCalls(), hits, e.hits())
	}
}

// A token the old panel refused is still passed to the destination, but
// the old panel is not asked again while the refusal is remembered.
func TestRefusedTokenIsAskedOnce(t *testing.T) {
	e := newEnv(t, "ali")
	tok := e.oldToken("ali", e.clock.now())
	forged := tok[:len(tok)-1] + flip(tok[len(tok)-1])
	hits := e.hits()
	for range 3 {
		if w := sendWith(e, "/sub/"+forged); w.Code != http.StatusNotFound {
			t.Fatal(w.Code)
		}
	}
	if e.oldCalls() != 1 || e.hits() != hits+3 {
		t.Fatalf("old calls %d, panel hits %d -> %d", e.oldCalls(), hits, e.hits())
	}
}
