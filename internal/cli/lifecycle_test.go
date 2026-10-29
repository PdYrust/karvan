package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/PdYrust/karvan/internal/health"
	"github.com/PdYrust/karvan/internal/netports"
	"github.com/PdYrust/karvan/internal/store"
)

// installed fakes an installer folder whose scripts the current user owns,
// and records what would run in place of the process.
func installed(t *testing.T, root bool) (dir string, ran *[]string) {
	t.Helper()
	dir = filepath.Join(t.TempDir(), "0.5.0")
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"get.sh", "install.sh"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("#!/bin/sh\n"), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	link := filepath.Join(filepath.Dir(dir), "current")
	if err := os.Symlink(dir, link); err != nil {
		t.Fatal(err)
	}
	var got []string
	oldDir, oldEuid, oldOwner, oldExec := lifecycleDir, geteuid, scriptOwner, execScript
	t.Cleanup(func() { lifecycleDir, geteuid, scriptOwner, execScript = oldDir, oldEuid, oldOwner, oldExec })
	lifecycleDir, scriptOwner = link, os.Getuid()
	geteuid = func() int {
		if root {
			return 0
		}
		return 1000
	}
	execScript = func(path string, args []string) error {
		got = append([]string{path}, args...)
		return nil
	}
	return dir, &got
}

func TestLifecycleRunsTheInstallerScript(t *testing.T) {
	dir, ran := installed(t, true)
	archive := filepath.Join(t.TempDir(), "karvan.tar.gz")
	if err := os.WriteFile(archive, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		args []string
		want []string
	}{
		{[]string{"update"}, []string{"get.sh", "--update"}},
		{[]string{"update", "-archive", archive}, []string{"get.sh", "--update", "--archive", archive}},
		{[]string{"rollback", "-yes"}, []string{"install.sh", "--rollback", "--yes"}},
		{[]string{"uninstall"}, []string{"install.sh", "--uninstall"}},
		{[]string{"uninstall", "-purge"}, []string{"install.sh", "--uninstall", "--purge"}},
	}
	for _, c := range cases {
		*ran = nil
		if code, _, errOut := run(t, c.args...); code != 0 {
			t.Fatalf("%v: exit %d, %s", c.args, code, errOut)
		}
		want := append([]string{filepath.Join(dir, c.want[0])}, c.want[1:]...)
		if !slices.Equal(*ran, want) {
			t.Errorf("%v ran %v, want %v", c.args, *ran, want)
		}
	}
}

func TestLifecycleRefusesWhenNotRoot(t *testing.T) {
	_, ran := installed(t, false)
	for _, cmd := range []string{"update", "rollback", "uninstall"} {
		code, _, errOut := run(t, cmd)
		if code != 1 || !strings.Contains(errOut, "as root") || *ran != nil {
			t.Errorf("%s: exit %d, %q, ran %v", cmd, code, errOut, *ran)
		}
	}
}

func TestLifecycleRefusesAnUnsafeScript(t *testing.T) {
	dir, ran := installed(t, true)
	script := filepath.Join(dir, "install.sh")
	refuse := func(why string) {
		t.Helper()
		code, _, errOut := run(t, "rollback")
		if code != 1 || !strings.Contains(errOut, why) || *ran != nil {
			t.Errorf("exit %d, %q, ran %v; want refusal %q", code, errOut, *ran, why)
		}
	}
	if err := os.Chmod(script, 0o775); err != nil {
		t.Fatal(err)
	}
	refuse("group or others can write")
	if err := os.Chmod(script, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(dir, 0o777); err != nil {
		t.Fatal(err)
	}
	refuse("group or others can write")
	if err := os.Chmod(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	scriptOwner = os.Getuid() + 1
	refuse("not owned by root")
	scriptOwner = os.Getuid()
	if err := os.Remove(script); err != nil {
		t.Fatal(err)
	}
	refuse("missing")
}

func TestUpdateRejectsAMissingArchive(t *testing.T) {
	_, ran := installed(t, true)
	code, _, errOut := run(t, "update", "-archive", filepath.Join(t.TempDir(), "none.tar.gz"))
	if code != 2 || !strings.Contains(errOut, "is not a file") || *ran != nil {
		t.Errorf("exit %d, %q, ran %v", code, errOut, *ran)
	}
}

func TestVersionJSON(t *testing.T) {
	code, out, _ := run(t, "version", "-json")
	var v versionJSON
	if code != 0 || json.Unmarshal([]byte(out), &v) != nil {
		t.Fatalf("exit %d, %q", code, out)
	}
	want, err := store.SchemaVersion()
	if err != nil || v.Version != "dev" || v.Schema != want || want == 0 {
		t.Errorf("got %+v, want version dev and schema %d (%v)", v, want, err)
	}
}

func TestPortsFreePicksFromTheRange(t *testing.T) {
	code, out, _ := run(t, "ports", "-free")
	p, err := strconv.Atoi(strings.TrimSpace(out))
	if code != 0 || err != nil || p < netports.FreeLow || p > netports.FreeHigh {
		t.Errorf("exit %d, %q", code, out)
	}
	if code, out, _ := run(t, "ports"); code != 0 {
		t.Errorf("ports: exit %d, %q", code, out)
	}
}

func TestHealthReadsTheHeartbeat(t *testing.T) {
	data := t.TempDir()
	args := []string{"health", "-data-dir", data, "-key-file", filepath.Join(t.TempDir(), "key")}
	if code, _, _ := run(t, args...); code != 1 {
		t.Errorf("no heartbeat: exit %d, want 1", code)
	}
	if err := health.Write(health.HeartbeatPath(data), time.Now()); err != nil {
		t.Fatal(err)
	}
	if code, out, errOut := run(t, args...); code != 0 {
		t.Errorf("fresh heartbeat: exit %d, %q %q", code, out, errOut)
	}
}
