// Package plan is the operator's migration plan as Karvan stores it: the
// category tree with its operations, the conflict resolutions, per-user
// decisions and the destination choices. It holds no secret, so it can be
// saved and shown.
package plan

import (
	"maps"
	"time"
)

// Plan is everything the operator decided for one migration.
type Plan struct {
	Version int       `json:"version"`
	Now     time.Time `json:"now"` // "now" for every time operation, fixed so a rebuild gives the same result

	// Admins is how each source admin lands, by its source username
	// (D35). An admin with no entry keeps its name and gets the default.
	Admins map[string]AdminPlan `json:"admins,omitempty"`
	// Admin is a plan from before Phase 4.1, for the login's admin. It is
	// read, never written: ForLogin moves it into Admins.
	Admin   AdminPlan `json:"admin,omitzero"`
	Targets []string  `json:"targets"` // destination group ids or "<protocol>/<tag>" inbounds

	// Tree is the operator's categories, in display order. Problem groups
	// are not stored; Karvan makes them from conflicts on every build.
	Tree []Node `json:"tree"`
	// Problems holds operations the operator added to a problem group.
	Problems map[string][]Op `json:"problems,omitempty"`

	// Resolutions are "apply to all similar" choices per conflict type.
	Resolutions map[string]Resolution `json:"resolutions,omitempty"`
	// Users are per-user decisions, keyed by source identity.
	Users map[string]UserDecision `json:"users,omitempty"`

	// ResetSchedule is how periodic resets continue (conflict C26):
	// "estimate" (from created_at) or "restart" (a new period now).
	ResetSchedule string `json:"resetSchedule,omitempty"`
}

// ForLogin returns p with an Admin entry moved to Admins under login, the
// source admin whose login reads the old panel (D35). Only a client that
// knows no Admins sends Admin, so it wins over the login's entry. p and
// its map are left as they were.
func (p Plan) ForLogin(login string) Plan {
	if p.Admin == (AdminPlan{}) {
		return p
	}
	admins := maps.Clone(p.Admins)
	if admins == nil {
		admins = map[string]AdminPlan{}
	}
	admins[login] = p.Admin
	p.Admins, p.Admin = admins, AdminPlan{}
	return p
}

// AdminPlan is how one source admin lands on the destination.
type AdminPlan struct {
	// Username is the final name; empty keeps the source name.
	Username string `json:"username,omitempty"`
	// Mode is "create" (default) or "merge" into an existing admin with
	// Username.
	Mode   string `json:"mode,omitempty"`
	RoleID int64  `json:"roleId,omitempty"` // PasarGuard; 0 means the operator role
}

// Node is one category: its members are its parent's members that match
// every condition. Operations run parent first, then child.
type Node struct {
	ID         string      `json:"id"`
	ParentID   string      `json:"parentId,omitempty"`
	Name       string      `json:"name"`
	Conditions []Condition `json:"conditions"`
	Ops        []Op        `json:"ops"`
}

// Condition is one filter on a source user. Field names the value, Op the
// comparison, Value its argument (always a string, parsed per field).
type Condition struct {
	Field string   `json:"field"`
	Op    string   `json:"op"`
	Value string   `json:"value,omitempty"`
	List  []string `json:"list,omitempty"`
}

// Op is one operation with its parameters.
type Op struct {
	Kind   string            `json:"kind"`
	Params map[string]string `json:"params,omitempty"`
}

// Resolution is a chosen fix for a conflict.
type Resolution struct {
	Option string            `json:"option"`
	Params map[string]string `json:"params,omitempty"`
}

// UserDecision overrides everything for one user.
type UserDecision struct {
	Exclude     bool                  `json:"exclude,omitempty"`
	Resolutions map[string]Resolution `json:"resolutions,omitempty"`
	Ops         []Op                  `json:"ops,omitempty"`
}

// ResetEstimate and ResetRestart are the C26 choices.
const (
	ResetEstimate = "estimate"
	ResetRestart  = "restart"
)

// AdminCreate and AdminMerge are the AdminPlan modes.
const (
	AdminCreate = "create"
	AdminMerge  = "merge"
)
