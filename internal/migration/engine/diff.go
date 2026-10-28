package engine

import (
	"cmp"
	"slices"
	"strings"
)

// Changes is how the outcome of a build differs from an earlier build's
// (D29): who moves, under which name, with which credentials and targets,
// which conflicts were fixed how, and where each admin lands. Traffic,
// online times and derived status are left out: they change all the time
// and are read fresh when the run writes.
type Changes struct {
	// Admin is the login's admin, for a client that reads no Admins.
	Admin     *AdminChange    `json:"admin"`
	Admins    []AdminChange   `json:"admins"` // every admin whose outcome differs, by source name
	Conflicts *ConflictChange `json:"conflicts"`
	Users     []UserChange    `json:"users"`
}

// Changed reports whether the outcome differs at all.
func (c Changes) Changed() bool {
	return c.Admin != nil || len(c.Admins) > 0 || c.Conflicts != nil || len(c.Users) > 0
}

// AdminChange is where a source admin lands before and after. Change is
// new, gone or changed, as for users; Fields names what differs for a
// changed one: username, merged, roleId.
type AdminChange struct {
	Source string       `json:"source"`
	Change string       `json:"change"`
	Fields []string     `json:"fields,omitempty"`
	Before AdminSummary `json:"before"`
	After  AdminSummary `json:"after"`
}

// AdminSummary is the admin's outcome: its name on the destination,
// whether it merges into an admin there, and its role.
type AdminSummary struct {
	Username string `json:"username"`
	Merged   bool   `json:"merged"`
	RoleID   int64  `json:"roleId"`
}

// ConflictChange is the job-level conflicts before and after.
type ConflictChange struct {
	Before []Conflict `json:"before"`
	After  []Conflict `json:"after"`
}

// What happened to a user between two builds.
const (
	UserNew     = "new"     // only in the later build
	UserGone    = "gone"    // only in the earlier one
	UserChanged = "changed" // in both, with another outcome
)

// UserChange is one user whose outcome differs. Fields names what differs
// for a changed user: include, admin, adopted, username, targets, credentials,
// conflicts.
type UserChange struct {
	Key    string       `json:"key"`
	Change string       `json:"change"`
	Fields []string     `json:"fields,omitempty"`
	Before *UserOutcome `json:"before,omitempty"`
	After  *UserOutcome `json:"after,omitempty"`
}

// UserOutcome is a user's outcome as Changes compares it. It holds no
// credential; Protocols names the ones the user has.
type UserOutcome struct {
	SourceUsername string     `json:"sourceUsername"`
	Admin          string     `json:"admin"` // the source admin
	Username       string     `json:"username"`
	Include        bool       `json:"include"`
	Adopted        bool       `json:"adopted"`
	Targets        []string   `json:"targets"`
	Protocols      []string   `json:"protocols"`
	Conflicts      []Conflict `json:"conflicts"` // type, severity and fix only
}

// Diff compares two builds. Without an earlier build there is nothing to
// compare with, and the changes are empty.
func Diff(before, after *Result) Changes {
	if before == nil || after == nil {
		return Changes{}
	}
	c := Changes{Admins: adminChanges(before.Admins, after.Admins), Users: userChanges(before.Items, after.Items)}
	for i := range c.Admins {
		if a, l := &c.Admins[i], after.Login(); l != nil && a.Source == l.Source.Username && a.Change == UserChanged {
			c.Admin = a
		}
	}
	if !slices.Equal(conflictKinds(before.Conflicts), conflictKinds(after.Conflicts)) {
		c.Conflicts = &ConflictChange{Before: append([]Conflict{}, before.Conflicts...), After: append([]Conflict{}, after.Conflicts...)}
	}
	return c
}

func adminChanges(before, after []AdminOutcome) []AdminChange {
	var out []AdminChange
	for _, b := range after {
		i := slices.IndexFunc(before, func(a AdminOutcome) bool { return a.Source.Username == b.Source.Username })
		switch {
		case i < 0:
			out = append(out, AdminChange{Source: b.Source.Username, Change: UserNew, After: b.Summary()})
		case adminChange(before[i], b) != nil:
			out = append(out, *adminChange(before[i], b))
		}
	}
	for _, a := range before {
		if !slices.ContainsFunc(after, func(b AdminOutcome) bool { return b.Source.Username == a.Source.Username }) {
			out = append(out, AdminChange{Source: a.Source.Username, Change: UserGone, Before: a.Summary()})
		}
	}
	slices.SortFunc(out, func(a, b AdminChange) int { return strings.Compare(a.Source, b.Source) })
	return out
}

func adminChange(a, b AdminOutcome) *AdminChange {
	var fields []string
	if a.Username != b.Username {
		fields = append(fields, "username")
	}
	if mergeID(a) != mergeID(b) {
		fields = append(fields, "merged")
	}
	if a.RoleID != b.RoleID {
		fields = append(fields, "roleId")
	}
	if fields == nil {
		return nil
	}
	return &AdminChange{Source: b.Source.Username, Change: UserChanged, Fields: fields, Before: a.Summary(), After: b.Summary()}
}

func mergeID(a AdminOutcome) int64 {
	if a.Merge == nil {
		return 0
	}
	return a.Merge.ID
}

// Summary is where the admin lands, as reports show it.
func (a AdminOutcome) Summary() AdminSummary {
	return AdminSummary{Username: a.Username, Merged: a.Merge != nil, RoleID: a.RoleID}
}

// conflictKinds is the job-level conflicts as compared: type, severity
// and source admin.
func conflictKinds(cs []Conflict) []string {
	out := make([]string, len(cs))
	for i, c := range cs {
		out[i] = c.Type + "|" + c.Severity + "|" + c.Detail["admin"]
	}
	slices.Sort(out)
	return out
}

func userChanges(before, after []*Item) []UserChange {
	left := make(map[string]*Item, len(before))
	for _, it := range before {
		left[it.Key] = it
	}
	var out []UserChange
	for _, it := range after {
		prev, ok := left[it.Key]
		delete(left, it.Key)
		switch fields := differs(prev, it); {
		case !ok:
			out = append(out, UserChange{Key: it.Key, Change: UserNew, After: outcomeOf(it)})
		case fields != nil:
			out = append(out, UserChange{Key: it.Key, Change: UserChanged, Fields: fields, Before: outcomeOf(prev), After: outcomeOf(it)})
		}
	}
	for _, it := range before {
		if _, gone := left[it.Key]; gone {
			out = append(out, UserChange{Key: it.Key, Change: UserGone, Before: outcomeOf(it)})
		}
	}
	slices.SortFunc(out, func(a, b UserChange) int { return compareKeys(a.Key, b.Key) })
	return out
}

// differs names the compared fields in which two outcomes of one user
// differ; nil when they are the same or a is missing.
func differs(a, b *Item) []string {
	if a == nil {
		return nil
	}
	var out []string
	for _, f := range []struct {
		name string
		same bool
	}{
		{"include", a.Include == b.Include},
		{"admin", a.Admin == b.Admin},
		{"adopted", adoptID(a) == adoptID(b)},
		{"username", a.User.Username == b.User.Username},
		{"targets", slices.Equal(a.Targets, b.Targets)},
		{"credentials", slices.Equal(credentialKeys(a), credentialKeys(b))},
		{"conflicts", slices.EqualFunc(marks(a), marks(b), sameMark)},
	} {
		if !f.same {
			out = append(out, f.name)
		}
	}
	return out
}

func adoptID(it *Item) int64 {
	if it.Adopt == nil {
		return 0
	}
	return it.Adopt.ID
}

// credentialKeys holds secrets: they are compared here and never leave.
func credentialKeys(it *Item) []string {
	keys := it.User.Credentials.Keys()
	out := make([]string, len(keys))
	for i, k := range keys {
		out[i] = string(k)
	}
	slices.Sort(out)
	return out
}

// marks is every conflict of a user, fixed or open, with its type,
// severity and fix, in one order.
func marks(it *Item) []Conflict {
	out := make([]Conflict, 0, len(it.Resolved)+len(it.Conflicts))
	for _, c := range append(slices.Clone(it.Resolved), it.Conflicts...) {
		out = append(out, Conflict{Type: c.Type, Severity: c.Severity, Fix: c.Fix})
	}
	slices.SortFunc(out, func(a, b Conflict) int {
		return cmp.Or(strings.Compare(a.Type, b.Type), strings.Compare(a.Fix, b.Fix), strings.Compare(a.Severity, b.Severity))
	})
	return out
}

func sameMark(a, b Conflict) bool {
	return a.Type == b.Type && a.Fix == b.Fix && a.Severity == b.Severity
}

func outcomeOf(it *Item) *UserOutcome {
	protocols := []string{}
	for _, k := range credentialKeys(it) {
		if p, _, _ := strings.Cut(k, ":"); !slices.Contains(protocols, p) {
			protocols = append(protocols, p)
		}
	}
	slices.Sort(protocols)
	return &UserOutcome{SourceUsername: it.Source.Username, Admin: it.Admin, Username: it.User.Username, Include: it.Include, Adopted: it.Adopt != nil,
		Targets: append([]string{}, it.Targets...), Protocols: protocols, Conflicts: marks(it)}
}
