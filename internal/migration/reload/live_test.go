//go:build paneltest

package reload_test

import (
	"context"
	"crypto/ecdh"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/PdYrust/karvan/internal/migration/engine"
	"github.com/PdYrust/karvan/internal/migration/livetest"
	"github.com/PdYrust/karvan/internal/migration/migrationtest"
	"github.com/PdYrust/karvan/internal/migration/plan"
	"github.com/PdYrust/karvan/internal/migration/reload"
	"github.com/PdYrust/karvan/internal/migration/run"
	"github.com/PdYrust/karvan/internal/panel"
)

// Marzban: Karvan's rows are invisible to the running Xray until a
// reload; touch and core restart both load them, touch keeps next_plan
// and sends one "user_updated" per user; the moved admin can log in.
func TestLiveMarzban(t *testing.T) {
	l := livetest.OpenLive(t, panel.Marzban)
	hook := livetest.StartWebhook(t)
	t.Run("touch", func(t *testing.T) {
		src := migrationtest.NewSource(t, panel.Marzban, 5)
		t.Cleanup(src.Close)
		livetest.Renamed(src, "mt_", func(i int, u map[string]any) {
			marzbanUser(1, i, u)
			if i == 0 {
				u["next_plan"] = map[string]any{"data_limit": 10 << 30, "expire": 2592000, "add_remaining_traffic": false, "fire_on_either": true}
			}
		})
		m := livetest.NewMover(t, l)
		rep := m.Move(src, run.ReloadTouch, marzbanPlan("mt_admin"))
		checkReload(t, rep, 5)
		waitXray(t, rep)
		names := movedNames(rep)
		if got := hook.Count("user_updated", names, len(names)); got != len(names) {
			t.Errorf("user_updated notifications: %d, want %d", got, len(names))
		}
		if np := l.User(0, "mt_user000")["next_plan"]; np == nil {
			t.Error("touch dropped next_plan")
		}
		checkMarzbanUsers(t, l, rep, 1)
		if code := l.Login("mt_admin", livetest.AdminPassword); code != http.StatusOK {
			t.Errorf("moved admin login: %d", code)
		}
	})
	t.Run("manual-then-core-restart", func(t *testing.T) {
		src := migrationtest.NewSource(t, panel.Marzban, 4)
		t.Cleanup(src.Close)
		livetest.Renamed(src, "mc_", func(i int, u map[string]any) { marzbanUser(2, i, u) })
		m := livetest.NewMover(t, l)
		busy := pollDuring(t, l)
		rep := m.Move(src, run.ReloadManual, marzbanPlan("mc_admin"))
		busy()
		if rep.Reload.Command != "marzban restart" {
			t.Errorf("manual command %q", rep.Reload.Command)
		}
		checkInXray(t, rep, false)
		res := reload.CoreRestart{}.Reload(context.Background(), run.ReloadInput{Kind: panel.Marzban, API: l.API, Users: livetest.Targets(rep, false)})
		if res.Error != "" || res.OK != 4 {
			t.Fatalf("core restart: %+v", res)
		}
		waitXray(t, rep)
		names := movedNames(rep)
		if got := hook.Count("user_updated", names, 1); got != 0 {
			t.Errorf("core restart sent %d user notifications", got)
		}
	})
}

// marzbanUser makes source user i of test run unique and active: the
// fake's expiry is in the past and its ids repeat across fakes.
func marzbanUser(run, i int, u map[string]any) {
	u["expire"] = 1900000000
	u["proxies"] = map[string]any{
		"vless": map[string]any{"id": vlessID(run, i), "flow": ""},
		"vmess": map[string]any{"id": fmt.Sprintf("00000000-0000-4%03d-9000-%012d", run, i)},
	}
}

func vlessID(run, i int) string { return fmt.Sprintf("00000000-0000-4%03d-8000-%012d", run, i) }

func marzbanPlan(admin string) func(p *plan.Plan) {
	return func(p *plan.Plan) {
		p.Targets = migrationtest.MarzbanTargets
		p.Admin.Username = admin
	}
}

func checkReload(t *testing.T, rep *run.Report, n int) {
	t.Helper()
	if rep.Reload.Error != "" || rep.Reload.OK != n || len(rep.Reload.Failed) > 0 {
		t.Fatalf("reload: %+v", rep.Reload)
	}
}

func movedNames(rep *run.Report) []string {
	var out []string
	for _, u := range rep.Users {
		if u.Moved {
			out = append(out, u.Username)
		}
	}
	return out
}

// checkInXray checks the moved users are (or are not) in the running
// Xray's VLESS inbound. Marzban names Xray clients "<id>.<username>".
func checkInXray(t *testing.T, rep *run.Report, want bool) {
	t.Helper()
	got := livetest.XrayEmails(t, "VLESS TCP")
	for _, u := range rep.Users {
		if u.Moved && got[fmt.Sprintf("%d.%s", u.DestID, u.Username)] != want {
			t.Errorf("%s in Xray = %v, want %v", u.Username, !want, want)
		}
	}
}

// waitXray waits for a restarted Xray to answer with the users.
func waitXray(t *testing.T, rep *run.Report) {
	t.Helper()
	for range 30 {
		got := livetest.XrayEmails(t, "VLESS TCP")
		all := true
		for _, u := range rep.Users {
			all = all && (!u.Moved || got[fmt.Sprintf("%d.%s", u.DestID, u.Username)])
		}
		if all {
			return
		}
		time.Sleep(time.Second)
	}
	checkInXray(t, rep, true)
}

// checkMarzbanUsers reads every moved user back through the panel API.
func checkMarzbanUsers(t *testing.T, l *livetest.Live, rep *run.Report, run int) {
	t.Helper()
	for i, u := range rep.Users {
		got := l.User(u.DestID, u.Username)
		proxies, _ := got["proxies"].(map[string]any)
		vless, _ := proxies["vless"].(map[string]any)
		if want := vlessID(run, i); vless["id"] != want {
			t.Errorf("%s vless id %v, want %s", u.Username, vless["id"], want)
		}
	}
}

// pollDuring calls the panel every 50 ms until the returned func is
// called, and fails on any error answer: the panel must keep working
// while Karvan writes into its SQLite file.
func pollDuring(t *testing.T, l *livetest.Live) func() {
	t.Helper()
	var stop atomic.Bool
	var worst atomic.Int64
	done := make(chan int)
	go func() {
		bad := 0
		for !stop.Load() {
			start := time.Now()
			if code := l.Call(http.MethodGet, "/api/admins", nil, nil); code != http.StatusOK {
				bad++
			}
			if d := int64(time.Since(start)); d > worst.Load() {
				worst.Store(d)
			}
			time.Sleep(50 * time.Millisecond)
		}
		done <- bad
	}()
	return func() {
		stop.Store(true)
		if bad := <-done; bad > 0 {
			t.Errorf("the panel failed %d calls while Karvan wrote", bad)
		}
		t.Logf("slowest panel call while Karvan wrote: %v", time.Duration(worst.Load()))
	}
}

// PasarGuard: touch gives WireGuard users peer IPs from the panel's own
// pool and keeps next_plan; core restart (node reconnect) succeeds; the
// moved admin can log in.
func TestLivePasarGuard(t *testing.T) {
	l := livetest.OpenLive(t, panel.PasarGuard)
	group := livetest.EnsureGroups(l).WireGuard
	t.Run("touch", func(t *testing.T) {
		rep := movePasarGuard(t, l, "pt_", group, run.ReloadTouch)
		checkReload(t, rep, 3)
		checkPeerIPs(t, l, rep, true)
		if np := l.User(rep.Users[0].DestID, "")["next_plan"]; np == nil {
			t.Error("touch dropped next_plan")
		}
		if code := l.Login("pt_admin", livetest.AdminPassword); code != http.StatusOK {
			t.Errorf("moved admin login: %d", code)
		}
	})
	t.Run("manual", func(t *testing.T) {
		rep := movePasarGuard(t, l, "pm_", group, run.ReloadManual)
		checkPeerIPs(t, l, rep, false)
	})
	t.Run("core-restart", func(t *testing.T) {
		rep := movePasarGuard(t, l, "pc_", group, run.ReloadCore)
		checkReload(t, rep, 3)
		checkPeerIPs(t, l, rep, true)
	})
}

func movePasarGuard(t *testing.T, l *livetest.Live, prefix string, group int64, method string) *run.Report {
	t.Helper()
	src := migrationtest.NewSource(t, panel.PasarGuard, 3)
	t.Cleanup(src.Close)
	livetest.Renamed(src, prefix, func(i int, u map[string]any) {
		u["proxy_settings"] = map[string]any{"wireguard": wireguardKeys(t, i)}
		if i == 0 {
			u["next_plan"] = map[string]any{"user_template_id": nil, "data_limit": 5 << 30, "expire": 86400, "add_remaining_traffic": true}
		}
	})
	m := livetest.NewMover(t, l)
	return m.Move(src, method, func(p *plan.Plan) {
		p.Targets = []string{strconv.FormatInt(group, 10)}
		p.Admin.Username = prefix + "admin"
		p.Resolutions = map[string]plan.Resolution{engine.WireGuardIPs: {Option: engine.OptAccept}}
	})
}

func wireguardKeys(t *testing.T, i int) map[string]any {
	t.Helper()
	k, err := ecdh.X25519().GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	return map[string]any{
		"private_key": base64.StdEncoding.EncodeToString(k.Bytes()),
		"public_key":  base64.StdEncoding.EncodeToString(k.PublicKey().Bytes()),
		"peer_ips":    []string{fmt.Sprintf("10.99.0.%d/32", i+2)},
	}
}

// checkPeerIPs checks every moved user has (or lacks) a peer IP from the
// panel's 10.77.0.0/24 pool.
func checkPeerIPs(t *testing.T, l *livetest.Live, rep *run.Report, want bool) {
	t.Helper()
	for _, u := range rep.Users {
		got := l.User(u.DestID, "")
		ps, _ := got["proxy_settings"].(map[string]any)
		wg, _ := ps["wireguard"].(map[string]any)
		ips, _ := wg["peer_ips"].([]any)
		has := len(ips) == 1 && len(fmt.Sprint(ips[0])) > 8 && fmt.Sprint(ips[0])[:8] == "10.77.0."
		if has != want {
			t.Errorf("%s peer_ips %v, want assigned=%v", u.Username, ips, want)
		}
	}
}

// The test PasarGuard serves a WireGuard group and a VLESS group: a user
// in each gets that group's link, and a second setup adds nothing.
func TestLivePasarGuardGroups(t *testing.T) {
	l := livetest.OpenLive(t, panel.PasarGuard)
	g := livetest.EnsureGroups(l)
	before := l.Setup()
	if again := livetest.EnsureGroups(l); again != g || l.Setup() != before {
		t.Errorf("second setup: %+v and %s, want %+v and %s", again, l.Setup(), g, before)
	}
	for name, group := range map[string]int64{"wireguard": g.WireGuard, "vless": g.VLESS} {
		links := l.Links(addPasarGuardUser(t, l, "gl_"+name, group))
		if len(links) != 1 || !strings.HasPrefix(links[0], name+"://") {
			t.Errorf("%s group links %q, want one %s:// link", name, links, name)
		}
	}
}

// addPasarGuardUser adds an active user in group through the panel API,
// removed again when the test ends.
func addPasarGuardUser(t *testing.T, l *livetest.Live, name string, group int64) int64 {
	t.Helper()
	del := func() { l.Call(http.MethodDelete, "/api/user/"+name, nil, nil) }
	del()
	t.Cleanup(del)
	var u struct {
		ID int64 `json:"id"`
	}
	body := map[string]any{"username": name, "group_ids": []int64{group}, "status": "active", "proxy_settings": map[string]any{}}
	if code := l.Call(http.MethodPost, "/api/user", body, &u); code != http.StatusCreated {
		t.Fatalf("add user %s: %d", name, code)
	}
	return u.ID
}
