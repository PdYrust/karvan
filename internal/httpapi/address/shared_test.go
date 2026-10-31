package address

import (
	"context"
	"crypto/tls"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// sharedSites is what the bridge holds now.
func (f *fixture) sharedSites() []Shared {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]Shared(nil), f.shared...)
}

// sharedFixture serves oldPath on a shared bridge port, as an install
// that had one before 0.5.8 does: no listener of its own.
func sharedFixture(t *testing.T) (*fixture, Site) {
	t.Helper()
	f := &fixture{t: t, done: make(chan error, 1)}
	cert := testCert(t)
	f.srv = &Server{
		Bind:        "127.0.0.1",
		Certificate: func(context.Context, string) (*tls.Certificate, error) { return cert, nil },
		Handler: func(base string) (http.Handler, error) {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				via := From(r.Context())
				fmt.Fprintf(w, "%s via %d/%s", base, via.Port, via.BasePath)
			}), nil
		},
		Save: func(_ context.Context, s Site) error {
			f.mu.Lock()
			defer f.mu.Unlock()
			f.saved = append(f.saved, s)
			return nil
		},
		HTTP: func(h http.Handler) *http.Server { return &http.Server{Handler: h, ReadHeaderTimeout: time.Second} },
		Share: func(sites []Shared) {
			f.mu.Lock()
			defer f.mu.Unlock()
			f.shared = sites
		},
		Log: slog.New(slog.DiscardHandler),
	}
	current := Site{Port: holdPort(t), BasePath: oldPath, Shared: true}
	ctx, cancel := context.WithCancel(context.Background())
	f.cancel = cancel
	go func() { f.done <- f.srv.Serve(ctx, current, nil) }()
	t.Cleanup(func() { _ = f.stop() })
	for f.srv.Status().Current != current {
		time.Sleep(5 * time.Millisecond)
	}
	return f, current
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

// A shared address stays with the bridge until a change of port gives the
// panel a listener of its own; a new path there stays shared.
func TestSharedAddressMovesToItsOwnPort(t *testing.T) {
	f, current := sharedFixture(t)
	if want := fmt.Sprintf("%s via %d/%s ", oldPath, current.Port, oldPath); f.viaBridge(current) != want {
		t.Fatalf("via the bridge: %s", f.viaBridge(current))
	}
	own := freePort(t)
	p, err := f.srv.Start(Change{Port: own})
	if err != nil || p.Shared || p.Port != own {
		t.Fatalf("port change from the shared address = %+v, %v", p, err)
	}
	f.viaBridge(current)
	if got := f.get(own, oldPath); got != fmt.Sprintf("%s via %d/%s", oldPath, own, oldPath) {
		t.Fatalf("the new port: %s", got)
	}
	if _, err := f.srv.Confirm(context.Background(), p.Site); err != nil {
		t.Fatal(err)
	}
	f.confirmed(p.Site)
	if s := f.sharedSites(); len(s) != 0 {
		t.Fatalf("shared after the confirm: %+v", s)
	}
}

// A new path on a shared port is shared too, next to the old one.
func TestNewPathOnASharedPortStaysShared(t *testing.T) {
	f, current := sharedFixture(t)
	p, err := f.srv.Start(Change{NewPath: true})
	if err != nil || !p.Shared || p.Port != current.Port {
		t.Fatalf("new path = %+v, %v", p, err)
	}
	if s := f.sharedSites(); len(s) != 2 {
		t.Fatalf("shared %+v", s)
	}
	if _, err := f.srv.Confirm(context.Background(), p.Site); err != nil {
		t.Fatal(err)
	}
	f.viaBridge(p.Site)
	if err := f.stop(); err != nil {
		t.Fatal(err)
	}
	if s := f.sharedSites(); len(s) != 0 {
		t.Fatalf("shared after the shutdown: %+v", s)
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
	current := Site{Port: 443, BasePath: oldPath, Shared: true}
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
