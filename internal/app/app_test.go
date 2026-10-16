package app

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/PdYrust/karvan/internal/auth"
	"github.com/PdYrust/karvan/internal/config"
	"github.com/PdYrust/karvan/internal/secret"
	"github.com/PdYrust/karvan/internal/settings"
	"github.com/PdYrust/karvan/internal/store"
)

func testConfig(t *testing.T) config.Config {
	t.Helper()
	dir := t.TempDir()
	cfg := config.Defaults()
	cfg.DataDir = filepath.Join(dir, "data")
	cfg.KeyFile = filepath.Join(dir, "etc", "secret.key")
	return cfg
}

func quiet() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }

func TestInitializeCreatesAWorkingInstallation(t *testing.T) {
	ctx := context.Background()
	cfg := testConfig(t)
	creds, err := Initialize(ctx, cfg, quiet())
	if err != nil {
		t.Fatalf("Initialize: %v", err)
	}
	if creds.Username == "" || creds.Password.IsZero() || creds.BasePath == "" || creds.Port < settings.MinRandomPort {
		t.Fatalf("credentials = %+v", creds)
	}

	a, err := Open(ctx, cfg, quiet())
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer a.Close()
	res, err := a.Auth.Login(ctx, auth.LoginRequest{Username: creds.Username, Password: creds.Password})
	if err != nil || res.Token == "" {
		t.Fatalf("sign-in with printed credentials: %v", err)
	}
	h, err := a.Settings.HTTP(ctx)
	if err != nil || h.BasePath != creds.BasePath || h.Port != creds.Port {
		t.Fatalf("settings = %+v, %v", h, err)
	}
}

func TestInitializeNeverStoresThePasswordInPlainText(t *testing.T) {
	cfg := testConfig(t)
	creds, err := Initialize(context.Background(), cfg, quiet())
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{store.FileName, store.FileName + "-wal"} {
		data, err := os.ReadFile(filepath.Join(cfg.DataDir, name))
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			t.Fatal(err)
		}
		if bytes.Contains(data, []byte(creds.Password.Reveal())) {
			t.Fatalf("%s contains the password", name)
		}
	}
}

func TestInitializeRefusesExistingInstallations(t *testing.T) {
	ctx := context.Background()
	cfg := testConfig(t)
	if _, err := Initialize(ctx, cfg, quiet()); err != nil {
		t.Fatal(err)
	}
	if _, err := Initialize(ctx, cfg, quiet()); err == nil {
		t.Fatal("second Initialize succeeded")
	}

	leftover := testConfig(t)
	if err := secret.CreateKeyFile(leftover.KeyFile); err != nil {
		t.Fatal(err)
	}
	if _, err := Initialize(ctx, leftover, quiet()); err == nil {
		t.Fatal("Initialize replaced an existing key file")
	}
}

func TestOpenBeforeInit(t *testing.T) {
	if _, err := Open(context.Background(), testConfig(t), quiet()); !errors.Is(err, settings.ErrNotInitialized) {
		t.Fatalf("err = %v, want ErrNotInitialized", err)
	}
}

func TestResetRegeneratesChosenValues(t *testing.T) {
	ctx := context.Background()
	cfg := testConfig(t)
	old, err := Initialize(ctx, cfg, quiet())
	if err != nil {
		t.Fatal(err)
	}
	a, err := Open(ctx, cfg, quiet())
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	session, err := a.Auth.Login(ctx, auth.LoginRequest{Username: old.Username, Password: old.Password})
	if err != nil {
		t.Fatal(err)
	}

	got, err := a.Reset(ctx, ResetOptions{Password: true, Port: 41000})
	if err != nil {
		t.Fatal(err)
	}
	if got.Username != "" || got.BasePath != "" || got.Password.IsZero() || got.Port != 41000 {
		t.Fatalf("reset output = %+v", got)
	}
	if _, err := a.Auth.Sessions.Lookup(ctx, session.Token); !errors.Is(err, auth.ErrNoSession) {
		t.Fatal("password reset kept the old session")
	}
	if _, err := a.Auth.Login(ctx, auth.LoginRequest{Username: old.Username, Password: old.Password}); err == nil {
		t.Fatal("old password still works")
	}
	if _, err := a.Auth.Login(ctx, auth.LoginRequest{Username: old.Username, Password: got.Password}); err != nil {
		t.Fatalf("new password: %v", err)
	}
}

func TestResetPathUserAndRandomPort(t *testing.T) {
	ctx := context.Background()
	cfg := testConfig(t)
	old, err := Initialize(ctx, cfg, quiet())
	if err != nil {
		t.Fatal(err)
	}
	a, err := Open(ctx, cfg, quiet())
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()

	again, err := a.Reset(ctx, ResetOptions{Path: true, Username: true, TOTP: true, Port: -1})
	if err != nil {
		t.Fatal(err)
	}
	h, _ := a.Settings.HTTP(ctx)
	if again.BasePath == old.BasePath || h.BasePath != again.BasePath || again.Username == old.Username || h.Port != again.Port {
		t.Fatalf("second reset = %+v, settings %+v", again, h)
	}
	if !(ResetOptions{TOTP: true}).Any() || (ResetOptions{}).Any() {
		t.Fatal("Any is wrong")
	}
}

func TestServeStopsWhenContextEnds(t *testing.T) {
	ctx := context.Background()
	cfg := testConfig(t)
	if _, err := Initialize(ctx, cfg, quiet()); err != nil {
		t.Fatal(err)
	}
	a, err := Open(ctx, cfg, quiet())
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(ctx)
	done := make(chan error, 1)
	go func() {
		done <- a.serveOn(ctx, ln, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusTeapot) }))
	}()
	res, err := http.Get("http://" + ln.Addr().String() + "/")
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != http.StatusTeapot {
		t.Fatalf("status %d", res.StatusCode)
	}

	// "OPTIONS *" must reach the handler, not net/http's built-in answer.
	conn, err := net.Dial("tcp", ln.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	fmt.Fprint(conn, "OPTIONS * HTTP/1.1\r\nHost: x\r\nConnection: close\r\n\r\n")
	line, _ := bufio.NewReader(conn).ReadString('\n')
	conn.Close()
	if !strings.Contains(line, "418") {
		t.Fatalf("OPTIONS * got %q, want the handler's 418", line)
	}
	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("serve returned %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("serve did not stop")
	}
}
