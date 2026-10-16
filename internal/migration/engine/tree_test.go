package engine

import (
	"slices"
	"testing"
	"time"

	"github.com/PdYrust/karvan/internal/migration/plan"
	"github.com/PdYrust/karvan/internal/panel"
)

func TestNestedCategoriesRunParentFirst(t *testing.T) {
	expired := mUser("old", 20)
	past := now.Add(-48 * time.Hour)
	expired.Expire, expired.Status = &past, panel.StatusExpired
	small := mUser("small", 2)
	small.Expire, small.Status = &past, panel.StatusExpired
	fresh := mUser("fresh", 1)

	p := basePlan()
	p.Tree = []plan.Node{
		{ID: "exp", Name: "Expired", Conditions: []plan.Condition{{Field: "expired", Op: "is", Value: "true"}},
			Ops: []plan.Op{{Kind: "time.restart", Params: map[string]string{"days": "30"}}, {Kind: "tag.add", Params: map[string]string{"tag": "renewed"}}}},
		{ID: "big", ParentID: "exp", Name: "Expired, heavy", Conditions: []plan.Condition{{Field: "used_gb", Op: "gt", Value: "10"}},
			Ops: []plan.Op{{Kind: "traffic.reset_used"}}},
		{ID: "all", Name: "Everyone", Ops: []plan.Op{{Kind: "note.append", Params: map[string]string{"text": "moved"}}}},
		{ID: "tagged", ParentID: "all", Conditions: []plan.Condition{{Field: "tag", Op: "has", Value: "renewed"}},
			Ops: []plan.Op{{Kind: "traffic.add", Params: map[string]string{"gb": "-10"}}}},
	}
	r := build(t, p, inputs(panel.Marzban, dest(panel.Marzban), expired, small, fresh))

	old := item(t, r, "old")
	if !slices.Equal(old.Nodes, []string{"exp", "big", "all", "tagged"}) {
		t.Fatalf("old is in %v", old.Nodes)
	}
	if old.User.UsedTraffic != 0 || old.User.Status != panel.StatusActive || !old.User.Expire.Equal(now.Add(30*24*time.Hour)) {
		t.Fatalf("old: used %d status %s expire %v", old.User.UsedTraffic, old.User.Status, old.User.Expire)
	}
	if old.User.DataLimit != 40*gib || old.User.Note != "moved" {
		t.Fatalf("old: limit %d note %q", old.User.DataLimit, old.User.Note)
	}
	if s := item(t, r, "small"); !slices.Equal(s.Nodes, []string{"exp", "all", "tagged"}) || s.User.UsedTraffic != 2*gib {
		t.Fatalf("small: %v used %d", s.Nodes, s.User.UsedTraffic)
	}
	if f := item(t, r, "fresh"); !slices.Equal(f.Nodes, []string{"all"}) || f.User.DataLimit != 50*gib {
		t.Fatalf("fresh: %v", f.Nodes)
	}
}

func TestOperationsAndStatus(t *testing.T) {
	cases := []struct {
		op    plan.Op
		check func(u panel.User) bool
	}{
		{plan.Op{Kind: "traffic.add", Params: map[string]string{"gb": "-100"}}, func(u panel.User) bool { return u.DataLimit == 1 && u.Status == panel.StatusLimited }},
		{plan.Op{Kind: "traffic.carry_remaining"}, func(u panel.User) bool { return u.DataLimit == 45*gib && u.UsedTraffic == 0 }},
		{plan.Op{Kind: "traffic.unlimited"}, func(u panel.User) bool { return u.DataLimit == 0 }},
		{plan.Op{Kind: "time.add", Params: map[string]string{"days": "-20"}}, func(u panel.User) bool { return u.Status == panel.StatusExpired }},
		{plan.Op{Kind: "time.unlimited"}, func(u panel.User) bool { return u.Expire == nil }},
		{plan.Op{Kind: "time.on_hold", Params: map[string]string{"days": "30"}}, func(u panel.User) bool {
			return u.Status == panel.StatusOnHold && u.Expire == nil && *u.OnHoldExpire == 30*24*time.Hour
		}},
		{plan.Op{Kind: "status.set", Params: map[string]string{"status": "disabled"}}, func(u panel.User) bool { return u.Status == panel.StatusDisabled }},
		{plan.Op{Kind: "name.replace", Params: map[string]string{"from": "user", "to": "member"}}, func(u panel.User) bool { return u.Username == "member5" }},
		// "u5" breaks Marzban's rules, so the default fix sanitizes it.
		{plan.Op{Kind: "name.replace", Params: map[string]string{"from": "user", "to": "u"}}, func(u panel.User) bool { return u.Username == "u5_" }},
	}
	for _, c := range cases {
		p := basePlan()
		p.Tree = []plan.Node{{ID: "all", Ops: []plan.Op{c.op}}}
		r := build(t, p, inputs(panel.Marzban, dest(panel.Marzban), mUser("user5", 5)))
		if u := r.Items[0].User; !c.check(u) {
			t.Errorf("%s: %+v", c.op.Kind, u)
		}
	}
}

func TestSkipAndPerUserDecisions(t *testing.T) {
	p := basePlan()
	p.Tree = []plan.Node{{ID: "drop", Conditions: []plan.Condition{{Field: "username", Op: "prefix", Value: "test"}}, Ops: []plan.Op{{Kind: "skip"}}}}
	p.Users = map[string]plan.UserDecision{"keep": {Exclude: true}, "vip": {Ops: []plan.Op{{Kind: "time.add", Params: map[string]string{"days": "5"}}}}}
	r := build(t, p, inputs(panel.Marzban, dest(panel.Marzban), mUser("test1", 1), mUser("keep", 2), mUser("vip", 3)))
	if item(t, r, "test1").Include || item(t, r, "keep").Include || !item(t, r, "vip").Include {
		t.Fatal("skip and exclude")
	}
	if !item(t, r, "vip").User.Expire.Equal(now.Add(15 * 24 * time.Hour)) {
		t.Fatal("per-user op")
	}
}
