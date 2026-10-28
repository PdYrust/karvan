package handlers_test

import (
	"context"
	"crypto/hmac"
	"crypto/sha1"
	"database/sql"
	"encoding/base32"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"testing/fstest"
	"time"

	"github.com/PdYrust/karvan/internal/audit"
	"github.com/PdYrust/karvan/internal/auth"
	"github.com/PdYrust/karvan/internal/bridge"
	"github.com/PdYrust/karvan/internal/certorder"
	"github.com/PdYrust/karvan/internal/dbstats"
	"github.com/PdYrust/karvan/internal/hoststats"
	"github.com/PdYrust/karvan/internal/httpapi"
	"github.com/PdYrust/karvan/internal/migration/migrationtest"
	"github.com/PdYrust/karvan/internal/migration/run"
	"github.com/PdYrust/karvan/internal/secret"
	"github.com/PdYrust/karvan/internal/settings"
	"github.com/PdYrust/karvan/internal/store"
)

const (
	base     = "abcDEF2345ghijkm"
	username = "owner"
	password = "correct horse battery"
	// testPanelPort stands for the web panel's own port in the bridge.
	testPanelPort = 9
)

var passwordHash = sync.OnceValue(func() string {
	h, err := auth.HashPassword(password)
	if err != nil {
		panic(err)
	}
	return h
})

// webBuild mimics the Vite output: two entries, a shared chunk, a lazy
// chunk only the login page loads, and a chunk only the app loads.
var webBuild = fstest.MapFS{
	"login.html":                  {Data: []byte(`<!doctype html><html><head><base href="/" /></head><body>login</body></html>`)},
	"index.html":                  {Data: []byte(`<!doctype html><html><head><base href="/" /></head><body>app</body></html>`)},
	"favicon.svg":                 {Data: []byte(`<svg xmlns="http://www.w3.org/2000/svg"/>`)},
	"manifest.webmanifest":        {Data: []byte(`{"name":"Karvan","start_url":"./","scope":"./"}`)},
	"icon-192.png":                {Data: []byte("\x89PNG")},
	"icon-512.png":                {Data: []byte("\x89PNG")},
	"icon-maskable-512.png":       {Data: []byte("\x89PNG")},
	"apple-touch-icon.png":        {Data: []byte("\x89PNG")},
	"assets/login-1.js":           {Data: []byte(`login()`)},
	"assets/shared-1.js":          {Data: []byte(`shared()`)},
	"assets/logo3d-1.js":          {Data: []byte(`logo3d()`)},
	"assets/login-1.css":          {Data: []byte(`body{}`)},
	"assets/font-1.woff2":         {Data: []byte(`font`)},
	"assets/app-1.js":             {Data: []byte(`app()`)},
	"assets/overview-1.js":        {Data: []byte(`overview()`)},
	"assets/not-in-manifest-1.js": {Data: []byte(`x`)},
	".vite/manifest.json": {Data: []byte(`{
		"login.html": {"file": "assets/login-1.js", "isEntry": true, "imports": ["_shared"], "dynamicImports": ["src/logo3d.ts"], "css": ["assets/login-1.css"], "assets": ["assets/font-1.woff2"]},
		"_shared": {"file": "assets/shared-1.js"},
		"src/logo3d.ts": {"file": "assets/logo3d-1.js", "imports": ["_shared"]},
		"index.html": {"file": "assets/app-1.js", "isEntry": true, "imports": ["_shared"], "dynamicImports": ["src/overview.tsx"]},
		"src/overview.tsx": {"file": "assets/overview-1.js"}
	}`)},
}

// testLog receives the access log; the secrets test reads it.
var testLog io.Writer = io.Discard

type env struct {
	t      *testing.T
	srv    *httptest.Server
	client *http.Client
	auth   *auth.Service
	csrf   string
	runs   *run.Service
	bridge *bridge.Bridge
	db     *sql.DB
	data   string // Karvan's data directory
	// closing is the server's shutdown signal; close it to end streams.
	closing chan struct{}
	host    *hoststats.Sampler
	dbstats *dbstats.Sampler
	// url and basePath are where do sends requests and login signs in.
	url      string
	basePath string
}

func newEnv(t *testing.T) *env {
	t.Helper()
	e, d := newEnvDeps(t)
	h, err := httpapi.New(d)
	if err != nil {
		t.Fatal(err)
	}
	e.srv = httptest.NewServer(h)
	t.Cleanup(e.srv.Close)
	e.url = e.srv.URL
	return e
}

// newEnvDeps sets up the services of a test server and the handler deps
// for them, with the test base path.
func newEnvDeps(t *testing.T) (*env, httpapi.Deps) {
	t.Helper()
	dir := t.TempDir()
	db, err := store.Open(context.Background(), filepath.Join(dir, "data"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	keyPath := filepath.Join(dir, "secret.key")
	if err := secret.CreateKeyFile(keyPath); err != nil {
		t.Fatal(err)
	}
	key, err := secret.LoadKeyFile(keyPath)
	if err != nil {
		t.Fatal(err)
	}
	ops := auth.NewOperators(db, key)
	if _, err := ops.Create(context.Background(), username, passwordHash()); err != nil {
		t.Fatal(err)
	}
	log := audit.New(db)
	svc := auth.NewService(ops, auth.NewSessions(db), auth.NewLimiter(db, auth.DefaultLockout), log)
	runs := run.New(run.Deps{DB: db, Key: key, DataDir: filepath.Join(dir, "data"), Tools: migrationtest.Tools})
	br := bridge.New(bridge.Deps{DB: db, Key: key, Log: slog.New(slog.DiscardHandler), DataDir: filepath.Join(dir, "data"),
		KeyFile: keyPath, EnvRoots: []string{filepath.Join(dir, "opt")}, PanelPorts: func() []int { return []int{testPanelPort} },
		ChallengePort: freePort(t)})
	// Nothing answers on port 1, so orders end as acme_unreachable.
	orders := certorder.New(certorder.Deps{DB: db, Key: key, Bridge: br, Settings: settings.New(db), Audit: log,
		Log: slog.New(slog.DiscardHandler), TestDirectory: "http://127.0.0.1:1/dir",
		LookupTXT: func(context.Context, string) ([]string, error) { return []string{"seen-value"}, nil }})
	closing := make(chan struct{})
	hostStore := hoststats.NewStore(db)
	host := &hoststats.Sampler{Reader: hoststats.System, Store: hostStore}
	dbs := &dbstats.Sampler{Store: dbstats.NewStore(db), Panels: func(ctx context.Context) ([]dbstats.PanelRef, error) {
		rows, err := runs.Panels(ctx)
		out := make([]dbstats.PanelRef, len(rows))
		for i, p := range rows {
			out[i] = dbstats.PanelRef{ID: p.ID, EnvPath: p.EnvPath, Name: p.Name}
		}
		return out, err
	}}
	d := httpapi.Deps{
		BasePath: base, Auth: svc, Audit: log, Web: webBuild, Version: "test",
		Logger: slog.New(slog.NewTextHandler(testLog, &slog.HandlerOptions{Level: slog.LevelDebug})),
		Runs:   runs, Bridge: br, Orders: orders, Settings: settings.New(db), Closing: closing,
		Host: host, HostStore: hostStore, HostReader: hoststats.System, DBStats: dbs,
	}
	jar, _ := cookiejar.New(nil)
	client := &http.Client{Jar: jar, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	return &env{t: t, client: client, auth: svc, runs: runs, bridge: br, db: db, data: filepath.Join(dir, "data"), closing: closing,
		host: host, dbstats: dbs, basePath: base}, d
}

type response struct {
	status int
	header http.Header
	body   []byte
}

func (r response) json(t *testing.T, v any) {
	t.Helper()
	if err := json.Unmarshal(r.body, v); err != nil {
		t.Fatalf("decode %s: %v", r.body, err)
	}
}

func (r response) errorCode(t *testing.T) string {
	t.Helper()
	var body struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	r.json(t, &body)
	return body.Error.Code
}

// do sends a request the way the web panel does: same-origin, with the
// current CSRF token.
func (e *env) do(method, path string, body any) response {
	e.t.Helper()
	var rd io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		rd = strings.NewReader(string(b))
	}
	req, _ := http.NewRequest(method, e.url+path, rd)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	req.Header.Set("X-CSRF-Token", e.csrf)
	res, err := e.client.Do(req)
	if err != nil {
		e.t.Fatal(err)
	}
	defer res.Body.Close()
	data, _ := io.ReadAll(res.Body)
	return response{status: res.StatusCode, header: res.Header, body: data}
}

func (e *env) raw(method, path string) []byte {
	e.t.Helper()
	req, _ := http.NewRequest(method, e.url+path, nil)
	res, err := http.DefaultTransport.RoundTrip(req)
	if err != nil {
		e.t.Fatal(err)
	}
	defer res.Body.Close()
	data, _ := io.ReadAll(res.Body)
	res.Header.Del("Date")
	var b strings.Builder
	fmt.Fprintf(&b, "%d\n", res.StatusCode)
	_ = res.Header.Write(&b)
	b.Write(data)
	return []byte(b.String())
}

func (e *env) fetchCSRF() {
	e.t.Helper()
	var s struct {
		CSRFToken string `json:"csrfToken"`
	}
	e.do(http.MethodGet, "/"+e.basePath+"/api/v1/session", nil).json(e.t, &s)
	e.csrf = s.CSRFToken
}

func (e *env) login(code string) response {
	e.t.Helper()
	return e.loginAs(username, password, code)
}

func (e *env) loginAs(user, pass, code string) response {
	e.t.Helper()
	e.fetchCSRF()
	res := e.do(http.MethodPost, "/"+e.basePath+"/api/v1/session", map[string]string{"username": user, "password": pass, "totpCode": code})
	if res.status == http.StatusOK {
		var s struct {
			CSRFToken string `json:"csrfToken"`
		}
		res.json(e.t, &s)
		e.csrf = s.CSRFToken
	}
	return res
}

func totpNow(secretText string) string {
	key, _ := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(secretText)
	var msg [8]byte
	binary.BigEndian.PutUint64(msg[:], uint64(time.Now().Unix()/30))
	mac := hmac.New(sha1.New, key)
	mac.Write(msg[:])
	sum := mac.Sum(nil)
	off := sum[len(sum)-1] & 0x0f
	return fmt.Sprintf("%06d", (binary.BigEndian.Uint32(sum[off:off+4])&0x7fffffff)%1_000_000)
}
