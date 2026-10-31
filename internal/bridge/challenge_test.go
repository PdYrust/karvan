package bridge

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

const challengePath = "/.well-known/acme-challenge/"

// challengeEnv is a bridge whose challenge port is a free port.
func challengeEnv(t *testing.T) *env {
	t.Helper()
	e := newCertEnv(t)
	e.b.ChallengePort = freePort(t)
	t.Cleanup(e.b.stopAll)
	return e
}

// ask sends method host path to the bridge handler of the challenge port.
func (e *env) ask(method, host, path string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(method, "http://"+host+path, nil)
	r.RemoteAddr = "198.51.100.7:40000"
	w := httptest.NewRecorder()
	e.b.Handler(e.b.ChallengePort).ServeHTTP(w, r)
	return w
}

func (e *env) openChallenge(holder string) {
	e.t.Helper()
	if err := e.b.WantChallengePort(context.Background(), holder, true); err != nil {
		e.t.Fatalf("open challenge port: %v", err)
	}
}

// On a port opened only for challenges, the live token on its own
// host answers and everything else gets the plain 404, with no redirect.
func TestChallengeOnlyPortAnswersTheLiveTokenAlone(t *testing.T) {
	e := challengeEnv(t)
	e.openChallenge("order:1")
	e.b.AddChallenge("tok_EN-1", "a.example.com", "tok_EN-1.thumb")
	w := e.ask(http.MethodGet, "A.example.com:80", challengePath+"tok_EN-1")
	if w.Code != 200 || w.Body.String() != "tok_EN-1.thumb" || w.Header().Get("Cache-Control") != "no-store" ||
		!strings.HasPrefix(w.Header().Get("Content-Type"), "text/plain") {
		t.Fatalf("live token: %d %q %v", w.Code, w.Body, w.Header())
	}
	if w := e.ask(http.MethodHead, "a.example.com", challengePath+"tok_EN-1"); w.Code != 200 {
		t.Fatalf("HEAD: %d", w.Code)
	}
	for _, c := range []struct{ method, host, path string }{
		{"GET", "a.example.com", challengePath + "other"},
		{"GET", "b.example.com", challengePath + "tok_EN-1"},
		{"GET", "a.example.com.evil.test", challengePath + "tok_EN-1"},
		{"GET", "a.example.com", challengePath + "tok_EN-1/x"},
		{"GET", "a.example.com", challengePath + "tok_EN-1/"},
		{"GET", "a.example.com", challengePath + "tok%5FEN-1"},
		{"GET", "a.example.com", "/.well-known/acme-challenge/../acme-challenge/tok_EN-1"},
		{"GET", "a.example.com", "//.well-known/acme-challenge/tok_EN-1"},
		{"GET", "a.example.com", challengePath + "tok_EN-1?x=1"},
		{"POST", "a.example.com", challengePath + "tok_EN-1"},
		{"PUT", "a.example.com", challengePath + "tok_EN-1"},
		{"GET", "a.example.com", challengePath},
		{"GET", "a.example.com", "/.well-known/"},
		{"GET", "a.example.com", "/"},
		{"GET", "a.example.com", "/sub/x"},
	} {
		w := e.ask(c.method, c.host, c.path)
		if c.path == challengePath+"tok_EN-1?x=1" { // the query is not part of the path
			if w.Code != 200 {
				t.Errorf("%+v: %d", c, w.Code)
			}
			continue
		}
		if w.Code != 404 || w.Header().Get("Location") != "" || w.Body.String() != plain404(t) {
			t.Errorf("%+v: %d %q %v", c, w.Code, w.Body, w.Header())
		}
	}
	e.b.RemoveChallenge("tok_EN-1")
	if w := e.ask(http.MethodGet, "a.example.com", challengePath+"tok_EN-1"); w.Code != 404 {
		t.Fatalf("after the order: %d", w.Code)
	}
}

// plain404 is the bridge's own 404 body.
func plain404(t *testing.T) string {
	t.Helper()
	w := httptest.NewRecorder()
	answer(w, http.StatusNotFound)
	return w.Body.String()
}

// The listener is there only while an order holds it, and a real client
// gets the token from it.
func TestChallengePortListensOnlyWhileHeld(t *testing.T) {
	e := challengeEnv(t)
	addr := "127.0.0.1:" + strconv.Itoa(e.b.ChallengePort)
	if dialable(addr) {
		t.Fatal("open before any order")
	}
	e.openChallenge("order:1")
	e.openChallenge("order:2")
	e.b.AddChallenge("t1", "a.example.com", "t1.key")
	req, _ := http.NewRequest(http.MethodGet, "http://"+addr+challengePath+"t1", nil)
	req.Host = "a.example.com"
	resp, err := (&http.Client{Timeout: 5 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error {
		return errors.New("redirected")
	}}).Do(req)
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK || string(body) != "t1.key" {
		t.Fatalf("%d %q", resp.StatusCode, body)
	}
	if err := e.b.WantChallengePort(t.Context(), "order:1", false); err != nil || !dialable(addr) {
		t.Fatalf("one holder left: %v", err)
	}
	if err := e.b.WantChallengePort(t.Context(), "order:2", false); err != nil || dialable(addr) {
		t.Fatalf("closed after the last order: %v", err)
	}
}

func dialable(addr string) bool {
	c, err := net.DialTimeout("tcp", addr, time.Second)
	if err == nil {
		_ = c.Close()
	}
	return err == nil
}

// On a port that rules also use, everything but the live token reaches
// the router as today: a prefix redirect on /.well-known saved before the
// reserved check still answers, while an order validates.
func TestChallengeOnARulePortLeavesOtherRequestsToTheRouter(t *testing.T) {
	e := challengeEnv(t)
	e.port = e.b.ChallengePort
	if _, err := e.b.DB.Exec("UPDATE legacy_sites SET listen_port = ?", e.port); err != nil {
		t.Fatal(err)
	}
	if _, err := e.b.DB.Exec("INSERT INTO link_redirects (name, domains, listen_port, from_path, match, to_url, created_at, "+
		"updated_at) VALUES ('wk', '[\"old.example.com\"]', ?, '/.well-known', 'prefix', 'https://new.example.com/wk', 1, 1)",
		e.port); err != nil {
		t.Fatal(err)
	}
	if err := e.b.Reload(t.Context()); err != nil {
		t.Fatal(err)
	}
	e.openChallenge("order:1")
	e.b.AddChallenge("live", "old.example.com", "live.key")
	if w := e.ask("GET", "old.example.com", challengePath+"live"); w.Code != 200 || w.Body.String() != "live.key" {
		t.Fatalf("live token: %d %q", w.Code, w.Body)
	}
	for path, loc := range map[string]string{
		challengePath + "other": "https://new.example.com/wk/acme-challenge/other",
		"/.well-known/x":        "https://new.example.com/wk/x",
	} {
		if w := e.ask("GET", "old.example.com", path); w.Code != 302 || w.Header().Get("Location") != loc {
			t.Errorf("%s: %d %q", path, w.Code, w.Header().Get("Location"))
		}
	}
	if w := e.ask("GET", "other.example.com", challengePath+"live"); w.Code != 404 {
		t.Errorf("other host: %d", w.Code)
	}
	if w := e.ask("POST", "old.example.com", challengePath+"live"); w.Code != http.StatusMethodNotAllowed {
		t.Errorf("POST goes to the router: %d", w.Code)
	}
}

// httpsRedirect is a redirect on the challenge port for a domain that
// has a certificate, so it is saved as HTTPS.
func (e *env) httpsRedirect(name string) Redirect {
	c, k := longCert(e.t, name)
	e.addDomainWith(name, CertChoice{Kind: ChoicePasted, CertPEM: c, KeyPEM: k})
	return Redirect{Name: name, Domains: []string{name}, Port: e.b.ChallengePort, FromPath: "/go", Match: MatchPrefix,
		ToURL: "https://new.example.com/"}
}

// While an order or keepPort80 holds the challenge port open,
// no HTTPS rule is saved on it; WantChallengePort refuses a port an HTTPS
// rule uses.
func TestChallengePortAndHTTPSRulesExcludeEachOther(t *testing.T) {
	e := challengeEnv(t)
	for _, holder := range []string{"order:1", "keep"} {
		e.openChallenge(holder)
		if _, err := e.b.SaveRedirect(t.Context(), e.httpsRedirect("tls.example.com")); !errors.Is(err, ErrPortMixed) {
			t.Fatalf("%s: HTTPS redirect saved: %v", holder, err)
		}
		s := e.site
		s.ID, s.Name, s.Port, s.HTTPS = 0, "tls rule", e.b.ChallengePort, nil
		s.Domains = []string{"tls.example.com"}
		if _, err := e.b.SaveSite(t.Context(), s); !errors.Is(err, ErrPortMixed) {
			t.Fatalf("%s: HTTPS panel rule saved: %v", holder, err)
		}
		if err := e.b.WantChallengePort(t.Context(), holder, false); err != nil {
			t.Fatal(err)
		}
	}
	e.saveRedirect(e.httpsRedirect("tls.example.com"))
	if err := e.b.WantChallengePort(t.Context(), "order:1", true); !errors.Is(err, ErrPortTLS) {
		t.Fatalf("opened next to an HTTPS rule: %v", err)
	}
	if e.b.challengeHeld() {
		t.Fatal("a refused holder stays")
	}
}

// A port another program holds is refused as busy, and nothing is kept.
func TestChallengePortBusy(t *testing.T) {
	e := challengeEnv(t)
	ln, err := net.Listen("tcp", ":"+strconv.Itoa(e.b.ChallengePort))
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	if err := e.b.WantChallengePort(t.Context(), "order:1", true); !errors.Is(err, ErrPortBusy) {
		t.Fatalf("busy port: %v", err)
	}
	if _, listed := e.b.Status()[e.b.ChallengePort]; e.b.challengeHeld() || listed {
		t.Fatalf("held %v, status %v", e.b.challengeHeld(), e.b.Status())
	}
}

func TestChallengePortErrors(t *testing.T) {
	for _, c := range []struct {
		err  error
		want error
	}{
		{&net.OpError{Op: "listen", Err: os.NewSyscallError("bind", syscall.EACCES)}, ErrPortDenied},
		{&net.OpError{Op: "listen", Err: os.NewSyscallError("bind", syscall.EADDRINUSE)}, ErrPortBusy},
	} {
		if got := portError(80, c.err); !errors.Is(got, c.want) {
			t.Errorf("%v: %v", c.err, got)
		}
	}
	if got := portError(80, errors.New("other")); errors.Is(got, ErrPortBusy) || errors.Is(got, ErrPortDenied) {
		t.Errorf("other error: %v", got)
	}
}

// A redirect cannot take the challenge path on POST or on a PATCH
// that changes fromPath; a stored one stays editable.
func TestRedirectFromPathUnderTheChallengeIsReserved(t *testing.T) {
	e := newEnv(t)
	for _, from := range []string{"/.well-known/acme-challenge", ".well-known/acme-challenge/x", "/.well-known/acme-challenge/a/b"} {
		_, err := e.b.SaveRedirect(t.Context(), e.redirectFor("r", from, MatchExact, "https://new.example.com/"))
		var invalid *RedirectInvalidError
		if !errors.As(err, &invalid) || invalid.Field != "fromPath" || invalid.Reason != ReasonReserved {
			t.Errorf("%s: %v", from, err)
		}
	}
	e.saveRedirect(e.redirectFor("ok", "/.well-known/other", MatchExact, "https://new.example.com/"))
	res, err := e.b.DB.Exec("INSERT INTO link_redirects (name, domains, listen_port, from_path, match, to_url, created_at, " +
		"updated_at) VALUES ('old', '[]', 1, '/.well-known/acme-challenge/x', 'exact', 'https://new.example.com/', 1, 1)")
	if err != nil {
		t.Fatal(err)
	}
	id, _ := res.LastInsertId()
	d := e.redirectFor("old renamed", ".well-known/acme-challenge/x", MatchExact, "https://new.example.com/y")
	d.ID, d.Domains = id, []string{}
	e.saveRedirect(d)
	d.FromPath = "/.well-known/acme-challenge/z"
	if _, err := e.b.SaveRedirect(t.Context(), d); !errors.Is(err, ErrInvalidRedirect) {
		t.Fatalf("changed to another reserved path: %v", err)
	}
}
