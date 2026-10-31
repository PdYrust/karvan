package handlers_test

import (
	"net/http"
	"os"
	"testing"
)

// GET /settings/install says whether Karvan runs as root, for the read-only
// row in Settings; signed out it is a 401 like every other route.
func TestInstallSettingsSayWhetherKarvanRunsAsRoot(t *testing.T) {
	e := newEnv(t)
	if res := e.do(http.MethodGet, api+"/settings/install", nil); res.status != http.StatusUnauthorized {
		t.Fatalf("signed out: %d", res.status)
	}
	e.login("")
	var out struct {
		RunsAsRoot *bool `json:"runsAsRoot"`
	}
	res := e.do(http.MethodGet, api+"/settings/install", nil)
	res.json(t, &out)
	if res.status != http.StatusOK || out.RunsAsRoot == nil || *out.RunsAsRoot != (os.Geteuid() == 0) {
		t.Fatalf("got %d %s", res.status, res.body)
	}
	if res := e.do(http.MethodPut, api+"/settings/install", map[string]bool{"runsAsRoot": true}); res.status != http.StatusMethodNotAllowed && res.status != http.StatusNotFound {
		t.Errorf("PUT: %d, want it read only", res.status)
	}
}
