package bridge

import (
	"fmt"
	"math/rand/v2"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
)

// redirectAt is a redirect rule as the router sees it.
func redirectAt(id int64, port int, path string, exact bool, domains ...string) rule {
	return rule{ref: RuleRef{Kind: RuleRedirect, ID: id, Name: fmt.Sprint("r", id)}, domains: domains, port: port,
		path: path, exact: exact}
}

func siteAt(id int64, port int, path string, domains ...string) Site {
	return Site{ID: id, Name: fmt.Sprint("s", id), Port: port, OldPath: path, Domains: domains}
}

// Every precedence pair: host before path, an exact name before a
// wildcard before no domains, an exact path before a prefix, the longer
// prefix first. A panel rule's path is "/<old path>/".
func TestRouterPrecedence(t *testing.T) {
	tb := newTable([]Site{
		siteAt(1, 443, "sub", "a.example.com"),
		siteAt(2, 443, "sub", "*.example.com"),
		siteAt(3, 443, "sub"),
	}, []rule{
		redirectAt(10, 443, "sub/x", false, "*.example.com"),
		redirectAt(11, 443, "sub/x/y", true, "*.example.com"),
		redirectAt(12, 443, "go", false, "b.example.com"),
		redirectAt(13, 8443, "sub", false),
	})
	for _, c := range []struct {
		port       int
		host, path string
		want       string // kind and id of the winner, or "" for none
		tail       string
	}{
		{443, "a.example.com", "/sub/tok", "panel 1", "/tok"},
		{443, "a.example.com", "/sub/x/abc", "panel 1", "/x/abc"}, // the host ranks before the path
		{443, "b.example.com", "/sub/tok/clash", "panel 2", "/tok/clash"},
		{443, "b.example.com", "/sub/x/abc", "redirect 10", "/abc"},
		{443, "b.example.com", "/sub/x", "redirect 10", ""},
		{443, "b.example.com", "/sub/x/y", "redirect 11", ""},
		{443, "b.example.com", "/sub/x/y/z", "redirect 10", "/y/z"},
		{443, "c.d.example.com", "/sub/tok", "panel 3", "/tok"}, // a wildcard covers one label
		{443, "example.com", "/sub/tok", "panel 3", "/tok"},
		{443, "other.org", "/sub/x/abc", "panel 3", "/x/abc"},
		{443, "b.example.com", "/go", "redirect 12", ""},
		{443, "b.example.com", "/go/a", "redirect 12", "/a"},
		{443, "b.example.com", "/gone", "", ""},
		{443, "a.example.com", "/sub", "", ""}, // a panel rule needs the slash after its path
		{443, "a.example.com", "/subx/tok", "", ""},
		{8443, "other.org", "/sub", "redirect 13", ""},
		{8443, "other.org", "/sub/z", "redirect 13", "/z"},
		{80, "a.example.com", "/sub/tok", "", ""},
	} {
		r, tail, ok := tb.match(c.port, c.host, c.path)
		got := ""
		if ok {
			got = fmt.Sprint(r.ref.Kind, " ", r.ref.ID)
		}
		if got != c.want || tail != c.tail {
			t.Errorf("%d %s%s: %q tail %q, want %q tail %q", c.port, c.host, c.path, got, tail, c.want, c.tail)
		}
		if ok && (r.panel == nil) != (r.ref.Kind == RuleRedirect) {
			t.Errorf("%s%s: route %+v", c.host, c.path, r)
		}
	}
}

// The winner answers alone: a malformed token or tail on it is 404 and
// never reaches the rule with no domains on the same path.
func TestMalformedTokenNeverFallsThrough(t *testing.T) {
	e := newEnv(t, "ali")
	all := e.site
	all.ID, all.Name, all.Domains = 0, "all", nil
	saved, err := e.b.st.Save(t.Context(), all)
	if err != nil {
		t.Fatalf("no domains next to a domain on one path: %v", err)
	}
	e.b.mu.Lock()
	e.b.load([]Site{e.site, saved}, nil)
	e.b.mu.Unlock()
	tok := e.oldToken("ali", e.clock.now())
	for host, want := range map[string]int64{"old.example.com": e.site.ID, "other.example.com": saved.ID} {
		q, ok := e.b.route(e.port, httptest.NewRequest(http.MethodGet, "http://"+host+"/sub/"+tok+"/clash", nil))
		if !ok || q.site.ID != want || q.token != tok || q.rest != "/clash" {
			t.Errorf("%s: %v %+v", host, ok, q)
		}
	}
	for _, path := range []string{"/sub/" + tok + "/a/b", "/sub/bad!token", "/sub/" + tok + "/" + strings.Repeat("x", 40)} {
		if q, ok := e.b.route(e.port, httptest.NewRequest(http.MethodGet, "http://old.example.com"+path, nil)); ok {
			t.Errorf("%q fell through to %s", path, q.site.Name)
		}
	}
}

// candidateRule is a random rule for the property test.
func candidateRule(rnd *rand.Rand, id int64) rule {
	patterns := []string{"a.example.com", "b.example.com", "*.example.com", "*.a.example.com", "example.com", "10.0.0.1"}
	paths := []string{"sub", "sub/x", "sub/x/y", "go"}
	var domains []string
	for range rnd.IntN(3) {
		domains = append(domains, patterns[rnd.IntN(len(patterns))])
	}
	slices.Sort(domains)
	domains = slices.Compact(domains)
	kind, exact := RulePanel, false
	path := paths[rnd.IntN(len(paths))]
	if rnd.IntN(2) == 0 {
		kind, exact = RuleRedirect, rnd.IntN(2) == 0
	} else {
		path = "sub"
		if rnd.IntN(3) == 0 {
			path = "go"
		}
	}
	return rule{ref: RuleRef{Kind: kind, ID: id, Name: fmt.Sprint(kind, id)}, domains: domains, port: 443 + rnd.IntN(2),
		path: path, exact: exact}
}

// After the save check no request meets two rules of equal rank, so it
// reaches at most one rule.
func TestNoRequestMeetsTwoRulesOfEqualRank(t *testing.T) {
	rnd := rand.New(rand.NewPCG(18, 52))
	hosts := []string{"a.example.com", "b.example.com", "x.a.example.com", "example.com", "c.example.com", "10.0.0.1", "other.org"}
	paths := []string{"/sub", "/sub/", "/sub/t", "/sub/x", "/sub/x/t", "/sub/x/y", "/sub/x/y/z", "/go", "/go/t", "/gone"}
	refused := 0
	for range 300 {
		tb := table{}
		var kept []rule
		for i := range 14 {
			c := candidateRule(rnd, int64(i+1))
			if slices.ContainsFunc(kept, func(o rule) bool { return c.conflict(o) != nil }) {
				refused++
				continue
			}
			kept = append(kept, c)
			tb[c.port] = append(tb[c.port], route{rule: c})
		}
		for _, port := range []int{443, 444} {
			for _, h := range hosts {
				for _, p := range paths {
					if n := topRanked(tb[port], h, p); n > 1 {
						t.Fatalf("%d %s%s meets %d rules of equal rank in %+v", port, h, p, n, tb[port])
					}
				}
			}
		}
	}
	if refused == 0 {
		t.Fatal("the check refused nothing, so the test proves nothing")
	}
}

// topRanked counts the routes that match with the best rank.
func topRanked(routes []route, host, path string) int {
	bh, bp, n := hostNone, 0, 0
	for _, r := range routes {
		h, p, _, ok := r.rank(host, path)
		switch {
		case !ok:
		case h > bh || h == bh && p > bp:
			bh, bp, n = h, p, 1
		case h == bh && p == bp:
			n++
		}
	}
	return n
}

// todayOverlaps is the overlap rule saves had before the precedence rules: one port, one
// old path and any shared host, or either site with no domains.
func todayOverlaps(s, o Site) bool {
	if s.Port != o.Port || s.OldPath != o.OldPath {
		return false
	}
	if len(s.Domains) == 0 || len(o.Domains) == 0 {
		return true
	}
	for _, a := range s.Domains {
		for _, b := range o.Domains {
			if a == b || matches(a, strings.TrimPrefix(b, "*.")) || matches(b, strings.TrimPrefix(a, "*.")) {
				return true
			}
		}
	}
	return false
}

// todayRoute is how a request picked its site before the precedence rules: the first site
// in id order on the port whose old path and domains fit.
func todayRoute(sites []Site, port int, host, path string) (int64, bool) {
	for _, s := range sites {
		if strings.HasPrefix(path, "/"+s.OldPath+"/") && s.Port == port && s.covers(host) {
			return s.ID, true
		}
	}
	return 0, false
}

// Sets of sites that today's save check allowed route exactly as before.
func TestTodaysSitesRouteAsBefore(t *testing.T) {
	rnd := rand.New(rand.NewPCG(4, 1))
	patterns := []string{"a.example.com", "b.example.com", "*.example.com", "*.b.example.com", "example.org"}
	hosts := []string{"a.example.com", "b.example.com", "x.b.example.com", "c.example.com", "example.org", "other.net"}
	for range 300 {
		var sites []Site
		for i := range 8 {
			s := siteAt(int64(i+1), 1+rnd.IntN(2), []string{"sub", "s2"}[rnd.IntN(2)])
			for range rnd.IntN(3) {
				s.Domains = append(s.Domains, patterns[rnd.IntN(len(patterns))])
			}
			if !slices.ContainsFunc(sites, func(o Site) bool { return todayOverlaps(s, o) }) {
				sites = append(sites, s)
			}
		}
		tb := newTable(sites, nil)
		for _, port := range []int{1, 2} {
			for _, h := range hosts {
				for _, p := range []string{"/sub/tok", "/s2/tok/clash", "/sub", "/other/tok"} {
					want, wok := todayRoute(sites, port, h, p)
					r, _, ok := tb.match(port, h, p)
					if ok != wok || ok && r.ref.ID != want {
						t.Fatalf("%d %s%s: %v %d, before %v %d (%+v)", port, h, p, ok, r.ref.ID, wok, want, sites)
					}
				}
			}
		}
	}
}
