// Package health tells systemd and Docker that serve is alive. While
// a self-probe of each panel listener succeeds, it sends WATCHDOG=1 to the
// systemd notification socket and writes the heartbeat file that
// "karvan health" reads.
package health

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	// Every is the rhythm of the probe, the watchdog ping and the heartbeat.
	Every = 10 * time.Second
	// MaxAge is how old the heartbeat may be for "karvan health" to pass.
	MaxAge = 30 * time.Second
	// startEvery is the rhythm until the first probe passes, so READY=1 is
	// not ten seconds late.
	startEvery = time.Second
)

// probeTimeout bounds one dial and the wait for the server's answer.
var probeTimeout = 3 * time.Second

// HeartbeatPath is the heartbeat file in dataDir.
func HeartbeatPath(dataDir string) string { return filepath.Join(dataDir, "run", "heartbeat") }

// Notify sends state to the systemd notification socket. An empty socket
// (not started by systemd) sends nothing. A name starting with @ is an
// abstract socket, which net handles.
func Notify(socket, state string) error {
	if socket == "" {
		return nil
	}
	conn, err := net.DialUnix("unixgram", nil, &net.UnixAddr{Name: socket, Net: "unixgram"})
	if err != nil {
		return fmt.Errorf("notify systemd: %w", err)
	}
	defer func() { _ = conn.Close() }()
	if _, err := conn.Write([]byte(state)); err != nil {
		return fmt.Errorf("notify systemd: %w", err)
	}
	return nil
}

// Probe connects to addr and sends a line that is not HTTP. Only a
// running server answers it or closes the connection; a hung one leaves
// it waiting in the kernel's queue until the timeout.
func Probe(ctx context.Context, addr string) error {
	d := net.Dialer{Timeout: probeTimeout}
	conn, err := d.DialContext(ctx, "tcp", addr)
	if err != nil {
		return fmt.Errorf("probe %s: %w", addr, err)
	}
	defer func() { _ = conn.Close() }()
	_ = conn.SetDeadline(time.Now().Add(probeTimeout))
	if _, err := conn.Write([]byte("PROBE\r\n\r\n")); err != nil {
		return fmt.Errorf("probe %s: %w", addr, err)
	}
	_, err = conn.Read(make([]byte, 1))
	var ne net.Error
	if errors.As(err, &ne) && ne.Timeout() {
		return fmt.Errorf("probe %s: no answer in %s", addr, probeTimeout)
	}
	return nil // bytes, a close or a reset all come from the server
}

// errNotServing is a round before the panel serves, with nothing to probe
// yet; it is not logged.
var errNotServing = errors.New("the panel is not served yet")

// Monitor runs the probe and reports while it passes.
type Monitor struct {
	Socket    string // NOTIFY_SOCKET
	Heartbeat string // file path
	// Addrs lists the addresses to probe, read again on every round; none
	// means the panel is not served yet.
	Addrs func() []string
	Log   *slog.Logger
}

// Run probes until ctx ends. The first passing round sends READY=1;
// every passing round pings the watchdog and writes the heartbeat.
func (m *Monitor) Run(ctx context.Context) {
	ready := false
	for {
		if err := m.probe(ctx); err != nil {
			if ctx.Err() != nil {
				return
			}
			if !errors.Is(err, errNotServing) {
				m.Log.Error("the self-probe failed", "err", err)
			}
		} else {
			if !ready {
				ready = true
				m.notify("READY=1")
			}
			m.notify("WATCHDOG=1")
			if err := Write(m.Heartbeat, time.Now()); err != nil {
				m.Log.Error("cannot write the heartbeat", "err", err)
			}
		}
		wait := Every
		if !ready {
			wait = startEvery
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(wait):
		}
	}
}

func (m *Monitor) probe(ctx context.Context) error {
	addrs := m.Addrs()
	if len(addrs) == 0 {
		return errNotServing
	}
	for _, a := range addrs {
		if err := Probe(ctx, a); err != nil {
			return err
		}
	}
	return nil
}

func (m *Monitor) notify(state string) {
	if err := Notify(m.Socket, state); err != nil {
		m.Log.Warn("cannot reach systemd", "state", state, "err", err)
	}
}

// Write stores t in the heartbeat file, replacing it in one step so a
// reader never sees half of it.
func Write(path string, t time.Time) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("heartbeat: %w", err)
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, []byte(strconv.FormatInt(t.Unix(), 10)+"\n"), 0o600); err != nil {
		return fmt.Errorf("heartbeat: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		return fmt.Errorf("heartbeat: %w", err)
	}
	return nil
}

// Check returns nil when the heartbeat at path is under MaxAge old at now.
func Check(path string, now time.Time) error {
	b, err := os.ReadFile(path) //nolint:gosec // G304: the heartbeat in Karvan's data folder.
	if err != nil {
		return fmt.Errorf("no heartbeat: %w", err)
	}
	sec, err := strconv.ParseInt(strings.TrimSpace(string(b)), 10, 64)
	if err != nil {
		return fmt.Errorf("heartbeat %q is not a time", strings.TrimSpace(string(b)))
	}
	age := now.Sub(time.Unix(sec, 0))
	if age >= MaxAge || age <= -MaxAge { // a clock set back counts the same
		return fmt.Errorf("the last heartbeat is %s old", age.Truncate(time.Second))
	}
	return nil
}
