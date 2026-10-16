package engine

import (
	"github.com/PdYrust/karvan/internal/migration/plan"
)

// tree is the compiled category tree: nodes in pre-order with their
// matchers, so a parent always runs before its children.
type tree struct {
	nodes []compiledNode
}

type compiledNode struct {
	id, parent string
	match      []matcher
	ops        []plan.Op
}

func compileTree(p plan.Plan) (*tree, error) {
	byID := map[string]plan.Node{}
	children := map[string][]string{}
	for _, n := range p.Tree {
		if n.ID == "" {
			return nil, planErr("category", "a category has no id")
		}
		if _, dup := byID[n.ID]; dup {
			return nil, planErr("category "+n.ID, "id used twice")
		}
		byID[n.ID] = n
		children[n.ParentID] = append(children[n.ParentID], n.ID)
	}
	for _, n := range p.Tree {
		if _, ok := byID[n.ParentID]; n.ParentID != "" && !ok {
			return nil, planErr("category "+n.ID, "parent %q does not exist", n.ParentID)
		}
	}
	t := &tree{}
	seen := map[string]bool{}
	var walk func(parent string) error
	walk = func(parent string) error {
		for _, id := range children[parent] {
			if seen[id] {
				return planErr("category "+id, "categories form a loop")
			}
			seen[id] = true
			cn, err := compileNode(byID[id], p)
			if err != nil {
				return err
			}
			t.nodes = append(t.nodes, cn)
			if err := walk(id); err != nil {
				return err
			}
		}
		return nil
	}
	if err := walk(""); err != nil {
		return nil, err
	}
	if len(seen) != len(byID) {
		return nil, planErr("category", "categories form a loop")
	}
	return t, checkPlanOps(p)
}

func compileNode(n plan.Node, p plan.Plan) (compiledNode, error) {
	cn := compiledNode{id: n.ID, parent: n.ParentID, ops: n.Ops}
	for _, c := range n.Conditions {
		m, err := compile(c, p.Now)
		if err != nil {
			return cn, err
		}
		cn.match = append(cn.match, m)
	}
	for _, o := range n.Ops {
		if err := checkOp(o); err != nil {
			return cn, err
		}
	}
	return cn, nil
}

func checkPlanOps(p plan.Plan) error {
	for _, ops := range p.Problems {
		for _, o := range ops {
			if err := checkOp(o); err != nil {
				return err
			}
		}
	}
	for _, d := range p.Users {
		for _, o := range d.Ops {
			if err := checkOp(o); err != nil {
				return err
			}
		}
	}
	for t, r := range p.Resolutions {
		if !allowed(t, r.Option) {
			return planErr("resolution "+t, "%q is not a fix for this conflict", r.Option)
		}
	}
	return nil
}

// run places it in every category it matches and applies their operations.
func (t *tree) run(it *Item, c opContext) {
	in := map[string]bool{"": true}
	for _, n := range t.nodes {
		if !in[n.parent] {
			continue
		}
		ok := true
		for _, m := range n.match {
			if !m(it) {
				ok = false
				break
			}
		}
		if !ok {
			continue
		}
		in[n.id] = true
		it.Nodes = append(it.Nodes, n.id)
		for _, o := range n.ops {
			apply(it, o, c)
		}
	}
}
