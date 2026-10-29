//go:build paneltest

package bridge_test

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/PdYrust/karvan/internal/bridge"
	"github.com/PdYrust/karvan/internal/migration/livetest"
	"github.com/PdYrust/karvan/internal/migration/migrationtest"
	"github.com/PdYrust/karvan/internal/migration/plan"
	"github.com/PdYrust/karvan/internal/migration/run"
	"github.com/PdYrust/karvan/internal/panel"
	"github.com/PdYrust/karvan/internal/panel/subtoken"
)

// An old Marzban link reaches the running Marzban through the bridge and
// gets the moved user's real subscription; a revoke on the new panel
// closes it.
func TestLiveBridgeMarzban(t *testing.T) {
	l := livetest.OpenLive(t, panel.Marzban)
	src := migrationtest.NewSource(t, panel.Marzban, 2)
	t.Cleanup(src.Close)
	livetest.Renamed(src, "mb_", func(i int, u map[string]any) {
		u["expire"] = 1900000000
		u["proxies"] = map[string]any{"vless": map[string]any{"id": vlessID(i), "flow": ""}}
	})
	m := livetest.NewMover(t, l)
	rep := m.Move(src, run.ReloadTouch, func(p *plan.Plan) {
		p.Admin.Username = "mb_admin"
	})
	b, port := startBridge(t, m, src)
	tok := subtoken.Marzban("mb_user000", "old-secret", time.Now().Add(-time.Minute))
	src.Subscribe(tok, "mb_user000")

	code, hdr, body := oldLink(t, port, tok, "v2rayNG/1.8.5")
	links, _ := base64.StdEncoding.DecodeString(body)
	if code != http.StatusOK || !strings.Contains(string(links), vlessID(0)) || hdr.Get("Subscription-Userinfo") == "" {
		t.Fatalf("old link: %d %v %q", code, hdr, links)
	}
	if code, _, _ := oldLink(t, port, tok, "v2rayNG/1.8.5"); code != http.StatusOK {
		t.Fatalf("second request: %d", code)
	}
	name := rep.Users[0].Username
	if code := l.Call(http.MethodPost, "/api/user/"+url.PathEscape(name)+"/revoke_sub", nil, nil); code != http.StatusOK {
		t.Fatalf("revoke on the new panel: %d", code)
	}
	if code, _, _ := oldLink(t, port, tok, "v2rayNG/1.8.5"); code != http.StatusNotFound {
		t.Fatalf("after revoke: %d", code)
	}
	if n, err := b.Sweep(context.Background()); err != nil || n != 0 {
		t.Fatalf("sweep after the request closed it: %d %v", n, err)
	}
}

func vlessID(i int) string { return fmt.Sprintf("00000000-0000-4b00-8000-%012d", i) }

func startBridge(t *testing.T, m *livetest.Mover, src migrationtest.Source) (*bridge.Bridge, int) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	_ = ln.Close()
	b := bridge.New(bridge.Deps{DB: m.DB, Key: m.Key})
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { _ = b.Run(ctx); close(done) }()
	t.Cleanup(func() { cancel(); <-done })
	if _, err := b.AddDomain(context.Background(), "old.example.com", bridge.CertChoice{Kind: bridge.ChoiceNone, PlainOK: true}); err != nil {
		t.Fatal(err)
	}
	if _, err := b.SaveSite(context.Background(), bridge.Site{Name: "old", PanelID: m.Panel, SourceKind: src.Kind,
		SourceURL: src.URL, SourceIP: "127.0.0.1", OldPath: "sub", Domains: []string{"old.example.com"}, Port: port,
		Lazy: true, PlainHTTPOK: true}); err != nil {
		t.Fatal(err)
	}
	return b, port
}

func oldLink(t *testing.T, port int, token, agent string) (int, http.Header, string) {
	t.Helper()
	req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("http://127.0.0.1:%d/sub/%s", port, token), nil)
	req.Host = "old.example.com"
	req.Header.Set("User-Agent", agent)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, resp.Header, string(b)
}
