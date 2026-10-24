package engine

import (
	"errors"
	"math/rand/v2"
	"slices"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/PdYrust/karvan/internal/migration/plan"
	"github.com/PdYrust/karvan/internal/panel"
	"github.com/PdYrust/karvan/internal/secret"
)

func TestTakenNamesGetTheFirstFreeNumber(t *testing.T) {
	d := dest(panel.Marzban, destUser{1, "ali", mUser("x", 900)}, destUser{2, "ali_2", mUser("y", 901)})
	r := build(t, basePlan(), inputs(panel.Marzban, d, mUser("ali", 1), mUser("bob", 2)))
	ali := item(t, r, "ali")
	if ali.User.Username != "ali_3" || !ali.Include || !ali.ChangesLink {
		t.Fatalf("ali = %+v", ali.User.Username)
	}
	if c := hasConflict(ali, NameTaken); c == nil || c.Fix != OptSuffix {
		t.Fatalf("conflict = %+v", c)
	}
	if bob := item(t, r, "bob"); bob.User.Username != "bob" || len(bob.Resolved) != 0 {
		t.Fatal("bob must be untouched")
	}
	if r.Blockers != 0 || !r.Stable {
		t.Fatalf("blockers %d stable %v", r.Blockers, r.Stable)
	}
}

// A group counts its users once each, fixed or open, and lists no keys:
// the page reads the members through the paged users list.
func TestGroupsCountTheirUsers(t *testing.T) {
	d := dest(panel.Marzban, destUser{1, "ali", mUser("x", 900)}, destUser{2, "reza", mUser("y", 901)})
	r := build(t, basePlan(), inputs(panel.Marzban, d, mUser("ali", 1), mUser("reza", 2), mUser("bob", 3)))
	if len(r.Groups) != 1 || r.Groups[0].Type != NameTaken || r.Groups[0].Count != 2 || r.Groups[0].Open != 0 {
		t.Fatalf("groups = %+v", r.Groups)
	}
}

func TestPrefixForAllAndManualRename(t *testing.T) {
	d := dest(panel.Marzban, destUser{1, "ali", mUser("x", 900)}, destUser{2, "reza", mUser("y", 901)}, destUser{3, "shop_reza", mUser("z", 902)})
	p := basePlan()
	p.Resolutions = map[string]plan.Resolution{NameTaken: {Option: OptPrefix, Params: map[string]string{"text": "shop_"}}}
	p.Users = map[string]plan.UserDecision{"reza": {Resolutions: map[string]plan.Resolution{NameTaken: {Option: OptRename, Params: map[string]string{"username": "ali"}}}}}
	r := build(t, p, inputs(panel.Marzban, d, mUser("ali", 1), mUser("reza", 2)))
	if got := item(t, r, "ali").User.Username; got != "shop_ali" {
		t.Errorf("prefix: %q", got)
	}
	// A typed name that is taken is refused, not silently numbered.
	reza := item(t, r, "reza")
	if c := hasConflict(reza, NameTaken); c == nil || c.Severity != Blocker || c.Detail["fixFailed"] != OptRename {
		t.Fatalf("manual rename to a taken name: %+v", c)
	}
	if r.Blockers == 0 {
		t.Fatal("the run must be blocked")
	}
}

func TestRenameCascadeSettles(t *testing.T) {
	// Renaming "ali" to "ali_2" collides with the source's own "ali_2",
	// which must then move on as well.
	d := dest(panel.Marzban, destUser{1, "ali", mUser("x", 900)})
	r := build(t, basePlan(), inputs(panel.Marzban, d, mUser("ali", 1), mUser("ali_2", 2)))
	got := map[string]string{}
	for _, it := range r.Items {
		got[it.Key] = it.User.Username
	}
	if got["ali"] == got["ali_2"] || got["ali"] == "ali" || !r.Stable {
		t.Fatalf("names %v stable %v", got, r.Stable)
	}
}

func TestCaseOnlyAndTwiceInSource(t *testing.T) {
	d := dest(panel.Marzban)
	d.Rules = foldRules{d.Rules}
	d.Index.Add("omid", &panel.ExistingUser{ID: 9, Username: "omid"})
	r := build(t, basePlan(), inputs(panel.PasarGuard, d, pUser(1, "Omid"), pUser(2, "Ali"), pUser(3, "ali")))
	if hasConflict(item(t, r, "1"), CaseOnly) == nil {
		t.Error("Omid vs omid on a case-insensitive column is C03")
	}
	if hasConflict(item(t, r, "3"), NameTwice) == nil || hasConflict(item(t, r, "2"), NameTwice) != nil {
		t.Error("only the later of Ali/ali is C05")
	}
}

func TestCredentialConflictsDefaultToSkip(t *testing.T) {
	owner := mUser("other", 1)
	d := dest(panel.Marzban, destUser{5, "other", owner})
	// Users are checked in key order, so "a_first" holds the UUID and
	// "z_dup", read later, is the duplicate.
	dup := mUser("z_dup", 7)
	dup.Credentials.VLESS.ID = uuid('b', 3)
	r := build(t, basePlan(), inputs(panel.Marzban, d, mUser("taken", 1), mUser("a_first", 3), dup))
	if it := item(t, r, "taken"); it.Include || hasConflict(it, CredTaken).Detail["username"] != "other" {
		t.Errorf("C07 must skip by default: %+v", it.Conflicts)
	}
	if it := item(t, r, "a_first"); !it.Include {
		t.Error("the first holder of a shared UUID moves")
	}
	if it := item(t, r, "z_dup"); it.Include || hasConflict(it, CredTwice) == nil {
		t.Error("the second holder is C06 and skipped")
	}
}

func TestNewCredentialIsFlaggedAndStable(t *testing.T) {
	d := dest(panel.Marzban, destUser{5, "other", mUser("other", 1)})
	p := basePlan()
	p.Resolutions = map[string]plan.Resolution{CredTaken: {Option: OptNewCredential}}
	in := inputs(panel.Marzban, d, mUser("taken", 1))
	first := item(t, build(t, p, in), "taken")
	if !first.Include || !first.ChangesConfig || first.User.Credentials.VLESS.ID == uuid('b', 1) {
		t.Fatalf("new credential not applied: %+v", first.User.Credentials)
	}
	second := item(t, build(t, p, in), "taken")
	if second.User.Credentials.VLESS.ID != first.User.Credentials.VLESS.ID {
		t.Fatal("a rebuild must reuse the generated UUID")
	}
}

func TestHistory(t *testing.T) {
	moved := mUser("moved", 1)
	d := dest(panel.Marzban, destUser{10, "moved", moved}, destUser{11, "here", mUser("here", 2)})
	in := inputs(panel.Marzban, d, moved, mUser("gone", 3), mUser("here", 2))
	in.Moved = map[string]Moved{
		"moved": {State: "done", DestID: 10, Username: "moved", Created: moved.CreatedAt},
		"gone":  {State: "done", DestID: 99, Username: "gone", Created: moved.CreatedAt},
	}
	r := build(t, basePlan(), in)
	if it := item(t, r, "moved"); it.Include || hasConflict(it, AlreadyMoved) == nil {
		t.Error("C16: moved before and still there")
	}
	if it := item(t, r, "gone"); it.Include || hasConflict(it, MovedGone) == nil {
		t.Error("C16b: deleted on the destination stays deleted by default")
	}
	if it := item(t, r, "here"); it.Include || it.Adopt == nil || it.Adopt.ID != 11 {
		t.Error("C28: same user already there is adopted")
	}
	p := basePlan()
	p.Users = map[string]plan.UserDecision{"gone": {Resolutions: map[string]plan.Resolution{MovedGone: {Option: OptMoveAgain}}}}
	if !item(t, build(t, p, in), "gone").Include {
		t.Error("move_again moves it")
	}
}

func TestConfigChangingConflictsBlock(t *testing.T) {
	u := mUser("flow", 1)
	u.Credentials.VLESS.Flow = "xtls-rprx-vision"
	ss := mUser("ss", 2)
	ss.Credentials.Shadowsocks = &panel.Shadowsocks{Password: secret.NewValue("short"), Method: "aes-128-gcm"}
	in := inputs(panel.Marzban, dest(panel.PasarGuard), u, ss)
	r := build(t, basePlan(), in)
	if c := hasConflict(item(t, r, "flow"), FlowDropped); c == nil || c.Severity != Blocker {
		t.Fatal("C11 must block")
	}
	if c := hasConflict(item(t, r, "ss"), ShadowsocksBad); c == nil || c.Severity != Blocker {
		t.Fatal("C12 must block")
	}
	p := basePlan()
	p.Resolutions = map[string]plan.Resolution{FlowDropped: {Option: OptAccept}, ShadowsocksBad: {Option: OptNewCredential}}
	r = build(t, p, in)
	if it := item(t, r, "flow"); it.User.Credentials.VLESS.Flow != "" || !it.ChangesConfig {
		t.Error("accept drops the flow and flags the config")
	}
	got := item(t, r, "ss").User.Credentials.Shadowsocks
	if len(got.Password.Reveal()) < 22 || got.Method != "aes-128-gcm" {
		t.Errorf("new shadowsocks password keeps the method: %+v", got)
	}
	if r.Blockers != 0 {
		t.Errorf("blockers left: %d", r.Blockers)
	}
}

func TestDeterministic(t *testing.T) {
	d := dest(panel.Marzban, destUser{1, "u3", mUser("x", 900)})
	var users []panel.User
	for i := range 50 {
		users = append(users, mUser("u"+string(rune('a'+i%26))+strings.Repeat("x", i/26), i))
	}
	users = append(users, mUser("u3", 3), mUser("u3_2", 4))
	a := build(t, basePlan(), inputs(panel.Marzban, d, users...))
	rand.Shuffle(len(users), func(i, j int) { users[i], users[j] = users[j], users[i] })
	b := build(t, basePlan(), inputs(panel.Marzban, d, users...))
	if Diff(a, b).Changed() {
		t.Fatal("the same input in another order must give the same plan")
	}
	moreTraffic := users[0]
	moreTraffic.UsedTraffic += 5 * gib
	users[0] = moreTraffic
	if Diff(a, build(t, basePlan(), inputs(panel.Marzban, d, users...))).Changed() {
		t.Fatal("traffic is volatile and must not change the outcome")
	}
}

func TestAdmin(t *testing.T) {
	d := dest(panel.Marzban)
	d.Admins["shop"] = panel.ExistingAdmin{ID: 4, Username: "shop"}
	r := build(t, basePlan(), inputs(panel.Marzban, d, mUser("a", 1)))
	if len(r.Conflicts) != 1 || r.Conflicts[0].Type != AdminTaken || r.Blockers == 0 {
		t.Fatalf("C01: %+v", r.Conflicts)
	}
	p := basePlan()
	p.Admin = plan.AdminPlan{Username: "shop2"}
	if r := build(t, p, inputs(panel.Marzban, d, mUser("a", 1))); r.Admin.Username != "shop2" || len(r.Conflicts) != 0 {
		t.Fatalf("rename: %+v", r.Conflicts)
	}
	p.Admin = plan.AdminPlan{Mode: plan.AdminMerge, Username: "shop"}
	if r := build(t, p, inputs(panel.Marzban, d, mUser("a", 1))); r.Admin.Merge == nil || r.Admin.Merge.ID != 4 {
		t.Fatal("merge")
	}
}

// A taken admin name comes with the first suggestion the check looked up
// and found free; one it did not look up may exist, so it is not offered.
func TestAdminTakenSuggestsAFreeName(t *testing.T) {
	d := dest(panel.Marzban)
	d.Admins["shop"] = panel.ExistingAdmin{ID: 4, Username: "shop"}
	d.Admins["shop_2"] = panel.ExistingAdmin{ID: 5, Username: "shop_2"}
	d.Asked = []string{"shop", "shop_2", "shop_3"}
	r := build(t, basePlan(), inputs(panel.Marzban, d, mUser("a", 1)))
	if len(r.Conflicts) != 1 || r.Conflicts[0].Detail["suggestion"] != "shop_3" {
		t.Fatalf("C01: %+v", r.Conflicts)
	}
	d.Asked = []string{"shop", "shop_2"}
	r = build(t, basePlan(), inputs(panel.Marzban, d, mUser("a", 1)))
	if _, ok := r.Conflicts[0].Detail["suggestion"]; ok {
		t.Fatalf("a name never looked up was offered: %+v", r.Conflicts)
	}
}

func TestAdminSuggestionsFitTheLimit(t *testing.T) {
	if got := AdminSuggestions("shop"); !slices.Equal(got, []string{"shop_2", "shop_3", "shop_4", "shop_5", "shop_6"}) {
		t.Fatalf("%q", got)
	}
	long := strings.Repeat("a", 33) + "é"
	for _, s := range AdminSuggestions(long) {
		if len(s) > maxAdminName || !utf8.ValidString(s) {
			t.Fatalf("%q does not fit", s)
		}
	}
	if got := AdminSuggestions(""); len(got) != 0 {
		t.Fatalf("an empty name: %q", got)
	}
}

func TestJobConflicts(t *testing.T) {
	// No targets chosen and none on the destination: nowhere to put users.
	in := inputs(panel.Marzban, dest(panel.Marzban), mUser("a", 1))
	in.Dest.Targets = nil
	in.Source.Skew = 3 * time.Minute
	p := basePlan()
	p.Targets = nil
	r := build(t, p, in)
	types := map[string]bool{}
	for _, c := range r.Conflicts {
		types[c.Type] = true
	}
	if !types[ClockSkew] || !types[NoTargets] {
		t.Fatalf("%+v", r.Conflicts)
	}
}

func TestPlanErrors(t *testing.T) {
	cases := map[string]plan.Plan{
		"unknown op":     {Tree: []plan.Node{{ID: "a", Ops: []plan.Op{{Kind: "time.travel"}}}}},
		"bad days":       {Tree: []plan.Node{{ID: "a", Ops: []plan.Op{{Kind: "time.add", Params: map[string]string{"days": "ten"}}}}}},
		"bad condition":  {Tree: []plan.Node{{ID: "a", Conditions: []plan.Condition{{Field: "colour", Op: "in"}}}}},
		"missing parent": {Tree: []plan.Node{{ID: "a", ParentID: "z"}}},
		"loop":           {Tree: []plan.Node{{ID: "a", ParentID: "b"}, {ID: "b", ParentID: "a"}}},
		"bad fix":        {Resolutions: map[string]plan.Resolution{CredTaken: {Option: OptPrefix}}},
	}
	for name, p := range cases {
		var pe *PlanError
		if _, err := Build(p, inputs(panel.Marzban, dest(panel.Marzban))); !errors.As(err, &pe) {
			t.Errorf("%s: err = %v", name, err)
		}
	}
}
