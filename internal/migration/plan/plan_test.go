package plan

import (
	"encoding/json"
	"strings"
	"testing"
)

// D35: a plan from before Phase 4.1 names one admin; it is the login's.
func TestForLoginReadsTheOldAdmin(t *testing.T) {
	var p Plan
	if err := json.Unmarshal([]byte(`{"admin":{"username":"boss","mode":"merge","roleId":7}}`), &p); err != nil {
		t.Fatal(err)
	}
	got := p.ForLogin("shop")
	if a := got.Admins["shop"]; a != (AdminPlan{Username: "boss", Mode: AdminMerge, RoleID: 7}) || len(got.Admins) != 1 {
		t.Fatalf("admins %+v", got.Admins)
	}
	raw, err := json.Marshal(got)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), `"admin"`) || !strings.Contains(string(raw), `"admins":{"shop":`) {
		t.Fatalf("written as %s", raw)
	}
	if p.Admins != nil {
		t.Fatal("ForLogin changed the plan it was given")
	}
}

// Only a client that knows no admins sends admin, so it wins over the
// login's entry; the other admins stay as they are.
func TestForLoginOldAdminWins(t *testing.T) {
	p := Plan{Admin: AdminPlan{RoleID: 2}, Admins: map[string]AdminPlan{"shop": {RoleID: 7}, "north": {Username: "n"}}}
	got := p.ForLogin("shop")
	if got.Admins["shop"].RoleID != 2 || got.Admins["north"].Username != "n" || got.Admin != (AdminPlan{}) {
		t.Fatalf("%+v", got)
	}
	if p.Admins["shop"].RoleID != 7 {
		t.Fatal("ForLogin changed the map it was given")
	}
	same := Plan{Admins: map[string]AdminPlan{"shop": {RoleID: 7}}}
	if same.ForLogin("shop").Admins["shop"].RoleID != 7 {
		t.Fatal("a plan with admins only changed")
	}
}
