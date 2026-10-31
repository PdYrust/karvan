package run

import (
	"context"
	"testing"
	"time"

	"github.com/PdYrust/karvan/internal/panel"
	"github.com/PdYrust/karvan/internal/panel/source"
	"github.com/PdYrust/karvan/internal/secret"
)

// The summary counts the old panel's users by expiry (an on hold user's
// time starts at its first connection, so it is not "no expiry") and
// protocol, names the destination and the admin the plan makes, and says
// whether the reseller's own password stays.
func TestSummarySumsUpTheOldPanel(t *testing.T) {
	h := newHarness(t, panel.Marzban, panel.PasarGuard, pasarGuardDB(t), 5)
	now := time.Now()
	h.src.Mutate(func(users *[]map[string]any) {
		u := *users
		u[0]["expire"] = nil
		u[1]["expire"] = now.Add(3 * 24 * time.Hour).Unix()
		u[2]["expire"] = now.Add(20 * 24 * time.Hour).Unix()
		u[3]["expire"] = now.Add(-time.Hour).Unix()
		delete(u[3]["proxies"].(map[string]any), "trojan")
		u[4]["expire"], u[4]["status"], u[4]["on_hold_expire_duration"] = nil, "on_hold", 7*24*3600
	})
	j := h.job(nil)
	sum, err := h.svc.Summary(j.ID)
	if err != nil {
		t.Fatal(err)
	}
	if want := (Expiries{Expired: 1, Week: 1, Month: 1, OnHold: 1, Never: 1}); sum.Expiries != want {
		t.Errorf("expiries %+v", sum.Expiries)
	}
	if p := sum.Protocols; p["vless"] != 5 || p["vmess"] != 5 || p["trojan"] != 4 || p["shadowsocks"] != 5 || len(p) != 4 {
		t.Errorf("protocols %v", p)
	}
	if sum.PanelID != h.panel || !sum.CanKeepPassword || sum.NewAdmin.Username != "reseller" || sum.NewAdmin.Merged {
		t.Errorf("summary %+v", sum)
	}
}

// Signed in with an API key, Karvan has no password to keep.
func TestSummaryWithAnAPIKeyKeepsNoPassword(t *testing.T) {
	h := newHarness(t, panel.PasarGuard, panel.PasarGuard, pasarGuardDB(t), 2)
	ctx := context.Background()
	j, err := h.svc.Create(ctx, Login{URL: h.src.URL, Kind: panel.PasarGuard, APIKey: secret.NewValue(h.src.APIKey)}, h.panel,
		source.Options{Sleep: func(context.Context, time.Duration) error { return nil }})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := h.svc.Check(ctx, j.ID); err != nil {
		t.Fatal(err)
	}
	if sum, err := h.svc.Summary(j.ID); err != nil || sum.CanKeepPassword {
		t.Fatalf("summary %+v %v", sum, err)
	}
}

// Summary and roles answer per source admin, the login's first.
func TestSummaryAndRolesPerAdmin(t *testing.T) {
	tg := pasarGuardDB(t)
	h := newHarness(t, panel.Marzban, panel.PasarGuard, tg, 6)
	giveToNorth(h.src)
	j := h.job(nil)
	sum, err := h.svc.Summary(j.ID)
	if err != nil {
		t.Fatal(err)
	}
	roles, err := h.svc.Roles(j.ID)
	if err != nil {
		t.Fatal(err)
	}
	checkAdminInfos(t, sum.Admins)
	checkAdminInfos(t, roles.Admins)
}

func checkAdminInfos(t *testing.T, got []AdminInfo) {
	t.Helper()
	if len(got) != 2 || got[0].Username != "reseller" || !got[0].IsLogin || got[0].Users != 3 {
		t.Fatalf("login %+v", got)
	}
	if n := got[1]; n.Username != "north" || n.IsLogin || n.Users != 3 || !n.Moves || n.NewAdmin.Username != "north" ||
		n.RoleID == 0 || n.RoleFixed || len(n.UserTargets) == 0 {
		t.Fatalf("north %+v", n)
	}
}
