package handlers

import (
	"errors"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/PdYrust/karvan/internal/audit"
	"github.com/PdYrust/karvan/internal/httpapi/respond"
	"github.com/PdYrust/karvan/internal/migration/dest"
	"github.com/PdYrust/karvan/internal/migration/engine"
	"github.com/PdYrust/karvan/internal/migration/run"
	"github.com/PdYrust/karvan/internal/panel"
	"github.com/PdYrust/karvan/internal/panel/source"
	"github.com/PdYrust/karvan/internal/panelctl"
	"github.com/PdYrust/karvan/internal/secret"
)

// Error codes for migration endpoints.
const (
	codeNotFound      = "not_found"
	codeStalePlan     = "stale_plan"
	codeBlocked       = "blocked"
	codeNotChecked    = "not_checked"
	codeLoginGone     = "login_gone"
	codeWeakPassword  = "weak_password"
	codePlanInvalid   = "plan_invalid"
	codePanelNotReady = "panel_not_ready"
	codeNameTaken     = "name_taken"
	codeConfirm       = "confirm_required"
	codeNothingLeft   = "gone"
	codePanelControl  = "panel_control_unavailable"
	// A rollback on request: the job is not failed or interrupted, or
	// another job holds its destination.
	codeNotRollbackable = "not_rollbackable"
	codeDestLocked      = "destination_locked"
	// Only a draft that never ran can be deleted.
	codeNotDeletable = "not_deletable"
)

// controlReason says why Karvan cannot stop and start a panel, as a code.
func controlReason(err error) string {
	switch {
	case errors.Is(err, panelctl.ErrOff):
		return "turned_off"
	case errors.Is(err, panelctl.ErrNoCompose):
		return "no_compose_file"
	case errors.Is(err, panelctl.ErrNoServices):
		return "no_panel_service"
	}
	return "docker_denied" // no Docker, or no permission to use it
}

// writeRunError maps migration errors to API errors. Old panel errors
// keep their code ("source_unreachable", "source_unauthorized"...), so the
// panel can say what went wrong there.
func writeRunError(w http.ResponseWriter, r *http.Request, err error) {
	var planErr *engine.PlanError
	var srcErr *source.Error
	var weak *run.WeakPasswordError
	switch {
	case errors.Is(err, run.ErrNoJob), errors.Is(err, run.ErrNotFound):
		respond.Error(w, respond.APIError{Status: http.StatusNotFound, Code: codeNotFound})
	case errors.Is(err, run.ErrBusy):
		respond.Error(w, respond.APIError{Status: http.StatusConflict, Code: codeBusy})
	case errors.Is(err, run.ErrNotDeletable):
		respond.Error(w, respond.APIError{Status: http.StatusConflict, Code: codeNotDeletable})
	case errors.Is(err, run.ErrBlocked):
		respond.Error(w, respond.APIError{Status: http.StatusConflict, Code: codeBlocked})
	case errors.Is(err, run.ErrNotChecked):
		respond.Error(w, respond.APIError{Status: http.StatusConflict, Code: codeNotChecked})
	case errors.Is(err, run.ErrNoLogin):
		respond.Error(w, respond.APIError{Status: http.StatusConflict, Code: codeLoginGone})
	case errors.Is(err, run.ErrNoPanelControl):
		respond.Error(w, respond.APIError{Status: http.StatusConflict, Code: codePanelControl, Extra: map[string]any{"reason": controlReason(err)}})
	case errors.Is(err, run.ErrPanelChanged):
		respond.Error(w, respond.APIError{Status: http.StatusConflict, Code: "panel_changed"})
	case errors.Is(err, run.ErrBadReload):
		respond.BadRequest(w)
	case errors.As(err, &weak):
		respond.Error(w, respond.APIError{Status: http.StatusBadRequest, Code: codeWeakPassword, Extra: map[string]any{"rule": weak.Rule}})
	case errors.Is(err, dest.ErrNotReady):
		respond.Error(w, respond.APIError{Status: http.StatusConflict, Code: codePanelNotReady})
	case errors.As(err, &planErr):
		respond.Error(w, respond.APIError{Status: http.StatusBadRequest, Code: codePlanInvalid,
			Extra: map[string]any{"where": planErr.Where, "problem": planErr.Problem}})
	case errors.As(err, &srcErr):
		respond.Error(w, respond.APIError{Status: http.StatusBadGateway, Code: "source_" + srcErr.Code, Extra: srcErr.Extra()})
	default:
		respond.Internal(w, r, err)
	}
}

type createBody struct {
	SourceURL  string `json:"sourceUrl"`
	SourceKind string `json:"sourceKind"`
	Username   string `json:"username"`
	Password   string `json:"password"`
	APIKey     string `json:"apiKey"`
	PinSHA256  string `json:"pinSha256"`
	PanelID    int64  `json:"panelId"`
}

func (b createBody) valid() bool {
	kinds := []string{"", string(panel.Marzban), string(panel.PasarGuard)}
	login := b.Username != "" && b.Password != "" || b.APIKey != ""
	return b.SourceURL != "" && len(b.SourceURL) <= 2048 && slices.Contains(kinds, b.SourceKind) && login &&
		len(b.Username) <= 128 && len(b.Password) <= 1024 && len(b.APIKey) <= 1024 && len(b.PinSHA256) <= 128 && b.PanelID > 0
}

// CreateMigration starts a job: the old panel's login and the destination.
// The login stays in memory only (D11).
func (a *API) CreateMigration(w http.ResponseWriter, r *http.Request) {
	var b createBody
	if err := respond.Decode(w, r, &b); err != nil || !b.valid() {
		respond.BadRequest(w)
		return
	}
	j, err := a.Runs.Create(r.Context(), run.Login{URL: b.SourceURL, Kind: panel.Kind(b.SourceKind), Username: b.Username,
		Password: secret.NewValue(b.Password), APIKey: secret.NewValue(b.APIKey), PinSHA256: b.PinSHA256}, b.PanelID, source.Options{})
	if err != nil {
		writeRunError(w, r, err)
		return
	}
	respond.JSON(w, http.StatusCreated, map[string]any{"id": j.ID, "state": j.State()})
}

// GetMigration reports a job's state and, once checked, its summary.
func (a *API) GetMigration(w http.ResponseWriter, r *http.Request) {
	sum, err := a.Runs.Summary(r.PathValue("id"))
	if errors.Is(err, run.ErrNotChecked) || errors.Is(err, run.ErrNoJob) {
		state, err := a.Runs.JobState(r.Context(), r.PathValue("id"))
		if err != nil {
			writeRunError(w, r, err)
			return
		}
		respond.JSON(w, http.StatusOK, map[string]any{"id": r.PathValue("id"), "state": state})
		return
	}
	if err != nil {
		writeRunError(w, r, err)
		return
	}
	respond.JSON(w, http.StatusOK, map[string]any{"id": r.PathValue("id"), "state": sum.State, "summary": sum})
}

// DeleteMigration deletes a draft that never ran, with its events, and
// the old panel's login Karvan holds for it. A job that ran keeps its
// record.
func (a *API) DeleteMigration(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	err := a.Runs.DeleteDraft(r.Context(), id)
	if errors.Is(err, run.ErrNoJob) {
		writeRunError(w, r, err)
		return
	}
	outcome := audit.Success
	if err != nil {
		outcome = audit.Failure
	}
	a.record(r, audit.ActionMigrationDelete, outcome, map[string]string{"job": id})
	if err != nil {
		writeRunError(w, r, err)
		return
	}
	respond.NoContent(w)
}

// Compare reads both panels again in the background and rebuilds the
// plan (steps 1 and 2, B2): reading many users can take longer than a
// request may. Progress and the outcome come on the job's event stream.
func (a *API) Compare(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := a.Runs.Compare(id); err != nil {
		writeRunError(w, r, err)
		return
	}
	respond.JSON(w, http.StatusAccepted, map[string]any{"id": id})
}

// Summary is step 1: who the admin is, how many users, how much traffic.
func (a *API) Summary(w http.ResponseWriter, r *http.Request) {
	sum, err := a.Runs.Summary(r.PathValue("id"))
	if err != nil {
		writeRunError(w, r, err)
		return
	}
	respond.JSON(w, http.StatusOK, sum)
}

// Groups lists the problem groups (made by Karvan from the conflicts),
// the job-level conflicts, and how many users each category holds.
func (a *API) Groups(w http.ResponseWriter, r *http.Request) {
	res, _, version, err := a.Runs.View(r.PathValue("id"))
	if err == nil && res == nil {
		err = run.ErrNotChecked
	}
	if err != nil {
		writeRunError(w, r, err)
		return
	}
	categories := map[string]int{}
	for _, it := range res.Items {
		for _, n := range it.Nodes {
			categories[n]++
		}
	}
	groups := res.Groups
	if groups == nil {
		groups = []engine.Group{}
	}
	respond.JSON(w, http.StatusOK, map[string]any{"planVersion": version, "groups": groups, "conflicts": orEmpty(res.Conflicts),
		"categories": categories, "blockers": res.Blockers, "stable": res.Stable})
}

func orEmpty[T any](s []T) []T {
	if s == nil {
		return []T{}
	}
	return s
}

// userJSON is one user in step 2 (as read) or step 4 (as it will land).
type userJSON struct {
	Key            string         `json:"key"`
	SourceUsername string         `json:"sourceUsername"`
	Username       string         `json:"username"`
	Include        bool           `json:"include"`
	Status         string         `json:"status"`
	UsedTraffic    int64          `json:"usedTraffic"`
	DataLimit      int64          `json:"dataLimit"`
	Expire         *time.Time     `json:"expire"`
	Note           string         `json:"note,omitempty"`
	Categories     []string       `json:"categories"`
	Targets        []string       `json:"targets,omitempty"`
	Conflicts      []conflictJSON `json:"conflicts"`
	Applied        []string       `json:"applied"`
	ChangesConfig  bool           `json:"changesConfig"`
	ChangesLink    bool           `json:"changesLink"`
	MovedBefore    bool           `json:"movedBefore,omitempty"`
}

type conflictJSON struct {
	Type     string `json:"type"`
	Severity string `json:"severity"`
	Fix      string `json:"fix,omitempty"`
}

func toUserJSON(it *engine.Item, final bool) userJSON {
	u := it.Source
	if final {
		u = it.User
	}
	out := userJSON{Key: it.Key, SourceUsername: it.Source.Username, Username: it.User.Username, Include: it.Include,
		Status: string(u.Status), UsedTraffic: u.UsedTraffic, DataLimit: u.DataLimit, Expire: u.Expire, Note: u.Note, Categories: orEmpty(it.Nodes),
		Applied: orEmpty(it.Applied), ChangesConfig: it.ChangesConfig, ChangesLink: it.ChangesLink, Conflicts: []conflictJSON{},
		MovedBefore: it.Adopt != nil}
	if final {
		out.Targets = it.Targets
	}
	for _, c := range it.Conflicts {
		out.Conflicts = append(out.Conflicts, conflictJSON{Type: c.Type, Severity: c.Severity, Fix: c.Fix})
	}
	return out
}

// pageLimit caps one page of users; lists of 100k users are read page by
// page.
const pageLimit = 200

// pageOf reads a page's cursor (an offset; anything else is 0) and its
// limit (1 to pageLimit; anything else is pageLimit).
func pageOf(q url.Values) (int, int) {
	cursor, _ := strconv.Atoi(q.Get("cursor"))
	limit, err := strconv.Atoi(q.Get("limit"))
	if err != nil || limit <= 0 || limit > pageLimit {
		limit = pageLimit
	}
	return max(cursor, 0), limit
}

// Users is step 2's list of users as read: filtered by group or category
// and a name search, in pages.
func (a *API) Users(w http.ResponseWriter, r *http.Request) { a.userPage(w, r, false) }

// Preview is step 4: the same list, each user as it will land.
func (a *API) Preview(w http.ResponseWriter, r *http.Request) { a.userPage(w, r, true) }

func (a *API) userPage(w http.ResponseWriter, r *http.Request, final bool) {
	items, err := a.Runs.Items(r.PathValue("id"))
	if err != nil {
		writeRunError(w, r, err)
		return
	}
	q := r.URL.Query()
	cursor, limit := pageOf(q)
	match := itemFilter(q.Get("group"), strings.ToLower(q.Get("q")))
	out, total := []userJSON{}, 0
	for _, it := range items {
		if !match(it) {
			continue
		}
		if total >= cursor && len(out) < limit {
			out = append(out, toUserJSON(it, final))
		}
		total++
	}
	page := map[string]any{"users": out, "total": total}
	if next := cursor + len(out); next < total {
		page["next"] = strconv.Itoa(next)
	}
	respond.JSON(w, http.StatusOK, page)
}

// itemFilter matches a group ("ready", "skipped", "renamed",
// "configChanged", a conflict type such as "C02", or "category:<id>") and
// a name search. renamed and configChanged count moving users only, as
// the plan's impact does.
func itemFilter(group, search string) func(*engine.Item) bool {
	return func(it *engine.Item) bool {
		if search != "" && !strings.Contains(strings.ToLower(it.Source.Username), search) &&
			!strings.Contains(strings.ToLower(it.User.Username), search) {
			return false
		}
		switch {
		case group == "":
			return true
		case group == "ready":
			return it.Include && len(it.Conflicts) == 0
		case group == "skipped":
			return !it.Include
		case group == "renamed":
			return it.Include && it.User.Username != it.Source.Username
		case group == "configChanged":
			return it.Include && it.ChangesConfig
		case strings.HasPrefix(group, "category:"):
			return slices.Contains(it.Nodes, strings.TrimPrefix(group, "category:"))
		}
		// The group's members, fixed or open, as GET groups counts them.
		is := func(c engine.Conflict) bool { return c.Type == group }
		return slices.ContainsFunc(it.Conflicts, is) || slices.ContainsFunc(it.Resolved, is)
	}
}
