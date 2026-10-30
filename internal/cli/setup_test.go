package cli

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

var passwordLine = regexp.MustCompile(`Password  (\S+)`)

// fingerprintLine is the self-signed certificate's fingerprint (D5).
var fingerprintLine = regexp.MustCompile(`(?m)^  ([0-9A-F]{2}:){31}[0-9A-F]{2}$`)

func TestInitPrintsCredentialsOnlyToStdout(t *testing.T) {
	dir := t.TempDir()
	code, out, errOut := doInit(t, dir)
	if code != 0 {
		t.Fatalf("init exit %d: %s", code, errOut)
	}
	m := passwordLine.FindStringSubmatch(out)
	if m == nil || !strings.Contains(out, "Username  ") || !strings.Contains(out, "Address   https://") || strings.Contains(out, "plain HTTP") ||
		!fingerprintLine.MatchString(out) {
		t.Fatalf("stdout = %q", out)
	}
	if strings.Contains(errOut, m[1]) {
		t.Fatal("password written to stderr, where logs go")
	}
	if code, _, errOut := doInit(t, dir); code != 1 || !strings.Contains(errOut, "already initialized") {
		t.Fatalf("second init: %d %s", code, errOut)
	}
}

// The installer passes the port the operator confirmed (D20).
func TestInitTakesThePort(t *testing.T) {
	for _, bad := range []string{"x", "65536", "-1"} {
		if code, _, errOut := doInit(t, t.TempDir(), "-port", bad); code != 2 || !strings.Contains(errOut, "-port") {
			t.Fatalf("-port %s: %d %s", bad, code, errOut)
		}
	}
	code, out, errOut := doInit(t, t.TempDir(), "-port", "31234")
	if code != 0 || !strings.Contains(out, "https://<server address>:31234/") {
		t.Fatalf("init -port: %d %q %s", code, out, errOut)
	}
}

func doInit(t *testing.T, dir string, flags ...string) (int, string, string) {
	t.Helper()
	var out, errOut bytes.Buffer
	args := append([]string{"init", "-data-dir", filepath.Join(dir, "data"), "-key-file", filepath.Join(dir, "secret.key")}, flags...)
	code := Run(context.Background(), args, IO{Stdout: &out, Stderr: &errOut, StdoutIsTerminal: true})
	return code, out.String(), errOut.String()
}

func doReset(t *testing.T, dir string, terminal bool, flags ...string) (int, string, string) {
	t.Helper()
	var out, errOut bytes.Buffer
	args := append([]string{"reset", "-data-dir", filepath.Join(dir, "data"), "-key-file", filepath.Join(dir, "secret.key")}, flags...)
	code := Run(context.Background(), args, IO{Stdout: &out, Stderr: &errOut, StdoutIsTerminal: terminal})
	return code, out.String(), errOut.String()
}

// wantResetOutput checks a reset of the password, port and two-step: the
// HTTPS address with the new port, the same fingerprint as at init, and
// the follow-up notes.
func wantResetOutput(t *testing.T, out, init string) {
	t.Helper()
	if !passwordLine.MatchString(out) || !strings.Contains(out, "https://<server address>:42000/<path>/") || strings.Contains(out, "Username") ||
		fingerprintLine.FindString(out) != fingerprintLine.FindString(init) {
		t.Fatalf("stdout = %q", out)
	}
	if !strings.Contains(out, "Restart Karvan") || !strings.Contains(out, "Two-step sign-in is off") {
		t.Fatalf("missing follow-up notes: %q", out)
	}
}

func TestResetCommand(t *testing.T) {
	dir := t.TempDir()
	code, first, errOut := doInit(t, dir)
	if code != 0 {
		t.Fatal(errOut)
	}
	if code, _, errOut := doReset(t, dir, true); code != 2 || !strings.Contains(errOut, "choose at least one") {
		t.Fatalf("reset without flags: %d %s", code, errOut)
	}

	code, out, errOut := doReset(t, dir, false, "-password", "-port-number", "42000", "-totp")
	if code != 0 {
		t.Fatalf("reset: %d %s", code, errOut)
	}
	wantResetOutput(t, out, first)
	if !strings.Contains(errOut, "not a terminal") {
		t.Fatalf("no warning for non-terminal output: %q", errOut)
	}

	code, out, _ = doReset(t, dir, true, "-path", "-user", "-port")
	if code != 0 || !strings.Contains(out, "Username  ") || strings.Contains(out, "<path>") || strings.Contains(out, "<port>") {
		t.Fatalf("second reset: %d %q", code, out)
	}
}

func TestCommandsBeforeInit(t *testing.T) {
	dir := t.TempDir()
	if code, _, errOut := doReset(t, dir, true, "-password"); code != 1 || !strings.Contains(errOut, "karvan init") {
		t.Fatalf("reset before init: %d %s", code, errOut)
	}
	var errOut bytes.Buffer
	code := Run(context.Background(), []string{"serve", "-data-dir", filepath.Join(dir, "data"), "-key-file", filepath.Join(dir, "k")},
		IO{Stdout: &bytes.Buffer{}, Stderr: &errOut})
	if code != 1 || !strings.Contains(errOut.String(), "karvan init") {
		t.Fatalf("serve before init: %d %s", code, errOut.String())
	}
}

func TestBadFlags(t *testing.T) {
	var errOut bytes.Buffer
	if code := Run(context.Background(), []string{"init", "-nope"}, IO{Stdout: &bytes.Buffer{}, Stderr: &errOut}); code != 2 {
		t.Fatalf("exit %d", code)
	}
	if code := Run(context.Background(), []string{"version", "-h"}, IO{Stdout: &bytes.Buffer{}, Stderr: &errOut}); code != 0 {
		t.Fatalf("help exit %d", code)
	}
}

// The installer's HTTPS choices: plain HTTP, the server's own address in
// the panel's address, and checks of the flags before anything is made.
func TestInitHTTPSChoices(t *testing.T) {
	code, out, errOut := doInit(t, t.TempDir(), "-tls", "off", "-host", "203.0.113.7", "-port", "31234")
	if code != 0 || !strings.Contains(out, "Address   http://203.0.113.7:31234/") || !strings.Contains(out, "plain HTTP") ||
		fingerprintLine.MatchString(out) {
		t.Fatalf("-tls off: %d %q %s", code, out, errOut)
	}
	code, out, errOut = doInit(t, t.TempDir(), "-host", "2001:db8::1", "-port", "31234")
	if code != 0 || !strings.Contains(out, "Address   https://[2001:db8::1]:31234/") || !fingerprintLine.MatchString(out) {
		t.Fatalf("-host IPv6: %d %q %s", code, out, errOut)
	}
	for _, bad := range [][]string{
		{"-tls", "ip"}, {"-tls", "domain"}, {"-domain", "a.example"}, {"-tls", "off", "-email", "a@b.example"},
		{"-tls", "domain", "-domain", "a.example", "-email", "Me <a@b.example>"}, {"-host", "a/b"}, {"-host", "x y"},
	} {
		dir := t.TempDir()
		if code, _, errOut := doInit(t, dir, bad...); code != 2 {
			t.Fatalf("%v: %d %s", bad, code, errOut)
		}
		if matches, _ := filepath.Glob(filepath.Join(dir, "*")); len(matches) != 0 {
			t.Fatalf("%v left %v", bad, matches)
		}
	}
}

// Domain mode falls back to the self-signed certificate, with the reason,
// when port 80 is taken or Let's Encrypt cannot be reached.
func TestInitDomainFallsBack(t *testing.T) {
	busy, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatal(err)
	}
	defer busy.Close()
	dead := httptest.NewServer(http.NotFoundHandler())
	dead.Close()
	for name, tc := range map[string]struct {
		port int
		want string
	}{
		"port busy": {busy.Addr().(*net.TCPAddr).Port, "is in use by another program"},
		"no CA":     {freeTCPPort(t), "Let's Encrypt could not be reached from this server"},
	} {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			cfg := filepath.Join(dir, "config.json")
			body := fmt.Sprintf(`{"acmeHttpPort": %d, "acmeTestDirectory": %q}`, tc.port, dead.URL+"/dir")
			if err := os.WriteFile(cfg, []byte(body), 0o600); err != nil {
				t.Fatal(err)
			}
			code, out, errOut := doInit(t, dir, "-config", cfg, "-tls", "domain", "-domain", "panel.example.com",
				"-host", "203.0.113.7", "-port", "31234")
			if code != 0 || !strings.Contains(out, tc.want) || !strings.Contains(out, "Karvan's own certificate") ||
				!strings.Contains(out, "Address   https://203.0.113.7:31234/") || !fingerprintLine.MatchString(out) {
				t.Fatalf("init: %d %q %s", code, out, errOut)
			}
		})
	}
}

func freeTCPPort(t *testing.T) int {
	t.Helper()
	ln, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	return ln.Addr().(*net.TCPAddr).Port
}
