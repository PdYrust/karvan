package bridge

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/PdYrust/karvan/internal/panel/subtoken"
)

// One address gets 3 first-time checks per 10 s; then 503 with
// Retry-After, and the old panel is not asked.
func TestFloodFromOneAddress(t *testing.T) {
	names := []string{"u1", "u2", "u3", "u4"}
	e := newEnv(t, names...)
	for i, n := range names {
		code, _ := e.get("/sub/"+e.oldToken(n, e.clock.now()), "203.0.113.5")
		if want := map[bool]int{true: http.StatusOK, false: http.StatusServiceUnavailable}[i < 3]; code != want {
			t.Fatalf("check %d: %d, want %d", i, code, want)
		}
	}
	if e.oldCalls() != 3 {
		t.Fatalf("old panel asked %d times", e.oldCalls())
	}
	e.clock.add(ipWindow)
	if code, _ := e.get("/sub/"+e.oldToken("u4", e.clock.now()), "203.0.113.5"); code != http.StatusOK {
		t.Fatalf("after the window: %d", code)
	}
}

// IPv6 clients count per /64.
func TestFloodFromOneIPv6Block(t *testing.T) {
	e := newEnv(t, "u1", "u2", "u3", "u4")
	for i, n := range []string{"u1", "u2", "u3", "u4"} {
		ip := fmt.Sprintf("[2001:db8:1:2::%x]", i+1)
		code, _ := e.get("/sub/"+e.oldToken(n, e.clock.now()), ip)
		if i == 3 && code != http.StatusServiceUnavailable {
			t.Fatalf("4th address in the /64: %d", code)
		}
	}
}

// Many addresses share the site budget of 10 checks per second.
func TestFloodFromManyAddresses(t *testing.T) {
	var names []string
	for i := range 12 {
		names = append(names, fmt.Sprintf("u%02d", i))
	}
	e := newEnv(t, names...)
	refused := 0
	for i, n := range names {
		if code, _ := e.get("/sub/"+e.oldToken(n, e.clock.now()), fmt.Sprintf("203.0.113.%d", i+1)); code == http.StatusServiceUnavailable {
			refused++
		}
	}
	if refused != 2 || e.oldCalls() != 10 {
		t.Fatalf("refused %d, old calls %d", refused, e.oldCalls())
	}
}

// An old panel that is down, overloaded or rate limiting gives 503, and
// nothing is cached: the next refresh checks again.
func TestOldPanelTroubleIsNotCached(t *testing.T) {
	for _, status := range []int{http.StatusTooManyRequests, http.StatusBadGateway, http.StatusForbidden} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			e := newEnv(t, "ali")
			tok := e.oldToken("ali", e.clock.now())
			e.old.FailSub(status)
			code, _ := e.get("/sub/"+tok, "198.51.100.7")
			if code != http.StatusServiceUnavailable {
				t.Fatalf("%d", code)
			}
			e.old.FailSub(0)
			if code, _ := e.get("/sub/"+tok, "198.51.100.7"); code != http.StatusOK {
				t.Fatalf("after recovery: %d", code)
			}
		})
	}
	e := newEnv(t, "ali")
	tok := e.oldToken("ali", e.clock.now())
	e.old.Close() // gone for good
	if code, _ := e.get("/sub/"+tok, "198.51.100.7"); code != http.StatusServiceUnavailable {
		t.Fatalf("old panel down: %d", code)
	}
}

// A day after the check, the next request checks again in the
// background; a revoke on the old panel then drops the token.
func TestDailyRecheckDropsRevokedTokens(t *testing.T) {
	e := newEnv(t, "ali", "bob")
	ali, bob := e.oldToken("ali", e.clock.now()), e.oldToken("bob", e.clock.now())
	e.get("/sub/"+ali, "198.51.100.7")
	e.get("/sub/"+bob, "198.51.100.7")
	e.old.Subscribe(ali, "") // revoked on the old panel
	e.clock.add(recheckAfter + time.Minute)
	for _, tok := range []string{ali, bob} {
		if code, _ := e.get("/sub/"+tok, "198.51.100.7"); code != http.StatusOK {
			t.Fatalf("served while the recheck runs: %d", code)
		}
	}
	e.b.Wait()
	if e.tokens() != 1 {
		t.Fatalf("tokens after recheck: %d", e.tokens())
	}
	e.old.Close() // an old panel that is gone keeps verified tokens
	e.clock.add(recheckAfter + time.Minute)
	e.get("/sub/"+bob, "198.51.100.7")
	e.b.Wait()
	if code, _ := e.get("/sub/"+bob, "198.51.100.7"); code != http.StatusOK || e.tokens() != 1 {
		t.Fatalf("with the old panel gone: %d, tokens %d", code, e.tokens())
	}
}

func TestLazyChecksCanBeOff(t *testing.T) {
	e := newEnv(t, "ali")
	e.b.sites[0].Lazy = false
	if code, _ := e.get("/sub/"+e.oldToken("ali", e.clock.now()), "198.51.100.7"); code != http.StatusNotFound || e.oldCalls() != 0 {
		t.Fatalf("%d, old calls %d", code, e.oldCalls())
	}
	e.b.sites[0].Lazy, e.b.sites[0].PlainHTTPOK = true, false // http:// old panel without the opt-in
	if code, _ := e.get("/sub/"+e.oldToken("ali", e.clock.now()), "198.51.100.7"); code != http.StatusNotFound || e.oldCalls() != 0 {
		t.Fatalf("plain http: %d, old calls %d", code, e.oldCalls())
	}
}

func TestRouting(t *testing.T) {
	e := newEnv(t, "ali")
	tok := e.oldToken("ali", e.clock.now())
	for _, c := range []struct {
		method, url string
		want        int
	}{
		{http.MethodPost, "http://old.example.com/sub/" + tok, http.StatusMethodNotAllowed},
		{http.MethodGet, "http://other.example.com/sub/" + tok, http.StatusNotFound},
		{http.MethodGet, "http://old.example.com/api/sub/" + tok, http.StatusNotFound},
		{http.MethodGet, "http://old.example.com/sub/" + tok + "/a/b", http.StatusNotFound},
		{http.MethodGet, "http://old.example.com/sub/short", http.StatusNotFound},
		{http.MethodGet, "http://old.example.com/sub/" + tok + "%2F..", http.StatusNotFound},
		{http.MethodHead, "http://OLD.example.com:8443/sub/" + tok, http.StatusOK},
	} {
		r := httptest.NewRequest(c.method, c.url, nil)
		w := httptest.NewRecorder()
		e.b.Handler(e.port).ServeHTTP(w, r)
		if w.Code != c.want {
			t.Errorf("%s %s: %d, want %d", c.method, c.url, w.Code, c.want)
		}
	}
}

func TestUploadStoresVerifiedTokens(t *testing.T) {
	e := newEnv(t, "ali", "bob", "cyrus")
	ali, bob := e.oldToken("ali", e.clock.now()), e.oldToken("bob", e.clock.now())
	stranger := e.oldToken("nobody", e.clock.now())
	unknown := subtoken.Marzban("ghost", oldSecret, e.clock.now()) // well formed, never issued
	list := "username,link\nali,https://old.example.com/sub/" + ali + "\nbob;" + bob + "\n" + stranger + "\n" +
		"https://old.example.com/sub/" + unknown + "/clash\nhttps://elsewhere.example.com/x/" + ali + "\n" + ali
	res, err := e.b.Upload(context.Background(), e.site.ID, strings.NewReader(list))
	if err != nil {
		t.Fatal(err)
	}
	if res.Found != 4 || res.Stored != 2 || res.NoUser != 2 {
		t.Fatalf("result %+v", res)
	}
	again, _ := e.b.Upload(context.Background(), e.site.ID, strings.NewReader(ali))
	if again.Known != 1 {
		t.Fatalf("again %+v", again)
	}
	calls := e.oldCalls()
	if code, _ := e.get("/sub/"+bob, "198.51.100.7"); code != http.StatusOK || e.oldCalls() != calls {
		t.Fatal("an uploaded token still needed a check")
	}
}
