package handlers_test

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

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
		{http.MethodPost, "/run", map[string]any{"planVersion": 0, "reload": "manual"}, http.StatusConflict, "blocked"},
		{http.MethodPost, "/rollback", nil, http.StatusConflict, "busy"},
		{http.MethodGet, "/report", nil, http.StatusNotFound, "not_found"},
		{http.MethodGet, "/passwords.csv", nil, http.StatusGone, "gone"},
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
	if err := m.runs.RestoreBackup(context.Background(), bs.Backups[0].ID); err == nil {
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
	up := m.withHeader("Content-Type", "text/plain")(http.MethodPost, path+"/tokens", nil)
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
		{http.MethodPost, "/migrations/" + j.ID + "/compare", nil, http.StatusConflict, "panel_not_ready"},
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
