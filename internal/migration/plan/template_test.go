package plan

import "testing"

func TestTemplateLeavesPickedUsersOut(t *testing.T) {
	p := Plan{
		Admin:   AdminPlan{Username: "shop2"},
		Targets: []string{"1"},
		Tree: []Node{
			{ID: "exp", Conditions: []Condition{{Field: "expired", Op: "is", Value: "true"}}},
			{ID: "picked", Conditions: []Condition{{Field: "users", Op: "in", List: []string{"ali"}}}},
			{ID: "child", ParentID: "picked"},
		},
		Users:       map[string]UserDecision{"ali": {Exclude: true}},
		Resolutions: map[string]Resolution{"C02": {Option: "prefix", Params: map[string]string{"text": "x_"}}},
	}
	tpl := TemplateOf("renew", p)
	if len(tpl.Tree) != 1 || tpl.Tree[0].ID != "exp" {
		t.Fatalf("tree = %+v", tpl.Tree)
	}
	next := tpl.Apply(Plan{Admin: AdminPlan{Username: "other"}, Targets: []string{"9"}})
	if next.Admin.Username != "other" || next.Targets[0] != "9" || next.Resolutions["C02"].Option != "prefix" {
		t.Fatalf("apply = %+v", next)
	}
}
