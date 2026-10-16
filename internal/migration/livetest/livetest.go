//go:build paneltest

package reload_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/PdYrust/karvan/internal/migration/dest"
	"github.com/PdYrust/karvan/internal/migration/migrationtest"
	"github.com/PdYrust/karvan/internal/migration/plan"
	"github.com/PdYrust/karvan/internal/migration/reload"
	"github.com/PdYrust/karvan/internal/migration/run"
	"github.com/PdYrust/karvan/internal/panel"
	"github.com/PdYrust/karvan/internal/panel/paneldb"
	"github.com/PdYrust/karvan/internal/panel/pasarguarddb"
	"github.com/PdYrust/karvan/internal/panel/source"
	"github.com/PdYrust/karvan/internal/secret"
	"github.com/PdYrust/karvan/internal/store"
)

// The panels come from scripts/testpanels/up.sh.
const (
	marzbanContainer = "karvan-testpanels-marzban-1"
	webhookAddr      = "127.0.0.1:18099"
	adminPassword    = "Karvan-Test-Pass-2026"
	fullAdminName    = "root"
)

func panelsDir() string {
	if d := os.Getenv("KARVAN_TEST_PANELS"); d != "" {
		return d
	}
	return "/tmp/karvan-panels"
}

// live is one running destination panel.
type live struct {
	t    *testing.T
	kind panel.Kind
	env  string
	api  dest.API
	tg   paneldb.Target
}

func openLive(t *testing.T, kind panel.Kind) *live {
	t.Helper()
	ctx := context.Background()
	env := filepath.Join(panelsDir(), string(kind), ".env")
	d, err := dest.Open(ctx, env)
	if err != nil {
		t.Fatalf("open %s (run scripts/testpanels/up.sh): %v", kind, err)
	}
	l := &live{t: t, kind: kind, env: env, tg: d.Target}
	l.api, err = d.API(ctx)
	if errors.Is(err, dest.ErrNoFullAdmin) {
		l.addFullAdmin()
		l.api, err = d.API(ctx)
	}
	if err != nil {
		t.Fatal(err)
	}
	if l.api.Admin != fullAdminName {
		t.Fatalf("full admin %q, want %q", l.api.Admin, fullAdminName)
	}
	return l
}

// addFullAdmin writes a full admin the way Karvan writes the moved one,
// with a Go bcrypt hash.
func (l *live) addFullAdmin() {
	l.t.Helper()
	ctx := context.Background()
	hash, err := bcrypt.GenerateFromPassword([]byte(adminPassword), 12)
	if err != nil {
		l.t.Fatal(err)
	}
	w, err := paneldb.OpenWriter(ctx, l.tg)
	if err != nil {
		l.t.Fatal(err)
	}
	defer w.Close()
	err = w.Batch(ctx, func(x paneldb.Execer) error {
		now := paneldb.TimeArg(l.tg.Dialect, time.Now().UTC())
		if l.kind == panel.Marzban {
			_, err := x.ExecContext(ctx, "INSERT INTO admins (username, hashed_password, created_at, is_sudo) VALUES (?, ?, ?, ?)",
				fullAdminName, string(hash), now, true)
			return err
		}
		_, err := pasarguarddb.Writer{Dialect: l.tg.Dialect}.InsertAdmin(ctx, x, panel.Admin{Username: fullAdminName}, string(hash), 1, time.Now().UTC())
		return err
	})
	if err != nil {
		l.t.Fatal(err)
	}
}

// call sends an API request as the full admin and decodes the answer.
func (l *live) call(method, path string, body, out any) int {
	l.t.Helper()
	var rd io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		rd = bytes.NewReader(b)
	}
	req, _ := http.NewRequest(method, l.api.BaseURL+path, rd)
	req.Header.Set("Authorization", "Bearer "+l.api.Token(time.Now()))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		l.t.Fatal(err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if out != nil && resp.StatusCode/100 == 2 {
		if err := json.Unmarshal(raw, out); err != nil {
			l.t.Fatalf("%s %s: %v: %s", method, path, err, raw)
		}
	}
	if resp.StatusCode/100 != 2 {
		l.t.Logf("%s %s: %d %s", method, path, resp.StatusCode, raw)
	}
	return resp.StatusCode
}

// login signs in through the panel's own login form.
func (l *live) login(user, password string) int {
	l.t.Helper()
	form := url.Values{"username": {user}, "password": {password}, "grant_type": {"password"}}
	resp, err := http.PostForm(l.api.BaseURL+"/api/admin/token", form)
	if err != nil {
		l.t.Fatal(err)
	}
	_ = resp.Body.Close()
	return resp.StatusCode
}

// user reads one user through the panel API.
func (l *live) user(id int64, name string) map[string]any {
	l.t.Helper()
	path := "/api/user/" + url.PathEscape(name)
	if l.kind == panel.PasarGuard {
		path = fmt.Sprintf("/api/user/by-id/%d", id)
	}
	var u map[string]any
	if code := l.call(http.MethodGet, path, nil, &u); code != http.StatusOK {
		l.t.Fatalf("GET %s: %d", path, code)
	}
	return u
}

// mover is a Karvan service pointed at one live panel.
type mover struct {
	t     *testing.T
	svc   *run.Service
	panel int64
	l     *live
}

func newMover(t *testing.T, l *live) *mover {
	t.Helper()
	ctx := context.Background()
	dir := t.TempDir()
	db, err := store.Open(ctx, dir)
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
	m := &mover{t: t, l: l, svc: run.New(run.Deps{DB: db, Key: key, DataDir: dir, Tools: migrationtest.Tools, Reloaders: reload.Methods()})}
	if m.panel, err = m.svc.RegisterPanel(ctx, l.env, "live", string(l.kind)); err != nil {
		t.Fatal(err)
	}
	return m
}

// move checks and runs a job from src and returns its report.
func (m *mover) move(src migrationtest.Source, method string, edit func(p *plan.Plan)) *run.Report {
	m.t.Helper()
	ctx := context.Background()
	j, err := m.svc.Create(ctx, run.Login{URL: src.URL, Username: src.Username, Password: secret.NewValue(src.Password)}, m.panel,
		source.Options{Sleep: func(context.Context, time.Duration) error { return nil }})
	if err != nil {
		m.t.Fatal(err)
	}
	if _, err := m.svc.Check(ctx, j.ID); err != nil {
		m.t.Fatal(err)
	}
	_, p, v, _ := m.svc.View(j.ID)
	edit(&p)
	r, v, err := m.svc.SetPlan(ctx, j.ID, v, p)
	if err != nil {
		m.t.Fatal(err)
	}
	if r.Blockers > 0 {
		m.t.Fatalf("blockers: %+v %+v", r.Conflicts, r.Groups)
	}
	if err := m.svc.Run(ctx, j.ID, run.Options{Version: v, Reload: method, AdminPassword: secret.NewValue(adminPassword)}); err != nil {
		m.t.Fatal(err)
	}
	m.svc.Wait()
	rep, err := m.svc.Report(j.ID)
	if err != nil {
		m.t.Fatal(err)
	}
	if rep.State != run.StateDone {
		m.t.Fatalf("job %s: %s", rep.State, rep.Error)
	}
	if rep.Counts.Moved != rep.Counts.Read {
		m.t.Fatalf("moved %d of %d; groups %+v", rep.Counts.Moved, rep.Counts.Read, r.Groups)
	}
	return rep
}

// targets lists the moved users the way the reload step does.
func targets(rep *run.Report, wireguard bool) []run.ReloadTarget {
	var out []run.ReloadTarget
	for _, u := range rep.Users {
		if u.Moved {
			out = append(out, run.ReloadTarget{ID: u.DestID, Username: u.Username, WireGuard: wireguard})
		}
	}
	return out
}

// renamed gives a fake source's users a prefix, so every test moves
// names the live panel has not seen.
func renamed(src migrationtest.Source, prefix string, edit func(i int, u map[string]any)) {
	src.Mutate(func(users *[]map[string]any) {
		for i, u := range *users {
			u["username"] = prefix + u["username"].(string)
			if edit != nil {
				edit(i, u)
			}
		}
	})
}

// xrayEmails lists the users Marzban's running Xray holds in an inbound,
// read through Xray's own API inside the container.
func xrayEmails(t *testing.T, tag string) map[string]bool {
	t.Helper()
	port := strings.TrimSpace(docker(t, "python", "-c", xrayPortScript))
	var out struct {
		Users []struct {
			Email string `json:"email"`
		} `json:"users"`
	}
	if err := json.Unmarshal([]byte(docker(t, "xray", "api", "inbounduser", "--server=127.0.0.1:"+port, "-tag="+tag)), &out); err != nil {
		t.Fatal(err)
	}
	got := map[string]bool{}
	for _, u := range out.Users {
		got[u.Email] = true
	}
	return got
}

// xrayPortScript prints the Xray API port: the one listening socket of
// the Xray process that is not a user inbound (Marzban picks it at random).
const xrayPortScript = `import os
pid = [p for p in os.listdir('/proc') if p.isdigit() and open('/proc/%s/cmdline' % p, 'rb').read().startswith(b'/usr/local/bin/xray\0run')][0]
links = [os.readlink('/proc/%s/fd/%s' % (pid, f)) for f in os.listdir('/proc/%s/fd' % pid)]
socks = {s[8:-1] for s in links if s.startswith('socket:')}
for line in open('/proc/net/tcp').readlines()[1:]:
    c = line.split()
    port = int(c[1].split(':')[1], 16)
    if c[3] == '0A' and c[9] in socks and not 18441 <= port <= 18444:
        print(port)
`

func docker(t *testing.T, args ...string) string {
	t.Helper()
	out, err := exec.Command("docker", append([]string{"exec", marzbanContainer}, args...)...).CombinedOutput()
	if err != nil {
		t.Fatalf("docker exec %v: %v: %s", args, err, out)
	}
	return string(out)
}

// webhook collects Marzban's notifications.
type webhook struct {
	mu      sync.Mutex
	actions map[string]int // "action username" -> count
}

func startWebhook(t *testing.T) *webhook {
	t.Helper()
	w := &webhook{actions: map[string]int{}}
	srv := &http.Server{Addr: webhookAddr, ReadHeaderTimeout: 5 * time.Second, Handler: http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		var events []struct {
			Action string `json:"action"`
			User   struct {
				Username string `json:"username"`
			} `json:"user"`
		}
		_ = json.NewDecoder(r.Body).Decode(&events)
		w.mu.Lock()
		for _, e := range events {
			w.actions[e.Action+" "+e.User.Username]++
		}
		w.mu.Unlock()
	})}
	go func() { _ = srv.ListenAndServe() }()
	t.Cleanup(func() { _ = srv.Close() })
	return w
}

// count waits up to 15 s for n notifications of action for names.
func (w *webhook) count(action string, names []string, n int) int {
	deadline := time.Now().Add(15 * time.Second)
	for {
		w.mu.Lock()
		got := 0
		for _, name := range names {
			got += w.actions[action+" "+name]
		}
		w.mu.Unlock()
		if got >= n || time.Now().After(deadline) {
			return got
		}
		time.Sleep(200 * time.Millisecond)
	}
}
