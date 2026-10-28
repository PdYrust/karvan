package run

import (
	"context"
	"net/http"
	"testing"

	"github.com/PdYrust/karvan/internal/migration/engine"
	"github.com/PdYrust/karvan/internal/panel"
	"github.com/PdYrust/karvan/internal/panel/marzbandb"
	"github.com/PdYrust/karvan/internal/panel/paneldb/paneldbtest"
)

// checkServed checks a job from a PasarGuard fake whose users sit in
// groups 1 (WireGuard) and 2 (Hysteria), answering its groups with
// status (0: normally), and returns the build.
func checkServed(t *testing.T, status int) *engine.Result {
	t.Helper()
	tg := paneldbtest.Load(t, paneldbtest.Fixture{Flavor: "sqlite", Path: "../../panel/marzbandb/testdata/" + marzbandb.Revision + ".sqlite.sql"})
	h := newHarness(t, panel.PasarGuard, panel.Marzban, tg, 2)
	h.src.ServeGroups([]map[string]any{
		{"id": 1, "name": "wireguard", "inbound_tags": []string{"wg0"}, "is_disabled": false},
		{"id": 2, "name": "hysteria", "inbound_tags": []string{"hy"}, "is_disabled": false},
	}, []map[string]any{{"tag": "wg0", "protocol": "wireguard"}, {"tag": "hy", "protocol": "hysteria"}}, status)
	j := h.created()
	r, err := h.svc.Check(context.Background(), j.ID)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func TestOldPanelGroupsGiveC38(t *testing.T) {
	r := checkServed(t, 0)
	for _, it := range r.Items {
		if len(it.Conflicts) == 0 || it.Conflicts[0].Type != engine.WireGuardOnly {
			t.Errorf("user %s: %+v, want C38", it.Key, it.Conflicts)
		}
	}
	for _, c := range r.Conflicts {
		if c.Type == engine.GroupsUnknown {
			t.Error("groups were read")
		}
	}
}

func TestForbiddenGroupsSayUnknown(t *testing.T) {
	r := checkServed(t, http.StatusForbidden)
	for _, it := range r.Items {
		for _, c := range it.Conflicts {
			if c.Type == engine.WireGuardOnly || c.Type == engine.NoProtocol {
				t.Errorf("user %s: %s raised with unknown groups", it.Key, c.Type)
			}
		}
	}
	found := false
	for _, c := range r.Conflicts {
		found = found || c.Type == engine.GroupsUnknown
	}
	if !found {
		t.Errorf("job conflicts %+v, want %s", r.Conflicts, engine.GroupsUnknown)
	}
}
