package handlers_test

import (
	"net/http"
	"testing"
)

func (m *migrationEnv) deleteAudits(outcome string) int {
	m.t.Helper()
	var n int
	if err := m.db.QueryRow("SELECT count(*) FROM audit_events WHERE action = 'migration.delete' AND outcome = ?", outcome).Scan(&n); err != nil {
		m.t.Fatal(err)
	}
	return n
}

func TestDeleteADraft(t *testing.T) {
	m := newMigrationEnv(t, 2)
	id := m.create()
	path := api + "/migrations/" + id
	token := m.csrf
	m.csrf = ""
	if res := m.do(http.MethodDelete, path, nil); res.status != http.StatusForbidden || res.errorCode(t) != "csrf" {
		t.Fatalf("without the CSRF token: %d %s", res.status, res.body)
	}
	m.csrf = token
	if res := m.do(http.MethodDelete, path, nil); res.status != http.StatusNoContent {
		t.Fatalf("delete: %d %s", res.status, res.body)
	}
	if res := m.do(http.MethodGet, path, nil); res.status != http.StatusNotFound {
		t.Errorf("the deleted draft still answers: %d", res.status)
	}
	if res := m.do(http.MethodDelete, path, nil); res.status != http.StatusNotFound || res.errorCode(t) != "not_found" {
		t.Errorf("delete again: %d %s", res.status, res.body)
	}
	if n := m.deleteAudits("success"); n != 1 {
		t.Errorf("%d successful deletes audited", n)
	}
}

func TestDeleteRefusesAJobThatRan(t *testing.T) {
	m := newMigrationEnv(t, 2)
	id := m.create()
	v := m.plan(id)
	if res := m.do(http.MethodPost, api+"/migrations/"+id+"/run", map[string]any{"planVersion": v, "reload": "manual"}); res.status != http.StatusAccepted {
		t.Fatalf("run: %d %s", res.status, res.body)
	}
	m.runs.Wait()
	if res := m.do(http.MethodDelete, api+"/migrations/"+id, nil); res.status != http.StatusConflict || res.errorCode(t) != "not_deletable" {
		t.Fatalf("delete a done job: %d %s", res.status, res.body)
	}
	if res := m.do(http.MethodGet, api+"/migrations/"+id+"/report", nil); res.status != http.StatusOK {
		t.Errorf("the report went: %d", res.status)
	}
	if n := m.deleteAudits("failure"); n != 1 {
		t.Errorf("%d refused deletes audited", n)
	}
}
