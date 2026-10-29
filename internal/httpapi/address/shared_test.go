package address

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/PdYrust/karvan/internal/settings"
)

var sharedTLS = settings.TLS{Mode: settings.TLSDomain, DomainID: 3, Shared: true}

// sharedSites is what the bridge holds now.
func (f *fixture) sharedSites() []Shared {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]Shared(nil), f.shared...)
}

// viaBridge is the answer of the one shared site to an HTTPS request, with
// its HSTS header.
func (f *fixture) viaBridge(want Site) string {
	f.t.Helper()
	s := f.sharedSites()
	if len(s) != 1 || s[0].Site != want {
		f.t.Fatalf("shared %+v, want %+v", s, want)
	}
	w := httptest.NewRecorder()
	s[0].Handler.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "https://panel.example.com/"+want.BasePath+"/", nil))
	return w.Body.String() + " " + w.Header().Get("Strict-Transport-Security")
}

// holdPort is a port another listener (the bridge) holds.
func holdPort(t *testing.T) int {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ln.Close() })
	return ln.Addr().(*net.TCPAddr).Port
}

// D12: a shared address opens no listener of its own; the bridge gets it
// next to the old address until it is confirmed, and a later change of
// port goes back to a listener of the panel's own.
func TestSharedChangeGoesToTheBridge(t *testing.T) {
	f := newFixture(t, 0)
	bridgePort := holdPort(t)
	p, err := f.srv.Start(Change{Port: bridgePort, TLS: &sharedTLS})
	if err != nil {
		t.Fatalf("Start = %v", err)
	}
	next := Site{Port: bridgePort, BasePath: oldPath, TLS: sharedTLS}
	if want := fmt.Sprintf("%s via %d/%s %s", oldPath, bridgePort, oldPath, hsts); f.viaBridge(next) != want {
		t.Fatalf("via the bridge: %s", f.viaBridge(next))
	}
	if got := f.get(f.port, oldPath); got != fmt.Sprintf("%s via %d/%s", oldPath, f.port, oldPath) {
		t.Fatalf("old address: %s", got)
	}
	if _, err := f.srv.Confirm(context.Background(), p.Site); err != nil {
		t.Fatal(err)
	}
	f.confirmed(next)
	if got := f.get(f.port, oldPath); got != "refused" {
		t.Fatalf("old port after the confirm: %s", got)
	}
	f.viaBridge(next)

	own := freePort(t)
	p, err = f.srv.Start(Change{Port: own})
	if err != nil || p.TLS.Shared || p.Port != own {
		t.Fatalf("port change from the shared address = %+v, %v", p, err)
	}
	f.viaBridge(next)
	if err := f.stop(); err != nil {
		t.Fatal(err)
	}
	if s := f.sharedSites(); len(s) != 0 {
		t.Fatalf("shared after the shutdown: %+v", s)
	}
}

// A shared change that is not confirmed in time leaves the bridge.
func TestUnconfirmedSharedChangeGoesAway(t *testing.T) {
	f := newFixture(t, 100*time.Millisecond)
	if _, err := f.srv.Start(Change{Port: holdPort(t), TLS: &sharedTLS}); err != nil {
		t.Fatal(err)
	}
	if len(f.sharedSites()) != 1 {
		t.Fatalf("shared %+v", f.sharedSites())
	}
	deadline := time.Now().Add(5 * time.Second)
	for f.srv.Status().Pending != nil && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if s := f.sharedSites(); len(s) != 0 || f.srv.Status().Pending != nil {
		t.Fatalf("after the TTL: shared %+v, status %+v", s, f.srv.Status())
	}
	if got := f.get(f.port, oldPath); got != fmt.Sprintf("%s via %d/%s", oldPath, f.port, oldPath) {
		t.Fatalf("old address: %s", got)
	}
}

// A shared current address is served with no listener: Serve takes nil.
func TestSharedCurrentServesWithoutAListener(t *testing.T) {
	var got []Shared
	srv := &Server{
		Bind:    "127.0.0.1",
		Handler: func(string) (http.Handler, error) { return http.NotFoundHandler(), nil },
		HTTP:    func(h http.Handler) *http.Server { return &http.Server{Handler: h, ReadHeaderTimeout: time.Second} },
		Share:   func(s []Shared) { got = s },
		Log:     slog.New(slog.DiscardHandler),
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	current := Site{Port: 443, BasePath: oldPath, TLS: sharedTLS}
	go func() { done <- srv.Serve(ctx, current, nil) }()
	for srv.Status().Current != current {
		time.Sleep(5 * time.Millisecond)
	}
	srv.mu.Lock()
	n, ports := len(got), len(srv.ports)
	srv.mu.Unlock()
	cancel()
	if err := <-done; err != nil || n != 1 || ports != 0 || got != nil {
		t.Fatalf("Serve = %v, shared %d, listeners %d, after the shutdown %v", err, n, ports, got)
	}
}
