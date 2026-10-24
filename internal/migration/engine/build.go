package engine

import (
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/PdYrust/karvan/internal/migration/plan"
	"github.com/PdYrust/karvan/internal/panel"
)

// maxPasses bounds the check-and-fix loop (D23).
const maxPasses = 5

// Result is the outcome of one build.
type Result struct {
	Items     []*Item
	Targets   []string // where users go unless an operation says otherwise
	Admin     AdminOutcome
	Conflicts []Conflict // job-level ones (admin, clock, targets, settling)
	Groups    []Group    // problem groups for the operator
	Stable    bool
	Blockers  int
}

// AdminOutcome is where the moved admin lands.
type AdminOutcome struct {
	Username string
	Merge    *panel.ExistingAdmin // set when merging into an existing admin
	RoleID   int64
}

// Group is one problem type with its users and ranked options.
type Group struct {
	Type    string   `json:"type"`
	Count   int      `json:"count"` // users with this conflict, fixed or open
	Options []Option `json:"options"`
	Open    int      `json:"open"` // users still without a fix
}

// Build runs the plan over the inputs. A *PlanError means the plan itself
// cannot run; everything else ends up in the result.
func Build(p plan.Plan, in Inputs) (*Result, error) {
	tree, err := compileTree(p)
	if err != nil {
		return nil, err
	}
	gen := generator(in)
	ctx := opContext{now: p.Now, generate: gen}
	res := &Result{}
	in.now = p.Now
	res.Admin, res.Conflicts = admin(p, in)
	res.Conflicts = append(res.Conflicts, chooseRole(p, &in, &res.Admin)...)
	res.Targets = p.Targets
	if len(res.Targets) == 0 {
		res.Targets = defaultTargets(in)
	}
	items := newItems(in, res.Targets)
	for _, it := range items {
		prepare(it, tree, p, in, ctx)
	}
	res.Items = items
	res.Stable = settle(p, in, items, gen, ctx)
	for _, it := range items {
		deriveStatus(&it.User, p.Now)
		if p.ResetSchedule == plan.ResetRestart && it.User.ResetStrategy != panel.ResetNever {
			it.note("reset:restart")
		}
	}
	res.Conflicts = append(res.Conflicts, jobConflicts(res.Targets, in, res.Stable)...)
	res.Conflicts = append(res.Conflicts, maxUsers(in, res.Admin, items)...)
	if out := outsideRole(in, items); len(out) > 0 {
		res.Conflicts = append(res.Conflicts, Conflict{Type: RoleGroups, Severity: Warning, Detail: map[string]string{"targets": joinSorted(out)}})
	}
	res.Groups, res.Blockers = groups(items, res.Conflicts)
	return res, nil
}

// prepare runs the categories' and the user's own operations, then gives
// the user every protocol its targets serve.
func prepare(it *Item, t *tree, p plan.Plan, in Inputs, ctx opContext) {
	fitProtocols(it, in.Dest.Kind)
	t.run(it, ctx)
	if d, ok := p.Users[it.Key]; ok {
		for _, o := range d.Ops {
			apply(it, o, ctx)
		}
		if d.Exclude {
			it.Include = false
			it.note("excluded")
		}
	}
	fillProtocols(it, in.Dest, ctx.generate)
}

// settle detects and fixes conflicts until a pass changes nothing.
// Operations the operator attached to a problem group run once, after the
// first detection.
func settle(p plan.Plan, in Inputs, items []*Item, gen func(string, string) panel.Credentials, ctx opContext) bool {
	problemOpsDone := false
	for pass := 0; pass < maxPasses; pass++ {
		detect(in, items)
		r := &resolver{p: p, in: in, rc: rank(items), taken: claimed(in, items), gen: gen}
		changed := 0
		for _, it := range items {
			changed += r.resolve(it)
		}
		if !problemOpsDone {
			changed += problemOps(p, items, ctx)
			problemOpsDone = true
		}
		if changed == 0 {
			return true
		}
	}
	detect(in, items)
	return false
}

func problemOps(p plan.Plan, items []*Item, ctx opContext) int {
	n := 0
	for _, it := range items {
		for _, c := range append(slices.Clone(it.Conflicts), it.Resolved...) {
			for _, o := range p.Problems[c.Type] {
				apply(it, o, ctx)
				n++
			}
		}
	}
	return n
}

func claimed(in Inputs, items []*Item) map[string]bool {
	out := map[string]bool{}
	for _, it := range items {
		if it.Include {
			out[in.Dest.Rules.UserKey(it.User.Username)] = true
		}
	}
	return out
}

func rank(items []*Item) rankContext {
	rc := rankContext{count: map[string]int{}}
	for _, it := range items {
		for _, c := range it.Conflicts {
			rc.count[c.Type]++
		}
	}
	return rc
}

// generator returns credentials for a fix, reusing what an earlier build
// generated for the same user and protocol (D20, D29).
func generator(in Inputs) func(key, protocol string) panel.Credentials {
	return func(key, protocol string) panel.Credentials {
		id := key + "/" + protocol
		if c, ok := in.Generated[id]; ok {
			return c
		}
		c := in.Generate(protocol)
		in.Generated[id] = c
		return c
	}
}

func admin(p plan.Plan, in Inputs) (AdminOutcome, []Conflict) {
	name := in.Source.Admin.Username
	if p.Admin.Username != "" {
		name = p.Admin.Username
	}
	out := AdminOutcome{Username: name, RoleID: p.Admin.RoleID}
	if p.Admin.Mode == "" && p.Admin.Username == "" && in.MovedAdmin != "" {
		if a, ok := in.Dest.Admins[in.MovedAdmin]; ok {
			out.Username, out.Merge = a.Username, &a
			return out, nil
		}
	}
	existing, taken := in.Dest.Admins[name]
	switch {
	case len(name) == 0 || len(name) > 34 || panel.HasControl(name) || strings.TrimSpace(name) != name:
		return out, []Conflict{{Type: AdminInvalid, Severity: Blocker, Detail: map[string]string{"username": name}}}
	case p.Admin.Mode == plan.AdminMerge && taken:
		out.Merge = &existing
	case p.Admin.Mode == plan.AdminMerge:
		return out, []Conflict{{Type: AdminTaken, Severity: Blocker, Detail: map[string]string{"username": name, "missing": "true"}}}
	case taken:
		d := map[string]string{"username": existing.Username}
		if s := freeAdminName(name, in.Dest); s != "" {
			d["suggestion"] = s
		}
		return out, []Conflict{{Type: AdminTaken, Severity: Blocker, Detail: d}}
	}
	return out, nil
}

// maxAdminName is the longest admin name the destination takes.
const maxAdminName = 34

// AdminSuggestions are the names offered when name is taken: name_2 to
// name_6, cut to fit. A check looks them up with the name itself.
func AdminSuggestions(name string) []string {
	out := make([]string, 0, 5)
	for n := 2; n <= 6; n++ {
		end := "_" + strconv.Itoa(n)
		base := []rune(name)
		for len(string(base))+len(end) > maxAdminName && len(base) > 0 {
			base = base[:len(base)-1]
		}
		if b := strings.TrimSpace(string(base)); b != "" {
			out = append(out, b+end)
		}
	}
	return out
}

// freeAdminName is the first suggestion for name that was looked up and
// is free, or "".
func freeAdminName(name string, d Dest) string {
	for _, s := range AdminSuggestions(name) {
		if _, taken := d.Admins[s]; !taken && slices.Contains(d.Asked, s) {
			return s
		}
	}
	return ""
}

// clockSkewLimit is when a source clock ahead of ours makes copied times
// lie in the destination's future (D12).
const clockSkewLimit = time.Minute

func jobConflicts(targets []string, in Inputs, stable bool) []Conflict {
	var out []Conflict
	if len(targets) == 0 {
		out = append(out, Conflict{Type: NoTargets, Severity: Blocker})
	}
	if in.Source.Skew > clockSkewLimit || in.Source.Skew < -clockSkewLimit {
		out = append(out, Conflict{Type: ClockSkew, Severity: Warning, Detail: map[string]string{"skew": in.Source.Skew.Round(time.Second).String()}})
	}
	if !stable {
		out = append(out, Conflict{Type: Unsettled, Severity: Blocker})
	}
	return out
}

func groups(items []*Item, job []Conflict) ([]Group, int) {
	byType := map[string]*Group{}
	blockers := 0
	for _, c := range job {
		if c.Severity == Blocker {
			blockers++
		}
	}
	rc := rank(items)
	for _, it := range items {
		var seen []string // the types this user is already counted in
		for _, c := range append(slices.Clone(it.Resolved), it.Conflicts...) {
			g := byType[c.Type]
			if g == nil {
				g = &Group{Type: c.Type, Options: Options(c.Type, rc)}
				byType[c.Type] = g
			}
			if !slices.Contains(seen, c.Type) {
				seen = append(seen, c.Type)
				g.Count++
			}
			if c.Fix == "" && c.Severity == Blocker {
				g.Open++
				blockers++
			}
		}
	}
	out := make([]Group, 0, len(byType))
	for _, g := range byType {
		out = append(out, *g)
	}
	slices.SortFunc(out, func(a, b Group) int { return strings.Compare(a.Type, b.Type) })
	return out, blockers
}
