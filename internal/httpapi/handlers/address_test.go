package handlers_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/PdYrust/karvan/internal/httpapi"
	"github.com/PdYrust/karvan/internal/httpapi/address"
	"github.com/PdYrust/karvan/internal/settings"
)

type addressEnv struct {
	*env
	srv  *address.Server
	port int
}

// newAddressEnv serves the test panel through an address.Server on a free
// local port, the way karvan serve does.
func newAddressEnv(t *testing.T, ttl time.Duration) *addressEnv {
	t.Helper()
	e, d := newEnvDeps(t)
	srv := &address.Server{
		Bind: "127.0.0.1", TTL: ttl, Log: d.Logger,
		HTTP: func(h http.Handler) *http.Server { return &http.Server{Handler: h, ReadHeaderTimeout: 5 * time.Second} },
		Save: func(ctx context.Context, s address.Site) error {
			return settings.New(e.db).SetHTTP(ctx, settings.HTTP{Bind: "127.0.0.1", Port: s.Port, BasePath: s.BasePath, TLS: s.TLS})
		},
	}
	d.Address = srv
	d.PanelTLS = panelTLS(t, e)
	srv.Certificate = d.PanelTLS.Certificate
	srv.Handler = func(b string) (http.Handler, error) {
		dd := d
		dd.BasePath = b
		return httpapi.New(dd)
	}
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- srv.Serve(ctx, address.Site{Port: port, BasePath: base}, ln) }()
	t.Cleanup(func() { cancel(); <-done })
	for srv.Status().Current.Port != port {
		time.Sleep(5 * time.Millisecond)
	}
	e.url = fmt.Sprintf("http://127.0.0.1:%d", port)
	return &addressEnv{env: e, srv: srv, port: port}
}

// at is this browser on another address: the same cookies, the page's own
// CSRF token.
func (a *addressEnv) at(port int, basePath string) *env {
	b := *a.env
	b.url, b.basePath, b.csrf = fmt.Sprintf("http://127.0.0.1:%d", port), basePath, ""
	b.fetchCSRF()
	return &b
}

type addressBody struct {
	Port    int `json:"port"`
	Pending *struct {
		Port        int    `json:"port"`
		PathChanged bool   `json:"pathChanged"`
		FromNew     bool   `json:"fromNew"`
		TLS         string `json:"tls"`
		Host        string `json:"host"`
	} `json:"pending"`
}

func (e *env) address(basePath string) addressBody {
	e.t.Helper()
	var a addressBody
	res := e.do(http.MethodGet, "/"+basePath+"/api/v1/settings/address", nil)
	if res.status != http.StatusOK {
		e.t.Fatalf("address: %d %s", res.status, res.body)
	}
	res.json(e.t, &a)
	return a
}

type newAddress struct {
	Port int    `json:"port"`
	Path string `json:"path"`
}

func (a *addressEnv) start(body map[string]any) newAddress {
	a.t.Helper()
	body["password"] = password
	res := a.do(http.MethodPost, api+"/settings/address", body)
	var n newAddress
	res.json(a.t, &n)
	if res.status != http.StatusOK || res.header.Get("Cache-Control") != "no-store" {
		a.t.Fatalf("start: %d %v %s", res.status, res.header, res.body)
	}
	return n
}

func (a *addressEnv) saved() (settings.HTTP, error) {
	return settings.New(a.db).HTTP(context.Background())
}

// notFromTheOldAddress checks that a change waits and cannot be confirmed
// from the old address.
func (a *addressEnv) notFromTheOldAddress(pathChanged bool) {
	a.t.Helper()
	if got := a.address(base); got.Port != a.port || got.Pending == nil || got.Pending.PathChanged != pathChanged || got.Pending.FromNew {
		a.t.Fatalf("status at the old address = %+v", got)
	}
	res := a.do(http.MethodPost, api+"/settings/address/confirm", nil)
	if res.status != http.StatusConflict || res.errorCode(a.t) != "not_new_address" {
		a.t.Fatalf("confirm from the old address: %d %s", res.status, res.body)
	}
	if _, err := a.saved(); !errors.Is(err, settings.ErrNotInitialized) {
		a.t.Fatalf("saved before the confirm: %v", err)
	}
}

func TestNewPathIsConfirmedFromTheNewAddress(t *testing.T) {
	a := newAddressEnv(t, 0)
	a.login("")
	n := a.start(map[string]any{"newPath": true})
	if n.Port != a.port || n.Path == base || len(n.Path) != 16 {
		t.Fatalf("new address = %+v", n)
	}
	a.notFromTheOldAddress(true)

	// Over plain HTTP the session cookie belongs to the old path, so the
	// new address asks for a sign-in first.
	next := a.at(a.port, n.Path)
	if res := next.do(http.MethodPost, "/"+n.Path+"/api/v1/settings/address/confirm", nil); res.status != http.StatusUnauthorized {
		t.Fatalf("confirm without a session there: %d", res.status)
	}
	next.login("")
	if got := next.address(n.Path); got.Pending == nil || !got.Pending.FromNew {
		t.Fatalf("status at the new address = %+v", got)
	}
	if res := next.do(http.MethodPost, "/"+n.Path+"/api/v1/settings/address/confirm", nil); res.status != http.StatusNoContent {
		t.Fatalf("confirm: %d %s", res.status, res.body)
	}
	if h, err := a.saved(); err != nil || h.BasePath != n.Path || h.Port != a.port {
		t.Fatalf("saved = %+v, %v", h, err)
	}
	if got := string(a.raw(http.MethodGet, "/"+base+"/")); got != string(a.raw(http.MethodGet, "/nothing-here/")) {
		t.Fatalf("the old path still answers: %s", got)
	}
	if got := next.address(n.Path); got.Pending != nil {
		t.Fatalf("still waiting: %+v", got)
	}
}

func TestNewPortIsConfirmedFromTheNewPort(t *testing.T) {
	a := newAddressEnv(t, 0)
	a.login("")
	port := freePort(t)
	n := a.start(map[string]any{"port": port})
	if n.Port != port || n.Path != base {
		t.Fatalf("new address = %+v", n)
	}
	a.notFromTheOldAddress(false)
	// Cookies do not depend on the port, so the session works there.
	next := a.at(port, base)
	if got := next.address(base); got.Port != a.port || got.Pending == nil || got.Pending.PathChanged || !got.Pending.FromNew {
		t.Fatalf("status at the new port = %+v", got)
	}
	if res := next.do(http.MethodPost, api+"/settings/address/confirm", nil); res.status != http.StatusNoContent {
		t.Fatalf("confirm: %d %s", res.status, res.body)
	}
	if h, err := a.saved(); err != nil || h.Port != port || h.BasePath != base {
		t.Fatalf("saved = %+v, %v", h, err)
	}
	if conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", a.port), time.Second); err == nil {
		conn.Close()
		t.Fatal("the old port still listens")
	}
	if got := next.address(base); got.Port != port || got.Pending != nil {
		t.Fatalf("status after the confirm = %+v", got)
	}
}

func TestUnconfirmedAddressChangeExpires(t *testing.T) {
	a := newAddressEnv(t, 150*time.Millisecond)
	a.login("")
	port := freePort(t)
	a.start(map[string]any{"newPath": true, "port": port})
	deadline := time.Now().Add(5 * time.Second)
	for a.address(base).Pending != nil {
		if time.Now().After(deadline) {
			t.Fatal("the change did not expire")
		}
		time.Sleep(20 * time.Millisecond)
	}
	if conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", port), time.Second); err == nil {
		conn.Close()
		t.Fatal("the expired port still listens")
	}
	res := a.do(http.MethodPost, api+"/settings/address/confirm", nil)
	if res.status != http.StatusConflict || res.errorCode(t) != "no_address_change" {
		t.Fatalf("confirm after expiry: %d %s", res.status, res.body)
	}
	if _, err := a.saved(); !errors.Is(err, settings.ErrNotInitialized) {
		t.Fatalf("an expired change was saved: %v", err)
	}
}

func TestAddressChangeNeedsThePassword(t *testing.T) {
	a := newAddressEnv(t, 0)
	a.login("")
	for _, body := range []map[string]any{
		{"newPath": true},
		{"password": password},
		{"password": password, "port": a.port},
		{"password": password, "port": 70000},
		{"password": password, "port": -2},
	} {
		if res := a.do(http.MethodPost, api+"/settings/address", body); res.status != http.StatusBadRequest {
			t.Fatalf("body %v: %d %s", body, res.status, res.body)
		}
	}
	res := a.do(http.MethodPost, api+"/settings/address", map[string]any{"password": "wrong password!", "newPath": true})
	if res.status != http.StatusForbidden || res.errorCode(t) != "invalid_credentials" {
		t.Fatalf("wrong password: %d %s", res.status, res.body)
	}
	busy, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer busy.Close()
	res = a.do(http.MethodPost, api+"/settings/address", map[string]any{"password": password, "port": busy.Addr().(*net.TCPAddr).Port})
	if res.status != http.StatusConflict || res.errorCode(t) != "port_unavailable" {
		t.Fatalf("busy port: %d %s", res.status, res.body)
	}
	if got := a.address(base); got.Pending != nil {
		t.Fatalf("a refused change is waiting: %+v", got)
	}
}

func TestAddressRoutesNeedTheAddressServer(t *testing.T) {
	e := newEnv(t)
	e.login("")
	if res := e.do(http.MethodGet, api+"/settings/address", nil); res.status != http.StatusNotFound {
		t.Fatalf("without an address server: %d", res.status)
	}
	if res := newAddressEnv(t, 0).do(http.MethodGet, api+"/settings/address", nil); res.status != http.StatusUnauthorized {
		t.Fatalf("signed out: %d", res.status)
	}
}

// The new login and the new path are only in the responses that made
// them: not in the logs, not in the audit log.
func TestOneTimeValuesStayOutOfLogsAndAudit(t *testing.T) {
	logs := &lockedBuffer{}
	testLog = logs
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(logs, &slog.HandlerOptions{Level: slog.LevelDebug})))
	t.Cleanup(func() { testLog = io.Discard; slog.SetDefault(prev) })

	a := newAddressEnv(t, 0)
	a.login("")
	n := a.start(map[string]any{"newPath": true, "port": freePort(t)})
	next := a.at(n.Port, n.Path)
	next.login("")
	if res := next.do(http.MethodPost, "/"+n.Path+"/api/v1/settings/address/confirm", nil); res.status != http.StatusNoContent {
		t.Fatalf("confirm: %d %s", res.status, res.body)
	}
	var login newLogin
	next.do(http.MethodPost, "/"+n.Path+"/api/v1/account/credentials", map[string]string{"password": password}).json(t, &login)
	if login.Password == "" {
		t.Fatal("no new login")
	}
	secrets := []string{n.Path, login.Username, login.Password}
	scanFor(t, "the logs", []byte(logs.String()), secrets)
	if !strings.Contains(logs.String(), "/api/v1/settings/address/confirm") {
		t.Fatal("the access log is empty: the scan would prove nothing")
	}
	audited := auditText(t, a.env)
	scanFor(t, "the audit log", []byte(audited), secrets)
	for _, want := range []string{"settings.address_start", "settings.address ", `"port":"` + strconv.Itoa(n.Port), "account.credentials"} {
		if !strings.Contains(audited, want) {
			t.Fatalf("audit log lacks %q:\n%s", want, audited)
		}
	}
}

// auditText is every stored audit event, one line each.
func auditText(t *testing.T, e *env) string {
	t.Helper()
	rows, err := e.db.Query(`SELECT actor || ' ' || ip || ' ' || action || ' ' || detail FROM audit_events`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var b strings.Builder
	for rows.Next() {
		var line string
		if err := rows.Scan(&line); err != nil {
			t.Fatal(err)
		}
		b.WriteString(line + "\n")
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return b.String()
}
