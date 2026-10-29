package cli

import (
	"bytes"
	"context"
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

func doInit(t *testing.T, dir string) (int, string, string) {
	t.Helper()
	var out, errOut bytes.Buffer
	code := Run(context.Background(), []string{"init", "-data-dir", filepath.Join(dir, "data"), "-key-file", filepath.Join(dir, "secret.key")},
		IO{Stdout: &out, Stderr: &errOut, StdoutIsTerminal: true})
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
