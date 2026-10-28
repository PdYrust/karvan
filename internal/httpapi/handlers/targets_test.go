package handlers_test

import (
	"maps"
	"net/http"
	"strings"
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

// The preview names the protocols a user gets that it lacked, and its
// totals count those users (D67): group 1 serves shadowsocks, which
// user000 no longer has.
func TestPreviewShowsAddedProtocols(t *testing.T) {
	m := newMigrationEnv(t, 2)
	m.src.Mutate(func(users *[]map[string]any) {
		u := (*users)[0]
		delete(u["proxies"].(map[string]any), "shadowsocks")
		delete(u["inbounds"].(map[string]any), "shadowsocks")
	})
	id := m.create()
	res := m.do(http.MethodGet, api+"/migrations/"+id+"/plan", nil)
	var got struct {
		Plan map[string]any `json:"plan"`
	}
	res.json(t, &got)
	got.Plan["resolutions"] = map[string]any{"C10": map[string]any{"option": "accept"}}
	var after struct {
		Impact map[string]int `json:"impact"`
	}
	out := m.withHeader("If-Match", res.header.Get("ETag"))(http.MethodPut, api+"/migrations/"+id+"/plan", got.Plan)
	out.json(t, &after)
	if out.status != http.StatusOK || after.Impact["moving"] != 2 || after.Impact["protocolsAdded"] != 1 {
		t.Fatalf("put plan: %d %s", out.status, out.body)
	}
	var page struct {
		Users []struct {
			Username       string   `json:"username"`
			AddedProtocols []string `json:"addedProtocols"`
		} `json:"users"`
	}
	m.do(http.MethodGet, api+"/migrations/"+id+"/preview", nil).json(t, &page)
	if len(page.Users) != 2 || strings.Join(page.Users[0].AddedProtocols, ",") != "shadowsocks" || page.Users[1].AddedProtocols != nil {
		t.Errorf("preview %+v", page)
	}
	var roles struct {
		RoleFixed *bool `json:"roleFixed"`
	}
	m.do(http.MethodGet, api+"/migrations/"+id+"/roles", nil).json(t, &roles)
	if roles.RoleFixed == nil || *roles.RoleFixed {
		t.Errorf("roleFixed %v, want false without a merge", roles.RoleFixed)
	}
}
