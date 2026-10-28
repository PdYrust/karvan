package handlers

import (
	"net/http"
	"strconv"

	"github.com/PdYrust/karvan/internal/httpapi/respond"
	"github.com/PdYrust/karvan/internal/migration/engine"
)

// Changes is what the last read of the panels changed in the outcome
// (B3): users new, gone or changed, a page at a time, with the admins and
// the job-level conflicts. Nothing changed since the operator reviewed it
// when changed is false; otherwise planVersion went up, and the next run
// must send it.
func (a *API) Changes(w http.ResponseWriter, r *http.Request) {
	c, version, at, err := a.Runs.Changes(r.PathValue("id"))
	if err != nil {
		writeRunError(w, r, err)
		return
	}
	cursor, limit := pageOf(r.URL.Query())
	counts := map[string]int{engine.UserNew: 0, engine.UserGone: 0, engine.UserChanged: 0}
	for _, u := range c.Users {
		counts[u.Change]++
	}
	users := c.Users[min(cursor, len(c.Users)):min(cursor+limit, len(c.Users))]
	page := map[string]any{"planVersion": version, "checkedAt": at, "changed": c.Changed(), "admin": c.Admin,
		"admins": orEmpty(c.Admins), "conflicts": c.Conflicts, "counts": counts, "users": orEmpty(users), "total": len(c.Users)}
	if next := cursor + len(users); next < len(c.Users) {
		page["next"] = strconv.Itoa(next)
	}
	respond.JSON(w, http.StatusOK, page)
}
