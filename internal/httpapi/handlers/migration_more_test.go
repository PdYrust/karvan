package handlers_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/PdYrust/karvan/internal/migration/migrationtest"
	"github.com/PdYrust/karvan/internal/migration/run"
)

func TestMigrationErrors(t *testing.T) {
	m := newMigrationEnv(t, 1)
	for _, c := range []struct {
		name   string
		body   map[string]any
		status int
		code   string
	}{
		{"no login", map[string]any{"sourceUrl": m.src.URL, "panelId": m.panelID}, http.StatusBadRequest, "bad_request"},
		{"bad kind", map[string]any{"sourceUrl": m.src.URL, "sourceKind": "x-ui", "username": "a", "password": "b", "panelId": m.panelID}, http.StatusBadRequest, "bad_request"},
		{"wrong password", map[string]any{"sourceUrl": m.src.URL, "username": "reseller", "password": "nope", "panelId": m.panelID}, http.StatusBadGateway, "source_unauthorized"},
	} {
		res := m.do(http.MethodPost, api+"/migrations", c.body)
		if res.status == http.StatusCreated {
			var j struct{ ID string }
			res.json(t, &j)
			res = m.do(http.MethodPost, api+"/migrations/"+j.ID+"/compare", nil)
		}
		if res.status != c.status || res.errorCode(t) != c.code {
			t.Errorf("%s: %d %s", c.name, res.status, res.body)
		}
	}
	id := m.create()
	for _, c := range []struct {
		method, path string
		body         any
		status       int
		code         string
	}{
		{http.MethodPost, "/rollback", nil, http.StatusConflict, "busy"},
		{http.MethodGet, "/report", nil, http.StatusNotFound, "not_found"},
		{http.MethodPost, "/passwords.csv", nil, http.StatusGone, "gone"},
		{http.MethodPut, "/plan", map[string]any{}, http.StatusPreconditionRequired, "stale_plan"},
	} {
		res := m.do(c.method, api+"/migrations/"+id+c.path, c.body)
		if res.status != c.status || res.errorCode(t) != c.code {
			t.Errorf("%s %s: %d %s", c.method, c.path, res.status, res.body)
		}
	}
	bad := m.withHeader("If-Match", `"0"`)(http.MethodPut, api+"/migrations/"+id+"/plan",
		map[string]any{"tree": []map[string]any{{"id": "a", "name": "A", "ops": []map[string]any{{"kind": "fly"}}}}})
	if bad.status != http.StatusBadRequest || bad.errorCode(t) != "plan_invalid" {
		t.Errorf("invalid plan: %d %s", bad.status, bad.body)
	}
	if res := m.do(http.MethodPost, api+"/migrations/"+id+"/cancel", nil); res.status != http.StatusNoContent {
		t.Errorf("cancel an idle job: %d", res.status)
	}
	if res := m.do(http.MethodGet, api+"/migrations/nope/events", nil); res.status != http.StatusNotFound {
		t.Errorf("events of an unknown job: %d", res.status)
	}
}

func TestReadOnlyLists(t *testing.T) {
	m := newMigrationEnv(t, 2)
	id := m.create()
	var ops struct {
		Operations []struct {
			Kind   string   `json:"kind"`
			Params []string `json:"params"`
		} `json:"operations"`
	}
	m.do(http.MethodGet, api+"/operations", nil).json(t, &ops)
	if len(ops.Operations) < 20 || ops.Operations[0].Kind > ops.Operations[1].Kind {
		t.Fatalf("operations %+v", ops)
	}
	m.plan(id)
	var page struct {
		Users []struct {
			Targets []string `json:"targets"`
		} `json:"users"`
	}
	m.do(http.MethodGet, api+"/migrations/"+id+"/preview?group=ready", nil).json(t, &page)
	if len(page.Users) == 0 || len(page.Users[0].Targets) == 0 {
		t.Fatalf("preview %+v", page)
	}
	var job struct {
		State   string      `json:"state"`
		Summary run.Summary `json:"summary"`
	}
	m.do(http.MethodGet, api+"/migrations/"+id, nil).json(t, &job)
	if job.State != run.StateChecked || job.Summary.Users != 2 {
		t.Fatalf("job %+v", job)
	}
	for _, p := range []string{"/templates", "/safe-mode/backups", "/migrations/" + id + "/users?group=skipped", "/migrations/" + id + "/users?group=C10",
		"/migrations/" + id + "/users?group=category:x"} {
		if res := m.do(http.MethodGet, api+p, nil); res.status != http.StatusOK {
			t.Errorf("%s: %d %s", p, res.status, res.body)
		}
	}
	if res := m.do(http.MethodDelete, api+"/safe-mode/backups/0123456789abcdef", nil); res.status != http.StatusNotFound {
		t.Errorf("delete an unknown backup: %d", res.status)
	}
	if res := m.do(http.MethodDelete, api+"/templates/99", nil); res.status != http.StatusNotFound {
		t.Errorf("delete an unknown template: %d", res.status)
	}
}

// A problem group lists its users once their conflict is fixed, as its
// count says.
func TestFixedUsersStayInTheirGroup(t *testing.T) {
	m := newMigrationEnv(t, 2)
	m.src.Mutate(func(users *[]map[string]any) {
		(*users)[0]["proxies"].(map[string]any)["vless"] = map[string]any{"id": "00000000-0000-4000-8000-000000000000", "flow": "xtls-rprx-vision"}
	})
	id := m.create()
	res := m.do(http.MethodGet, api+"/migrations/"+id+"/plan", nil)
	var got struct {
		Plan map[string]any `json:"plan"`
	}
	res.json(t, &got)
	got.Plan["targets"] = migrationtest.PasarGuardTargets
	got.Plan["resolutions"] = map[string]any{"C11": map[string]any{"option": "accept"}}
	if out := m.withHeader("If-Match", res.header.Get("ETag"))(http.MethodPut, api+"/migrations/"+id+"/plan", got.Plan); out.status != http.StatusOK {
		t.Fatalf("put plan: %d %s", out.status, out.body)
	}
	var page struct {
		Total int `json:"total"`
		Users []struct {
			Applied []string `json:"applied"`
		} `json:"users"`
	}
	m.do(http.MethodGet, api+"/migrations/"+id+"/users?group=C11", nil).json(t, &page)
	if page.Total != 1 || !slices.Contains(page.Users[0].Applied, "fix:C11:accept") {
		t.Fatalf("group C11: %+v", page)
	}
}

// A run with Safe Mode on leaves a backup the operator can list and delete.
func TestBackupsAfterARun(t *testing.T) {
	m := newMigrationEnv(t, 2)
	id := m.create()
	v := m.plan(id)
	if res := m.do(http.MethodPost, api+"/migrations/"+id+"/run", map[string]any{"planVersion": v, "reload": "manual"}); res.status != http.StatusAccepted {
		t.Fatalf("run: %d %s", res.status, res.body)
	}
	m.runs.Wait()
	var bs struct {
		Backups []run.BackupInfo `json:"backups"`
	}
	m.do(http.MethodGet, api+"/safe-mode/backups", nil).json(t, &bs)
	if len(bs.Backups) != 1 || bs.Backups[0].JobID != id {
		t.Fatalf("backups %+v", bs)
	}
	if res := m.do(http.MethodDelete, api+"/safe-mode/backups/"+bs.Backups[0].ID, nil); res.status != http.StatusNoContent {
		t.Fatalf("delete backup: %d", res.status)
	}
	if err := m.runs.RestoreBackup(context.Background(), bs.Backups[0].ID, true); err == nil {
		t.Fatal("a deleted backup restored")
	}
}

func TestSiteChangesAndTokenUpload(t *testing.T) {
	m := newMigrationEnv(t, 1)
	site := map[string]any{"name": "old", "panelId": m.panelID, "sourceKind": "marzban", "sourceUrl": m.src.URL, "oldPath": "sub",
		"port": freePort(t), "lazy": true}
	created := m.do(http.MethodPost, api+"/legacy-sites", site)
	var s struct{ ID int64 }
	created.json(t, &s)
	path := api + "/legacy-sites/" + strconv.FormatInt(s.ID, 10)
	site["name"] = "renamed"
	if res := m.do(http.MethodPatch, path, site); res.status != http.StatusOK || !strings.Contains(string(res.body), "renamed") {
		t.Fatalf("patch: %d %s", res.status, res.body)
	}
	upload := m.withHeader("Content-Type", "text/plain")
	if up := upload(http.MethodPost, path+"/tokens", nil); up.status != http.StatusConflict || up.errorCode(t) != "plain_http_off" {
		t.Fatalf("plain http without the opt-in: %d %s", up.status, up.body)
	}
	site["plainHttpOk"] = true
	m.do(http.MethodPatch, path, site)
	up := upload(http.MethodPost, path+"/tokens", nil)
	if up.status != http.StatusOK || !strings.Contains(string(up.body), `"found":0`) {
		t.Fatalf("empty upload: %d %s", up.status, up.body)
	}
	if res := m.do(http.MethodDelete, path, nil); res.status != http.StatusNoContent {
		t.Fatalf("delete: %d", res.status)
	}
	for _, c := range []struct{ method, path string }{{http.MethodDelete, path}, {http.MethodPatch, path}, {http.MethodPost, path + "/tokens"}} {
		if res := m.do(c.method, c.path, site); res.status != http.StatusNotFound {
			t.Errorf("%s %s after delete: %d", c.method, c.path, res.status)
		}
	}
}

func TestMoreErrorCodes(t *testing.T) {
	m := newMigrationEnv(t, 1)
	broken, err := m.runs.RegisterPanel(context.Background(), "/nonexistent/panel/.env", "gone", "marzban")
	if err != nil {
		t.Fatal(err)
	}
	res := m.do(http.MethodPost, api+"/migrations", map[string]any{"sourceUrl": m.src.URL, "username": m.src.Username,
		"password": m.src.Password, "panelId": broken})
	var j struct{ ID string }
	res.json(t, &j)
	var got struct{ State string }
	m.do(http.MethodGet, api+"/migrations/"+j.ID, nil).json(t, &got)
	if got.State != run.StateCreated {
		t.Errorf("a new job: %q", got.State)
	}
	for _, c := range []struct {
		method, path string
		body         any
		status       int
		code         string
	}{
		{http.MethodGet, "/migrations/" + j.ID + "/groups", nil, http.StatusConflict, "not_checked"},
		{http.MethodGet, "/migrations/" + j.ID + "/users", nil, http.StatusConflict, "not_checked"},
		{http.MethodPost, "/templates", map[string]any{"jobId": j.ID, "name": " "}, http.StatusConflict, "template_name"},
		{http.MethodPost, "/migrations", map[string]any{"sourceUrl": m.src.URL, "username": "a", "password": "b", "panelId": 999}, http.StatusNotFound, "not_found"},
		{http.MethodGet, "/migrations/nope", nil, http.StatusNotFound, "not_found"},
		{http.MethodPost, "/migrations/nope/compare", nil, http.StatusNotFound, "not_found"},
		{http.MethodGet, "/migrations/nope/plan", nil, http.StatusNotFound, "not_found"},
		{http.MethodGet, "/migrations/nope/reload-options", nil, http.StatusNotFound, "not_found"},
		{http.MethodGet, "/migrations/nope/links.csv", nil, http.StatusNotFound, "not_found"},
		{http.MethodPost, "/migrations/nope/rollback", nil, http.StatusNotFound, "not_found"},
		{http.MethodPost, "/migrations/nope/cancel", nil, http.StatusNotFound, "not_found"},
		{http.MethodPost, "/templates", map[string]any{"jobId": "nope", "name": "x"}, http.StatusNotFound, "not_found"},
		{http.MethodDelete, "/templates/abc", nil, http.StatusBadRequest, "bad_request"},
		{http.MethodPut, "/settings/safe-mode", map[string]any{"enabled": "yes"}, http.StatusBadRequest, "bad_request"},
	} {
		res := m.do(c.method, api+c.path, c.body)
		if res.status != c.status || res.errorCode(t) != c.code {
			t.Errorf("%s %s: %d %s", c.method, c.path, res.status, res.body)
		}
	}
	if res := m.do(http.MethodPut, api+"/settings/safe-mode", map[string]any{"enabled": true}); res.status != http.StatusOK {
		t.Errorf("turn on: %d", res.status)
	}
	// The compare reads the old panel, then finds the destination unusable.
	if code := compareLog(t, m, j.ID)["code"]; code != "panel_not_ready" {
		t.Errorf("compare on a broken panel: %v", code)
	}
}

// Events resume from ?after= too, for clients that cannot set headers.
func TestEventsAfterQuery(t *testing.T) {
	m := newMigrationEnv(t, 1)
	id := m.create()
	v := m.plan(id)
	m.do(http.MethodPost, api+"/migrations/"+id+"/run", map[string]any{"planVersion": v, "reload": "manual"})
	m.runs.Wait()
	all := readEvents(t, m, id, "")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, m.srv.URL+api+"/migrations/"+id+"/events?after="+strconv.FormatInt(all[len(all)-2].Seq, 10), nil)
	resp, err := m.client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	buf := make([]byte, 4096)
	n, _ := resp.Body.Read(buf)
	if !strings.Contains(string(buf[:n]), "event: done") || strings.Count(string(buf[:n]), "id: ") != 1 {
		t.Fatalf("after: %q", buf[:n])
	}
}

// An open event stream ends when the server shuts down, so the shutdown
// does not wait for it.
func TestEventStreamEndsOnShutdown(t *testing.T) {
	m := newMigrationEnv(t, 1)
	id := m.create()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, m.srv.URL+api+"/migrations/"+id+"/events", nil)
	resp, err := m.client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	close(m.closing)
	if _, err := io.Copy(io.Discard, resp.Body); err != nil {
		t.Fatalf("stream: %v", err)
	}
	if ctx.Err() != nil {
		t.Fatal("the stream outlived the shutdown")
	}
}

func TestUnknownReloadIsABadRequest(t *testing.T) {
	m := newMigrationEnv(t, 1)
	id := m.create()
	v := m.plan(id)
	if res := m.do(http.MethodPost, api+"/migrations/"+id+"/run", map[string]any{"planVersion": v, "reload": "fly"}); res.status != http.StatusBadRequest {
		t.Fatalf("%d %s", res.status, res.body)
	}
}

// A typed admin password that breaks the destination's policy says which
// rule it broke, so the page can say what to change.
func TestWeakPasswordNamesTheRule(t *testing.T) {
	m := newMigrationEnv(t, 1)
	id := m.create()
	v := m.plan(id)
	for pw, rule := range map[string]string{"Short1!": "password_short", "longbutonlylowercase": "password_weak"} {
		res := m.do(http.MethodPost, api+"/migrations/"+id+"/run", map[string]any{"planVersion": v, "reload": "manual", "adminPassword": pw})
		var body struct {
			Error struct{ Code, Rule string } `json:"error"`
		}
		res.json(t, &body)
		if res.status != http.StatusBadRequest || body.Error.Code != "weak_password" || body.Error.Rule != rule {
			t.Errorf("%s: %d %s", pw, res.status, res.body)
		}
	}
}

// An old panel with a certificate Karvan cannot verify answers
// source_tls_untrusted with the certificate's fingerprint, subject and
// expiry, for the operator to compare before trusting it: at once when
// Karvan signs in to tell the panel's kind, else on the compare's stream.
func TestUntrustedCertificateIsDescribed(t *testing.T) {
	m := newMigrationEnv(t, 0)
	srv := httptest.NewTLSServer(http.NotFoundHandler())
	defer srv.Close()
	login := map[string]any{"sourceUrl": srv.URL, "username": "admin", "password": "secret-pass", "panelId": m.panelID}
	res := m.do(http.MethodPost, api+"/migrations", login)
	var body struct {
		Error map[string]any `json:"error"`
	}
	res.json(t, &body)
	sum := sha256.Sum256(srv.Certificate().Raw)
	want := map[string]any{"code": "source_tls_untrusted", "sha256": hex.EncodeToString(sum[:]),
		"subject": srv.Certificate().Subject.String(), "notAfter": srv.Certificate().NotAfter.UTC().Format(time.RFC3339)}
	if res.status != http.StatusBadGateway || fmt.Sprint(body.Error) != fmt.Sprint(want) {
		t.Fatalf("detect: %d %s", res.status, res.body)
	}
	login["sourceKind"] = "marzban" // nothing is called until the compare
	res = m.do(http.MethodPost, api+"/migrations", login)
	var job struct{ ID string }
	res.json(t, &job)
	if got := compareLog(t, m, job.ID); fmt.Sprint(got) != fmt.Sprint(want) {
		t.Fatalf("compare: %v", got)
	}
}

// The rollback choice is a Safe Mode setting; a run with "backup" needs
// Karvan to be able to stop and start the panel, and says why it cannot.
func TestRollbackSettingAndPanelControl(t *testing.T) {
	m := newMigrationEnv(t, 1)
	res := m.do(http.MethodPut, api+"/settings/safe-mode", map[string]any{"enabled": true, "rollback": "backup"})
	var sm struct {
		Enabled  bool   `json:"enabled"`
		Rollback string `json:"rollback"`
	}
	res.json(t, &sm)
	if res.status != http.StatusOK || !sm.Enabled || sm.Rollback != "backup" {
		t.Fatalf("set: %d %s", res.status, res.body)
	}
	if bad := m.do(http.MethodPut, api+"/settings/safe-mode", map[string]any{"enabled": true, "rollback": "both"}); bad.status != http.StatusBadRequest {
		t.Fatalf("bad rollback: %d", bad.status)
	}
	id := m.create()
	v := m.plan(id)
	run := m.do(http.MethodPost, api+"/migrations/"+id+"/run", map[string]any{"planVersion": v, "reload": "manual"})
	if run.status != http.StatusConflict || run.errorCode(t) != "panel_control_unavailable" || !strings.Contains(string(run.body), `"reason"`) {
		t.Fatalf("run without panel control: %d %s", run.status, run.body)
	}
	var panels struct {
		Panels []struct {
			Control struct {
				Available bool   `json:"available"`
				Reason    string `json:"reason"`
			} `json:"control"`
		} `json:"panels"`
	}
	m.do(http.MethodGet, api+"/panels", nil).json(t, &panels)
	if len(panels.Panels) != 1 || panels.Panels[0].Control.Available || panels.Panels[0].Control.Reason == "" {
		t.Fatalf("panels %+v", panels)
	}
	var n int
	_ = m.db.QueryRow("SELECT count(*) FROM audit_events WHERE action = 'safe_mode.rollback'").Scan(&n)
	if n != 1 {
		t.Fatalf("audit rows: %d", n)
	}
}

func TestRestoreNeedsTheTypedID(t *testing.T) {
	m := newMigrationEnv(t, 1)
	path := api + "/safe-mode/backups/0123456789abcdef/restore"
	if res := m.do(http.MethodPost, path, map[string]any{"confirm": "yes"}); res.status != http.StatusBadRequest || res.errorCode(t) != "confirm_required" {
		t.Fatalf("wrong confirm: %d %s", res.status, res.body)
	}
	if res := m.do(http.MethodPost, path, map[string]any{"confirm": "0123456789abcdef"}); res.status != http.StatusNotFound {
		t.Fatalf("unknown backup: %d %s", res.status, res.body)
	}
}

// The final admin step lists PasarGuard's roles with their groups, and
// users go to the chosen role's groups when no targets were picked.
func TestRolesAndDefaultTargets(t *testing.T) {
	m := newMigrationEnv(t, 2)
	id := m.create()
	var got struct {
		RoleID int64 `json:"roleId"`
		Roles  []struct {
			ID        int64 `json:"id"`
			Name      string
			AllGroups bool `json:"allGroups"`
			Groups    []struct {
				ID        string   `json:"id"`
				Protocols []string `json:"protocols"`
			} `json:"groups"`
		} `json:"roles"`
	}
	m.do(http.MethodGet, api+"/migrations/"+id+"/roles", nil).json(t, &got)
	if got.RoleID != 3 || len(got.Roles) != 3 || got.Roles[2].Name != "operator" || !got.Roles[2].AllGroups ||
		len(got.Roles[2].Groups) != 1 || got.Roles[2].Groups[0].ID != "1" {
		t.Fatalf("roles %+v", got)
	}
	var sum run.Summary
	m.do(http.MethodGet, api+"/migrations/"+id+"/summary", nil).json(t, &sum)
	if strings.Join(sum.UserTargets, ",") != "1" || sum.RoleID != 3 || sum.Blockers != 0 {
		t.Fatalf("summary %+v", sum)
	}
}
