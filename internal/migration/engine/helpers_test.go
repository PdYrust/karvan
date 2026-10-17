package engine

import (
	"fmt"
	"testing"
	"time"

	"github.com/PdYrust/karvan/internal/migration/plan"
	"github.com/PdYrust/karvan/internal/panel"
	"github.com/PdYrust/karvan/internal/panel/marzban"
	"github.com/PdYrust/karvan/internal/panel/pasarguard"
	"github.com/PdYrust/karvan/internal/secret"
)

var now = time.Date(2026, 10, 16, 12, 0, 0, 0, time.UTC)

func uuid(prefix byte, n int) panel.UUID {
	return panel.UUID(fmt.Sprintf("%c0000000-0000-4000-8000-%012d", prefix, n))
}

// mUser is a Marzban-sourced user with vmess, vless and trojan.
func mUser(name string, n int) panel.User {
	exp := now.Add(10 * 24 * time.Hour)
	return panel.User{
		Username: name, Status: panel.StatusActive, ResetStrategy: panel.ResetNever,
		CreatedAt: time.Date(2025, 1, 2, 3, 4, 5, 0, time.UTC), Expire: &exp,
		DataLimit: 50 * gib, UsedTraffic: int64(n) * gib,
		Credentials: panel.Credentials{
			VMess:  &panel.VMess{ID: uuid('a', n)},
			VLESS:  &panel.VLESS{ID: uuid('b', n)},
			Trojan: &panel.Trojan{Password: secret.NewValue(fmt.Sprintf("tp-%d", n))},
		},
	}
}

type destUser struct {
	id   int64
	name string
	user panel.User // credentials and created_at
}

func dest(kind panel.Kind, users ...destUser) Dest {
	var rules panel.Rules = marzban.NewRules(false)
	if kind == panel.PasarGuard {
		rules = pasarguard.Rules{}
	}
	x := panel.NewIndex()
	for _, d := range users {
		u := &panel.ExistingUser{ID: d.id, Username: d.name, CreatedAt: d.user.CreatedAt}
		x.Add(rules.UserKey(d.name), u)
		x.AddCredentials(d.user.Credentials, d.id)
	}
	d := Dest{
		Kind: kind, Rules: rules, Index: x, Admins: map[string]panel.ExistingAdmin{},
		Targets:         []panel.Target{{ID: "1", Name: "Main"}},
		TargetProtocols: map[string][]string{"1": {"vmess", "vless", "trojan", "shadowsocks"}},
	}
	if kind == panel.PasarGuard {
		d.Roles = builtInRoles()
	}
	return d
}

func inputs(kind panel.Kind, d Dest, users ...panel.User) Inputs {
	n := 0
	return Inputs{
		Source: Source{Kind: kind, Origin: "https://old.example", Admin: panel.Admin{Username: "shop"}, Users: users},
		Dest:   d,
		Moved:  map[string]Moved{},
		Generate: func(protocol string) panel.Credentials {
			n++
			switch protocol {
			case "shadowsocks":
				return panel.Credentials{Shadowsocks: &panel.Shadowsocks{Password: secret.NewValue(fmt.Sprintf("generated-shadowsocks-%04d", n)), Method: "aes-256-gcm"}}
			case "trojan":
				return panel.Credentials{Trojan: &panel.Trojan{Password: secret.NewValue(fmt.Sprintf("gen-%d", n))}}
			}
			id := uuid('f', n)
			return panel.Credentials{VMess: &panel.VMess{ID: id}, VLESS: &panel.VLESS{ID: id}}
		},
		Generated: map[string]panel.Credentials{},
	}
}

func basePlan() plan.Plan { return plan.Plan{Now: now, Targets: []string{"1"}} }

func build(t *testing.T, p plan.Plan, in Inputs) *Result {
	t.Helper()
	r, err := Build(p, in)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func item(t *testing.T, r *Result, key string) *Item {
	t.Helper()
	for _, it := range r.Items {
		if it.Key == key {
			return it
		}
	}
	t.Fatalf("no item %q", key)
	return nil
}

func hasConflict(it *Item, typ string) *Conflict {
	for i, c := range it.Conflicts {
		if c.Type == typ {
			return &it.Conflicts[i]
		}
	}
	for i, c := range it.Resolved {
		if c.Type == typ {
			return &it.Resolved[i]
		}
	}
	return nil
}

// pUser is a PasarGuard-sourced user.
func pUser(id int64, name string) panel.User {
	u := mUser(name, int(id))
	u.SourceID = id
	return u
}

// foldRules is a destination whose users.username ignores case.
type foldRules struct{ panel.Rules }

func (foldRules) UserKey(name string) string { return marzban.NewRules(true).UserKey(name) }

// builtInRoles are PasarGuard's three default roles, allowing everything.
func builtInRoles() []panel.Role {
	all := panel.RoleFeatures{ResetStrategy: true, NextPlan: true}
	return []panel.Role{{ID: 1, Name: "owner", IsOwner: true, Features: all}, {ID: 2, Name: "administrator", Features: all},
		{ID: 3, Name: "operator", Features: all}}
}
