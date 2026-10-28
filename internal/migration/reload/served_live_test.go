//go:build paneltest

package reload_test

import (
	"net/http"
	"slices"
	"testing"

	"github.com/PdYrust/karvan/internal/migration/engine"
	"github.com/PdYrust/karvan/internal/migration/livetest"
	"github.com/PdYrust/karvan/internal/migration/plan"
	"github.com/PdYrust/karvan/internal/panel"
)

// D36 to D38 against the running panels, read as old panels over their
// API: a PasarGuard user in the WireGuard group gets C38 on the way to
// Marzban, a Marzban user under a role with only the WireGuard group gets
// C10, and one under a role with only the VLESS group gets neither. Nothing is moved.
func TestLiveCurrentProtocols(t *testing.T) {
	pg := livetest.OpenLive(t, panel.PasarGuard)
	mz := livetest.OpenLive(t, panel.Marzban)
	g := livetest.EnsureGroups(pg)
	t.Run("pasarguard-to-marzban", func(t *testing.T) {
		addPasarGuardUser(t, pg, "cp_wireguard", g.WireGuard)
		addPasarGuardUser(t, pg, "cp_vless", g.VLESS)
		r := livetest.NewMover(t, mz).Check(pg, func(*plan.Plan) {})
		wantTypes(t, r, "cp_wireguard", engine.WireGuardOnly)
		wantTypes(t, r, "cp_vless")
	})
	t.Run("marzban-to-pasarguard", func(t *testing.T) {
		addMarzbanUser(t, mz, "cp_marzban")
		m := livetest.NewMover(t, pg)
		under := func(role int64) *engine.Result {
			return m.Check(mz, func(p *plan.Plan) { p.Admin.RoleID = role })
		}
		wantTypes(t, under(livetest.EnsureRole(pg, "karvan_wireguard", g.WireGuard)), "cp_marzban", engine.NoProtocol)
		wantTypes(t, under(livetest.EnsureRole(pg, "karvan_vless", g.VLESS)), "cp_marzban")
	})
}

// wantTypes checks the C10 and C38 conflicts of the user named name.
func wantTypes(t *testing.T, r *engine.Result, name string, want ...string) {
	t.Helper()
	for _, c := range r.Conflicts {
		if c.Type == engine.GroupsUnknown {
			t.Errorf("%s: the full admin could not read the groups", name)
		}
	}
	for _, it := range r.Items {
		if it.Source.Username != name {
			continue
		}
		var got []string
		for _, c := range it.Conflicts {
			if c.Type == engine.NoProtocol || c.Type == engine.WireGuardOnly {
				got = append(got, c.Type)
			}
		}
		if !slices.Equal(got, want) {
			t.Errorf("%s: %v, want %v", name, got, want)
		}
		return
	}
	t.Errorf("%s was not read", name)
}

// addMarzbanUser adds a user on the VLESS inbound through the panel API,
// removed again when the test ends.
func addMarzbanUser(t *testing.T, l *livetest.Live, name string) {
	t.Helper()
	del := func() { l.Call(http.MethodDelete, "/api/user/"+name, nil, nil) }
	del()
	t.Cleanup(del)
	body := map[string]any{"username": name, "status": "active", "proxies": map[string]any{"vless": map[string]any{}},
		"inbounds": map[string]any{"vless": []string{"VLESS TCP"}}}
	if code := l.Call(http.MethodPost, "/api/user", body, nil); code != http.StatusOK {
		t.Fatalf("add user %s: %d", name, code)
	}
}
