package run

import (
	"context"
	"testing"
	"time"

	"github.com/PdYrust/karvan/internal/panel"
	"github.com/PdYrust/karvan/internal/panel/source"
	"github.com/PdYrust/karvan/internal/secret"
)

// The summary counts the old panel's users by expiry and protocol, names
// the destination and the admin the plan makes, and says whether the
// reseller's own password stays.
func TestSummarySumsUpTheOldPanel(t *testing.T) {
	h := newHarness(t, panel.Marzban, panel.PasarGuard, pasarGuardDB(t), 4)
	now := time.Now()
	h.src.Mutate(func(users *[]map[string]any) {
		u := *users
		u[0]["expire"] = nil
		u[1]["expire"] = now.Add(3 * 24 * time.Hour).Unix()
		u[2]["expire"] = now.Add(20 * 24 * time.Hour).Unix()
		u[3]["expire"] = now.Add(-time.Hour).Unix()
		delete(u[3]["proxies"].(map[string]any), "trojan")
	})
	j := h.job(nil)
	sum, err := h.svc.Summary(j.ID)
	if err != nil {
		t.Fatal(err)
	}
	if want := (Expiries{Expired: 1, Week: 1, Month: 1, Never: 1}); sum.Expiries != want {
		t.Errorf("expiries %+v", sum.Expiries)
	}
	if p := sum.Protocols; p["vless"] != 4 || p["vmess"] != 4 || p["trojan"] != 3 || p["shadowsocks"] != 4 || len(p) != 4 {
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
