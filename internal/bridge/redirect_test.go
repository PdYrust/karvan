package bridge

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"slices"
	"strconv"
	"strings"
	"testing"
)

// redirectFor is a plain redirect rule for old.example.com on the env's port.
func (e *env) redirectFor(name, from, match, to string) Redirect {
	return Redirect{Name: name, Domains: []string{"old.example.com"}, Port: e.port, FromPath: from, Match: match, ToURL: to}
}

func (e *env) saveRedirect(r Redirect) RedirectView {
	e.t.Helper()
	v, err := e.b.SaveRedirect(context.Background(), r)
	if err != nil {
		e.t.Fatalf("save redirect %s: %v", r.Name, err)
	}
	return v
}

// send requests path on old.example.com with method.
func (e *env) send(method, path string) *httptest.ResponseRecorder {
	e.t.Helper()
	r := httptest.NewRequest(method, "http://old.example.com"+path, nil)
	r.RemoteAddr = "198.51.100.7:40000"
	w := httptest.NewRecorder()
	e.b.Handler(e.port).ServeHTTP(w, r)
	return w
}

// Every field is checked against the redirect limits, and the stored form is clean.
func TestRedirectFieldsAreChecked(t *testing.T) {
	e := newEnv(t)
	long := strings.Repeat("x", 1000)
	for _, c := range []struct {
		edit  func(r *Redirect)
		field string
	}{
		{func(r *Redirect) { r.Name = " " }, "name"},
		{func(r *Redirect) { r.Name = strings.Repeat("n", 65) }, "name"},
		{func(r *Redirect) { r.Name = "a\x00b" }, "name"},
		{func(r *Redirect) { r.Domains = []string{"a..b"} }, "domains"},
		{func(r *Redirect) { r.Port = 0 }, "port"},
		{func(r *Redirect) { r.Port = 65536 }, "port"},
		{func(r *Redirect) { r.FromPath = "" }, "fromPath"},
		{func(r *Redirect) { r.FromPath = "/" }, "fromPath"},
		{func(r *Redirect) { r.FromPath = "//evil.example.com" }, "fromPath"},
		{func(r *Redirect) { r.FromPath = "/a/../b" }, "fromPath"},
		{func(r *Redirect) { r.FromPath = "/./a" }, "fromPath"},
		{func(r *Redirect) { r.FromPath = "/a/" }, "fromPath"},
		{func(r *Redirect) { r.FromPath = "/a%2fb" }, "fromPath"},
		{func(r *Redirect) { r.FromPath = "/a b" }, "fromPath"},
		{func(r *Redirect) { r.FromPath = "/a/b/c/d/e/f/g/h/i" }, "fromPath"},
		{func(r *Redirect) { r.FromPath = "/" + long + "x" }, "fromPath"},
		{func(r *Redirect) { r.FromPath = "/" + long + "/" + long + "/" + long }, "fromPath"},
		{func(r *Redirect) { r.Match = "glob" }, "match"},
		{func(r *Redirect) { r.ToURL = "javascript:alert(1)" }, "toUrl"},
		{func(r *Redirect) { r.ToURL = "//evil.example.com/x" }, "toUrl"},
		{func(r *Redirect) { r.ToURL = "/relative" }, "toUrl"},
		{func(r *Redirect) { r.ToURL = "ftp://new.example.com/" }, "toUrl"},
		{func(r *Redirect) { r.ToURL = "https://user@new.example.com/" }, "toUrl"},
		{func(r *Redirect) { r.ToURL = "https://new.example.com/?a=1" }, "toUrl"},
		{func(r *Redirect) { r.ToURL = "https://new.example.com/?" }, "toUrl"},
		{func(r *Redirect) { r.ToURL = "https://new.example.com/#top" }, "toUrl"},
		{func(r *Redirect) { r.ToURL = "https:///no-host" }, "toUrl"},
		{func(r *Redirect) { r.ToURL = `https://new.example.com\@evil.example.com/` }, "toUrl"},
		{func(r *Redirect) { r.ToURL = "https://evil.example.com%40new.example.com/" }, "toUrl"},
		{func(r *Redirect) { r.ToURL = "https://*.example.com/" }, "toUrl"},
		{func(r *Redirect) { r.ToURL = "https://new.example.com:0/" }, "toUrl"},
		{func(r *Redirect) { r.ToURL = "https://new.example.com:/" }, "toUrl"},
		{func(r *Redirect) { r.ToURL = "https://new.example.com/" + strings.Repeat("p", 2048) }, "toUrl"},
	} {
		r := e.redirectFor("r", "/go", MatchPrefix, "https://new.example.com/")
		c.edit(&r)
		_, err := e.b.SaveRedirect(t.Context(), r)
		var invalid *RedirectInvalidError
		if !errors.As(err, &invalid) || invalid.Field != c.field || invalid.Reason != "" {
			t.Errorf("%+v: %v, want field %s", r, err, c.field)
		}
	}
	v := e.saveRedirect(Redirect{Name: " go ", Domains: []string{"Old.Example.com."}, Port: e.port, FromPath: "go/a", Match: MatchExact,
		ToURL: "HTTPS://New.Example.COM:8443/x%2Fy"})
	if v.Name != "go" || v.Domains[0] != "old.example.com" || v.FromPath != "/go/a" || v.ToURL != "https://new.example.com:8443/x%2Fy" ||
		v.Certificate != nil || !slices.Equal(v.Warnings, []string{WarnNoCertificate}) {
		t.Fatalf("stored %+v", v)
	}
	var n int
	if err := e.b.DB.QueryRow("SELECT count(*) FROM domains WHERE name = 'old.example.com'").Scan(&n); err != nil || n != 1 {
		t.Fatalf("domain row: %d %v", n, err)
	}
}

// A target that a redirect rule of Karvan answers is a loop, the rule
// itself included; no lookup is made to decide it.
func TestRedirectLoopIsRefused(t *testing.T) {
	e := newEnv(t)
	e.saveRedirect(e.redirectFor("other", "/away", MatchPrefix, "https://new.example.com/"))
	port := ":" + strconv.Itoa(e.port)
	for _, to := range []string{
		"http://old.example.com" + port + "/go",
		"http://OLD.example.com" + port + "/go/x/y",
		"http://old.example.com" + port + "/away/z",
	} {
		_, err := e.b.SaveRedirect(t.Context(), e.redirectFor("go", "/go", MatchPrefix, to))
		var invalid *RedirectInvalidError
		if !errors.As(err, &invalid) || invalid.Field != "toUrl" || invalid.Reason != ReasonLoop {
			t.Errorf("%s: %v, want a loop", to, err)
		}
	}
	for _, to := range []string{
		"http://old.example.com" + port + "/sub/x", // the panel rule answers it
		"http://old.example.com" + port + "/gone",
		"https://old.example.com/go", // port 443
		"http://other.example.com" + port + "/go",
	} {
		if _, err := e.b.SaveRedirect(t.Context(), e.redirectFor("ok", "/go", MatchExact, to)); err != nil {
			t.Errorf("%s: %v", to, err)
		}
		if err := e.b.DB.QueryRow("DELETE FROM link_redirects WHERE name = 'ok' RETURNING id").Scan(new(int64)); err != nil {
			t.Fatal(err)
		}
	}
}

// Conflicts go through the router's check: Karvan's own port, a name,
// a port served the other way, and the same requests.
func TestRedirectConflicts(t *testing.T) {
	e := newEnv(t)
	first := e.saveRedirect(e.redirectFor("first", "/go", MatchPrefix, "https://new.example.com/"))
	certPEM, keyPEM := selfSigned(t, "t.example.com")
	e.certify("t.example.com", certPEM, keyPEM)
	for _, c := range []struct {
		r    Redirect
		want error
	}{
		{e.redirectFor("first", "/other", MatchPrefix, "https://new.example.com/"), ErrNameTaken},
		{e.redirectFor("same", "/go", MatchPrefix, "https://new.example.com/b"), ErrOverlap},
		{e.redirectFor("panel", "/sub", MatchPrefix, "https://new.example.com/b"), ErrOverlap},
		{func() Redirect {
			r := e.redirectFor("tls", "/t", MatchExact, "https://new.example.com/")
			r.Domains = []string{"t.example.com"} // HTTPS by default: its domain has a certificate
			return r
		}(),
			ErrPortMixed},
	} {
		if _, err := e.b.SaveRedirect(t.Context(), c.r); !errors.Is(err, c.want) {
			t.Errorf("%s: %v, want %v", c.r.Name, err, c.want)
		}
	}
	e.b.st.panelPorts = func() []int { return []int{e.port} }
	if _, err := e.b.SaveRedirect(t.Context(), e.redirectFor("own", "/own", MatchExact, "https://new.example.com/")); !errors.Is(err, ErrPortTaken) {
		t.Fatalf("Karvan's port: %v", err)
	}
	first.Name = "renamed"
	if _, err := e.b.SaveRedirect(t.Context(), first.Redirect); !errors.Is(err, ErrPortTaken) {
		t.Fatalf("a change on Karvan's port: %v", err)
	}
	e.b.st.panelPorts = nil
	if v, err := e.b.SaveRedirect(t.Context(), first.Redirect); err != nil || v.ID != first.ID || v.Name != "renamed" {
		t.Fatalf("change: %+v %v", v, err)
	}
	first.ID = 999
	if _, err := e.b.SaveRedirect(t.Context(), first.Redirect); !errors.Is(err, ErrNotFound) {
		t.Fatalf("unknown id: %v", err)
	}
}

// A redirect answers 302 to its stored target, with a checked tail on a
// prefix match, and nothing else; every bad path is the plain 404.
func TestRedirectServing(t *testing.T) {
	e := newEnv(t)
	e.saveRedirect(e.redirectFor("exact", "/exact", MatchExact, "https://new.example.com/landing"))
	e.saveRedirect(e.redirectFor("prefix", "/go", MatchPrefix, "https://new.example.com/base/"))
	eight := strings.Repeat("/s", 8)
	for path, want := range map[string]string{
		"/exact":                    "https://new.example.com/landing",
		"/exact?to=https://evil.io": "https://new.example.com/landing",
		"/go":                       "https://new.example.com/base/",
		"/go/a/b?x=1":               "https://new.example.com/base/a/b",
		"/go/a.b~c=d-e_f":           "https://new.example.com/base/a.b~c=d-e_f",
		"/go" + eight:               "https://new.example.com/base" + eight,
	} {
		w := e.send(http.MethodGet, path)
		if w.Code != http.StatusFound || w.Header().Get("Location") != want || w.Header().Get("Cache-Control") != "no-store" ||
			w.Body.Len() != 0 {
			t.Errorf("%s: %d %q %q %q", path, w.Code, w.Header().Get("Location"), w.Header().Get("Cache-Control"), w.Body.String())
		}
	}
	for _, path := range []string{
		"/exact/more", "/exactly", "/go/", "/go//evil.example.com", `/go/\evil.example.com`, "/go/%2f%2fevil.example.com",
		"/go/%2e%2e/x", "/go/../x", "/go/a/./b", "/go/@evil.example.com", "/go/https:/evil.example.com", "/go/a%40b",
		"/go" + eight + "/s", "/go/" + strings.Repeat("x", 1001), "/gone", "//evil.example.com",
	} {
		if w := e.send(http.MethodGet, path); w.Code != http.StatusNotFound || w.Header().Get("Location") != "" {
			t.Errorf("%s: %d %q", path, w.Code, w.Header().Get("Location"))
		}
	}
	if w := e.send(http.MethodHead, "/go/a"); w.Code != http.StatusFound || w.Header().Get("Location") != "https://new.example.com/base/a" ||
		w.Body.Len() != 0 {
		t.Fatalf("HEAD: %d %q", w.Code, w.Header().Get("Location"))
	}
	if w := e.send(http.MethodPost, "/go/a"); w.Code != http.StatusMethodNotAllowed || w.Header().Get("Location") != "" {
		t.Fatalf("POST: %d", w.Code)
	}
}

// The Location never takes its scheme or host from the request: random
// tails and hosts all land on the stored host or get 404.
func TestRedirectNeverLeavesItsHost(t *testing.T) {
	e := newEnv(t)
	all := e.redirectFor("all", "/go", MatchPrefix, "https://new.example.com/b")
	all.Domains = nil
	e.saveRedirect(all)
	for _, tail := range []string{"/x", "/evil.example.com", "/http:", "/a=b", "/~", "/-", "/..a", "/a..", "/.a"} {
		for _, host := range []string{"old.example.com", "evil.example.com", "evil.example.com:443", "[::1]"} {
			r := httptest.NewRequest(http.MethodGet, "http://placeholder/go"+tail, nil)
			r.Host = host
			w := httptest.NewRecorder()
			e.b.Handler(e.port).ServeHTTP(w, r)
			loc := w.Header().Get("Location")
			if w.Code == http.StatusFound && loc != "https://new.example.com/b"+tail || w.Code != http.StatusFound && loc != "" {
				t.Errorf("%s %s: %d %q", host, tail, w.Code, loc)
			}
		}
	}
}

// A single link redirect (a prefix on one old link's path) wins over the
// batch rule on the same host for that one link only.
func TestSingleLinkRedirectWinsForOneLink(t *testing.T) {
	e := newEnv(t, "ali", "reza")
	ali, reza := e.oldToken("ali", e.clock.now()), e.oldToken("reza", e.clock.now())
	e.saveRedirect(e.redirectFor("ali", "/sub/"+ali, MatchPrefix, "https://elsewhere.example.com/ali"))
	w := e.send(http.MethodGet, "/sub/"+ali+"/clash")
	if w.Code != http.StatusFound || w.Header().Get("Location") != "https://elsewhere.example.com/ali/clash" {
		t.Fatalf("the single link: %d %q", w.Code, w.Header().Get("Location"))
	}
	if code, body := e.get("/sub/"+reza, "198.51.100.7"); code != http.StatusOK || !strings.Contains(body, "new_reza") {
		t.Fatalf("the batch rule's other link: %d %q", code, body)
	}
}

// The redirect code path opens nothing: no destination, no old panel, no
// dial and no lookup, on save or on a request.
func TestRedirectMakesNoOutboundRequest(t *testing.T) {
	e := newEnv(t)
	fail := func(what string) error { t.Errorf("outbound %s", what); return errors.New("no outbound") }
	e.b.OpenDest = func(context.Context, string) (Destination, error) { return nil, fail("destination") }
	e.b.OpenOld = func(Site) (OldPanel, error) { return nil, fail("old panel") }
	dial := func(context.Context, string, string) (net.Conn, error) { return nil, fail("dial") }
	transport, resolver := http.DefaultTransport, net.DefaultResolver
	http.DefaultTransport = &http.Transport{DialContext: dial}
	net.DefaultResolver = &net.Resolver{PreferGo: true, Dial: dial}
	t.Cleanup(func() { http.DefaultTransport, net.DefaultResolver = transport, resolver })
	e.saveRedirect(e.redirectFor("go", "/go", MatchPrefix, "https://unresolvable.invalid/x"))
	for _, path := range []string{"/go", "/go/a", "/go/../a", "/go/%2f"} {
		e.send(http.MethodGet, path)
		e.send(http.MethodHead, path)
	}
}

// A redirect with a certificate listens with TLS on its port and serves
// that certificate; deleting it stops the route.
func TestRedirectOverTLS(t *testing.T) {
	e := newEnv(t)
	ctx := context.Background()
	t.Cleanup(e.b.stopAll)
	certPEM, keyPEM := selfSigned(t, "old.example.com")
	c := e.certify("old.example.com", certPEM, keyPEM).Certificate
	e.addDomain("other.example.com")
	r := e.redirectFor("tls", "/go", MatchPrefix, "https://new.example.com/")
	on := true
	r.Port, r.Domains, r.HTTPS = freePort(t), []string{"old.example.com", "other.example.com"}, &on
	var invalid *RedirectInvalidError
	if _, err := e.b.SaveRedirect(ctx, r); !errors.As(err, &invalid) || invalid.Field != "domains" || invalid.Reason != ReasonNoCert ||
		!slices.Equal(invalid.Names, []string{"other.example.com"}) {
		t.Fatalf("https with a domain without a certificate: %v", err)
	}
	r.Domains, r.HTTPS = []string{"old.example.com"}, nil
	e.redirectServesItsDomainCertificate(r, c.ID, certPEM)
}

// redirectServesItsDomainCertificate saves r, checks it serves certPEM
// (certificate id) over TLS, then deletes it and checks the route goes.
func (e *env) redirectServesItsDomainCertificate(r Redirect, id int64, certPEM string) {
	t := e.t
	ctx := context.Background()
	v := e.saveRedirect(r)
	if v.Certificate == nil || v.Certificate.ID != id || slices.Contains(v.Warnings, WarnDomain) || !*v.HTTPS {
		t.Fatalf("view %+v", v)
	}
	block, _ := pem.Decode([]byte(certPEM))
	sum := sha256.Sum256(block.Bytes)
	if got := handshake(t, r.Port, "old.example.com"); got != hex.EncodeToString(sum[:]) {
		t.Fatalf("handshake %s", got)
	}
	list, err := e.b.Redirects(ctx)
	if err != nil || len(list) != 1 || list[0].ID != v.ID {
		t.Fatalf("list %+v %v", list, err)
	}
	if err := e.b.DeleteRedirect(ctx, v.ID); err != nil {
		t.Fatal(err)
	}
	if got := handshake(t, r.Port, "old.example.com"); got != "error" {
		t.Fatalf("after delete: %s", got)
	}
	if err := e.b.DeleteRedirect(ctx, v.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("second delete: %v", err)
	}
}
