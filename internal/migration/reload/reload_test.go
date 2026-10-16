package reload

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/PdYrust/karvan/internal/migration/dest"
	"github.com/PdYrust/karvan/internal/migration/run"
	"github.com/PdYrust/karvan/internal/panel"
)

// fake records the calls a reloader makes and answers GETs with user.
type fake struct {
	mu    sync.Mutex
	calls []string
	puts  map[string]map[string]json.RawMessage
	user  string
	fail  string // a path that answers 500
}

func (f *fake) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, r.Method+" "+r.URL.EscapedPath())
	if !strings.HasPrefix(r.Header.Get("Authorization"), "Bearer eyJ") {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	if r.URL.EscapedPath() == f.fail {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"detail":"boom"}`))
		return
	}
	switch r.Method {
	case http.MethodGet:
		_, _ = w.Write([]byte(f.user))
	case http.MethodPut:
		var body map[string]json.RawMessage
		_ = json.NewDecoder(r.Body).Decode(&body)
		f.puts[r.URL.EscapedPath()] = body
		_, _ = w.Write([]byte(`{}`))
	default:
		_, _ = w.Write([]byte(`{}`))
	}
}

func serve(t *testing.T, user string) (*fake, dest.API) {
	t.Helper()
	f := &fake{puts: map[string]map[string]json.RawMessage{}, user: user}
	srv := httptest.NewServer(f)
	t.Cleanup(srv.Close)
	return f, dest.API{BaseURL: srv.URL, Admin: "root"}
}

func TestTouchKeepsNextPlanAndGroups(t *testing.T) {
	f, api := serve(t, `{"username":"a b","next_plan":{"data_limit":5},"group_ids":[3,4]}`)
	api.Kind = panel.PasarGuard
	res := Touch{}.Reload(context.Background(), run.ReloadInput{Kind: panel.PasarGuard, API: api,
		Users: []run.ReloadTarget{{ID: 9, Username: "a b"}}})
	if res.OK != 1 || res.Error != "" {
		t.Fatalf("result %+v", res)
	}
	body := f.puts["/api/user/by-id/9"]
	if string(body["next_plan"]) != `{"data_limit":5}` || string(body["group_ids"]) != "[3,4]" || len(body) != 2 {
		t.Fatalf("PUT body %v", body)
	}
}

func TestTouchMarzbanSendsAnEmptyEdit(t *testing.T) {
	f, api := serve(t, `{"username":"a b","next_plan":null,"group_ids":[1]}`)
	api.Kind = panel.Marzban
	res := Touch{}.Reload(context.Background(), run.ReloadInput{Kind: panel.Marzban, API: api,
		Users: []run.ReloadTarget{{ID: 9, Username: "a b"}, {ID: 10, Username: "c"}}})
	if res.OK != 2 {
		t.Fatalf("result %+v", res)
	}
	if body, ok := f.puts["/api/user/a%20b"]; !ok || len(body) != 0 {
		t.Fatalf("PUT body %v (ok %v)", body, ok)
	}
}

func TestFailuresAreCountedNotFatal(t *testing.T) {
	f, api := serve(t, `{}`)
	f.fail = "/api/user/bad"
	api.Kind = panel.Marzban
	res := Touch{}.Reload(context.Background(), run.ReloadInput{Kind: panel.Marzban, API: api,
		Users: []run.ReloadTarget{{Username: "ok"}, {Username: "bad"}}})
	if res.OK != 1 || len(res.Failed) != 1 || !strings.Contains(res.Failed[0], "bad: GET /api/user/bad: 500 boom") {
		t.Fatalf("result %+v", res)
	}
}

func TestCoreRestart(t *testing.T) {
	f, api := serve(t, `{"group_ids":[2]}`)
	api.Kind = panel.PasarGuard
	users := []run.ReloadTarget{{ID: 1, Username: "a", WireGuard: true}, {ID: 2, Username: "b"}}
	res := CoreRestart{}.Reload(context.Background(), run.ReloadInput{Kind: panel.PasarGuard, API: api, Users: users})
	if res.OK != 2 || res.Error != "" {
		t.Fatalf("result %+v", res)
	}
	want := []string{"GET /api/user/by-id/1", "PUT /api/user/by-id/1", "POST /api/nodes/reconnect"}
	if strings.Join(f.calls, ",") != strings.Join(want, ",") {
		t.Fatalf("calls %v", f.calls)
	}
	f.calls = nil
	api.Kind = panel.Marzban
	res = CoreRestart{}.Reload(context.Background(), run.ReloadInput{Kind: panel.Marzban, API: api, Users: users})
	if res.OK != 2 || strings.Join(f.calls, ",") != "POST /api/core/restart" {
		t.Fatalf("result %+v calls %v", res, f.calls)
	}
	f.fail = "/api/core/restart"
	if res = (CoreRestart{}).Reload(context.Background(), run.ReloadInput{Kind: panel.Marzban, API: api}); res.Error == "" {
		t.Fatal("a failed restart reads as done")
	}
}

// The panel's certificate is pinned; any other one is refused.
func TestHTTPSPinsTheCertificate(t *testing.T) {
	f := &fake{puts: map[string]map[string]json.RawMessage{}, user: `{}`}
	srv := httptest.NewTLSServer(f)
	t.Cleanup(srv.Close)
	dir := t.TempDir()
	good := filepath.Join(dir, "good.pem")
	writeCert(t, good, srv.TLS.Certificates[0])
	api := dest.API{Kind: panel.Marzban, BaseURL: srv.URL, CertFile: good, Admin: "root"}
	if res := (CoreRestart{}).Reload(context.Background(), run.ReloadInput{Kind: panel.Marzban, API: api}); res.Error != "" {
		t.Fatalf("pinned call: %+v", res)
	}
	bad := filepath.Join(dir, "bad.pem")
	if err := os.WriteFile(bad, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: []byte("not this one")}), 0o600); err != nil {
		t.Fatal(err)
	}
	api.CertFile = bad
	if res := (CoreRestart{}).Reload(context.Background(), run.ReloadInput{Kind: panel.Marzban, API: api}); !strings.Contains(res.Error, "different certificate") {
		t.Fatalf("unpinned call: %+v", res)
	}
	api.CertFile = filepath.Join(dir, "missing.pem")
	if res := (Touch{}).Reload(context.Background(), run.ReloadInput{Kind: panel.Marzban, API: api}); res.Error == "" {
		t.Fatal("a missing certificate file reads as fine")
	}
}

func writeCert(t *testing.T, path string, c tls.Certificate) {
	t.Helper()
	b := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: []byte("skipped")})
	b = append(b, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: c.Certificate[0]})...)
	if err := os.WriteFile(path, b, 0o600); err != nil {
		t.Fatal(err)
	}
}
