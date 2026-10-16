package reload

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"sync"

	"github.com/PdYrust/karvan/internal/migration/run"
	"github.com/PdYrust/karvan/internal/panel"
)

// workers is how many users are touched at once. The panel writes one
// row per call, so more would only queue on its database.
const workers = 4

// maxFailed is how many failures the report lists by name.
const maxFailed = 100

// Methods are the API reloaders, keyed for run.Deps.Reloaders.
func Methods() map[string]run.Reloader {
	return map[string]run.Reloader{run.ReloadTouch: Touch{}, run.ReloadCore: CoreRestart{}}
}

// Touch sends each written user through the panel's own edit path, with
// nothing changed. The panel then adds the user to its core and nodes
// and, on PasarGuard, gives WireGuard users their peer IPs. Each edit is
// a "user modified" event, so the panel sends its usual notifications.
type Touch struct{}

// Reload implements run.Reloader.
func (Touch) Reload(ctx context.Context, in run.ReloadInput) run.ReloadResult {
	res := run.ReloadResult{Method: run.ReloadTouch}
	c, err := newClient(in.API)
	if err != nil {
		res.Error = err.Error()
		return res
	}
	touchAll(ctx, c, in.Kind, in.Users, &res)
	return res
}

// CoreRestart restarts the core once. Marzban rebuilds its core config
// from the database and restarts the core and every connected node;
// PasarGuard pushes every user to every node again. Users stay connected
// to nothing for the few seconds a restart takes. PasarGuard WireGuard
// users are touched first, because only an edit gives them a peer IP.
type CoreRestart struct{}

// Reload implements run.Reloader.
func (CoreRestart) Reload(ctx context.Context, in run.ReloadInput) run.ReloadResult {
	res := run.ReloadResult{Method: run.ReloadCore}
	c, err := newClient(in.API)
	if err != nil {
		res.Error = err.Error()
		return res
	}
	path := "/api/core/restart"
	if in.Kind == panel.PasarGuard {
		path = "/api/nodes/reconnect"
		var wg []run.ReloadTarget
		for _, u := range in.Users {
			if u.WireGuard {
				wg = append(wg, u)
			}
		}
		touchAll(ctx, c, in.Kind, wg, &res)
	}
	if err := c.do(ctx, http.MethodPost, path, nil, nil); err != nil {
		res.Error = err.Error()
		return res
	}
	res.OK = len(in.Users) - len(res.Failed)
	return res
}

// touchAll touches users with a few workers and counts the outcome into
// res. It stops early when ctx ends.
func touchAll(ctx context.Context, c *client, k panel.Kind, users []run.ReloadTarget, res *run.ReloadResult) {
	jobs := make(chan run.ReloadTarget)
	var mu sync.Mutex
	var wg sync.WaitGroup
	failed := 0
	for range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for u := range jobs {
				err := touch(ctx, c, k, u)
				mu.Lock()
				switch {
				case err == nil:
					res.OK++
				case failed < maxFailed:
					res.Failed = append(res.Failed, u.Username+": "+err.Error())
					failed++
				default:
					failed++
				}
				mu.Unlock()
			}
		}()
	}
	for _, u := range users {
		if ctx.Err() != nil {
			break
		}
		jobs <- u
	}
	close(jobs)
	wg.Wait()
	if failed > maxFailed {
		res.Failed = append(res.Failed, fmt.Sprintf("%d more", failed-maxFailed))
	}
	if ctx.Err() != nil && res.Error == "" {
		res.Error = ctx.Err().Error()
	}
}

// touch reads the user and writes back what an empty edit would lose:
// both panels delete next_plan when the edit leaves it out, and
// PasarGuard runs its WireGuard allocation only when group_ids is set.
func touch(ctx context.Context, c *client, k panel.Kind, u run.ReloadTarget) error {
	path := "/api/user/" + url.PathEscape(u.Username)
	if k == panel.PasarGuard {
		path = "/api/user/by-id/" + strconv.FormatInt(u.ID, 10)
	}
	var cur map[string]json.RawMessage
	if err := c.do(ctx, http.MethodGet, path, nil, &cur); err != nil {
		return err
	}
	body := map[string]json.RawMessage{}
	if np := cur["next_plan"]; set(np) {
		body["next_plan"] = np
	}
	if g := cur["group_ids"]; k == panel.PasarGuard && set(g) && string(g) != "[]" {
		body["group_ids"] = g
	}
	return c.do(ctx, http.MethodPut, path, body, nil)
}

func set(v json.RawMessage) bool { return len(v) > 0 && string(v) != "null" }
