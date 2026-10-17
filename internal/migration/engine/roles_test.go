package engine

import (
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/PdYrust/karvan/internal/migration/plan"
	"github.com/PdYrust/karvan/internal/panel"
)

func ptr(v int64) *int64 { return &v }

// roleDest is a PasarGuard destination with groups 1 (vless), 2 (vmess,
// disabled) and 3 (trojan), and a "cheap" role allowing groups 1 and 2.
func roleDest() Dest {
	d := dest(panel.PasarGuard)
	d.Targets = []panel.Target{{ID: "1", Name: "A"}, {ID: "2", Name: "B", Disabled: true}, {ID: "3", Name: "C"}}
	d.TargetProtocols = map[string][]string{"1": {"vless"}, "2": {"vmess"}, "3": {"trojan"}}
	all := panel.RoleFeatures{ResetStrategy: true, NextPlan: true}
	d.Roles = append(builtInRoles(), panel.Role{ID: 7, Name: "cheap", Groups: []string{"1", "2"}, Features: all})
	return d
}

func vlessOnly(name string, n int) panel.User {
	u := mUser(name, n)
	u.Credentials = panel.Credentials{VLESS: u.Credentials.VLESS}
	u.SourceID = int64(n) // a PasarGuard source keys items by id
	return u
}

func noTargets() plan.Plan {
	p := basePlan()
	p.Targets = nil
	return p
}

// D65: the source role's name picks the destination role; the owner role
// never comes by default.
func TestDefaultRole(t *testing.T) {
	for source, want := range map[string]int64{"cheap": 7, "owner": operatorRole, "": operatorRole, "gold": operatorRole} {
		in := inputs(panel.PasarGuard, roleDest(), vlessOnly("alice", 1))
		in.Source.Admin.RoleName = source
		if r := build(t, noTargets(), in); r.Admin.RoleID != want {
			t.Errorf("source role %q: got %d, want %d", source, r.Admin.RoleID, want)
		}
	}
	in := inputs(panel.PasarGuard, roleDest(), vlessOnly("alice", 1))
	p := noTargets()
	p.Admin.RoleID = 99
	r := build(t, p, in)
	if !slices.ContainsFunc(r.Conflicts, func(c Conflict) bool { return c.Type == RoleMissing && c.Severity == Blocker }) {
		t.Errorf("a role not on the destination: %+v", r.Conflicts)
	}
	p.Admin.RoleID = 1
	if r := build(t, p, in); !slices.ContainsFunc(r.Conflicts, func(c Conflict) bool { return c.Type == RoleOwner }) {
		t.Error("the owner role chosen by hand is not flagged")
	}
}

// D66, D67: with no targets chosen, users get every enabled group the
// role allows, and credentials for every protocol those groups serve. The
// old credentials stay as they were.
func TestAccessFollowsTheRole(t *testing.T) {
	src := vlessOnly("alice", 1)
	in := inputs(panel.PasarGuard, roleDest(), src)
	in.Source.Admin.RoleName = "cheap"
	r := build(t, noTargets(), in)
	it := item(t, r, "1")
	if strings.Join(r.Targets, ",") != "1" || strings.Join(it.Targets, ",") != "1" {
		t.Fatalf("targets %v / %v", r.Targets, it.Targets)
	}
	in = inputs(panel.PasarGuard, roleDest(), src) // operator allows all groups
	it = item(t, build(t, noTargets(), in), "1")
	c := it.User.Credentials
	if strings.Join(it.Targets, ",") != "1,3" || c.Trojan == nil || c.VMess != nil || c.VLESS.ID != src.Credentials.VLESS.ID {
		t.Fatalf("targets %v creds %+v", it.Targets, c)
	}
	if it.ChangesConfig || !slices.Contains(it.Applied, "added:trojan") {
		t.Errorf("added a protocol: changes config %v, applied %v", it.ChangesConfig, it.Applied)
	}
	if r.Blockers != 0 {
		t.Errorf("blockers %d: %+v", r.Blockers, r.Conflicts)
	}
}

func TestMarzbanGetsEveryInbound(t *testing.T) {
	d := dest(panel.Marzban)
	d.Targets = []panel.Target{{ID: "vless/A", Protocol: "vless"}, {ID: "vmess/B", Protocol: "vmess"}}
	d.TargetProtocols = map[string][]string{"vless/A": {"vless"}, "vmess/B": {"vmess"}}
	it := item(t, build(t, noTargets(), inputs(panel.PasarGuard, d, vlessOnly("alice", 1))), "1")
	if strings.Join(it.Targets, ",") != "vless/A,vmess/B" || it.User.Credentials.VMess == nil {
		t.Fatalf("targets %v creds %+v", it.Targets, it.User.Credentials)
	}
}

// A group given by hand outside the role is a warning, not a blocker.
func TestGroupOutsideTheRole(t *testing.T) {
	in := inputs(panel.PasarGuard, roleDest(), vlessOnly("alice", 1))
	in.Source.Admin.RoleName = "cheap"
	p := basePlan()
	p.Targets = []string{"1", "3"}
	r := build(t, p, in)
	i := slices.IndexFunc(r.Conflicts, func(c Conflict) bool { return c.Type == RoleGroups })
	if i < 0 || r.Conflicts[i].Detail["targets"] != "3" || r.Blockers != 0 {
		t.Fatalf("%+v", r.Conflicts)
	}
}

// D68: a user beyond the role's limits moves as it is unless the operator
// chooses to fit it; the admin's own overrides win over the role.
func TestRoleLimits(t *testing.T) {
	d := roleDest()
	d.Roles[3].Limits = panel.RoleLimits{DataLimitMax: ptr(50 * gib), ExpireMax: ptr(int64(30 * 24 * 3600)), MaxUsers: ptr(1)}
	big := vlessOnly("big", 1)
	big.DataLimit = 100 * gib
	far := basePlan().Now.Add(90 * 24 * time.Hour)
	big.Expire = &far
	small := vlessOnly("small", 2)
	small.DataLimit, small.Expire = gib, nil
	in := inputs(panel.PasarGuard, d, big, small)
	in.Source.Admin.RoleName = "cheap"
	p := noTargets()
	r := build(t, p, in)
	if c := hasConflict(item(t, r, "1"), RoleLimit); c == nil || c.Fix != OptKeep || c.Detail["fields"] != "data_limit,expire" {
		t.Fatalf("big: %+v", item(t, r, "1").Conflicts)
	}
	if c := hasConflict(item(t, r, "2"), RoleLimit); c == nil || c.Detail["fields"] != "expire" {
		t.Fatalf("an unlimited expiry breaks a maximum: %+v", item(t, r, "2").Conflicts)
	}
	if item(t, r, "1").User.DataLimit != 100*gib || r.Blockers != 0 {
		t.Fatal("keep must leave the user as it was and block nothing")
	}
	if !slices.ContainsFunc(r.Conflicts, func(c Conflict) bool { return c.Type == RoleMaxUsers && c.Detail["users"] == "2" }) {
		t.Errorf("max users: %+v", r.Conflicts)
	}
	p.Resolutions = map[string]plan.Resolution{RoleLimit: {Option: OptFit}}
	it := item(t, build(t, p, in), "1")
	open := slices.ContainsFunc(it.Conflicts, func(c Conflict) bool { return c.Type == RoleLimit })
	if it.User.DataLimit != 50*gib || !it.User.Expire.Equal(p.Now.Add(30*24*time.Hour)) || open {
		t.Fatalf("fit: %d %v %+v", it.User.DataLimit, it.User.Expire, it.Conflicts)
	}
	in.Source.Admin.Overrides = &panel.RoleLimits{DataLimitMax: ptr(200 * gib), ExpireMax: ptr(int64(365 * 24 * 3600))}
	if c := hasConflict(item(t, build(t, noTargets(), in), "1"), RoleLimit); c != nil {
		t.Errorf("the admin's own limits win: %+v", c)
	}
}

func TestRoleFeatures(t *testing.T) {
	d := roleDest()
	d.Roles[3].Features = panel.RoleFeatures{ResetStrategy: true, NextPlan: false}
	u := vlessOnly("alice", 1)
	u.NextPlan = &panel.NextPlan{DataLimit: gib}
	in := inputs(panel.PasarGuard, d, u)
	in.Source.Admin.RoleName = "cheap"
	p := noTargets()
	if c := hasConflict(item(t, build(t, p, in), "1"), RoleFeature); c == nil || c.Detail["fields"] != "next_plan" {
		t.Fatal("a next plan the role forbids is not flagged")
	}
	p.Resolutions = map[string]plan.Resolution{RoleFeature: {Option: OptRemove}}
	if it := item(t, build(t, p, in), "1"); it.User.NextPlan != nil {
		t.Fatal("remove keeps the next plan")
	}
}

// An admin merged into keeps its own role; the role counts its users.
func TestMergeKeepsTheAdminsRole(t *testing.T) {
	d := roleDest()
	d.Roles[3].Limits.MaxUsers = ptr(5)
	d.Admins = map[string]panel.ExistingAdmin{"shop": {ID: 4, Username: "shop", RoleID: 7, Users: 5}}
	in := inputs(panel.PasarGuard, d, vlessOnly("alice", 1))
	p := noTargets()
	p.Admin.Mode, p.Admin.RoleID = plan.AdminMerge, 2
	r := build(t, p, in)
	if r.Admin.RoleID != 7 || !slices.ContainsFunc(r.Conflicts, func(c Conflict) bool { return c.Type == RoleMaxUsers && c.Detail["users"] == "6" }) {
		t.Fatalf("role %d conflicts %+v", r.Admin.RoleID, r.Conflicts)
	}
}
