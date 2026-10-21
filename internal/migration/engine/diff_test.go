package engine

import (
	"encoding/json"
	"slices"
	"strings"
	"testing"

	"github.com/PdYrust/karvan/internal/panel"
)

func TestDiffOfABuildWithItselfIsEmpty(t *testing.T) {
	d := dest(panel.Marzban, destUser{1, "ali", mUser("x", 900)})
	r := build(t, basePlan(), inputs(panel.Marzban, d, mUser("ali", 1), mUser("bob", 2)))
	again := build(t, basePlan(), inputs(panel.Marzban, d, mUser("ali", 1), mUser("bob", 2)))
	if c := Diff(r, again); c.Changed() {
		t.Fatalf("the same read changed: %+v", c)
	}
	if c := Diff(nil, r); c.Changed() {
		t.Fatalf("a first build has nothing to compare with: %+v", c)
	}
}

// A user can be new, gone or changed; a changed one names the fields that
// differ and shows both outcomes, in key order.
func TestDiffNamesNewGoneAndChangedUsers(t *testing.T) {
	before := build(t, basePlan(), inputs(panel.Marzban, dest(panel.Marzban),
		mUser("ali", 1), mUser("bob", 2), mUser("carl", 3)))
	// Meanwhile ali's name was taken on the destination, bob was deleted,
	// carl got new credentials and dana was created.
	d := dest(panel.Marzban, destUser{9, "ali", mUser("x", 900)})
	after := build(t, basePlan(), inputs(panel.Marzban, d, mUser("ali", 1), mUser("carl", 33), mUser("dana", 4)))
	c := Diff(before, after)
	got := make([]string, len(c.Users))
	for i, u := range c.Users {
		got[i] = u.Key + ":" + u.Change + ":" + strings.Join(u.Fields, ",")
	}
	want := []string{"ali:changed:username,conflicts", "bob:gone:", "carl:changed:credentials", "dana:new:"}
	if !slices.Equal(got, want) || c.Admin != nil || c.Conflicts != nil || !c.Changed() {
		t.Fatalf("changes %v", got)
	}
	ali := c.Users[0]
	if ali.Before.Username != "ali" || ali.After.Username != "ali_2" || len(ali.After.Conflicts) != 1 || ali.After.Conflicts[0].Fix != OptSuffix {
		t.Fatalf("ali before %+v after %+v", ali.Before, ali.After)
	}
	if c.Users[1].After != nil || c.Users[1].Before.Username != "bob" || c.Users[3].Before != nil || !c.Users[3].After.Include {
		t.Fatalf("gone %+v new %+v", c.Users[1], c.Users[3])
	}
}

// New credentials show as a changed field, never as values: the outcome
// only names the protocols.
func TestDiffKeepsCredentialsOut(t *testing.T) {
	before := build(t, basePlan(), inputs(panel.Marzban, dest(panel.Marzban), mUser("carl", 3)))
	after := build(t, basePlan(), inputs(panel.Marzban, dest(panel.Marzban), mUser("carl", 33)))
	c := Diff(before, after)
	raw, err := json.Marshal(c)
	if err != nil {
		t.Fatal(err)
	}
	for _, secret := range []string{string(uuid('a', 3)), string(uuid('b', 33)), "tp-3", "tp-33"} {
		if strings.Contains(string(raw), secret) {
			t.Fatalf("the changes show a credential: %s", raw)
		}
	}
	// The target serves shadowsocks too, so the build added it.
	want := []string{"shadowsocks", "trojan", "vless", "vmess"}
	if u := c.Users[0]; !slices.Equal(u.Fields, []string{"credentials"}) || !slices.Equal(u.Before.Protocols, want) || !slices.Equal(u.After.Protocols, want) {
		t.Fatalf("protocols %+v %+v", u.Before, u.After)
	}
}

// The admin's outcome and the job-level conflicts are compared too.
func TestDiffSeesTheAdminAndJobConflicts(t *testing.T) {
	before := build(t, basePlan(), inputs(panel.Marzban, dest(panel.Marzban), mUser("ali", 1)))
	taken := dest(panel.Marzban)
	taken.Admins["shop"] = panel.ExistingAdmin{ID: 5, Username: "shop"}
	c := Diff(before, build(t, basePlan(), inputs(panel.Marzban, taken, mUser("ali", 1))))
	if c.Admin != nil || c.Conflicts == nil || len(c.Conflicts.Before) != 0 || len(c.Conflicts.After) != 1 || c.Conflicts.After[0].Type != AdminTaken {
		t.Fatalf("admin taken: %+v %+v", c.Admin, c.Conflicts)
	}
	// An earlier run moved this admin, so the next one merges into it.
	in := inputs(panel.Marzban, taken, mUser("ali", 1))
	in.MovedAdmin = "shop"
	c = Diff(before, build(t, basePlan(), in))
	if c.Admin == nil || !slices.Equal(c.Admin.Fields, []string{"merged"}) || c.Admin.Before.Merged || !c.Admin.After.Merged {
		t.Fatalf("merge: %+v", c.Admin)
	}
}
