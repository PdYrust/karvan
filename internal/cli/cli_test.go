package cli

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func run(t *testing.T, args ...string) (code int, stdout, stderr string) {
	t.Helper()
	var out, errOut bytes.Buffer
	code = Run(context.Background(), args, IO{Stdout: &out, Stderr: &errOut})
	return code, out.String(), errOut.String()
}

func TestVersionPrintsBuildInfo(t *testing.T) {
	code, out, _ := run(t, "version")
	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	if !strings.HasPrefix(out, "karvan dev ") {
		t.Errorf("stdout = %q, want it to start with %q", out, "karvan dev ")
	}
}

func TestVersionRejectsArguments(t *testing.T) {
	code, _, errOut := run(t, "version", "extra")
	if code != 2 {
		t.Fatalf("exit code = %d, want 2", code)
	}
	if !strings.Contains(errOut, "unexpected argument") {
		t.Errorf("stderr = %q, want an unexpected argument message", errOut)
	}
}

func TestUnknownCommandExitsWithUsage(t *testing.T) {
	code, _, errOut := run(t, "fly")
	if code != 2 {
		t.Fatalf("exit code = %d, want 2", code)
	}
	if !strings.Contains(errOut, `unknown command "fly"`) || !strings.Contains(errOut, "Commands:") {
		t.Errorf("stderr = %q, want unknown command and usage", errOut)
	}
}

func TestNoArgumentsPrintsUsage(t *testing.T) {
	code, out, _ := run(t)
	if code != 0 || !strings.Contains(out, "version") {
		t.Fatalf("code = %d, stdout = %q", code, out)
	}
}
