package health

import (
	"context"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// fakeSystemd listens where NOTIFY_SOCKET would point and returns the
// messages it gets.
func fakeSystemd(t *testing.T) (string, <-chan string) {
	t.Helper()
	dir, err := os.MkdirTemp("", "kh") // short: socket paths are limited to 108 bytes
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	path := filepath.Join(dir, "notify")
	conn, err := net.ListenUnixgram("unixgram", &net.UnixAddr{Name: path, Net: "unixgram"})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	got := make(chan string, 16)
	go func() {
		b := make([]byte, 256)
		for {
			n, err := conn.Read(b)
			if err != nil {
				return
			}
			got <- string(b[:n])
		}
	}()
	return path, got
}

func next(t *testing.T, got <-chan string) string {
	t.Helper()
	select {
	case m := <-got:
		return m
	case <-time.After(5 * time.Second):
		t.Fatal("no message from the monitor")
		return ""
	}
}

func TestNotifySendsTheState(t *testing.T) {
	path, got := fakeSystemd(t)
	if err := Notify(path, "READY=1"); err != nil {
		t.Fatal(err)
	}
	if m := next(t, got); m != "READY=1" {
		t.Errorf("got %q, want READY=1", m)
	}
}

func TestNotifyWithoutSocketDoesNothing(t *testing.T) {
	if err := Notify("", "READY=1"); err != nil {
		t.Errorf("err = %v, want nil when not started by systemd", err)
	}
	if err := Notify(filepath.Join(t.TempDir(), "none"), "READY=1"); err == nil {
		t.Error("a missing socket gave no error")
	}
}

func TestProbeAnswersForHTTPAndHTTPS(t *testing.T) {
	for _, srv := range []*httptest.Server{
		httptest.NewServer(http.NotFoundHandler()),
		httptest.NewTLSServer(http.NotFoundHandler()),
	} {
		if err := Probe(context.Background(), srv.Listener.Addr().String()); err != nil {
			t.Errorf("%s: %v", srv.URL, err)
		}
		srv.Close()
	}
}

func TestProbeFailsOnAHungOrClosedListener(t *testing.T) {
	old := probeTimeout
	probeTimeout = 200 * time.Millisecond
	t.Cleanup(func() { probeTimeout = old })
	ln, err := net.Listen("tcp", "127.0.0.1:0") // never accepts
	if err != nil {
		t.Fatal(err)
	}
	addr := ln.Addr().String()
	if err := Probe(context.Background(), addr); err == nil || !strings.Contains(err.Error(), "no answer") {
		t.Errorf("hung: err = %v, want no answer", err)
	}
	_ = ln.Close()
	if err := Probe(context.Background(), addr); err == nil {
		t.Error("closed: no error")
	}
}

func TestMonitorSendsReadyThenWatchdogAndWritesTheHeartbeat(t *testing.T) {
	socket, got := fakeSystemd(t)
	srv := httptest.NewServer(http.NotFoundHandler())
	defer srv.Close()
	beat := filepath.Join(t.TempDir(), "run", "heartbeat")
	m := &Monitor{Socket: socket, Heartbeat: beat, Log: slog.New(slog.DiscardHandler),
		Addrs: func() []string { return []string{srv.Listener.Addr().String()} }}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { m.Run(ctx); close(done) }()
	if a, b := next(t, got), next(t, got); a != "READY=1" || b != "WATCHDOG=1" {
		t.Errorf("messages %q, %q; want READY=1, WATCHDOG=1", a, b)
	}
	cancel()
	<-done
	if err := Check(beat, time.Now()); err != nil {
		t.Errorf("heartbeat: %v", err)
	}
	if fi, err := os.Stat(beat); err != nil || fi.Mode().Perm() != 0o600 {
		t.Errorf("heartbeat mode: %v %v", fi, err)
	}
}

func TestMonitorStaysSilentWhileTheProbeFails(t *testing.T) {
	socket, got := fakeSystemd(t)
	beat := filepath.Join(t.TempDir(), "heartbeat")
	// A port this test opened and closed again: a fixed low port can be in
	// use by another package's tests where binding it needs no root.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	closed := ln.Addr().String()
	_ = ln.Close()
	m := &Monitor{Socket: socket, Heartbeat: beat, Log: slog.New(slog.DiscardHandler),
		Addrs: func() []string { return []string{closed} }}
	ctx, cancel := context.WithTimeout(context.Background(), 1500*time.Millisecond)
	defer cancel()
	m.Run(ctx)
	select {
	case msg := <-got:
		t.Errorf("sent %q while the probe failed", msg)
	default:
	}
	if _, err := os.Stat(beat); err == nil {
		t.Error("wrote a heartbeat while the probe failed")
	}
}

func TestCheckHeartbeatAge(t *testing.T) {
	path := filepath.Join(t.TempDir(), "run", "heartbeat")
	now := time.Unix(1_800_000_000, 0)
	if err := Check(path, now); err == nil {
		t.Error("a missing heartbeat passed")
	}
	cases := []struct {
		age  time.Duration
		pass bool
	}{{0, true}, {29 * time.Second, true}, {30 * time.Second, false}, {5 * time.Minute, false}, {-time.Minute, false}}
	for _, c := range cases {
		if err := Write(path, now.Add(-c.age)); err != nil {
			t.Fatal(err)
		}
		if err := Check(path, now); (err == nil) != c.pass {
			t.Errorf("age %s: err = %v, want pass %v", c.age, err, c.pass)
		}
	}
	if err := os.WriteFile(path, []byte("soon\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := Check(path, now); err == nil {
		t.Error("a heartbeat that is not a time passed")
	}
}
