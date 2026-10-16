package plan

// Template is a reusable set of choices (D28): the category tree with its
// operations, problem-group operations and fixes. It names no user, admin
// or destination, so it fits any migration.
type Template struct {
	Name          string                `json:"name"`
	Tree          []Node                `json:"tree"`
	Problems      map[string][]Op       `json:"problems,omitempty"`
	Resolutions   map[string]Resolution `json:"resolutions,omitempty"`
	ResetSchedule string                `json:"resetSchedule,omitempty"`
}

// TemplateOf takes the reusable part of p. Categories that pick users by
// name are left out, because those users belong to this migration only.
func TemplateOf(name string, p Plan) Template {
	t := Template{Name: name, Problems: p.Problems, Resolutions: p.Resolutions, ResetSchedule: p.ResetSchedule}
	dropped := map[string]bool{}
	for _, n := range p.Tree {
		if dropped[n.ParentID] || picksUsers(n) {
			dropped[n.ID] = true
			continue
		}
		t.Tree = append(t.Tree, n)
	}
	return t
}

func picksUsers(n Node) bool {
	for _, c := range n.Conditions {
		if c.Field == "users" {
			return true
		}
	}
	return false
}

// Apply replaces p's reusable choices with t's and keeps the rest.
func (t Template) Apply(p Plan) Plan {
	p.Tree, p.Problems, p.Resolutions, p.ResetSchedule = t.Tree, t.Problems, t.Resolutions, t.ResetSchedule
	return p
}
