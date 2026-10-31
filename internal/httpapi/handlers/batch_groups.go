package handlers

import (
	"net/http"
	"strconv"

	"github.com/PdYrust/karvan/internal/httpapi/respond"
	"github.com/PdYrust/karvan/internal/migration/batch"
)

// groupRuns reads ?runs= (0 to batch.MaxRuns, default batch.MaxRuns).
func groupRuns(r *http.Request) (int, bool) {
	v := r.URL.Query().Get("runs")
	if v == "" {
		return batch.MaxRuns, true
	}
	n, err := strconv.Atoi(v)
	return n, err == nil && n >= 0 && n <= batch.MaxRuns
}

// BatchGroups lists the old panels moved onto each destination panel,
// the latest run first, a page at a time: one group per old panel with
// its totals, rules and newest runs.
// ?panel=<id>&state=active|inactive&runs=0..5&cursor=&limit=.
func (a *API) BatchGroups(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	limit, ok := batchLimit(r)
	runs, okRuns := groupRuns(r)
	panel, okPanel := queryID(r, "panel")
	if !ok || !okRuns || !okPanel {
		respond.BadRequest(w)
		return
	}
	list, next, err := a.Runs.Batches().Groups(r.Context(), batch.Filter{PanelID: panel, State: q.Get("state")}, q.Get("cursor"), limit, runs)
	if err != nil {
		writeBatchError(w, r, err)
		return
	}
	out := map[string]any{"groups": list}
	if next != "" {
		out["next"] = next
	}
	respond.Lists(w, http.StatusOK, out)
}

// BatchGroup reads the group of any of its batches: ?runs=0..5.
func (a *API) BatchGroup(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r, "id")
	runs, okRuns := groupRuns(r)
	if !ok || !okRuns {
		respond.BadRequest(w)
		return
	}
	g, err := a.Runs.Batches().GetGroup(r.Context(), id, runs)
	if err != nil {
		writeBatchError(w, r, err)
		return
	}
	respond.Lists(w, http.StatusOK, g)
}
