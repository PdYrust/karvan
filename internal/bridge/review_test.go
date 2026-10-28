package bridge

import (
	"context"
	"errors"
	"net/http"
	"path/filepath"
	"strings"
	"testing"

	"github.com/PdYrust/karvan/internal/panel"
	kstore "github.com/PdYrust/karvan/internal/store"
)

// A ledger read that fails for a moment answers 503 and keeps the proof,
// which may never be made again once the old panel is gone.
func TestPassingFaultKeepsAVerifiedToken(t *testing.T) {
	e := newEnv(t, "ali")
	tok := e.oldToken("ali", e.clock.now())
	if code, _ := e.get("/sub/"+tok, "198.51.100.7"); code != http.StatusOK {
		t.Fatal(code)
	}
	broken, err := kstore.Open(context.Background(), filepath.Join(t.TempDir(), "other"))
	if err != nil {
		t.Fatal(err)
	}
	_ = broken.Close()
	good := e.b.st.db
	e.b.st.db = broken
	if code, _ := e.get("/sub/"+tok, "198.51.100.7"); code != http.StatusServiceUnavailable {
		t.Fatalf("ledger down: %d", code)
	}
	e.b.st.db = good
	if e.tokens() != 1 {
		t.Fatal("the proof was dropped")
	}
}

// A token proven on one site of an old panel counts on its other sites.
func TestProofCountsOnEverySiteOfTheOldPanel(t *testing.T) {
	e := newEnv(t, "ali")
	tok := e.oldToken("ali", e.clock.now())
	if code, _ := e.get("/sub/"+tok, "198.51.100.7"); code != http.StatusOK {
		t.Fatal(code)
	}
	other := e.site
	other.ID, other.Name, other.Port = 0, "plain", 2
	saved, err := e.b.st.Save(context.Background(), other)
	if err != nil {
		t.Fatal(err)
	}
	e.b.mu.Lock()
	e.b.load(append(e.b.sites, saved), nil)
	e.b.mu.Unlock()
	calls := e.oldCalls()
	e.port = 2
	if code, _ := e.get("/sub/"+tok, "198.51.100.7"); code != http.StatusOK || e.oldCalls() != calls {
		t.Fatalf("on the sibling site: %d, old calls %d -> %d", code, calls, e.oldCalls())
	}
}

// Forged tokens naming a real user, each with a new signature, pass the
// negative cache; the per-address limit stops them before any database.
func TestForgeriesAreLimitedBeforeTheDatabase(t *testing.T) {
	e := newEnv(t, "ali")
	tok := e.oldToken("ali", e.clock.now())
	var codes []int
	reads := 0
	for _, c := range "ABCDEFG" {
		e.dest.mu.Lock()
		before := e.dest.reads
		e.dest.mu.Unlock()
		code, _ := e.get("/sub/"+tok[:len(tok)-1]+string(c), "203.0.113.9")
		codes = append(codes, code)
		e.dest.mu.Lock()
		if len(codes) > 3 {
			reads += e.dest.reads - before
		}
		e.dest.mu.Unlock()
	}
	if codes[3] != http.StatusServiceUnavailable || reads != 0 {
		t.Fatalf("codes %v, destination reads after the limit %d", codes, reads)
	}
}

func TestLazyChecksNeedThePinnedAddress(t *testing.T) {
	s := Site{Name: "s", PanelID: 1, SourceKind: panel.Marzban, SourceURL: "https://old.example.com", OldPath: "sub", Port: 443, Lazy: true}
	if err := s.normalize(); !errors.Is(err, ErrInvalidSite) {
		t.Fatalf("no address: %v", err)
	}
	s.SourceIP = "203.0.113.4"
	if err := s.normalize(); err != nil {
		t.Fatal(err)
	}
}

func TestUploadRefusesPlainHTTPWithoutTheOptIn(t *testing.T) {
	e := newEnv(t, "ali")
	e.b.sites[0].PlainHTTPOK = false
	if _, err := e.b.Upload(context.Background(), e.site.ID, strings.NewReader(e.oldToken("ali", e.clock.now()))); !errors.Is(err, ErrPlainHTTP) {
		t.Fatalf("err %v", err)
	}
}

// With SNI for no listed domain, the port's catch-all site answers.
func TestCatchAllSiteOwnsUnknownNames(t *testing.T) {
	b := &Bridge{}
	b.load([]Site{{ID: 1, Port: 443, Domains: []string{"a.example.com"}}, {ID: 2, Port: 443}, {ID: 3, Port: 8443}}, nil)
	for name, want := range map[string]int64{"a.example.com": 1, "b.example.com": 2, "": 2} {
		if r, ok := d53(b.routes[443], name); !ok || r.ref.ID != want {
			t.Errorf("%q: site %d", name, r.ref.ID)
		}
	}
	if _, ok := d53(b.routes[80], "x"); ok {
		t.Error("a port with no site answered")
	}
}

// Overlapping reloads leave each port in the mode its sites want.
func TestConcurrentReloads(t *testing.T) {
	e := newEnv(t)
	t.Cleanup(e.b.stopAll)
	s := e.site
	s.Port = freePort(t)
	done := make(chan struct{})
	for range 4 {
		go func() {
			_, _ = e.b.SaveSite(context.Background(), s)
			done <- struct{}{}
		}()
	}
	for range 4 {
		<-done
	}
	e.b.mu.Lock()
	defer e.b.mu.Unlock()
	if l := e.b.servers[s.Port]; l == nil || l.tls {
		t.Fatalf("listener %+v", l)
	}
}
