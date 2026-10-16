package handlers_test

import (
	"bufio"
	"context"
	"encoding/json"
	"net"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/PdYrust/karvan/internal/migration/migrationtest"
	"github.com/PdYrust/karvan/internal/migration/run"
	"github.com/PdYrust/karvan/internal/panel"
	"github.com/PdYrust/karvan/internal/panel/paneldb/paneldbtest"
	"github.com/PdYrust/karvan/internal/panel/pasarguarddb"
)

const api = "/" + base + "/api/v1"

// migrationEnv is a signed-in panel with one destination (a PasarGuard
// fixture) and a fake old Marzban with users.
type migrationEnv struct {
	*env
	src     migrationtest.Source
	panelID int64
}

func newMigrationEnv(t *testing.T, users int) *migrationEnv {
	t.Helper()
	e := newEnv(t)
	tg := paneldbtest.Load(t, paneldbtest.Fixture{Flavor: "sqlite", Path: "../../panel/pasarguarddb/testdata/" + pasarguarddb.Revision + ".sqlite.sql"})
	envPath := migrationtest.PanelEnv(t, tg, panel.PasarGuard)
	id, err := e.runs.RegisterPanel(context.Background(), envPath, "dest", string(panel.PasarGuard))
	if err != nil {
		t.Fatal(err)
	}
	src := migrationtest.NewSource(t, panel.Marzban, users)
	t.Cleanup(src.Close)
	if res := e.login(""); res.status != http.StatusOK {
		t.Fatalf("login: %d", res.status)
	}
	return &migrationEnv{env: e, src: src, panelID: id}
}

// create starts and checks a job and returns its id.
func (m *migrationEnv) create() string {
	m.t.Helper()
	res := m.do(http.MethodPost, api+"/migrations", map[string]any{"sourceUrl": m.src.URL, "username": m.src.Username,
		"password": m.src.Password, "panelId": m.panelID})
	if res.status != http.StatusCreated {
		m.t.Fatalf("create: %d %s", res.status, res.body)
	}
	var job struct{ ID string }
	res.json(m.t, &job)
	if res := m.do(http.MethodPost, api+"/migrations/"+job.ID+"/compare", nil); res.status != http.StatusOK {
		m.t.Fatalf("compare: %d %s", res.status, res.body)
	}
	return job.ID
}

// plan reads the plan, puts the users in group 1 and returns the new version.
func (m *migrationEnv) plan(id string) int {
	m.t.Helper()
	res := m.do(http.MethodGet, api+"/migrations/"+id+"/plan", nil)
	var got struct {
		PlanVersion int            `json:"planVersion"`
		Plan        map[string]any `json:"plan"`
	}
	res.json(m.t, &got)
	if res.header.Get("ETag") != strconv.Quote(strconv.Itoa(got.PlanVersion)) {
		m.t.Fatalf("etag %q", res.header.Get("ETag"))
	}
	got.Plan["targets"] = migrationtest.PasarGuardTargets
	put := m.withHeader("If-Match", res.header.Get("ETag"))
	out := put(http.MethodPut, api+"/migrations/"+id+"/plan", got.Plan)
	if out.status != http.StatusOK {
		m.t.Fatalf("put plan: %d %s", out.status, out.body)
	}
	var after struct {
		PlanVersion int            `json:"planVersion"`
		Blockers    int            `json:"blockers"`
		Impact      map[string]int `json:"impact"`
	}
	out.json(m.t, &after)
	if after.Blockers != 0 || after.Impact["moving"] == 0 {
		m.t.Fatalf("after plan: %+v %s", after, out.body)
	}
	if stale := put(http.MethodPut, api+"/migrations/"+id+"/plan", got.Plan); stale.status != http.StatusPreconditionFailed ||
		stale.errorCode(m.t) != "stale_plan" {
		m.t.Fatalf("stale If-Match: %d %s", stale.status, stale.body)
	}
	return after.PlanVersion
}

// withHeader returns do with one more header.
func (m *migrationEnv) withHeader(k, v string) func(method, path string, body any) response {
	return func(method, path string, body any) response {
		old := m.client.Transport
		m.client.Transport = headerTransport{k: k, v: v, next: http.DefaultTransport}
		defer func() { m.client.Transport = old }()
		return m.do(method, path, body)
	}
}

type headerTransport struct {
	k, v string
	next http.RoundTripper
}

func (h headerTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	r = r.Clone(r.Context())
	r.Header.Set(h.k, h.v)
	return h.next.RoundTrip(r)
}

// The owner's five steps over the API: summary, compare, plan, preview,
// run with live events, then the downloads.
func TestMigrationOverTheAPI(t *testing.T) {
	m := newMigrationEnv(t, 3)
	id := m.create()
	var sum run.Summary
	m.do(http.MethodGet, api+"/migrations/"+id+"/summary", nil).json(t, &sum)
	if sum.Users != 3 || sum.Admin != "reseller" || sum.UsedTraffic != 3000 || len(sum.Targets) == 0 {
		t.Fatalf("summary %+v", sum)
	}
	checkPages(t, m, id)
	v := m.plan(id)
	var opts struct {
		Options []run.ReloadOption `json:"options"`
	}
	m.do(http.MethodGet, api+"/migrations/"+id+"/reload-options", nil).json(t, &opts)
	if len(opts.Options) != 3 || opts.Options[0].Method != run.ReloadManual {
		t.Fatalf("reload options %+v", opts)
	}
	if res := m.do(http.MethodPost, api+"/migrations/"+id+"/run", map[string]any{"planVersion": v - 1, "reload": "manual"}); res.status != http.StatusPreconditionFailed {
		t.Fatalf("run on an old version: %d", res.status)
	}
	if res := m.do(http.MethodPost, api+"/migrations/"+id+"/run", map[string]any{"planVersion": v, "reload": "manual"}); res.status != http.StatusAccepted {
		t.Fatalf("run: %d %s", res.status, res.body)
	}
	events := readEvents(t, m, id, "")
	if events[len(events)-1].Type != run.EventDone {
		t.Fatalf("last event %+v", events[len(events)-1])
	}
	if again := readEvents(t, m, id, strconv.FormatInt(events[2].Seq, 10)); again[0].Seq != events[3].Seq {
		t.Fatalf("Last-Event-ID resume started at %d", again[0].Seq)
	}
	checkDownloads(t, m, id)
}

func checkPages(t *testing.T, m *migrationEnv, id string) {
	t.Helper()
	var page struct {
		Users []map[string]any `json:"users"`
		Total int              `json:"total"`
		Next  string           `json:"next"`
	}
	m.do(http.MethodGet, api+"/migrations/"+id+"/users?limit=2", nil).json(t, &page)
	if len(page.Users) != 2 || page.Total != 3 || page.Next != "2" {
		t.Fatalf("page 1: %+v", page)
	}
	m.do(http.MethodGet, api+"/migrations/"+id+"/users?q=user001", nil).json(t, &page)
	if page.Total != 1 || page.Users[0]["sourceUsername"] != "user001" {
		t.Fatalf("search: %+v", page)
	}
	if res := m.do(http.MethodGet, api+"/migrations/"+id+"/groups", nil); res.status != http.StatusOK {
		t.Fatalf("groups: %d", res.status)
	}
	if res := m.do(http.MethodGet, api+"/migrations/nope/summary", nil); res.status != http.StatusNotFound {
		t.Fatalf("unknown job: %d", res.status)
	}
}

func checkDownloads(t *testing.T, m *migrationEnv, id string) {
	t.Helper()
	var rep run.Report
	m.do(http.MethodGet, api+"/migrations/"+id+"/report", nil).json(t, &rep)
	if rep.State != run.StateDone || rep.Counts.Moved != 3 || rep.Users[0].NewLink != "" {
		t.Fatalf("report: %+v", rep)
	}
	links := m.do(http.MethodGet, api+"/migrations/"+id+"/links.csv", nil)
	if links.status != http.StatusOK || strings.Count(string(links.body), "\n") != 4 || links.header.Get("Cache-Control") != "no-store" {
		t.Fatalf("links.csv: %d %q", links.status, links.body)
	}
	if res := m.do(http.MethodGet, api+"/migrations/"+id+"/report.csv", nil); res.status != http.StatusOK {
		t.Fatalf("report.csv: %d", res.status)
	}
	first := m.do(http.MethodGet, api+"/migrations/"+id+"/passwords.csv", nil)
	if first.status != http.StatusOK || !strings.Contains(string(first.body), "reseller,") {
		t.Fatalf("passwords.csv: %d %q", first.status, first.body)
	}
	if again := m.do(http.MethodGet, api+"/migrations/"+id+"/passwords.csv", nil); again.status != http.StatusGone {
		t.Fatalf("second passwords.csv: %d", again.status)
	}
}

// readEvents reads the event stream until the done event.
func readEvents(t *testing.T, m *migrationEnv, id, lastID string) []run.Event {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, m.srv.URL+api+"/migrations/"+id+"/events", nil)
	if lastID != "" {
		req.Header.Set("Last-Event-ID", lastID)
	}
	resp, err := m.client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.Header.Get("Content-Type") != "text/event-stream" {
		t.Fatalf("content type %q", resp.Header.Get("Content-Type"))
	}
	var out []run.Event
	sc := bufio.NewScanner(resp.Body)
	for sc.Scan() {
		data, ok := strings.CutPrefix(sc.Text(), "data: ")
		if !ok {
			continue
		}
		var e run.Event
		if err := json.Unmarshal([]byte(data), &e); err != nil {
			t.Fatal(err)
		}
		out = append(out, e)
		if e.Type == run.EventDone {
			return out
		}
	}
	t.Fatalf("stream ended without done after %d events: %v", len(out), sc.Err())
	return nil
}

func TestSafeModeNeedsTheTypedConfirmation(t *testing.T) {
	m := newMigrationEnv(t, 0)
	if res := m.do(http.MethodPut, api+"/settings/safe-mode", map[string]any{"enabled": false}); res.status != http.StatusBadRequest ||
		res.errorCode(t) != "confirm_required" {
		t.Fatalf("off without confirm: %d %s", res.status, res.body)
	}
	res := m.do(http.MethodPut, api+"/settings/safe-mode", map[string]any{"enabled": false, "confirm": "turn-off-safe-mode"})
	var got struct{ Enabled bool }
	res.json(t, &got)
	if res.status != http.StatusOK || got.Enabled {
		t.Fatalf("off: %d %s", res.status, res.body)
	}
	m.do(http.MethodGet, api+"/settings/safe-mode", nil).json(t, &got)
	if got.Enabled {
		t.Fatal("still on")
	}
	var n int
	_ = m.db.QueryRow("SELECT count(*) FROM audit_events WHERE action = 'safe_mode.off'").Scan(&n)
	if n != 1 {
		t.Fatalf("audit rows: %d", n)
	}
}

func TestMigrationRoutesNeedASession(t *testing.T) {
	e := newEnv(t)
	for _, path := range []string{"/panels", "/migrations/x/summary", "/legacy-sites", "/safe-mode/backups"} {
		if res := e.do(http.MethodGet, api+path, nil); res.status != http.StatusUnauthorized {
			t.Errorf("%s: %d", path, res.status)
		}
	}
	m := newMigrationEnv(t, 0)
	m.csrf = "wrong"
	if res := m.do(http.MethodPost, api+"/migrations", map[string]any{}); res.status != http.StatusForbidden {
		t.Fatalf("no CSRF token: %d", res.status)
	}
}

func TestTemplatesAndSites(t *testing.T) {
	m := newMigrationEnv(t, 1)
	id := m.create()
	res := m.do(http.MethodPost, api+"/templates", map[string]any{"jobId": id, "name": "monthly"})
	if res.status != http.StatusCreated {
		t.Fatalf("save template: %d %s", res.status, res.body)
	}
	if dup := m.do(http.MethodPost, api+"/templates", map[string]any{"jobId": id, "name": "monthly"}); dup.status != http.StatusConflict {
		t.Fatalf("duplicate template: %d", dup.status)
	}
	var tpl struct{ ID int64 }
	res.json(t, &tpl)
	if del := m.do(http.MethodDelete, api+"/templates/"+strconv.FormatInt(tpl.ID, 10), nil); del.status != http.StatusNoContent {
		t.Fatalf("delete template: %d", del.status)
	}
	site := map[string]any{"name": "old", "panelId": m.panelID, "sourceKind": "marzban", "sourceUrl": m.src.URL, "oldPath": "sub",
		"domains": []string{"old.example.com"}, "port": freePort(t), "lazy": true}
	created := m.do(http.MethodPost, api+"/legacy-sites", site)
	if created.status != http.StatusCreated {
		t.Fatalf("add site: %d %s", created.status, created.body)
	}
	t.Cleanup(func() { _ = m.bridge.DeleteSite(context.Background(), 1) })
	site["name"] = "again"
	if dup := m.do(http.MethodPost, api+"/legacy-sites", site); dup.status != http.StatusConflict || dup.errorCode(t) != "site_overlap" {
		t.Fatalf("overlapping site: %d %s", dup.status, dup.body)
	}
	site["oldPath"] = "a/b"
	if bad := m.do(http.MethodPost, api+"/legacy-sites", site); bad.status != http.StatusBadRequest {
		t.Fatalf("invalid site: %d", bad.status)
	}
	var list struct {
		Sites []map[string]any `json:"sites"`
	}
	m.do(http.MethodGet, api+"/legacy-sites", nil).json(t, &list)
	if len(list.Sites) != 1 {
		t.Fatalf("sites: %+v", list)
	}
}

func freePort(t *testing.T) int {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	return ln.Addr().(*net.TCPAddr).Port
}

func TestPanelsScanAndList(t *testing.T) {
	m := newMigrationEnv(t, 0)
	rows, err := m.runs.Panels(context.Background())
	if err != nil || len(rows) != 1 {
		t.Fatalf("panels %v %v", rows, err)
	}
	res := m.do(http.MethodPost, api+"/panels/scan", map[string]any{"extra": []string{rows[0].EnvPath}})
	var got struct {
		Panels []struct {
			ID    int64  `json:"id"`
			Kind  string `json:"kind"`
			Ready bool   `json:"ready"`
		} `json:"panels"`
	}
	res.json(t, &got)
	if res.status != http.StatusOK || len(got.Panels) != 1 || got.Panels[0].ID != rows[0].ID || got.Panels[0].Kind != "pasarguard" {
		t.Fatalf("scan: %d %s", res.status, res.body)
	}
	if bad := m.do(http.MethodPost, api+"/panels/scan", map[string]any{"extra": []string{"/etc/passwd"}}); bad.status != http.StatusBadRequest {
		t.Fatalf("scan of a non-ENV file: %d", bad.status)
	}
	m.do(http.MethodGet, api+"/panels", nil).json(t, &got)
	if len(got.Panels) != 1 || !got.Panels[0].Ready {
		t.Fatalf("list: %+v", got)
	}
	if strings.Contains(string(m.do(http.MethodGet, api+"/panels", nil).body), "databaseUrl") {
		t.Fatal("the panel list shows the database URL")
	}
}
