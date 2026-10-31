package engine

import (
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/PdYrust/karvan/internal/migration/plan"
	"github.com/PdYrust/karvan/internal/panel"
)

// maxPasses bounds the check-and-fix loop.
const maxPasses = 5

// Result is the outcome of one build.
type Result struct {
	Items     []*Item
	Admins    []AdminOutcome // the login's first
	Conflicts []Conflict     // job-level ones (admins, clock, targets, settling)
	Groups    []Group        // problem groups for the operator
	Stable    bool
	Blockers  int
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
	p = p.ForLogin(in.Source.Admin.Username)
	gen := generator(in)
	ctx := opContext{now: p.Now, generate: gen}
	res := &Result{}
	in.now = p.Now
	res.Admins, res.Conflicts = landAdmins(p, in) // targets never from p.Targets
	in.admins = map[string]*AdminOutcome{}
	for i := range res.Admins {
		in.admins[res.Admins[i].Source.Username] = &res.Admins[i]
	}
	items := newItems(in)
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
	countUsers(res.Admins, items)
	res.Conflicts = dropIdle(res.Admins, res.Conflicts)
	res.Conflicts = append(res.Conflicts, jobConflicts(in, res.Stable)...)
	res.Conflicts = append(res.Conflicts, maxUsers(res.Admins)...)
	res.Groups, res.Blockers = groups(items, res.Conflicts)
	return res, nil
}

// prepare runs the categories' and the user's own operations, then gives
// the user every protocol its targets serve.
func prepare(it *Item, t *tree, p plan.Plan, in Inputs, ctx opContext) {
	markLostProtocols(it, in)
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
// generated for the same user and protocol.
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

// freeAdminName is the first suggestion for name that was looked up, is
// free and no admin of the job uses (used holds lower case names), or "".
func freeAdminName(name string, d Dest, used map[string]bool) string {
	for _, s := range AdminSuggestions(name) {
		if _, taken := d.Admins[s]; !taken && !used[strings.ToLower(s)] && slices.Contains(d.Asked, s) {
			return s
		}
	}
	return ""
}

// clockSkewLimit is when a source clock ahead of ours makes copied times
// lie in the destination's future.
const clockSkewLimit = time.Minute

func jobConflicts(in Inputs, stable bool) []Conflict {
	var out []Conflict
	if in.Source.Skew > clockSkewLimit || in.Source.Skew < -clockSkewLimit {
		out = append(out, Conflict{Type: ClockSkew, Severity: Warning, Detail: map[string]string{"skew": in.Source.Skew.Round(time.Second).String()}})
	}
	if !stable {
		out = append(out, Conflict{Type: Unsettled, Severity: Blocker})
	}
	if in.Source.Kind == panel.PasarGuard && !in.Source.GroupsRead {
		out = append(out, Conflict{Type: GroupsUnknown, Severity: Info})
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
