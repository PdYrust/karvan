//go:build paneltest

// Package livetest drives the running panels from test/panels: it opens
// them as destinations, acts as their full admin, and moves users into
// them with Karvan. Only `make test-panels` builds it.
package livetest

import (
	"bytes"
	"context"
	"database/sql"
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
	"github.com/PdYrust/karvan/internal/migration/engine"
	"github.com/PdYrust/karvan/internal/migration/migrationtest"
	"github.com/PdYrust/karvan/internal/migration/plan"
	"github.com/PdYrust/karvan/internal/migration/reload"
	"github.com/PdYrust/karvan/internal/migration/run"
	"github.com/PdYrust/karvan/internal/panel"
	"github.com/PdYrust/karvan/internal/panel/paneldb"
	"github.com/PdYrust/karvan/internal/panel/pasarguarddb"
	"github.com/PdYrust/karvan/internal/panel/source"
	"github.com/PdYrust/karvan/internal/panelctl"
	"github.com/PdYrust/karvan/internal/secret"
	"github.com/PdYrust/karvan/internal/store"
)

// The panels come from scripts/testpanels/up.sh.
const (
	marzbanContainer = "karvan-test-marzban-marzban-1"
	webhookAddr      = "127.0.0.1:18099"
	// AdminPassword is every admin password the tests set.
	AdminPassword = "Karvan-Test-Pass-2026" //nolint:gosec // G101: a test value for the test panels
	// FullAdminName is the full admin the tests add when a panel has none.
	FullAdminName = "root"
)

// PanelsDir is where scripts/testpanels/up.sh put the panels.
func PanelsDir() string {
	if d := os.Getenv("KARVAN_TEST_PANELS"); d != "" {
		return d
	}
	return "/tmp/karvan-panels"
}

// Live is one running destination panel.
type Live struct {
	t      *testing.T
	Kind   panel.Kind
	Env    string
	API    dest.API
	Target paneldb.Target
}

// OpenLive opens a running panel as a destination, adding a full admin
// when it has none.
func OpenLive(t *testing.T, kind panel.Kind) *Live {
	t.Helper()
	ctx := context.Background()
	env := filepath.Join(PanelsDir(), "karvan-test-"+string(kind), ".env")
	d, err := dest.Open(ctx, env)
	if err != nil {
		t.Fatalf("open %s (run scripts/testpanels/up.sh): %v", kind, err)
	}
	l := &Live{t: t, Kind: kind, Env: env, Target: d.Target}
	l.API, err = d.API(ctx)
	if errors.Is(err, dest.ErrNoFullAdmin) {
		l.addFullAdmin()
		l.API, err = d.API(ctx)
	}
	if err != nil {
		t.Fatal(err)
	}
	if l.API.Admin != FullAdminName {
		t.Fatalf("full admin %q, want %q", l.API.Admin, FullAdminName)
	}
	return l
}

// addFullAdmin writes a full admin the way Karvan writes the moved one,
// with a Go bcrypt hash.
func (l *Live) addFullAdmin() {
	l.t.Helper()
	ctx := context.Background()
	hash, err := bcrypt.GenerateFromPassword([]byte(AdminPassword), 12)
	if err != nil {
		l.t.Fatal(err)
	}
	w, err := paneldb.OpenWriter(ctx, l.Target)
	if err != nil {
		l.t.Fatal(err)
	}
	defer w.Close()
	err = w.Chunk(ctx, func(x paneldb.Execer) error {
		now := paneldb.TimeArg(l.Target.Dialect, time.Now().UTC())
		if l.Kind == panel.Marzban {
			_, err := x.ExecContext(ctx, "INSERT INTO admins (username, hashed_password, created_at, is_sudo) VALUES (?, ?, ?, ?)",
				FullAdminName, string(hash), now, true)
			return err
		}
		_, err := pasarguarddb.Writer{Dialect: l.Target.Dialect}.InsertAdmin(ctx, x, panel.Admin{Username: FullAdminName}, string(hash), 1, time.Now().UTC())
		return err
	})
	if err != nil {
		l.t.Fatal(err)
	}
}

// Call sends an API request as the full admin and decodes the answer.
func (l *Live) Call(method, path string, body, out any) int {
	l.t.Helper()
	var rd io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		rd = bytes.NewReader(b)
	}
	req, _ := http.NewRequest(method, l.API.BaseURL+path, rd)
	req.Header.Set("Authorization", "Bearer "+l.API.Token(time.Now()))
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

// Login signs in through the panel's own login form.
func (l *Live) Login(user, password string) int {
	l.t.Helper()
	form := url.Values{"username": {user}, "password": {password}, "grant_type": {"password"}}
	resp, err := http.PostForm(l.API.BaseURL+"/api/admin/token", form)
	if err != nil {
		l.t.Fatal(err)
	}
	_ = resp.Body.Close()
	return resp.StatusCode
}

// User reads one user through the panel API.
func (l *Live) User(id int64, name string) map[string]any {
	l.t.Helper()
	path := "/api/user/" + url.PathEscape(name)
	if l.Kind == panel.PasarGuard {
		path = fmt.Sprintf("/api/user/by-id/%d", id)
	}
	var u map[string]any
	if code := l.Call(http.MethodGet, path, nil, &u); code != http.StatusOK {
		l.t.Fatalf("GET %s: %d", path, code)
	}
	return u
}

// Mover is a Karvan service pointed at one Live panel.
type Mover struct {
	t     *testing.T
	Svc   *run.Service
	Panel int64
	DB    *sql.DB // Karvan's own database, with the ledger
	Key   *secret.Key
	l     *Live
}

// NewMover starts a Karvan service with its own store, pointed at l.
func NewMover(t *testing.T, l *Live) *Mover {
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
	ctl := panelctl.Opener{Mode: panelctl.ModeDocker}
	m := &Mover{t: t, l: l, DB: db, Key: key, Svc: run.New(run.Deps{DB: db, Key: key, DataDir: dir, Tools: migrationtest.Tools,
		Reloaders: reload.Methods(), Control: func(ctx context.Context, env string) (run.PanelControl, error) {
			c, err := ctl.Open(ctx, filepath.Dir(env))
			if err != nil {
				return nil, err
			}
			return c, nil
		}})}
	if m.Panel, err = m.Svc.RegisterPanel(ctx, l.Env, "Live", string(l.Kind)); err != nil {
		t.Fatal(err)
	}
	return m
}

// Move checks and runs a job from src, expects every user moved, and
// returns its report.
func (m *Mover) Move(src migrationtest.Source, method string, edit func(p *plan.Plan)) *run.Report {
	m.t.Helper()
	rep := m.Run(src, run.Options{Reload: method}, edit)
	if rep.State != run.StateDone {
		m.t.Fatalf("job %s: %s", rep.State, rep.Error)
	}
	if rep.Counts.Moved != rep.Counts.Read {
		m.t.Fatalf("moved %d of %d", rep.Counts.Moved, rep.Counts.Read)
	}
	return rep
}

// Run checks and runs a job from src with o (its version and password
// are filled in) and returns the report, however the job ended.
func (m *Mover) Run(src migrationtest.Source, o run.Options, edit func(p *plan.Plan)) *run.Report {
	m.t.Helper()
	ctx := context.Background()
	j, err := m.Svc.Create(ctx, run.Login{URL: src.URL, Username: src.Username, Password: secret.NewValue(src.Password)}, m.Panel,
		source.Options{Sleep: func(context.Context, time.Duration) error { return nil }})
	if err != nil {
		m.t.Fatal(err)
	}
	if _, err := m.Svc.Check(ctx, j.ID); err != nil {
		m.t.Fatal(err)
	}
	_, p, v, _ := m.Svc.View(j.ID)
	edit(&p)
	r, v, err := m.Svc.SetPlan(ctx, j.ID, v, p)
	if err != nil {
		m.t.Fatal(err)
	}
	if r.Blockers > 0 {
		m.t.Fatalf("blockers: %+v %+v", r.Conflicts, r.Groups)
	}
	o.Version, o.AdminPassword = v, secret.NewValue(AdminPassword)
	if err := m.Svc.Run(ctx, j.ID, o); err != nil {
		m.t.Fatal(err)
	}
	m.Svc.Wait()
	rep, err := m.Svc.Report(j.ID)
	if err != nil {
		m.t.Fatal(err)
	}
	return rep
}

// Check reads the running panel from as an old panel, signed in as its
// full admin, and builds the plan edit gives for m's panel. Nothing is
// written to either panel.
func (m *Mover) Check(from *Live, edit func(p *plan.Plan)) *engine.Result {
	m.t.Helper()
	ctx := context.Background()
	j, err := m.Svc.Create(ctx, run.Login{URL: from.API.BaseURL, Username: FullAdminName, Password: secret.NewValue(AdminPassword)}, m.Panel,
		source.Options{Sleep: func(context.Context, time.Duration) error { return nil }})
	if err != nil {
		m.t.Fatal(err)
	}
	if _, err := m.Svc.Check(ctx, j.ID); err != nil {
		m.t.Fatal(err)
	}
	_, p, v, _ := m.Svc.View(j.ID)
	edit(&p)
	r, _, err := m.Svc.SetPlan(ctx, j.ID, v, p)
	if err != nil {
		m.t.Fatal(err)
	}
	return r
}

// Targets lists the moved users the way the reload step does.
func Targets(rep *run.Report, wireguard bool) []run.ReloadTarget {
	var out []run.ReloadTarget
	for _, u := range rep.Users {
		if u.Moved {
			out = append(out, run.ReloadTarget{ID: u.DestID, Username: u.Username, WireGuard: wireguard})
		}
	}
	return out
}

// Renamed gives a fake source's users a prefix, so every test moves
// names the Live panel has not seen.
func Renamed(src migrationtest.Source, prefix string, edit func(i int, u map[string]any)) {
	src.Mutate(func(users *[]map[string]any) {
		for i, u := range *users {
			u["username"] = prefix + u["username"].(string)
			if edit != nil {
				edit(i, u)
			}
		}
	})
}

// XrayEmails lists the users Marzban's running Xray holds in an inbound,
// read through Xray's own API inside the container.
func XrayEmails(t *testing.T, tag string) map[string]bool {
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
	out, err := exec.Command("docker", append([]string{"exec", marzbanContainer}, args...)...).CombinedOutput() //nolint:gosec // G204: fixed test commands
	if err != nil {
		t.Fatalf("docker exec %v: %v: %s", args, err, out)
	}
	return string(out)
}

// Webhook collects Marzban's notifications.
type Webhook struct {
	mu      sync.Mutex
	actions map[string]int // "action username" -> count
}

// StartWebhook listens where the test Marzban sends its webhooks.
func StartWebhook(t *testing.T) *Webhook {
	t.Helper()
	w := &Webhook{actions: map[string]int{}}
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

// Count waits up to 15 s for n notifications of action for names.
func (w *Webhook) Count(action string, names []string, n int) int {
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
