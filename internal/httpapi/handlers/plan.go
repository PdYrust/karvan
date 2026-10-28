package handlers

import (
	"errors"
	"net/http"
	"slices"
	"strconv"
	"strings"

	"github.com/PdYrust/karvan/internal/audit"
	"github.com/PdYrust/karvan/internal/httpapi/respond"
	"github.com/PdYrust/karvan/internal/migration/engine"
	"github.com/PdYrust/karvan/internal/migration/plan"
	"github.com/PdYrust/karvan/internal/migration/run"
	"github.com/PdYrust/karvan/internal/secret"
)

// maxPlanBytes caps a plan body: categories plus per-user choices for
// 100k users fit well under it.
const maxPlanBytes = 8 << 20

// Operations lists the ready-made operations and their parameters
// (step 3).
func (a *API) Operations(w http.ResponseWriter, _ *http.Request) {
	type opJSON struct {
		Kind   string   `json:"kind"`
		Params []string `json:"params"`
	}
	out := make([]opJSON, 0, len(engine.Ops))
	for k, p := range engine.Ops {
		out = append(out, opJSON{Kind: k, Params: orEmpty(p)})
	}
	slices.SortFunc(out, func(a, b opJSON) int { return strings.Compare(a.Kind, b.Kind) })
	respond.JSON(w, http.StatusOK, map[string]any{"operations": out})
}

// GetPlan returns the plan with its version, also as the ETag that PUT
// must send back in If-Match.
func (a *API) GetPlan(w http.ResponseWriter, r *http.Request) {
	res, p, version, err := a.Runs.View(r.PathValue("id"))
	if err != nil {
		writeRunError(w, r, err)
		return
	}
	w.Header().Set("ETag", strconv.Quote(strconv.Itoa(version)))
	out := map[string]any{"planVersion": version, "plan": p}
	if res != nil {
		out["impact"] = impact(res)
	}
	respond.JSON(w, http.StatusOK, out)
}

// PutPlan replaces the plan if If-Match names the current version, and
// returns what it changes (steps 3 and 4).
func (a *API) PutPlan(w http.ResponseWriter, r *http.Request) {
	version, err := strconv.Atoi(strings.Trim(r.Header.Get("If-Match"), `"`))
	if err != nil {
		respond.Error(w, respond.APIError{Status: http.StatusPreconditionRequired, Code: codeStalePlan})
		return
	}
	var p plan.Plan
	if err := respond.DecodeLimit(w, r, &p, maxPlanBytes); err != nil {
		respond.BadRequest(w)
		return
	}
	res, now, err := a.Runs.SetPlan(r.Context(), r.PathValue("id"), version, p)
	if errors.Is(err, run.ErrStaleVer) {
		respond.Error(w, respond.APIError{Status: http.StatusPreconditionFailed, Code: codeStalePlan, Extra: map[string]any{"planVersion": now}})
		return
	}
	if err != nil {
		writeRunError(w, r, err)
		return
	}
	w.Header().Set("ETag", strconv.Quote(strconv.Itoa(now)))
	respond.JSON(w, http.StatusOK, map[string]any{"planVersion": now, "blockers": res.Blockers, "stable": res.Stable,
		"groups": orEmpty(res.Groups), "conflicts": orEmpty(res.Conflicts), "impact": impact(res)})
}

// impact counts what the plan does, for the operator before step 5.
func impact(res *engine.Result) map[string]int {
	out := map[string]int{"users": len(res.Items)}
	for _, it := range res.Items {
		switch {
		case !it.Include:
			out["skipped"]++
		default:
			out["moving"]++
		}
		if !it.Include && it.AlreadyMoved() {
			out["alreadyMoved"]++
		}
		if it.Include && it.ChangesConfig {
			out["configChanged"]++
		}
		if it.Include && it.User.Username != it.Source.Username {
			out["renamed"]++
		}
		if it.Include && len(it.Applied) > 0 {
			out["changed"]++
		}
		if it.Include && len(addedProtocols(it)) > 0 {
			out["protocolsAdded"]++
		}
	}
	return out
}

// addedProtocols lists the protocols a user gets that it lacked (D67).
func addedProtocols(it *engine.Item) []string {
	var out []string
	for _, a := range it.Applied {
		if p, ok := strings.CutPrefix(a, "added:"); ok {
			out = append(out, p)
		}
	}
	return out
}

// ReloadOptions lists every reload method for the final step (Q5).
func (a *API) ReloadOptions(w http.ResponseWriter, r *http.Request) {
	opts, err := a.Runs.ReloadOptions(r.Context(), r.PathValue("id"))
	if err != nil {
		writeRunError(w, r, err)
		return
	}
	respond.JSON(w, http.StatusOK, map[string]any{"options": opts})
}

type runBody struct {
	PlanVersion   int    `json:"planVersion"`
	Reload        string `json:"reload"`
	AdminPassword string `json:"adminPassword"`
}

// RunMigration starts step 5. Safe Mode comes from the setting, never
// from the request, so turning it off always goes through its own
// confirmation.
func (a *API) RunMigration(w http.ResponseWriter, r *http.Request) {
	var b runBody
	if err := respond.Decode(w, r, &b); err != nil || b.Reload == "" || len(b.AdminPassword) > 1024 {
		respond.BadRequest(w)
		return
	}
	sm, err := a.safeMode(r)
	if err != nil {
		respond.Internal(w, r, err)
		return
	}
	safe := sm.Enabled
	id := r.PathValue("id")
	err = a.Runs.Run(r.Context(), id, run.Options{Version: b.PlanVersion, Reload: b.Reload, SafeMode: safe,
		Rollback: sm.Rollback, AdminPassword: secret.NewValue(b.AdminPassword)})
	if errors.Is(err, run.ErrStaleVer) {
		respond.Error(w, respond.APIError{Status: http.StatusPreconditionFailed, Code: codeStalePlan})
		return
	}
	if err != nil {
		writeRunError(w, r, err)
		return
	}
	a.record(r, audit.ActionMigrationRun, audit.Success, map[string]string{"job": id, "reload": b.Reload, "safeMode": strconv.FormatBool(safe), "rollback": sm.Rollback})
	respond.JSON(w, http.StatusAccepted, map[string]any{"id": id, "state": run.StateRunning})
}

// CancelMigration stops a running job between chunks.
func (a *API) CancelMigration(w http.ResponseWriter, r *http.Request) {
	if err := a.Runs.Cancel(r.PathValue("id")); err != nil {
		writeRunError(w, r, err)
		return
	}
	a.record(r, audit.ActionMigrationStop, audit.Success, map[string]string{"job": r.PathValue("id")})
	respond.NoContent(w)
}

// RollbackMigration starts undoing a failed or interrupted job (D39, B5);
// the outcome follows on the job's event stream.
func (a *API) RollbackMigration(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	err := a.Runs.Rollback(r.Context(), id)
	switch {
	case errors.Is(err, run.ErrNoRollback):
		respond.Error(w, respond.APIError{Status: http.StatusConflict, Code: codeNotRollbackable})
		return
	case errors.Is(err, run.ErrLocked):
		respond.Error(w, respond.APIError{Status: http.StatusConflict, Code: codeDestLocked})
		return
	}
	if err != nil {
		a.record(r, audit.ActionMigrationUndo, audit.Failure, map[string]string{"job": id})
		writeRunError(w, r, err)
		return
	}
	a.record(r, audit.ActionMigrationUndo, audit.Success, map[string]string{"job": id})
	respond.JSON(w, http.StatusAccepted, map[string]any{"id": id})
}

// Roles lists the destination's admin roles for the final admin step:
// each with its groups, host counts, limits and features, the role the
// moved admin gets now (D65) and whether a merge fixes it (D33). Empty on
// a Marzban destination.
func (a *API) Roles(w http.ResponseWriter, r *http.Request) {
	roles, err := a.Runs.Roles(r.PathValue("id"))
	if err != nil {
		writeRunError(w, r, err)
		return
	}
	respond.JSON(w, http.StatusOK, roles)
}
