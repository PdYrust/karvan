package engine

import (
	"cmp"
	"slices"
	"strings"

	"github.com/PdYrust/karvan/internal/migration/plan"
	"github.com/PdYrust/karvan/internal/panel"
)

// AdminOutcome is where one source admin lands.
type AdminOutcome struct {
	Source   panel.Admin          // as read; a user's owner the login cannot read is known by name only
	IsLogin  bool                 // the admin whose login reads the old panel
	Username string               // its name on the destination
	Merge    *panel.ExistingAdmin // set when merging into an existing admin
	RoleID   int64
	Targets  []string // where its users go: its role's groups, or every inbound
	Users    int      // its users this build moves

	// role is the PasarGuard role it gets, and limits that role's limits
	// with the admin's own overrides.
	role   *panel.Role
	limits panel.RoleLimits
}

// Login is the login's admin, first in every build.
func (r *Result) Login() *AdminOutcome {
	if len(r.Admins) == 0 {
		return nil
	}
	return &r.Admins[0]
}

// Moves reports whether the run writes the admin: the login's always,
// another only when it has users to move.
func (o AdminOutcome) Moves() bool { return o.IsLogin || o.Users > 0 }

// dropIdle drops the conflicts of admins that do not move.
func dropIdle(admins []AdminOutcome, cs []Conflict) []Conflict {
	return slices.DeleteFunc(cs, func(c Conflict) bool {
		i := slices.IndexFunc(admins, func(a AdminOutcome) bool { return a.Source.Username == c.Detail["admin"] })
		return i >= 0 && !admins[i].Moves()
	})
}

// ownerOf is the source admin u belongs to: its owner, else the login's.
func ownerOf(u panel.User, login string) string { return cmp.Or(u.OwnerUsername, login) }

// sourceAdmins are the admins that move: the login's first, then
// every other owner of a source user, by name. One read by the source
// keeps its facts; another is known by its name only.
func sourceAdmins(src Source) []panel.Admin {
	login := src.Admin.Username
	seen := map[string]bool{login: true}
	var names []string
	for _, u := range src.Users {
		if o := ownerOf(u, login); !seen[o] {
			seen[o] = true
			names = append(names, o)
		}
	}
	slices.Sort(names)
	out := []panel.Admin{src.Admin}
	for _, n := range names {
		a := panel.Admin{Username: n}
		if i := slices.IndexFunc(src.Admins, func(x panel.Admin) bool { return x.Username == n }); i >= 0 {
			a = src.Admins[i]
		}
		out = append(out, a)
	}
	return out
}

// LookupNames are the destination admin names a build of p over src
// looks up: per source admin its source name, its name in the plan with
// that name's suggestions, and where an earlier run moved it (moved, by
// source admin).
func LookupNames(p plan.Plan, src Source, moved map[string]string) []string {
	p = p.ForLogin(src.Admin.Username)
	var out []string
	add := func(n string) {
		if n != "" && !slices.Contains(out, n) {
			out = append(out, n)
		}
	}
	for _, a := range sourceAdmins(src) {
		name := cmp.Or(p.Admins[a.Username].Username, a.Username)
		add(name)
		for _, s := range AdminSuggestions(name) {
			add(s)
		}
		add(a.Username)
		add(moved[a.Username])
	}
	return out
}

// landAdmins decides where every source admin lands, with its role and
// targets. Each conflict names its source admin in Detail["admin"].
func landAdmins(p plan.Plan, in Inputs) ([]AdminOutcome, []Conflict) {
	admins := sourceAdmins(in.Source)
	out := make([]AdminOutcome, len(admins))
	found := make([][]Conflict, len(admins))
	for i, a := range admins {
		out[i], found[i] = land(a, p.Admins[a.Username], in.MovedAdmins[a.Username], in.Dest)
		out[i].IsLogin = i == 0
	}
	// Two admins of the job cannot be created under one name: the later
	// one is flagged. Names compare without case, as a destination on
	// MySQL does.
	used := map[string]bool{}
	for i, o := range out {
		switch k := strings.ToLower(o.Username); {
		case found[i] != nil || o.Merge != nil:
		case used[k]:
			found[i] = []Conflict{{Type: AdminTaken, Severity: Blocker, Detail: map[string]string{"username": o.Username}}}
		default:
			used[k] = true
		}
	}
	var cs []Conflict
	for i := range out {
		o := &out[i]
		for _, c := range found[i] {
			if c.Type == AdminTaken && c.Detail["missing"] == "" {
				if s := freeAdminName(o.Username, in.Dest, used); s != "" {
					c.Detail["suggestion"], used[strings.ToLower(s)] = s, true
				}
			}
		}
		found[i] = append(found[i], chooseRole(p.Admins[o.Source.Username], o, in.Dest)...)
		o.Targets = defaultTargets(in.Dest, o.role)
		if len(o.Targets) == 0 {
			found[i] = append(found[i], Conflict{Type: NoTargets, Severity: Blocker})
		}
		for _, c := range found[i] {
			c.Detail = merge(c.Detail, map[string]string{"admin": o.Source.Username})
			cs = append(cs, c)
		}
	}
	return out, cs
}

// land is where admin a goes by its plan entry: a re-run merges into the
// admin an earlier run moved it to (moved) unless the plan says otherwise.
func land(a panel.Admin, ap plan.AdminPlan, moved string, d Dest) (AdminOutcome, []Conflict) {
	name := cmp.Or(ap.Username, a.Username)
	out := AdminOutcome{Source: a, Username: name, RoleID: ap.RoleID}
	if ap.Mode == "" && ap.Username == "" && moved != "" {
		if e, ok := d.Admins[moved]; ok {
			out.Username, out.Merge = e.Username, &e
			return out, nil
		}
	}
	existing, taken := d.Admins[name]
	switch {
	case len(name) == 0 || len(name) > maxAdminName || panel.HasControl(name) || strings.TrimSpace(name) != name:
		return out, []Conflict{{Type: AdminInvalid, Severity: Blocker, Detail: map[string]string{"username": name}}}
	case ap.Mode == plan.AdminMerge && taken:
		out.Merge = &existing
	case ap.Mode == plan.AdminMerge:
		return out, []Conflict{{Type: AdminTaken, Severity: Blocker, Detail: map[string]string{"username": name, "missing": "true"}}}
	case taken:
		return out, []Conflict{{Type: AdminTaken, Severity: Blocker, Detail: map[string]string{"username": existing.Username}}}
	}
	return out, nil
}

// countUsers sets each admin's moved users.
func countUsers(admins []AdminOutcome, items []*Item) {
	n := map[string]int{}
	for _, it := range items {
		if it.Include {
			n[it.Admin]++
		}
	}
	for i := range admins {
		admins[i].Users = n[admins[i].Source.Username]
	}
}
