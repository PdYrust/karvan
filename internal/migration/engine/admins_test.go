package engine

import (
	"encoding/json"
	"slices"
	"strings"
	"testing"

	"github.com/PdYrust/karvan/internal/migration/plan"
	"github.com/PdYrust/karvan/internal/panel"
)

func owned(u panel.User, owner string) panel.User {
	u.OwnerUsername = owner
	return u
}

// adminConflict is the job-level conflict of type t for source admin a.
func adminConflict(r *Result, t, a string) *Conflict {
	for i, c := range r.Conflicts {
		if c.Type == t && c.Detail["admin"] == a {
			return &r.Conflicts[i]
		}
	}
	return nil
}

func sources(r *Result) string {
	var out []string
	for _, a := range r.Admins {
		out = append(out, a.Source.Username)
	}
	return strings.Join(out, ",")
}

// D34: each admin moves with its own users; users with no admin, and the
// login's own, go to the login's admin. The login comes first, the rest
// by name.
func TestAdminsFromTheUsersOwners(t *testing.T) {
	in := inputs(panel.Marzban, dest(panel.Marzban),
		owned(mUser("a", 1), "shop"), owned(mUser("b", 2), "south"), mUser("c", 3), owned(mUser("d", 4), "north"), owned(mUser("e", 5), "north"))
	r := build(t, basePlan(), in)
	if got := sources(r); got != "shop,north,south" {
		t.Fatalf("admins %s", got)
	}
	for key, want := range map[string]string{"a": "shop", "b": "south", "c": "shop", "d": "north", "e": "north"} {
		if got := item(t, r, key).Admin; got != want {
			t.Errorf("%s belongs to %q, want %q", key, got, want)
		}
	}
	login := r.Login()
	if login == nil || !login.IsLogin || login.Source.Username != "shop" || login.Users != 2 || r.Admins[1].IsLogin {
		t.Fatalf("login %+v", login)
	}
	if n := r.Admins[1]; n.Username != "north" || n.Users != 2 || n.Merge != nil || len(n.Targets) != 1 {
		t.Fatalf("north %+v", n)
	}
	if len(r.Conflicts) != 0 || r.Blockers != 0 {
		t.Fatalf("conflicts %+v", r.Conflicts)
	}
}

// The login's admin moves even when it owns no user; an admin known
// from Source.Admins keeps what was read about it.
func TestLoginAlwaysMovesAndAdminsKeepTheirFacts(t *testing.T) {
	in := inputs(panel.Marzban, dest(panel.Marzban), owned(mUser("a", 1), "north"))
	in.Source.Admins = []panel.Admin{{Username: "north", Note: "the north shop"}, {Username: "idle"}}
	r := build(t, basePlan(), in)
	if got := sources(r); got != "shop,north" {
		t.Fatalf("admins %s: an admin with no user does not move", got)
	}
	if r.Login().Users != 0 || r.Admins[1].Source.Note != "the north shop" {
		t.Fatalf("%+v", r.Admins)
	}
}

// An admin other than the login's whose users all stay behind is not
// written, so its own conflicts go too; the login's admin always moves.
func TestAnAdminWithNothingToMoveIsLeftOut(t *testing.T) {
	d := dest(panel.Marzban)
	d.Admins["north"] = panel.ExistingAdmin{ID: 4, Username: "north"}
	d.Admins["shop"] = panel.ExistingAdmin{ID: 5, Username: "shop"}
	in := inputs(panel.Marzban, d, owned(mUser("b", 2), "north"))
	p := basePlan()
	p.Users = map[string]plan.UserDecision{"b": {Exclude: true}}
	r := build(t, p, in)
	if r.Admins[1].Moves() || !r.Login().Moves() {
		t.Fatalf("moves: %+v", r.Admins)
	}
	if adminConflict(r, AdminTaken, "north") != nil || adminConflict(r, AdminTaken, "shop") == nil {
		t.Fatalf("conflicts %+v", r.Conflicts)
	}
}

// C01 and C30 are per admin, and a plan entry fixes only its own admin.
func TestAdminConflictsPerAdmin(t *testing.T) {
	d := dest(panel.Marzban)
	d.Admins["north"] = panel.ExistingAdmin{ID: 4, Username: "north"}
	d.Asked = []string{"shop", "north", "north_2"}
	in := inputs(panel.Marzban, d, mUser("a", 1), owned(mUser("b", 2), "north"))
	r := build(t, basePlan(), in)
	c := adminConflict(r, AdminTaken, "north")
	if c == nil || c.Severity != Blocker || c.Detail["username"] != "north" || c.Detail["suggestion"] != "north_2" {
		t.Fatalf("north: %+v", r.Conflicts)
	}
	if adminConflict(r, AdminTaken, "shop") != nil || r.Blockers != 1 {
		t.Fatalf("only north is taken: %+v", r.Conflicts)
	}
	p := basePlan()
	p.Admins = map[string]plan.AdminPlan{"north": {Mode: plan.AdminMerge}}
	r = build(t, p, in)
	if n := r.Admins[1]; n.Merge == nil || n.Merge.ID != 4 || len(r.Conflicts) != 0 {
		t.Fatalf("merge: %+v %+v", n, r.Conflicts)
	}
	p.Admins = map[string]plan.AdminPlan{"north": {Username: " north"}}
	if c := adminConflict(build(t, p, in), AdminInvalid, "north"); c == nil || c.Severity != Blocker {
		t.Fatal("an invalid name for north is not flagged")
	}
}

// Two admins of one job cannot be created under one name: the later one
// gets C01, with a suggestion no admin of the job uses. Merging both into
// one existing admin is fine.
func TestTwoAdminsUnderOneName(t *testing.T) {
	d := dest(panel.Marzban)
	d.Asked = []string{"shop", "shop_2", "shop_3"}
	in := inputs(panel.Marzban, d, mUser("a", 1), owned(mUser("b", 2), "north"), owned(mUser("c", 3), "south"))
	p := basePlan()
	p.Admins = map[string]plan.AdminPlan{"north": {Username: "shop"}, "south": {Username: "shop_2"}}
	r := build(t, p, in)
	c := adminConflict(r, AdminTaken, "north")
	if c == nil || c.Detail["username"] != "shop" || c.Detail["suggestion"] != "shop_3" {
		t.Fatalf("north: %+v", r.Conflicts)
	}
	if adminConflict(r, AdminTaken, "shop") != nil || adminConflict(r, AdminTaken, "south") != nil {
		t.Fatalf("the login and south keep their names: %+v", r.Conflicts)
	}
	p.Admins = map[string]plan.AdminPlan{"north": {Username: "Shop"}}
	if adminConflict(build(t, p, in), AdminTaken, "north") == nil {
		t.Fatal("two admins whose names differ only in case")
	}
	d.Admins["root"] = panel.ExistingAdmin{ID: 9, Username: "root"}
	p.Admins = map[string]plan.AdminPlan{"north": {Username: "root", Mode: plan.AdminMerge}, "south": {Username: "root", Mode: plan.AdminMerge}}
	if r := build(t, p, in); len(r.Conflicts) != 0 || r.Admins[1].Merge.ID != 9 || r.Admins[2].Merge.ID != 9 {
		t.Fatalf("two merges into root: %+v", r.Conflicts)
	}
	p.Admins = map[string]plan.AdminPlan{"north": {Username: "root", Mode: plan.AdminMerge}, "south": {Username: "root"}}
	if adminConflict(build(t, p, in), AdminTaken, "south") == nil {
		t.Fatal("creating an admin that another merges into")
	}
}

// D35: an old plan's admin is the login's; the other admins keep theirs.
func TestOldPlanAdminIsTheLogins(t *testing.T) {
	in := inputs(panel.Marzban, dest(panel.Marzban), mUser("a", 1), owned(mUser("b", 2), "north"))
	p := basePlan()
	p.Admin = plan.AdminPlan{Username: "boss"}
	r := build(t, p, in)
	if r.Login().Username != "boss" || r.Admins[1].Username != "north" {
		t.Fatalf("%+v", r.Admins)
	}
}

// A re-run merges each admin into the one an earlier run moved it to.
func TestReRunMergesPerAdmin(t *testing.T) {
	d := dest(panel.Marzban)
	d.Admins["north_old"] = panel.ExistingAdmin{ID: 6, Username: "north_old"}
	in := inputs(panel.Marzban, d, mUser("a", 1), owned(mUser("b", 2), "north"))
	in.MovedAdmins = map[string]string{"north": "north_old"}
	r := build(t, basePlan(), in)
	if r.Login().Merge != nil || r.Admins[1].Merge == nil || r.Admins[1].Username != "north_old" || len(r.Conflicts) != 0 {
		t.Fatalf("%+v %+v", r.Admins, r.Conflicts)
	}
}

// rolePerAdmin has users over the "cheap" role's limits for the login,
// north (cheap by the plan) and south (cheap by its source role name).
func rolePerAdmin() (plan.Plan, Inputs) {
	d := roleDest()
	d.Roles[3].Limits = panel.RoleLimits{DataLimitMax: ptr(10 * gib), MaxUsers: ptr(1)}
	big := func(name string, n int, owner string) panel.User {
		u := owned(vlessOnly(name, n), owner)
		u.DataLimit = 100 * gib
		return u
	}
	in := inputs(panel.PasarGuard, d, big("a", 1, "shop"), big("b", 2, "north"), big("c", 3, "north"), big("e", 5, "south"))
	in.Source.Admins = []panel.Admin{{Username: "south", RoleName: "cheap"}}
	p := basePlan()
	p.Admins = map[string]plan.AdminPlan{"north": {RoleID: 7}}
	return p, in
}

// D31 per admin: each admin's users get the groups of that admin's role,
// whose limits judge only its own users.
func TestRolePerAdmin(t *testing.T) {
	p, in := rolePerAdmin()
	r := build(t, p, in)
	if r.Login().RoleID != operatorRole || r.Admins[1].RoleID != 7 || r.Admins[2].RoleID != 7 {
		t.Fatalf("roles %+v", r.Admins)
	}
	if got := strings.Join(item(t, r, "1").Targets, ","); got != "1,3" {
		t.Errorf("the login's user gets %s, want every enabled group", got)
	}
	for _, key := range []string{"2", "3", "5"} {
		it := item(t, r, key)
		if got := strings.Join(it.Targets, ","); got != "1" || hasConflict(it, RoleLimit) == nil {
			t.Errorf("%s: targets %s conflicts %+v", key, got, it.Conflicts)
		}
	}
	if hasConflict(item(t, r, "1"), RoleLimit) != nil {
		t.Error("the operator role has no limit")
	}
	if c := adminConflict(r, RoleMaxUsers, "north"); c == nil || c.Detail["users"] != "2" {
		t.Fatalf("north owns two users over a maximum of one: %+v", r.Conflicts)
	}
	if adminConflict(r, RoleMaxUsers, "south") != nil || adminConflict(r, RoleMaxUsers, "shop") != nil {
		t.Fatalf("%+v", r.Conflicts)
	}
	if got := strings.Join(r.Admins[1].Targets, ","); got != "1" {
		t.Errorf("north targets %s", got)
	}
}

func TestFitUsesEachAdminsRole(t *testing.T) {
	p, in := rolePerAdmin()
	p.Resolutions = map[string]plan.Resolution{RoleLimit: {Option: OptFit}}
	r := build(t, p, in)
	if item(t, r, "2").User.DataLimit != 10*gib || item(t, r, "1").User.DataLimit != 100*gib {
		t.Fatal("fit uses each admin's own role")
	}
}

// Two admins merged into one existing admin count against its role
// together.
func TestMaxUsersCountsEveryAdminMergedIn(t *testing.T) {
	d := roleDest()
	d.Roles[3].Limits.MaxUsers = ptr(3)
	d.Admins = map[string]panel.ExistingAdmin{"root": {ID: 4, Username: "root", RoleID: 7, Users: 1}}
	in := inputs(panel.PasarGuard, d, owned(vlessOnly("a", 1), "north"), owned(vlessOnly("b", 2), "south"), owned(vlessOnly("c", 3), "south"))
	p := basePlan()
	p.Admins = map[string]plan.AdminPlan{"north": {Username: "root", Mode: plan.AdminMerge}, "south": {Username: "root", Mode: plan.AdminMerge}}
	r := build(t, p, in)
	for _, a := range []string{"north", "south"} {
		if c := adminConflict(r, RoleMaxUsers, a); c == nil || c.Detail["users"] != "4" {
			t.Fatalf("%s: %+v", a, r.Conflicts)
		}
	}
}

// An admin whose role allows no enabled group gets C09; the others move.
func TestNoTargetsPerAdmin(t *testing.T) {
	d := roleDest()
	d.Roles = append(d.Roles, panel.Role{ID: 8, Name: "off", Groups: []string{"2"}, Features: d.Roles[3].Features})
	in := inputs(panel.PasarGuard, d, vlessOnly("a", 1), owned(vlessOnly("b", 2), "north"))
	p := basePlan()
	p.Admins = map[string]plan.AdminPlan{"north": {RoleID: 8}}
	r := build(t, p, in)
	if adminConflict(r, NoTargets, "north") == nil || adminConflict(r, NoTargets, "shop") != nil {
		t.Fatalf("%+v", r.Conflicts)
	}
}

// The same reading in another order gives the same result.
func TestAdminsAreDeterministic(t *testing.T) {
	users := []panel.User{mUser("a", 1), owned(mUser("b", 2), "zed"), owned(mUser("c", 3), "north"), owned(mUser("d", 4), "mid")}
	d := dest(panel.Marzban)
	d.Admins["north"] = panel.ExistingAdmin{ID: 4, Username: "north"}
	d.Admins["zed"] = panel.ExistingAdmin{ID: 5, Username: "zed"}
	once := build(t, basePlan(), inputs(panel.Marzban, d, users...))
	slices.Reverse(users)
	again := build(t, basePlan(), inputs(panel.Marzban, d, users...))
	a, _ := json.Marshal(struct {
		A []AdminOutcome
		C []Conflict
	}{once.Admins, once.Conflicts})
	b, _ := json.Marshal(struct {
		A []AdminOutcome
		C []Conflict
	}{again.Admins, again.Conflicts})
	if string(a) != string(b) || Diff(once, again).Changed() {
		t.Fatalf("%s\n%s", a, b)
	}
	if got := sources(once); got != "shop,mid,north,zed" {
		t.Fatalf("admins %s", got)
	}
}

// Diff sees every admin, not only the login's.
func TestDiffSeesEveryAdmin(t *testing.T) {
	in := inputs(panel.Marzban, dest(panel.Marzban), mUser("a", 1), owned(mUser("b", 2), "north"))
	before := build(t, basePlan(), in)
	p := basePlan()
	p.Admins = map[string]plan.AdminPlan{"north": {Username: "n2"}}
	c := Diff(before, build(t, p, in))
	if c.Admin != nil || len(c.Admins) != 1 || c.Admins[0].Source != "north" || c.Admins[0].Change != UserChanged ||
		!slices.Equal(c.Admins[0].Fields, []string{"username"}) || c.Admins[0].After.Username != "n2" {
		t.Fatalf("%+v", c.Admins)
	}
	in2 := inputs(panel.Marzban, dest(panel.Marzban), mUser("a", 1), owned(mUser("b", 2), "south"))
	c = Diff(before, build(t, basePlan(), in2))
	got := []string{}
	for _, a := range c.Admins {
		got = append(got, a.Source+":"+a.Change)
	}
	if !slices.Equal(got, []string{"north:gone", "south:new"}) {
		t.Fatalf("%v", got)
	}
	if u := c.Users[0]; u.Key != "b" || !slices.Contains(u.Fields, "admin") {
		t.Fatalf("b moved to another admin: %+v", c.Users)
	}
}
