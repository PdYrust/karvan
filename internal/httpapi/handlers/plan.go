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
	_, p, version, err := a.Runs.View(r.PathValue("id"))
	if err != nil {
		writeRunError(w, r, err)
		return
	}
	w.Header().Set("ETag", strconv.Quote(strconv.Itoa(version)))
	respond.JSON(w, http.StatusOK, map[string]any{"planVersion": version, "plan": p})
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
		if it.Include && it.ChangesConfig {
			out["configChanged"]++
		}
		if it.Include && it.User.Username != it.Source.Username {
			out["renamed"]++
		}
		if it.Include && len(it.Applied) > 0 {
			out["changed"]++
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
	safe, err := a.Settings.SafeMode(r.Context())
	if err != nil {
		respond.Internal(w, r, err)
		return
	}
	id := r.PathValue("id")
	err = a.Runs.Run(r.Context(), id, run.Options{Version: b.PlanVersion, Reload: b.Reload, SafeMode: safe,
		AdminPassword: secret.NewValue(b.AdminPassword)})
	if errors.Is(err, run.ErrStaleVer) {
		respond.Error(w, respond.APIError{Status: http.StatusPreconditionFailed, Code: codeStalePlan})
		return
	}
	if err != nil {
		writeRunError(w, r, err)
		return
	}
	a.record(r, audit.ActionMigrationRun, audit.Success, map[string]string{"job": id, "reload": b.Reload, "safeMode": strconv.FormatBool(safe)})
	respond.JSON(w, http.StatusAccepted, map[string]any{"id": id, "state": run.StateRunning})
}

// CancelMigration stops a running job between batches.
func (a *API) CancelMigration(w http.ResponseWriter, r *http.Request) {
	if err := a.Runs.Cancel(r.PathValue("id")); err != nil {
		writeRunError(w, r, err)
		return
	}
	a.record(r, audit.ActionMigrationStop, audit.Success, map[string]string{"job": r.PathValue("id")})
	respond.NoContent(w)
}

// RollbackMigration undoes a failed or interrupted job (D39).
func (a *API) RollbackMigration(w http.ResponseWriter, r *http.Request) {
	rep, err := a.Runs.Rollback(r.Context(), r.PathValue("id"))
	if errors.Is(err, run.ErrNoRollback) {
		respond.Error(w, respond.APIError{Status: http.StatusConflict, Code: codeBusy})
		return
	}
	outcome := audit.Success
	if err != nil || rep.State != run.StateRolledBack {
		outcome = audit.Failure
	}
	a.record(r, audit.ActionMigrationUndo, outcome, map[string]string{"job": r.PathValue("id")})
	if err != nil {
		writeRunError(w, r, err)
		return
	}
	respond.JSON(w, http.StatusOK, rep)
}

// Templates lists saved templates.
func (a *API) Templates(w http.ResponseWriter, r *http.Request) {
	ts, err := a.Runs.Templates(r.Context())
	if err != nil {
		respond.Internal(w, r, err)
		return
	}
	respond.JSON(w, http.StatusOK, map[string]any{"templates": orEmpty(ts)})
}

// SaveTemplate keeps the reusable part of a job's plan.
func (a *API) SaveTemplate(w http.ResponseWriter, r *http.Request) {
	var b struct {
		JobID string `json:"jobId"`
		Name  string `json:"name"`
	}
	if err := respond.Decode(w, r, &b); err != nil || b.JobID == "" {
		respond.BadRequest(w)
		return
	}
	t, err := a.Runs.SaveTemplate(r.Context(), b.JobID, b.Name)
	if err != nil {
		writeRunError(w, r, err)
		return
	}
	respond.JSON(w, http.StatusCreated, t)
}

// DeleteTemplate removes a template.
func (a *API) DeleteTemplate(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		respond.BadRequest(w)
		return
	}
	if err := a.Runs.DeleteTemplate(r.Context(), id); err != nil {
		writeRunError(w, r, err)
		return
	}
	respond.NoContent(w)
}
