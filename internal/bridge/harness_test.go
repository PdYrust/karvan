package bridge

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/PdYrust/karvan/internal/migration/ledger"
	"github.com/PdYrust/karvan/internal/panel"
	"github.com/PdYrust/karvan/internal/panel/fakepanel"
	"github.com/PdYrust/karvan/internal/panel/marzbanapi"
	"github.com/PdYrust/karvan/internal/panel/source"
	"github.com/PdYrust/karvan/internal/panel/subtoken"
	"github.com/PdYrust/karvan/internal/secret"
	kstore "github.com/PdYrust/karvan/internal/store"
)

const oldSecret = "old-panel-secret-for-tests-only-0123456789"

// clock is a test clock.
type clock struct {
	mu sync.Mutex
	t  time.Time
}

func (c *clock) now() time.Time { c.mu.Lock(); defer c.mu.Unlock(); return c.t }
func (c *clock) add(d time.Duration) {
	c.mu.Lock()
	c.t = c.t.Add(d)
	c.mu.Unlock()
}

// fakeDest is a destination panel: rows in memory and an HTTP server
// that answers its own minted tokens and one native token.
type fakeDest struct {
	mu    sync.Mutex
	rows  map[int64]Row
	srv   *httptest.Server
	last  *http.Request
	hits  int
	reads int // Rows calls
}

func newFakeDest(t *testing.T) *fakeDest {
	f := &fakeDest{rows: map[int64]Row{}}
	f.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		f.last, f.hits = r.Clone(context.Background()), f.hits+1
		f.mu.Unlock()
		switch {
		case strings.HasPrefix(r.URL.Path, "/sub/dest-"), r.URL.Path == "/sub/native-token-0123456789":
			w.Header().Set("Subscription-Userinfo", "upload=0; download=1")
			w.Header().Set("Connection", "X-Secret-Hop")
			w.Header().Set("X-Secret-Hop", "drop me")
			_, _ = io.WriteString(w, "config "+r.URL.Path)
		default:
			http.Error(w, `{"detail":"Not Found"}`, http.StatusNotFound)
		}
	}))
	t.Cleanup(f.srv.Close)
	return f
}

func (f *fakeDest) Key() string          { return "fake-dest" }
func (f *fakeDest) SubPath() string      { return "sub" }
func (f *fakeDest) BaseURL() string      { return f.srv.URL }
func (f *fakeDest) Client() *http.Client { return f.srv.Client() }
func (f *fakeDest) Close()               {}
func (f *fakeDest) Token(id int64, username string, _ time.Time) string {
	return fmt.Sprintf("dest-%d-%s", id, username)
}

func (f *fakeDest) Rows(_ context.Context, ids []int64) (map[int64]Row, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.reads++
	out := map[int64]Row{}
	for _, id := range ids {
		if r, ok := f.rows[id]; ok {
			out[id] = r
		}
	}
	return out, nil
}

func (f *fakeDest) set(id int64, fn func(r *Row)) {
	f.mu.Lock()
	defer f.mu.Unlock()
	r := f.rows[id]
	fn(&r)
	f.rows[id] = r
}

// env is a bridge with one site, a fake old Marzban and a fake new panel.
type env struct {
	t     *testing.T
	b     *Bridge
	clock *clock
	old   *fakepanel.Marzban
	dest  *fakeDest
	site  Site
	users map[string]ledger.Entry // by source username
	port  int
	setup int // old panel calls made by the setup
}

// oldCalls is how many /info calls the bridge made.
func (e *env) oldCalls() int { return e.old.SubCalls() - e.setup }

var created = time.Date(2025, 1, 2, 3, 4, 5, 123456000, time.UTC)

func newEnv(t *testing.T, names ...string) *env {
	t.Helper()
	ctx := context.Background()
	dir := t.TempDir()
	db, err := kstore.Open(ctx, dir)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	keyPath := filepath.Join(dir, "key")
	if err := secret.CreateKeyFile(keyPath); err != nil {
		t.Fatal(err)
	}
	key, err := secret.LoadKeyFile(keyPath)
	if err != nil {
		t.Fatal(err)
	}
	users := make([]map[string]any, len(names))
	for i, n := range names {
		users[i] = fakepanel.MarzbanUser(n, i+1, nil)
	}
	e := &env{t: t, clock: &clock{t: time.Date(2026, 10, 16, 12, 0, 0, 0, time.UTC)}, users: map[string]ledger.Entry{}, port: 1}
	e.old = fakepanel.NewMarzban(fakepanel.MarzbanAdmin{Username: "reseller", Password: "pw"}, users, fakepanel.Faults{})
	t.Cleanup(e.old.Close)
	e.dest = newFakeDest(t)
	if _, err := db.ExecContext(ctx, "INSERT INTO panels (id, env_path, name, kind, scanned_at) VALUES (1, '/opt/new/.env', 'new', 'pasarguard', 0)"); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, "INSERT INTO domains (name, created_at, updated_at) VALUES ('old.example.com', 1, 1)"); err != nil {
		t.Fatal(err)
	}
	e.b = New(Deps{DB: db, Key: key, Now: e.clock.now, Log: slog.New(slog.DiscardHandler),
		OpenDest: func(context.Context, string) (Destination, error) { return e.dest, nil }})
	e.site, err = e.b.st.Save(ctx, Site{Name: "old", PanelID: 1, SourceKind: panel.Marzban, SourceURL: e.old.URL,
		SourceIP: "127.0.0.1", OldPath: "sub", Domains: []string{"old.example.com"}, Port: e.port, Lazy: true, PlainHTTPOK: true})
	if err != nil {
		t.Fatal(err)
	}
	e.b.mu.Lock()
	e.b.load([]Site{e.site}, nil)
	e.b.mu.Unlock()
	for i, n := range names {
		e.addUser(n, int64(100+i))
	}
	e.setup = e.old.SubCalls()
	return e
}

// addUser records n as moved to destination id, with the fingerprint the
// old panel's own answer gives.
func (e *env) addUser(name string, id int64) {
	e.t.Helper()
	ctx := context.Background()
	tok := e.oldToken(name, e.clock.now().Add(-time.Hour))
	c, _ := source.New(source.Options{BaseURL: e.old.URL, Attempts: 1})
	u, err := marzbanapi.SubscriptionUser(ctx, c, "sub", tok)
	if err != nil {
		e.t.Fatal(err)
	}
	e.old.Subscribe(tok, "") // setup only
	row := panel.Row{ID: id, Username: "new_" + name, CreatedAt: u.CreatedAt}
	ids, err := ledger.New(e.b.DB).Add(ctx, []ledger.Entry{{Kind: ledger.KindUser, JobID: "job", State: ledger.Done,
		SourceOrigin: e.site.origin, SourceKey: name, SourceCreated: u.CreatedAt, SourceUsername: name,
		CredPrint: ledger.Fingerprint(u.Credentials), Dest: "fake-dest", Row: row}})
	if err != nil {
		e.t.Fatal(err)
	}
	got, _ := ledger.New(e.b.DB).Get(ctx, ids[0])
	e.users[name] = got
	e.dest.set(id, func(r *Row) { r.Username, r.CreatedAt = row.Username, row.CreatedAt })
}

// oldToken mints an old link token for name and makes the old panel know it.
func (e *env) oldToken(name string, at time.Time) string {
	tok := subtoken.Marzban(name, oldSecret, at)
	e.old.Subscribe(tok, name)
	return tok
}

// get requests an old link from ip and returns status and body.
func (e *env) get(path, ip string) (int, string) { return e.request("old.example.com", path, ip) }

func (e *env) getHost(host, path string) (int, string) { return e.request(host, path, "198.51.100.7") }

func (e *env) request(host, path, ip string) (int, string) {
	e.t.Helper()
	r := httptest.NewRequest(http.MethodGet, "http://"+host+path, nil)
	r.RemoteAddr = ip + ":40000"
	r.Header.Set("User-Agent", "v2rayNG/1.9")
	r.Header.Set("X-Hwid", "device-1")
	w := httptest.NewRecorder()
	e.b.Handler(e.port).ServeHTTP(w, r)
	return w.Code, w.Body.String()
}

func (e *env) tokens() int {
	e.t.Helper()
	var n int
	if err := e.b.DB.QueryRow("SELECT count(*) FROM legacy_tokens").Scan(&n); err != nil {
		e.t.Fatal(err)
	}
	return n
}
