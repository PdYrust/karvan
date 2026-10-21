package handlers

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/PdYrust/karvan/internal/httpapi/respond"
	"github.com/PdYrust/karvan/internal/migration/run"
)

// defaultPage is how many jobs one page holds when the request does not say.
const defaultPage = 20

// readJobFilter reads ?panel=<id>&state=a,b&limit=<n>.
func readJobFilter(r *http.Request) (run.JobFilter, int, bool) {
	q := r.URL.Query()
	var f run.JobFilter
	if p := q.Get("panel"); p != "" {
		id, err := strconv.ParseInt(p, 10, 64)
		if err != nil || id <= 0 {
			return f, 0, false
		}
		f.PanelID = id
	}
	if s := q.Get("state"); s != "" {
		f.States = strings.Split(s, ",")
	}
	limit := defaultPage
	if l := q.Get("limit"); l != "" {
		n, err := strconv.Atoi(l)
		if err != nil || n < 1 || n > run.MaxPage {
			return f, 0, false
		}
		limit = n
	}
	return f, limit, true
}

// ListMigrations lists jobs newest first, a page at a time; next, when
// present, is the cursor of the following page.
func (a *API) ListMigrations(w http.ResponseWriter, r *http.Request) {
	f, limit, ok := readJobFilter(r)
	if !ok {
		respond.BadRequest(w)
		return
	}
	rows, next, err := a.Runs.Jobs(r.Context(), f, r.URL.Query().Get("cursor"), limit)
	switch {
	case errors.Is(err, run.ErrBadCursor), errors.Is(err, run.ErrBadState):
		respond.BadRequest(w)
		return
	case err != nil:
		respond.Internal(w, r, err)
		return
	}
	out := map[string]any{"jobs": rows}
	if next != "" {
		out["next"] = next
	}
	respond.JSON(w, http.StatusOK, out)
}
