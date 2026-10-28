package handlers_test

import (
	"net"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"
)

type acmeErrorOut struct {
	Error struct {
		Code   string   `json:"code"`
		Field  string   `json:"field"`
		Reason string   `json:"reason"`
		Seen   []string `json:"seen"`
	} `json:"error"`
}

func wantACME(t *testing.T, res response, status int, code, field, reason string) {
	t.Helper()
	var e acmeErrorOut
	res.json(t, &e)
	if res.status != status || e.Error.Code != code || e.Error.Field != field || e.Error.Reason != reason {
		t.Errorf("want %d %s %s %s: %d %s", status, code, field, reason, res.status, res.body)
	}
}

func acmeWay(method string, agree bool) map[string]any {
	return map[string]any{"kind": "acme", "method": method, "agreeTos": agree}
}

// orderOut is the part of an AcmeOrder the tests read.
type orderOut struct {
	ID     int64  `json:"id"`
	Method string `json:"method"`
	State  string `json:"state"`
	Error  *struct {
		Code string `json:"code"`
	} `json:"error"`
}

// waitOrder polls GET /domains/{id}/acme until the order reaches state.
func (m *migrationEnv) waitOrder(t *testing.T, path, state string) orderOut {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for {
		var st struct {
			Order *orderOut `json:"order"`
		}
		res := m.do(http.MethodGet, path, nil)
		res.json(t, &st)
		if res.status == http.StatusOK && st.Order != nil && st.Order.State == state {
			return *st.Order
		}
		if time.Now().After(deadline) {
			t.Fatalf("order: %d %s", res.status, res.body)
		}
		time.Sleep(20 * time.Millisecond)
	}
}

// Adding a domain by Let's Encrypt checks the request and the port before
// anything is stored, then answers 201 with its order in the background.
func TestAddDomainByLetsEncrypt(t *testing.T) {
	m := newMigrationEnv(t, 0)
	domains := api + "/domains"
	for _, c := range []struct {
		name, method  string
		agree         bool
		field, reason string
	}{
		{"a.example.com", "http-01", false, "agreeTos", ""},
		{"a.example.com", "manual", true, "method", ""},
		{"*.a.example.com", "http-01", true, "method", "wildcard_needs_dns"},
		{"192.0.2.1", "dns-01", true, "name", "ip"},
	} {
		wantACME(t, m.do(http.MethodPost, domains, map[string]any{"name": c.name, "certificate": acmeWay(c.method, c.agree)}),
			http.StatusBadRequest, "acme_invalid", c.field, c.reason)
	}
	ln, err := net.Listen("tcp", ":"+strconv.Itoa(m.bridge.ChallengePort))
	if err != nil {
		t.Fatal(err)
	}
	wantACME(t, m.do(http.MethodPost, domains, map[string]any{"name": "a.example.com", "certificate": acmeWay("http-01", true)}),
		http.StatusConflict, "acme_port_busy", "", "")
	_ = ln.Close()
	var n int
	if err := m.db.QueryRow("SELECT count(*) FROM domains WHERE name = 'a.example.com'").Scan(&n); err != nil || n != 0 {
		t.Fatalf("refused adds stored %d domains: %v", n, err)
	}
	res := m.do(http.MethodPost, domains, map[string]any{"name": "a.example.com", "certificate": acmeWay("dns-01", true)})
	var d struct {
		ID   int64 `json:"id"`
		ACME struct {
			Order orderOut `json:"order"`
		} `json:"acme"`
	}
	res.json(t, &d)
	if res.status != http.StatusCreated || d.ACME.Order.Method != "dns-01" {
		t.Fatalf("add: %d %s", res.status, res.body)
	}
	path := domains + "/" + strconv.FormatInt(d.ID, 10) + "/acme"
	if o := m.waitOrder(t, path, "failed"); o.Error == nil || o.Error.Code != "acme_unreachable" {
		t.Fatalf("order: %+v", o)
	}
	if ok := len(m.audited("domain.add", "success")); ok != 1 {
		t.Fatalf("add audits %d", ok)
	}
}

// The order routes: start, busy, Check, cancel and their codes.
func TestLetsEncryptOrderRoutes(t *testing.T) {
	m := newMigrationEnv(t, 0)
	wantError(t, m.do(http.MethodGet, api+"/domains/999/acme", nil), http.StatusNotFound, "not_found")
	wantError(t, m.do(http.MethodGet, api+"/domains/x/acme", nil), http.StatusBadRequest, "bad_request")
	id := m.domainID(t, "old.example.com")
	path := api + "/domains/" + strconv.FormatInt(id, 10) + "/acme"
	if res := m.do(http.MethodGet, path, nil); res.status != http.StatusOK || string(res.body) != `{"order":null,"renewal":null}`+"\n" {
		t.Fatalf("no order yet: %d %q", res.status, res.body)
	}
	wantError(t, m.do(http.MethodPost, path+"/check", map[string]any{}), http.StatusConflict, "acme_state")
	wantError(t, m.do(http.MethodDelete, path, nil), http.StatusNotFound, "not_found")
	if _, err := m.db.Exec("INSERT INTO acme_accounts (id, directory, url, created_at) VALUES (1, 'x', '', 1)"); err != nil {
		t.Fatal(err)
	}
	if _, err := m.db.Exec("INSERT INTO acme_orders (domain_id, account_id, method, state, dns_name, dns_value, created_at, "+
		"updated_at) VALUES (?, 1, 'dns-01', 'waiting_dns', '_acme-challenge.old.example.com', 'want', 1, 1)", id); err != nil {
		t.Fatal(err)
	}
	wantACME(t, m.do(http.MethodPost, path, map[string]any{"method": "dns-01"}), http.StatusConflict, "acme_busy", "", "")
	res := m.do(http.MethodPost, path+"/check", map[string]any{})
	var e acmeErrorOut
	res.json(t, &e)
	if res.status != http.StatusConflict || e.Error.Code != "acme_dns_not_found" || strings.Join(e.Error.Seen, ",") != "seen-value" {
		t.Fatalf("check: %d %s", res.status, res.body)
	}
	if res := m.do(http.MethodDelete, path, nil); res.status != http.StatusNoContent {
		t.Fatalf("cancel: %d %s", res.status, res.body)
	}
	res = m.do(http.MethodPost, path, map[string]any{"method": "dns-01", "agreeTos": true})
	var o orderOut
	res.json(t, &o)
	if res.status != http.StatusAccepted || o.Method != "dns-01" {
		t.Fatalf("start: %d %s", res.status, res.body)
	}
	m.waitOrder(t, path, "failed")
	wantError(t, m.do(http.MethodPost, path, map[string]any{"method": "dns-01", "x": 1}), http.StatusBadRequest, "bad_request")
	for action, want := range map[string]int{"acme.order": 2, "acme.check": 2, "acme.cancel": 2} {
		if got := len(m.audited(action, "success")) + len(m.audited(action, "failure")); got != want {
			t.Errorf("%s audits: %d", action, got)
		}
	}
}

// PUT certificate with the acme way starts an order and keeps the
// domain's certificate (202); another way cancels the running order.
func TestDomainCertificateByLetsEncrypt(t *testing.T) {
	m := newMigrationEnv(t, 0)
	id := m.domainID(t, "old.example.com")
	path := api + "/domains/" + strconv.FormatInt(id, 10)
	res := m.do(http.MethodPut, path+"/certificate", map[string]any{"certificate": acmeWay("dns-01", true)})
	if res.status != http.StatusAccepted || !strings.Contains(string(res.body), `"acme":{"method":""`) {
		t.Fatalf("acme way: %d %s", res.status, res.body)
	}
	m.waitOrder(t, path+"/acme", "failed")
	if _, err := m.db.Exec("INSERT INTO acme_orders (domain_id, account_id, method, state, created_at, updated_at) " +
		"SELECT domain_id, account_id, method, 'waiting_dns', 2, 2 FROM acme_orders"); err != nil {
		t.Fatal(err)
	}
	if res := m.do(http.MethodPut, path+"/certificate", map[string]any{"certificate": plain}); res.status != http.StatusOK {
		t.Fatalf("plain: %d %s", res.status, res.body)
	}
	m.waitOrder(t, path+"/acme", "cancelled")
}

// GET and PUT /settings/acme: the email is checked, keepPort80 opens the
// challenge port first and refuses when it cannot.
func TestLetsEncryptSettings(t *testing.T) {
	m := newMigrationEnv(t, 0)
	path := api + "/settings/acme"
	if res := m.do(http.MethodGet, path, nil); res.status != http.StatusOK ||
		string(res.body) != `{"staging":false,"email":"","keepPort80":false,"testDirectory":true,"hasAccount":false}`+"\n" {
		t.Fatalf("defaults: %d %s", res.status, res.body)
	}
	wantACME(t, m.do(http.MethodPut, path, map[string]any{"email": "Ops <ops@example.com>"}), http.StatusBadRequest, "acme_invalid", "email", "")
	ln, err := net.Listen("tcp", ":"+strconv.Itoa(m.bridge.ChallengePort))
	if err != nil {
		t.Fatal(err)
	}
	wantACME(t, m.do(http.MethodPut, path, map[string]any{"keepPort80": true}), http.StatusConflict, "acme_port_busy", "", "")
	_ = ln.Close()
	res := m.do(http.MethodPut, path, map[string]any{"staging": true, "email": "ops@example.com", "keepPort80": true})
	if res.status != http.StatusOK || !strings.Contains(string(res.body), `"staging":true,"email":"ops@example.com","keepPort80":true`) {
		t.Fatalf("save: %d %s", res.status, res.body)
	}
	if res := m.do(http.MethodPut, path, map[string]any{"keepPort80": false}); res.status != http.StatusOK {
		t.Fatalf("off: %d %s", res.status, res.body)
	}
	if got := len(m.audited("acme.settings", "success")); got != 2 {
		t.Errorf("settings audits: %d", got)
	}
}
