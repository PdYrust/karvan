package handlers_test

import (
	"net/http"
	"strconv"
	"strings"
	"testing"

	"github.com/PdYrust/karvan/internal/migration/batch"
)

// doneBatch runs a job of m's users to done and returns the job and the
// batch it became.
func doneBatch(t *testing.T, m *migrationEnv) (string, batch.Batch) {
	t.Helper()
	id := m.create()
	v := m.plan(id)
	if res := m.do(http.MethodPost, api+"/migrations/"+id+"/run", map[string]any{"planVersion": v, "reload": "manual"}); res.status != http.StatusAccepted {
		t.Fatalf("run: %d %s", res.status, res.body)
	}
	m.runs.Wait()
	var list struct {
		Batches []batch.Batch `json:"batches"`
	}
	m.do(http.MethodGet, api+"/batches", nil).json(t, &list)
	if len(list.Batches) != 1 || list.Batches[0].JobID != id {
		t.Fatalf("batches %+v", list.Batches)
	}
	return id, list.Batches[0]
}

func (m *migrationEnv) exec(q string, args ...any) {
	m.t.Helper()
	if _, err := m.db.Exec(q, args...); err != nil {
		m.t.Fatalf("%s: %v", q, err)
	}
}

func (m *migrationEnv) audited(action, outcome string) []string {
	m.t.Helper()
	rows, err := m.db.Query("SELECT detail FROM audit_events WHERE action = ? AND outcome = ? ORDER BY id", action, outcome)
	if err != nil {
		m.t.Fatal(err)
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var d string
		if err := rows.Scan(&d); err != nil {
			m.t.Fatal(err)
		}
		out = append(out, d)
	}
	if err := rows.Err(); err != nil {
		m.t.Fatal(err)
	}
	return out
}

func wantError(t *testing.T, res response, status int, code string) {
	t.Helper()
	if res.status != status || res.errorCode(t) != code {
		t.Fatalf("want %d %s, got %d %s", status, code, res.status, res.body)
	}
}

func TestBatchListAndGet(t *testing.T) {
	m := newMigrationEnv(t, 3)
	jobID, b := doneBatch(t, m)
	if b.Name != "reseller" || b.State != "active" || b.Counts.Users != 3 || b.Panel.ID != m.panelID || len(b.Admins) != 1 {
		t.Fatalf("batch %+v", b)
	}
	path := api + "/batches/" + strconv.FormatInt(b.ID, 10)
	var got batch.Batch
	if res := m.do(http.MethodGet, path, nil); res.status != http.StatusOK {
		t.Fatalf("get: %d %s", res.status, res.body)
	} else {
		res.json(t, &got)
	}
	if got.ID != b.ID || got.JobID != jobID {
		t.Fatalf("get %+v", got)
	}
	for _, q := range []string{"?state=deleted", "?state=x", "?limit=0", "?limit=101", "?panel=x", "?panel=0", "?cursor=!"} {
		wantError(t, m.do(http.MethodGet, api+"/batches"+q, nil), http.StatusBadRequest, "bad_request")
	}
	var page struct {
		Batches []batch.Batch `json:"batches"`
		Next    string        `json:"next"`
	}
	m.do(http.MethodGet, api+"/batches?state=inactive&panel="+strconv.FormatInt(m.panelID, 10), nil).json(t, &page)
	if len(page.Batches) != 0 || page.Next != "" {
		t.Fatalf("inactive %+v", page)
	}
	wantError(t, m.do(http.MethodGet, api+"/batches/999", nil), http.StatusNotFound, "not_found")
	wantError(t, m.do(http.MethodGet, api+"/batches/x", nil), http.StatusBadRequest, "bad_request")
}

// The migration names its batch, on its row and on its own.
func TestMigrationsNameTheirBatch(t *testing.T) {
	m := newMigrationEnv(t, 1)
	jobID, b := doneBatch(t, m)
	var jobs struct {
		Jobs []struct {
			Batch *batch.Ref `json:"batch"`
		} `json:"jobs"`
	}
	m.do(http.MethodGet, api+"/migrations", nil).json(t, &jobs)
	if len(jobs.Jobs) != 1 || jobs.Jobs[0].Batch == nil || jobs.Jobs[0].Batch.ID != b.ID {
		t.Fatalf("job rows %+v", jobs)
	}
	var one struct {
		Batch *batch.Ref `json:"batch"`
	}
	m.do(http.MethodGet, api+"/migrations/"+jobID, nil).json(t, &one)
	if one.Batch == nil || *one.Batch != (batch.Ref{ID: b.ID, Name: "reseller", State: "active"}) {
		t.Fatalf("job batch %+v", one.Batch)
	}
}

// The user list says who is in the batch and never carries a token or
// MAC.
func TestBatchUsers(t *testing.T) {
	m := newMigrationEnv(t, 3)
	_, b := doneBatch(t, m)
	m.exec(`INSERT INTO legacy_sites (id, name, panel_id, source_kind, source_url, old_path, listen_port, created_at, updated_at)
		VALUES (1, 'old', ?, 'marzban', 'https://old.example', 'sub', 443, 1, 1)`, m.panelID)
	m.exec(`INSERT INTO legacy_tokens (mac, site_id, ledger_id, issued_at, verified_at, checked_at)
		SELECT x'0102030405', 1, min(id), 11, 12, 13 FROM ledger WHERE batch_id = ?`, b.ID)
	path := api + "/batches/" + strconv.FormatInt(b.ID, 10) + "/users"
	res := m.do(http.MethodGet, path+"?q=user001", nil)
	var page struct {
		Users []batch.User `json:"users"`
	}
	res.json(t, &page)
	if len(page.Users) != 1 || page.Users[0].SourceUsername != "user001" || page.Users[0].Joined != "moved" || page.Users[0].Admin == "" {
		t.Fatalf("users %s", res.body)
	}
	all := m.do(http.MethodGet, path, nil)
	body := strings.ToLower(string(all.body))
	for _, bad := range []string{"mac", "token", "0102030405", "issued", "aqidbau"} {
		if strings.Contains(body, bad) {
			t.Errorf("%q in %s", bad, all.body)
		}
	}
	if !strings.Contains(string(all.body), `"verified":1`) {
		t.Errorf("verified count missing: %s", all.body)
	}
	for _, q := range []string{"?limit=0", "?limit=101", "?cursor=x", "?q=" + strings.Repeat("a", 257)} {
		wantError(t, m.do(http.MethodGet, path+q, nil), http.StatusBadRequest, "bad_request")
	}
	wantError(t, m.do(http.MethodGet, api+"/batches/999/users", nil), http.StatusNotFound, "not_found")
}

// Show new link: built on request, never cached, only for a user of the
// batch, audited without the link.
func TestBatchShowLink(t *testing.T) {
	m := newMigrationEnv(t, 2)
	_, b := doneBatch(t, m)
	var users struct {
		Users []batch.User `json:"users"`
	}
	bp := api + "/batches/" + strconv.FormatInt(b.ID, 10)
	m.do(http.MethodGet, bp+"/users", nil).json(t, &users)
	path := bp + "/users/" + strconv.FormatInt(users.Users[0].ID, 10) + "/link"
	res := m.do(http.MethodPost, path, nil)
	var got struct {
		URL string `json:"url"`
	}
	res.json(t, &got)
	if res.status != http.StatusOK || strings.Count(got.URL, "/") < 2 || res.header.Get("Cache-Control") != "no-store" {
		t.Fatalf("link: %d %s %q", res.status, res.body, res.header.Get("Cache-Control"))
	}
	entries := m.audited("batch.show_link", "success")
	token := got.URL[strings.LastIndex(got.URL, "/")+1:]
	if len(entries) != 1 || len(token) < 10 || strings.Contains(entries[0], token) || strings.Contains(entries[0], "url") ||
		!strings.Contains(entries[0], strconv.FormatInt(users.Users[0].ID, 10)) {
		t.Fatalf("audit %v", entries)
	}
	var admin int64
	if err := m.db.QueryRow("SELECT id FROM ledger WHERE kind = 'admin'").Scan(&admin); err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{bp + "/users/999/link", bp + "/users/" + strconv.FormatInt(admin, 10) + "/link", api + "/batches/999/users/1/link"} {
		wantError(t, m.do(http.MethodPost, p, nil), http.StatusNotFound, "not_found")
	}
	if got := m.do(http.MethodGet, path, nil); got.status != http.StatusNotFound && got.status != http.StatusMethodNotAllowed {
		t.Fatalf("GET link: %d", got.status)
	}
	m.csrf = ""
	wantError(t, m.do(http.MethodPost, path, nil), http.StatusForbidden, "csrf")
}

func TestBatchRename(t *testing.T) {
	m := newMigrationEnv(t, 1)
	_, b := doneBatch(t, m)
	path := api + "/batches/" + strconv.FormatInt(b.ID, 10)
	var got batch.Batch
	res := m.do(http.MethodPatch, path, map[string]any{"name": "  spring  "})
	res.json(t, &got)
	if res.status != http.StatusOK || got.Name != "spring" {
		t.Fatalf("rename: %d %s", res.status, res.body)
	}
	for _, body := range []any{map[string]any{"name": ""}, map[string]any{"name": "a\u0007b"}, map[string]any{"name": strings.Repeat("n", 65)}, map[string]any{}} {
		res := m.do(http.MethodPatch, path, body)
		wantError(t, res, http.StatusBadRequest, "batch_invalid")
		if !strings.Contains(string(res.body), `"field":"name"`) {
			t.Fatalf("field: %s", res.body)
		}
	}
	wantError(t, m.do(http.MethodPatch, path, map[string]any{"name": "x", "state": "active"}), http.StatusBadRequest, "bad_request")
	wantError(t, m.do(http.MethodPatch, api+"/batches/999", map[string]any{"name": "x"}), http.StatusNotFound, "not_found")
	if n := len(m.audited("batch.rename", "success")); n != 1 {
		t.Fatalf("rename audits %d", n)
	}
	if n := len(m.audited("batch.rename", "failure")); n < 4 {
		t.Fatalf("failed rename audits %d", n)
	}
	m.csrf = ""
	wantError(t, m.do(http.MethodPatch, path, map[string]any{"name": "y"}), http.StatusForbidden, "csrf")
}

func TestBatchDelete(t *testing.T) {
	m := newMigrationEnv(t, 2)
	_, b := doneBatch(t, m)
	path := api + "/batches/" + strconv.FormatInt(b.ID, 10)
	wantError(t, m.do(http.MethodDelete, path, map[string]any{"confirm": b.Name}), http.StatusConflict, "batch_active")
	m.exec("UPDATE batches SET state = 'inactive', inactive_reason = 'restoring' WHERE id = ?", b.ID)
	wantError(t, m.do(http.MethodDelete, path, map[string]any{"confirm": b.Name}), http.StatusConflict, "busy")
	m.exec("UPDATE batches SET inactive_reason = 'rolled_back' WHERE id = ?", b.ID)
	m.exec(`INSERT INTO legacy_sites (id, name, panel_id, source_kind, source_url, old_path, listen_port, mode, created_at, updated_at)
		VALUES (4, 'rule', ?, 'marzban', 'https://old.example', 'sub', 443, 'batches', 1, 1)`, m.panelID)
	m.exec("INSERT INTO legacy_site_batches (site_id, batch_id) VALUES (4, ?)", b.ID)
	res := m.do(http.MethodDelete, path, map[string]any{"confirm": b.Name})
	wantError(t, res, http.StatusConflict, "batch_in_use")
	if !strings.Contains(string(res.body), `"rules":[{"id":4,"name":"rule"}]`) {
		t.Fatalf("rules: %s", res.body)
	}
	m.exec("DELETE FROM legacy_site_batches")
	res = m.do(http.MethodDelete, path, map[string]any{"confirm": "nope"})
	wantError(t, res, http.StatusBadRequest, "confirm_required")
	if !strings.Contains(string(res.body), `"confirm":"`+b.Name+`"`) {
		t.Fatalf("confirm: %s", res.body)
	}
	wantError(t, m.do(http.MethodDelete, path, nil), http.StatusBadRequest, "bad_request")
	token := m.csrf
	m.csrf = ""
	wantError(t, m.do(http.MethodDelete, path, map[string]any{"confirm": b.Name}), http.StatusForbidden, "csrf")
	m.csrf = token
	if res := m.do(http.MethodDelete, path, map[string]any{"confirm": b.Name}); res.status != http.StatusNoContent {
		t.Fatalf("delete: %d %s", res.status, res.body)
	}
	wantError(t, m.do(http.MethodGet, path, nil), http.StatusNotFound, "not_found")
	wantError(t, m.do(http.MethodGet, path+"/users", nil), http.StatusNotFound, "not_found")
	var list struct {
		Batches []batch.Batch `json:"batches"`
	}
	m.do(http.MethodGet, api+"/batches", nil).json(t, &list)
	if len(list.Batches) != 0 {
		t.Fatalf("a deleted batch listed: %+v", list.Batches)
	}
	if ok, failed := len(m.audited("batch.delete", "success")), len(m.audited("batch.delete", "failure")); ok != 1 || failed < 4 {
		t.Fatalf("delete audits %d ok %d failed", ok, failed)
	}
}

func TestBatchReactivate(t *testing.T) {
	m := newMigrationEnv(t, 1)
	_, b := doneBatch(t, m)
	path := api + "/batches/" + strconv.FormatInt(b.ID, 10) + "/reactivate"
	wantError(t, m.do(http.MethodPost, path, nil), http.StatusConflict, "not_reactivatable")
	m.exec("UPDATE batches SET state = 'inactive', inactive_reason = 'restore_failed', inactive_at = 5 WHERE id = ?", b.ID)
	res := m.do(http.MethodPost, path, nil)
	var got batch.Batch
	res.json(t, &got)
	if res.status != http.StatusOK || got.State != "active" || got.InactiveReason != "" {
		t.Fatalf("reactivate: %d %s", res.status, res.body)
	}
	wantError(t, m.do(http.MethodPost, api+"/batches/999/reactivate", nil), http.StatusNotFound, "not_found")
	if ok, failed := len(m.audited("batch.reactivate", "success")), len(m.audited("batch.reactivate", "failure")); ok != 1 || failed != 2 {
		t.Fatalf("reactivate audits %d ok %d failed", ok, failed)
	}
	m.csrf = ""
	wantError(t, m.do(http.MethodPost, path, nil), http.StatusForbidden, "csrf")
}

func TestBatchRoutesNeedASession(t *testing.T) {
	e := newEnv(t)
	for _, path := range []string{"/batches", "/batches/1", "/batches/1/users"} {
		if res := e.do(http.MethodGet, api+path, nil); res.status != http.StatusUnauthorized {
			t.Errorf("%s: %d", path, res.status)
		}
	}
}
