package panelctl

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

// fakeEngine plays the Docker daemon on a unix socket.
type fakeEngine struct {
	mu     sync.Mutex
	socket string
	calls  []string
	status int // answer to stop and start; 0 means 204
}

func newFakeEngine(t *testing.T) *fakeEngine {
	t.Helper()
	f := &fakeEngine{socket: filepath.Join(t.TempDir(), "docker.sock")}
	l, err := (&net.ListenConfig{}).Listen(context.Background(), "unix", f.socket)
	if err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewUnstartedServer(http.HandlerFunc(f.serve))
	srv.Listener = l
	srv.Start()
	t.Cleanup(srv.Close)
	return f
}

func (f *fakeEngine) serve(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	f.calls = append(f.calls, r.Method+" "+r.URL.Path)
	status := f.status
	f.mu.Unlock()
	if r.Method == http.MethodGet && r.URL.Path == "/containers/json" {
		if r.URL.Query().Get("all") != "1" || r.URL.Query().Get("filters") != `{"label":["com.docker.compose.project=marzban"]}` {
			http.Error(w, `{"message":"bad query"}`, http.StatusBadRequest)
			return
		}
		_ = json.NewEncoder(w).Encode([]map[string]any{
			{"Id": "c2", "Image": "pasarguard/panel@sha256:abc", "Labels": map[string]string{serviceLabel: "node-worker"}},
			{"Id": "c3", "Image": "mysql:8.4", "Labels": map[string]string{serviceLabel: "mysql"}},
			{"Id": "c1", "Image": "gozargah/marzban:v0.8.4", "Labels": map[string]string{serviceLabel: "marzban"}},
		})
		return
	}
	if status == 0 {
		status = http.StatusNoContent
	}
	w.WriteHeader(status)
	if status >= 400 {
		_, _ = w.Write([]byte(`{"message":"cannot stop container c1"}`))
	}
}

func (f *fakeEngine) answer(status int) {
	f.mu.Lock()
	f.status = status
	f.mu.Unlock()
}

func (f *fakeEngine) log() string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return strings.Join(f.calls, "\n")
}

// The socket finds the panel's containers by the compose labels and stops
// and starts only those, so the database keeps running.
func TestSocketStopsOnlyThePanelContainers(t *testing.T) {
	f := newFakeEngine(t)
	ctx := context.Background()
	c, err := Opener{Socket: f.socket}.openSocket(ctx, panelDir(t, "docker-compose.yml"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(c.Services, ",") != "marzban,node-worker" || c.Project != "marzban" {
		t.Fatalf("services %v project %q", c.Services, c.Project)
	}
	if err := c.Stop(ctx); err != nil {
		t.Fatal(err)
	}
	if err := c.Start(ctx); err != nil {
		t.Fatal(err)
	}
	want := "GET /containers/json\nPOST /containers/c1/stop\nPOST /containers/c2/stop\nPOST /containers/c1/start\nPOST /containers/c2/start"
	if got := f.log(); got != want {
		t.Fatalf("calls:\n%s", got)
	}
}

// Mode auto tries the Docker CLI first, then the socket.
func TestAutoFallsBackToTheSocket(t *testing.T) {
	f := newFakeEngine(t)
	denied := &recorder{mode: "denied"}
	c, err := Opener{Mode: ModeAuto, Run: denied.run, Socket: f.socket}.Open(context.Background(), panelDir(t, "docker-compose.yml"))
	if err != nil {
		t.Fatal(err)
	}
	if c.engine == nil || f.log() != "GET /containers/json" {
		t.Fatalf("not through the socket: %s", f.log())
	}
}

func TestSocketErrors(t *testing.T) {
	ctx := context.Background()
	f := newFakeEngine(t)
	if _, err := (Opener{Socket: f.socket}).openSocket(ctx, panelDir(t, "")); !errors.Is(err, ErrNoCompose) {
		t.Errorf("no compose file: %v", err)
	}
	if _, err := (Opener{Socket: filepath.Join(t.TempDir(), "none.sock")}).openSocket(ctx, panelDir(t, "compose.yml")); err == nil ||
		!strings.Contains(err.Error(), "docker socket") {
		t.Errorf("no socket: %v", err)
	}
	c, err := Opener{Socket: f.socket}.openSocket(ctx, panelDir(t, "compose.yml"))
	if err != nil {
		t.Fatal(err)
	}
	f.answer(http.StatusNotModified) // already stopped
	if err := c.Stop(ctx); err != nil {
		t.Errorf("already stopped: %v", err)
	}
	f.answer(http.StatusInternalServerError)
	if err := c.Stop(ctx); err == nil || !strings.Contains(err.Error(), "cannot stop container c1") {
		t.Errorf("daemon error: %v", err)
	}
}

func TestProjectName(t *testing.T) {
	for in, want := range map[string]string{"/opt/marzban": "marzban", "/opt/My Panel.1": "mypanel1", "/opt/_x-Y": "x-y"} {
		if got := projectName(in); got != want {
			t.Errorf("%q: %q, want %q", in, got, want)
		}
	}
}
