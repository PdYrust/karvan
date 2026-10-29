package panelctl

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

// The test binary plays docker: KARVAN_FAKE_DOCKER selects the behavior.
func TestMain(m *testing.M) {
	if mode := os.Getenv("KARVAN_FAKE_DOCKER"); mode != "" {
		os.Exit(fakeDocker(mode, os.Args[1:]))
	}
	os.Exit(m.Run())
}

const config = `{"services": {"marzban": {"image": "gozargah/marzban:v0.8.4"}, "mysql": {"image": "mysql:8.4"},
 "node-worker": {"image": "pasarguard/panel@sha256:abc"}, "nginx": {"image": "nginx"}}}`

func fakeDocker(mode string, args []string) int {
	cmd := strings.Join(args, " ")
	switch {
	case strings.Contains(cmd, "version"):
		if mode == "legacy" && strings.HasPrefix(cmd, "compose") {
			return 1
		}
		return 0
	case strings.Contains(cmd, " config "):
		fmt.Print(config)
		return 0
	case mode == "denied" && strings.Contains(cmd, " ps "):
		fmt.Fprintln(os.Stderr, "permission denied while trying to connect to the Docker daemon socket\nmore")
		return 1
	}
	return 0
}

// recorder runs the fake docker and records each command line.
type recorder struct {
	mu    sync.Mutex
	mode  string
	lines []string
}

func (r *recorder) run(ctx context.Context, name string, args ...string) *exec.Cmd {
	r.mu.Lock()
	r.lines = append(r.lines, name+" "+strings.Join(args, " "))
	r.mu.Unlock()
	cmd := exec.CommandContext(ctx, os.Args[0], args...)
	cmd.Env = append(os.Environ(), "KARVAN_FAKE_DOCKER="+r.mode)
	return cmd
}

func panelDir(t *testing.T, compose string) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "marzban")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	if compose != "" {
		if err := os.WriteFile(filepath.Join(dir, compose), []byte("services: {}\n"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

// Like the official scripts: the folder name is the project, and only the
// panel's own services stop, so the database keeps running.
func TestStopsOnlyThePanelServices(t *testing.T) {
	dir := panelDir(t, "docker-compose.yml")
	r := &recorder{mode: "ok"}
	c, err := Opener{Mode: ModeDocker, Run: r.run}.Open(context.Background(), dir)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(c.Services, ",") != "marzban,node-worker" || c.Project != "marzban" {
		t.Fatalf("services %v project %q", c.Services, c.Project)
	}
	if err := c.Stop(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := c.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(dir, "docker-compose.yml")
	want := []string{
		"docker compose version",
		"docker compose -f " + file + " -p marzban config --format json",
		"docker compose -f " + file + " -p marzban ps -q marzban node-worker",
		"docker compose -f " + file + " -p marzban stop marzban node-worker",
		"docker compose -f " + file + " -p marzban start marzban node-worker",
	}
	if got := strings.Join(r.lines, "\n"); got != strings.Join(want, "\n") {
		t.Fatalf("commands:\n%s", got)
	}
}

func TestOldDockerCompose(t *testing.T) {
	r := &recorder{mode: "legacy"}
	c, err := Opener{Mode: ModeDocker, Run: r.run}.Open(context.Background(), panelDir(t, "compose.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(r.lines[len(r.lines)-1], "docker-compose -f ") || !strings.HasSuffix(c.File, "compose.yaml") {
		t.Fatalf("commands %v", r.lines)
	}
}

func TestWhyNot(t *testing.T) {
	ctx := context.Background()
	if _, err := (Opener{Mode: ModeOff}).Open(ctx, t.TempDir()); !errors.Is(err, ErrOff) {
		t.Errorf("off: %v", err)
	}
	r := &recorder{mode: "ok"}
	if _, err := (Opener{Mode: ModeDocker, Run: r.run}).Open(ctx, panelDir(t, "")); !errors.Is(err, ErrNoCompose) {
		t.Errorf("no compose file: %v", err)
	}
	denied := &recorder{mode: "denied"}
	_, err := Opener{Mode: ModeAuto, Run: denied.run, Socket: filepath.Join(t.TempDir(), "none.sock")}.Open(ctx, panelDir(t, "docker-compose.yml"))
	if err == nil || !strings.Contains(err.Error(), "permission denied while trying") || strings.Contains(err.Error(), "more") {
		t.Errorf("denied: %v", err)
	}
	if _, err := panelServices([]byte(`{"services": {"db": {"image": "mysql"}}}`)); !errors.Is(err, ErrNoServices) {
		t.Errorf("no panel service: %v", err)
	}
}
