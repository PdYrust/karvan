package address

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/PdYrust/karvan/internal/settings"
)

const oldPath = "currentPath123"

type fixture struct {
	t      *testing.T
	srv    *Server
	port   int
	saved  []Site
	shared []Shared // the last sites handed to the bridge
	mu     sync.Mutex
	closed atomic.Int32
	done   chan error
	cancel context.CancelFunc
}

// newFixture serves oldPath on a free local port. Each hidden path answers
// with its own name and the address the request came through.
func newFixture(t *testing.T, ttl time.Duration) *fixture {
	t.Helper()
	f := &fixture{t: t, done: make(chan error, 1)}
	f.srv = &Server{
		Bind: "127.0.0.1",
		TTL:  ttl,
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
		OnShutdown: func() { f.closed.Add(1) },
		Log:        slog.New(slog.DiscardHandler),
	}
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	f.port = ln.Addr().(*net.TCPAddr).Port
	ctx, cancel := context.WithCancel(context.Background())
	f.cancel = cancel
	go func() { f.done <- f.srv.Serve(ctx, Site{Port: f.port, BasePath: oldPath}, ln) }()
	t.Cleanup(func() { _ = f.stop() })
	for f.srv.Status().Current.Port != f.port {
		time.Sleep(5 * time.Millisecond)
	}
	return f
}

func (f *fixture) stop() error {
	f.cancel()
	select {
	case err := <-f.done:
		f.done <- err
		return err
	case <-time.After(15 * time.Second):
		f.t.Fatal("Serve did not return")
		return nil
	}
}

// get returns the body at port and path, or "refused" when nothing listens.
func (f *fixture) get(port int, path string) string {
	f.t.Helper()
	c := &http.Client{Timeout: 5 * time.Second, Transport: &http.Transport{DisableKeepAlives: true}}
	res, err := c.Get(fmt.Sprintf("http://127.0.0.1:%d/%s/", port, path))
	if err != nil {
		return "refused"
	}
	defer res.Body.Close()
	b, _ := io.ReadAll(res.Body)
	if res.StatusCode != http.StatusOK {
		return fmt.Sprint(res.StatusCode)
	}
	return string(b)
}

func (f *fixture) savedSites() []Site {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]Site(nil), f.saved...)
}

func freePort(t *testing.T) int {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	return ln.Addr().(*net.TCPAddr).Port
}

func TestPathChangeServesBothPathsUntilConfirmed(t *testing.T) {
	f := newFixture(t, 0)
	p, err := f.srv.Start(Change{NewPath: true})
	if err != nil || p.Port != f.port || p.BasePath == oldPath || time.Until(p.Expires) < TTL-time.Minute {
		t.Fatalf("Start = %+v, %v", p, err)
	}
	old := Site{Port: f.port, BasePath: oldPath}
	if got := f.get(f.port, oldPath); got != fmt.Sprintf("%s via %d/%s", oldPath, f.port, oldPath) {
		t.Fatalf("old path: %s", got)
	}
	if got := f.get(f.port, p.BasePath); got != fmt.Sprintf("%s via %d/%s", p.BasePath, f.port, p.BasePath) {
		t.Fatalf("new path: %s", got)
	}
	if _, err := f.srv.Confirm(context.Background(), old); !errors.Is(err, ErrNotNew) || len(f.savedSites()) != 0 {
		t.Fatalf("confirm from the old address: %v, saved %v", err, f.savedSites())
	}

	before, err := f.srv.Confirm(context.Background(), p.Site)
	if err != nil || before != old {
		t.Fatalf("Confirm = %+v, %v", before, err)
	}
	f.confirmed(p.Site)
	if got := f.get(f.port, oldPath); got != "404" {
		t.Fatalf("old path after the confirm: %s", got)
	}
}

// confirmed checks that site was saved once and is now the only address.
func (f *fixture) confirmed(site Site) {
	f.t.Helper()
	if s := f.savedSites(); len(s) != 1 || s[0] != site {
		f.t.Fatalf("saved %v", s)
	}
	if st := f.srv.Status(); st.Current != site || st.Pending != nil {
		f.t.Fatalf("status %+v", st)
	}
	if _, err := f.srv.Confirm(context.Background(), site); !errors.Is(err, ErrNoPending) {
		f.t.Fatalf("second confirm: %v", err)
	}
}

func TestPortChangeOpensASecondListener(t *testing.T) {
	f := newFixture(t, 0)
	port := freePort(t)
	p, err := f.srv.Start(Change{Port: port})
	if err != nil || p.Site != (Site{Port: port, BasePath: oldPath}) {
		t.Fatalf("Start = %+v, %v", p, err)
	}
	if got := f.get(f.port, oldPath); got != fmt.Sprintf("%s via %d/%s", oldPath, f.port, oldPath) {
		t.Fatalf("old port: %s", got)
	}
	if got := f.get(port, oldPath); got != fmt.Sprintf("%s via %d/%s", oldPath, port, oldPath) {
		t.Fatalf("new port: %s", got)
	}
	if _, err := f.srv.Confirm(context.Background(), p.Site); err != nil {
		t.Fatal(err)
	}
	if got := f.get(f.port, oldPath); got != "refused" {
		t.Fatalf("old port after the confirm: %s", got)
	}
	if got := f.get(port, oldPath); got == "refused" {
		t.Fatal("the new port stopped")
	}
}

func TestUnconfirmedChangeGoesAway(t *testing.T) {
	f := newFixture(t, 100*time.Millisecond)
	port := freePort(t)
	p, err := f.srv.Start(Change{NewPath: true, Port: port})
	if err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(5 * time.Second)
	for f.srv.Status().Pending != nil {
		if time.Now().After(deadline) {
			t.Fatal("the change did not expire")
		}
		time.Sleep(20 * time.Millisecond)
	}
	if got := f.get(port, p.BasePath); got != "refused" {
		t.Fatalf("expired address: %s", got)
	}
	if _, err := f.srv.Confirm(context.Background(), p.Site); !errors.Is(err, ErrNoPending) {
		t.Fatalf("confirm after expiry: %v", err)
	}
	if got := f.get(f.port, oldPath); got == "refused" || got == "404" || len(f.savedSites()) != 0 {
		t.Fatalf("old address %s, saved %v", got, f.savedSites())
	}
}

func TestStartRefusesNoChangeAndABusyPort(t *testing.T) {
	f := newFixture(t, 0)
	for _, c := range []Change{{}, {Port: f.port}} {
		if _, err := f.srv.Start(c); !errors.Is(err, ErrNoChange) {
			t.Fatalf("Start(%+v) = %v", c, err)
		}
	}
	busy, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer busy.Close()
	if _, err := f.srv.Start(Change{Port: busy.Addr().(*net.TCPAddr).Port}); !errors.Is(err, ErrPortUnavailable) {
		t.Fatalf("busy port: %v", err)
	}
	if f.srv.Status().Pending != nil {
		t.Fatal("a refused change is waiting")
	}
}

func TestNewStartReplacesTheWaitingOne(t *testing.T) {
	f := newFixture(t, 0)
	port := freePort(t)
	first, err := f.srv.Start(Change{Port: port})
	if err != nil {
		t.Fatal(err)
	}
	second, err := f.srv.Start(Change{NewPath: true, Port: port})
	if err != nil {
		t.Fatalf("the same port again: %v", err)
	}
	if got := f.get(port, first.BasePath); got != "404" {
		t.Fatalf("the replaced path: %s", got)
	}
	if got := f.get(port, second.BasePath); got == "refused" || got == "404" {
		t.Fatalf("the new one: %s", got)
	}
	if _, err := f.srv.Confirm(context.Background(), first.Site); !errors.Is(err, ErrNotNew) {
		t.Fatalf("confirm the replaced one: %v", err)
	}
}

func TestRandomPortAndShutdown(t *testing.T) {
	f := newFixture(t, 0)
	p, err := f.srv.Start(Change{Port: -1})
	if err != nil || p.Port == f.port || p.Port < settings.MinRandomPort || p.Port > settings.MaxRandomPort || p.BasePath != oldPath {
		t.Fatalf("Start = %+v, %v", p, err)
	}
	if err := f.stop(); err != nil {
		t.Fatalf("Serve = %v", err)
	}
	if f.closed.Load() != 1 {
		t.Fatalf("OnShutdown ran %d times", f.closed.Load())
	}
	if f.get(f.port, oldPath) != "refused" || f.get(p.Port, oldPath) != "refused" {
		t.Fatal("a port still answers after the shutdown")
	}
	if _, err := f.srv.Start(Change{NewPath: true}); err == nil {
		t.Fatal("Start after the shutdown")
	}
}

func TestFailedSaveKeepsBothAddresses(t *testing.T) {
	f := newFixture(t, 0)
	f.srv.Save = func(context.Context, Site) error { return errors.New("disk full") }
	p, err := f.srv.Start(Change{NewPath: true})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.srv.Confirm(context.Background(), p.Site); err == nil {
		t.Fatal("Confirm ignored the failed save")
	}
	if st := f.srv.Status(); st.Pending == nil || st.Current.BasePath != oldPath {
		t.Fatalf("status %+v", st)
	}
	if f.get(f.port, oldPath) == "404" || f.get(f.port, p.BasePath) == "404" {
		t.Fatal("an address stopped after a failed save")
	}
}
