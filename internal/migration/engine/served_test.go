package engine

import (
	"testing"

	"github.com/PdYrust/karvan/internal/migration/plan"
	"github.com/PdYrust/karvan/internal/panel"
	"github.com/PdYrust/karvan/internal/secret"
)

// pgSource is a PasarGuard source whose groups were read: group 1 serves
// WireGuard, group 2 VLESS, group 3 Hysteria and group 4 (disabled) VLESS.
func pgSource(in Inputs) Inputs {
	in.Source.Kind = panel.PasarGuard
	in.Source.GroupsRead = true
	in.Source.Groups = []panel.SourceGroup{
		{ID: 1, Protocols: []string{"wireguard"}},
		{ID: 2, Protocols: []string{"vless"}},
		{ID: 3, Protocols: []string{"hysteria"}},
		{ID: 4, Protocols: []string{"vless"}, Disabled: true},
	}
	return in
}

// pgUser is a PasarGuard user with every protocol (as the panel fills
// them) in groups.
func pgUser(id int64, groups ...int64) panel.User {
	u := pUser(id, "u"+string(rune('a'+id)))
	u.Credentials.WireGuard = &panel.WireGuard{PrivateKey: secret.NewValue("k"), PublicKey: "p"}
	u.Credentials.Hysteria = &panel.Hysteria{Auth: secret.NewValue("h")}
	u.SourceGroups = groups
	return u
}

func marzbanDest() Dest {
	d := dest(panel.Marzban)
	d.TargetProtocols = map[string][]string{"1": {"vless"}}
	return d
}

func TestCurrentProtocols(t *testing.T) {
	src := pgSource(Inputs{}).Source
	u := pgUser(1, 1, 2, 4, 9)
	u.Credentials.VLESS = nil // no credentials for group 2's protocol
	if got, ok := currentProtocols(src, u); !ok || len(got) != 1 || got[0] != "wireguard" {
		t.Errorf("PasarGuard current = %v %v, want [wireguard] (no VLESS key, group 4 disabled, group 9 unknown)", got, ok)
	}
	src.GroupsRead = false
	if _, ok := currentProtocols(src, u); ok {
		t.Error("groups not read: current protocols must be unknown")
	}
	m := mUser("a", 1)
	m.SourceInbounds = map[string][]string{"vless": {"VLESS TCP"}, "vmess": {}}
	if got, ok := currentProtocols(Source{Kind: panel.Marzban}, m); !ok || len(got) != 1 || got[0] != "vless" {
		t.Errorf("Marzban current = %v %v, want [vless] (vmess has no inbound)", got, ok)
	}
}

func TestWireGuardOnlyToMarzbanIsC38(t *testing.T) {
	in := pgSource(inputs(panel.PasarGuard, marzbanDest(), pgUser(1, 1), pgUser(2, 1, 3), pgUser(3, 2)))
	r := build(t, basePlan(), in)
	for _, key := range []string{"1", "2"} {
		it := item(t, r, key)
		if c := hasConflict(it, WireGuardOnly); c == nil || c.Severity != Blocker {
			t.Errorf("user %s: C38 = %+v, want a blocker", key, c)
		}
		if hasConflict(it, NoProtocol) != nil {
			t.Errorf("user %s: C38 replaces C10, so one user has one reason", key)
		}
	}
	if it := item(t, r, "3"); len(it.Conflicts) != 0 {
		t.Errorf("a VLESS user to a VLESS inbound has %+v, want none", it.Conflicts)
	}
	if r.Blockers != 2 {
		t.Errorf("blockers = %d, want 2", r.Blockers)
	}
}

func TestNoCurrentProtocolServedIsC10(t *testing.T) {
	d := dest(panel.PasarGuard)
	d.TargetProtocols = map[string][]string{"1": {"wireguard"}, "2": {"vless"}}
	m := mUser("a", 1)
	m.SourceInbounds = map[string][]string{"vless": {"VLESS TCP"}}
	in := inputs(panel.Marzban, d, m)
	r := build(t, basePlan(), in)
	if c := hasConflict(item(t, r, "a"), NoProtocol); c == nil || c.Severity != Blocker {
		t.Fatalf("C10 = %+v, want a blocker", c)
	}
	p := basePlan()
	p.Targets = []string{"2"}
	if it := item(t, build(t, p, in), "a"); hasConflict(it, NoProtocol) != nil {
		t.Errorf("into the VLESS group: %+v, want no C10", it.Conflicts)
	}
}

func TestNoCurrentProtocolsRaisesNothing(t *testing.T) {
	d := dest(panel.PasarGuard)
	d.TargetProtocols = map[string][]string{"1": {"wireguard"}}
	r := build(t, basePlan(), inputs(panel.Marzban, d, mUser("a", 1)))
	if it := item(t, r, "a"); hasConflict(it, NoProtocol) != nil {
		t.Errorf("a user with no current protocol has no config to lose: %+v", it.Conflicts)
	}
}

func TestMoveAnywayChangesConfig(t *testing.T) {
	in := pgSource(inputs(panel.PasarGuard, marzbanDest(), pgUser(1, 1)))
	p := basePlan()
	p.Resolutions = map[string]plan.Resolution{WireGuardOnly: {Option: OptAccept}}
	r := build(t, p, in)
	it := item(t, r, "1")
	if !r.Stable || r.Blockers != 0 || !it.Include || !it.ChangesConfig {
		t.Errorf("accept: stable %v, blockers %d, include %v, changesConfig %v", r.Stable, r.Blockers, it.Include, it.ChangesConfig)
	}
	if it.User.Credentials.VLESS == nil {
		t.Error("move anyway must give the Marzban protocol's credentials")
	}
	p.Resolutions = map[string]plan.Resolution{WireGuardOnly: {Option: OptSkip}}
	if it := item(t, build(t, p, in), "1"); it.Include {
		t.Error("skip must leave the user out")
	}
}

func TestC10AcceptKeepsUserAndFlagsConfig(t *testing.T) {
	d := dest(panel.PasarGuard)
	d.TargetProtocols = map[string][]string{"1": {"wireguard"}}
	m := mUser("a", 1)
	m.SourceInbounds = map[string][]string{"vless": {"VLESS TCP"}}
	p := basePlan()
	p.Resolutions = map[string]plan.Resolution{NoProtocol: {Option: OptAccept}}
	r := build(t, p, inputs(panel.Marzban, d, m))
	it := item(t, r, "a")
	if !r.Stable || r.Blockers != 0 || !it.Include || !it.ChangesConfig {
		t.Errorf("accept: stable %v, blockers %d, include %v, changesConfig %v", r.Stable, r.Blockers, it.Include, it.ChangesConfig)
	}
}

// D39: a PasarGuard user to Marzban who also uses WireGuard or Hysteria
// loses that config.
func TestLostProtocolCountsAsConfigChange(t *testing.T) {
	in := pgSource(inputs(panel.PasarGuard, marzbanDest(), pgUser(1, 1, 2), pgUser(2, 2)))
	r := build(t, basePlan(), in)
	if it := item(t, r, "1"); !it.ChangesConfig || len(it.Conflicts) != 0 {
		t.Errorf("WireGuard and VLESS user: changesConfig %v, conflicts %+v; want a change and no conflict", it.ChangesConfig, it.Conflicts)
	}
	if it := item(t, r, "2"); it.ChangesConfig {
		t.Error("a VLESS-only user keeps its config")
	}
}

func TestGroupsUnknownRaisesNoBlocker(t *testing.T) {
	in := pgSource(inputs(panel.PasarGuard, marzbanDest(), pgUser(1, 1)))
	in.Source.GroupsRead = false
	r := build(t, basePlan(), in)
	if it := item(t, r, "1"); hasConflict(it, WireGuardOnly) != nil || hasConflict(it, NoProtocol) != nil || it.ChangesConfig {
		t.Errorf("groups unknown: %+v, changesConfig %v; want no C10, C38 or change", it.Conflicts, it.ChangesConfig)
	}
	found := false
	for _, c := range r.Conflicts {
		found = found || c.Type == GroupsUnknown && c.Severity == Info && c.User == ""
	}
	if !found || r.Blockers != 0 {
		t.Errorf("job conflicts %+v, blockers %d; want the info line only", r.Conflicts, r.Blockers)
	}
	m := build(t, basePlan(), inputs(panel.Marzban, marzbanDest(), mUser("a", 1)))
	for _, c := range m.Conflicts {
		if c.Type == GroupsUnknown {
			t.Error("a Marzban source has no groups to read")
		}
	}
}
