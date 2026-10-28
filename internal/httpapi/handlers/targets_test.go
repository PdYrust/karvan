package handlers_test

import (
	"maps"
	"net/http"
	"testing"
)

// D32: a plan save with targets or with the place.targets operation is
// refused, and the operations list no longer offers it.
func TestPlanRefusesTargets(t *testing.T) {
	m := newMigrationEnv(t, 1)
	id := m.create()
	var ops struct {
		Operations []struct {
			Kind string `json:"kind"`
		} `json:"operations"`
	}
	m.do(http.MethodGet, api+"/operations", nil).json(t, &ops)
	for _, o := range ops.Operations {
		if o.Kind == "place.targets" {
			t.Error("place.targets is still offered")
		}
	}
	res := m.do(http.MethodGet, api+"/migrations/"+id+"/plan", nil)
	var got struct {
		Plan map[string]any `json:"plan"`
	}
	res.json(t, &got)
	put := m.withHeader("If-Match", res.header.Get("ETag"))
	place := []map[string]any{{"kind": "place.targets", "params": map[string]string{"ids": "1"}}}
	for name, edit := range map[string]func(p map[string]any){
		"targets":  func(p map[string]any) { p["targets"] = []string{"1"} },
		"category": func(p map[string]any) { p["tree"] = []map[string]any{{"id": "a", "name": "A", "ops": place}} },
	} {
		p := maps.Clone(got.Plan)
		edit(p)
		var body struct {
			Error struct {
				Code, Where, Problem string
			} `json:"error"`
		}
		out := put(http.MethodPut, api+"/migrations/"+id+"/plan", p)
		out.json(t, &body)
		want := map[string]string{"targets": "targets/not_allowed", "category": "ops/place.targets"}[name]
		if out.status != http.StatusBadRequest || body.Error.Code != "plan_invalid" || body.Error.Where+"/"+body.Error.Problem != want {
			t.Errorf("%s: %d %s", name, out.status, out.body)
		}
	}
}
