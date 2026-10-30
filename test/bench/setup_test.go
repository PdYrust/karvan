//go:build benchtest

// Package bench holds the load tests of D37 and D38: the bridge under
// open-loop load with a million stored tokens, and the HTTP API at 50
// clients. They run with make bench-bridge and make bench-api, never
// in make check.
package bench

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/PdYrust/karvan/internal/bridge"
	"github.com/PdYrust/karvan/internal/migration/ledger"
	"github.com/PdYrust/karvan/internal/panel"
	"github.com/PdYrust/karvan/internal/panel/source"
	"github.com/PdYrust/karvan/internal/secret"
	kstore "github.com/PdYrust/karvan/internal/store"
)

const (
	oldURL    = "http://127.0.0.1:9/"
	oldDomain = "old.example.com"
	destKey   = "bench-dest"
)

// created is when every destination user was made; issued is the time
// inside every stored token, after it.
var (
	created = time.Date(2025, 1, 2, 3, 4, 5, 0, time.UTC)
	issued  = created.Add(24 * time.Hour)
)

// body is a subscription answer of a usual size.
var body = strings.Repeat("dmxlc3M6Ly9iZW5jaG1hcmstY29uZmlnLWxpbmUK", 100)

// fakePanel is the new panel: it answers every /sub/ link with body and
// knows users 1 to n by id, without a table in memory.
type fakePanel struct {
	srv    *httptest.Server
	client *http.Client // one, as the bridge's panel client is
}

func newFakePanel(t *testing.T) *fakePanel {
	f := &fakePanel{srv: httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/sub/") {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Subscription-Userinfo", "upload=0; download=1; total=0; expire=0")
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = io.WriteString(w, body)
	}))}
	f.srv.Config.ReadHeaderTimeout = 10 * time.Second
	f.client = &http.Client{Transport: &http.Transport{MaxIdleConnsPerHost: 512, IdleConnTimeout: time.Minute}}
	t.Cleanup(f.srv.Close)
	return f
}

func (f *fakePanel) Key() string     { return destKey }
func (f *fakePanel) SubPath() string { return "sub" }
func (f *fakePanel) BaseURL() string { return f.srv.URL }
func (f *fakePanel) Close()          {}

func (f *fakePanel) Client() *http.Client { return f.client }

func (f *fakePanel) Token(id int64, _ string, _ time.Time) string { return fmt.Sprintf("dest-%d", id) }

func (f *fakePanel) Rows(_ context.Context, ids []int64) (map[int64]bridge.Row, error) {
	out := make(map[int64]bridge.Row, len(ids))
	for _, id := range ids {
		out[id] = bridge.Row{Username: userName(id), CreatedAt: created}
	}
	return out, nil
}

func userName(id int64) string { return fmt.Sprintf("user%07d", id) }

// oldToken is user id's old link token; every one is stored as verified.
func oldToken(id int64) string { return fmt.Sprintf("oldtoken%016d", id) }

// rig is a bridge serving one panel rule with users stored tokens.
type rig struct {
	panel  *fakePanel
	bridge string // the bridge's address
	users  int64
}

func newRig(t *testing.T, users int64) *rig {
	t.Helper()
	ctx := context.Background()
	dir := t.TempDir()
	db, err := kstore.Open(ctx, dir)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	key := newKey(t, dir)
	f := newFakePanel(t)
	b := bridge.New(bridge.Deps{DB: db, Key: key, Log: slog.New(slog.DiscardHandler),
		OpenDest: func(context.Context, string) (bridge.Destination, error) { return f, nil }})
	exec(t, db, "INSERT INTO panels (id, env_path, name, kind, scanned_at) VALUES (1, '/opt/new/.env', 'new', 'pasarguard', 0)")
	exec(t, db, "INSERT INTO domains (name, created_at, updated_at) VALUES (?, 1, 1)", oldDomain)
	port := freePort(t)
	off := false
	site, err := b.SaveSite(ctx, bridge.Site{Name: "bench", PanelID: 1, SourceKind: panel.Marzban, SourceURL: oldURL,
		OldPath: "sub", Domains: []string{oldDomain}, Port: port, PlainHTTPOK: true, HTTPS: &off})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = b.DeleteSite(context.Background(), site.ID) })
	start := time.Now()
	fill(t, db, key.Derive("bridge/token-mac"), site.ID, users)
	t.Logf("stored %d users and tokens in %.1f s", users, time.Since(start).Seconds())
	return &rig{panel: f, bridge: fmt.Sprintf("127.0.0.1:%d", port), users: users}
}

func newKey(t *testing.T, dir string) *secret.Key {
	t.Helper()
	p := filepath.Join(dir, "key")
	if err := secret.CreateKeyFile(p); err != nil {
		t.Fatal(err)
	}
	k, err := secret.LoadKeyFile(p)
	if err != nil {
		t.Fatal(err)
	}
	return k
}

func exec(t *testing.T, db *sql.DB, q string, args ...any) {
	t.Helper()
	if _, err := db.Exec(q, args...); err != nil {
		t.Fatal(err)
	}
}

func freePort(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port
}

// fill stores users moved users with one verified old token each, as
// the migration and a first lazy check would, in transactions of
// fillChunk rows, then folds the write-ahead log into the database.
func fill(t *testing.T, db *sql.DB, macKey []byte, siteID, users int64) {
	t.Helper()
	u, err := source.ParseBase(oldURL)
	if err != nil {
		t.Fatal(err)
	}
	origin := ledger.Origin(u)
	for from := int64(1); from <= users; from += fillChunk {
		fillRange(t, db, macKey, siteID, origin, from, min(users, from+fillChunk-1))
	}
	exec(t, db, "PRAGMA wal_checkpoint(TRUNCATE)")
}

const fillChunk = 20_000

func fillRange(t *testing.T, db *sql.DB, macKey []byte, siteID int64, origin string, from, to int64) {
	t.Helper()
	at := created.UnixMicro()
	tx, err := db.Begin()
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback() //nolint:errcheck // a no-op after Commit
	ins, err := tx.Prepare("INSERT INTO ledger (id, kind, job_id, state, source_origin, source_key, source_created, source_username, " +
		"dest, dest_id, dest_username, dest_created, created_at, updated_at) VALUES (?, 'user', 'bench', 'done', ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)")
	if err != nil {
		t.Fatal(err)
	}
	defer ins.Close()
	tok, err := tx.Prepare("INSERT INTO legacy_tokens (mac, site_id, ledger_id, issued_at, verified_at, checked_at) VALUES (?, ?, ?, ?, ?, ?)")
	if err != nil {
		t.Fatal(err)
	}
	defer tok.Close()
	now := time.Now().Unix()
	for id := from; id <= to; id++ {
		name := userName(id)
		if _, err := ins.Exec(id, origin, name, at, name, destKey, id, name, at, at, at); err != nil {
			t.Fatal(err)
		}
		h := hmac.New(sha256.New, macKey)
		h.Write([]byte(oldToken(id)))
		if _, err := tok.Exec(h.Sum(nil), siteID, id, issued.Unix(), now, now); err != nil {
			t.Fatal(err)
		}
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
}
