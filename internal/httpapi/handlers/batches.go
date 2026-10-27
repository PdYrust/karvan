package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/PdYrust/karvan/internal/audit"
	"github.com/PdYrust/karvan/internal/httpapi/respond"
	"github.com/PdYrust/karvan/internal/migration/batch"
)

// defaultBatchPage is how many batches or batch users one page holds when
// the request does not say.
const defaultBatchPage = 50

// maxBatchQuery bounds ?q= on a batch's users.
const maxBatchQuery = 256

func writeBatchError(w http.ResponseWriter, r *http.Request, err error) {
	var inUse *batch.InUseError
	var confirm *batch.ConfirmError
	switch {
	case errors.Is(err, batch.ErrNotFound):
		respond.Error(w, respond.APIError{Status: http.StatusNotFound, Code: codeNotFound})
	case errors.Is(err, batch.ErrBadCursor), errors.Is(err, batch.ErrBadState):
		respond.BadRequest(w)
	case errors.Is(err, batch.ErrInvalidName):
		respond.Error(w, respond.APIError{Status: http.StatusBadRequest, Code: "batch_invalid", Extra: map[string]any{"field": "name"}})
	case errors.As(err, &confirm):
		respond.Error(w, respond.APIError{Status: http.StatusBadRequest, Code: codeConfirm, Extra: map[string]any{"confirm": confirm.Name}})
	case errors.Is(err, batch.ErrActive):
		respond.Error(w, respond.APIError{Status: http.StatusConflict, Code: "batch_active"})
	case errors.As(err, &inUse):
		respond.Error(w, respond.APIError{Status: http.StatusConflict, Code: "batch_in_use", Extra: map[string]any{"rules": inUse.Rules}})
	case errors.Is(err, batch.ErrBusy):
		respond.Error(w, respond.APIError{Status: http.StatusConflict, Code: codeBusy})
	case errors.Is(err, batch.ErrNotReactivatable):
		respond.Error(w, respond.APIError{Status: http.StatusConflict, Code: "not_reactivatable"})
	default:
		writeRunError(w, r, err)
	}
}

// pathID reads a positive integer path value.
func pathID(r *http.Request, name string) (int64, bool) {
	id, err := strconv.ParseInt(r.PathValue(name), 10, 64)
	return id, err == nil && id > 0
}

// batchLimit reads ?limit= (1 to batch.MaxPage, default defaultBatchPage).
func batchLimit(r *http.Request) (int, bool) {
	l := r.URL.Query().Get("limit")
	if l == "" {
		return defaultBatchPage, true
	}
	n, err := strconv.Atoi(l)
	return n, err == nil && n >= 1 && n <= batch.MaxPage
}

// Batches lists batches newest first, a page at a time, never deleted
// ones: ?panel=<id>&state=active|inactive&cursor=&limit=.
func (a *API) Batches(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	f := batch.Filter{State: q.Get("state")}
	limit, ok := batchLimit(r)
	if p := q.Get("panel"); p != "" {
		id, err := strconv.ParseInt(p, 10, 64)
		ok = ok && err == nil && id > 0
		f.PanelID = id
	}
	if !ok {
		respond.BadRequest(w)
		return
	}
	list, next, err := a.Runs.Batches().List(r.Context(), f, q.Get("cursor"), limit)
	if err != nil {
		writeBatchError(w, r, err)
		return
	}
	out := map[string]any{"batches": list}
	if next != "" {
		out["next"] = next
	}
	respond.JSON(w, http.StatusOK, out)
}

// Batch reads one batch; a deleted one is not found.
func (a *API) Batch(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r, "id")
	if !ok {
		respond.BadRequest(w)
		return
	}
	b, err := a.Runs.Batches().Get(r.Context(), id)
	if err != nil {
		writeBatchError(w, r, err)
		return
	}
	respond.JSON(w, http.StatusOK, b)
}

// BatchUsers lists a batch's users, a page at a time: ?q=&cursor=&limit=.
// No old token or MAC is in it (D57).
func (a *API) BatchUsers(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r, "id")
	limit, okLimit := batchLimit(r)
	q := r.URL.Query()
	if !ok || !okLimit || len(q.Get("q")) > maxBatchQuery {
		respond.BadRequest(w)
		return
	}
	users, next, err := a.Runs.Batches().Users(r.Context(), id, q.Get("q"), q.Get("cursor"), limit)
	if err != nil {
		writeBatchError(w, r, err)
		return
	}
	out := map[string]any{"users": users}
	if next != "" {
		out["next"] = next
	}
	respond.JSON(w, http.StatusOK, out)
}

// BatchUserLink builds a batch user's new subscription link on the
// destination panel (D57, Q7 B). The link is never stored, never cached
// and never written to the audit log.
func (a *API) BatchUserLink(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r, "id")
	user, okUser := pathID(r, "ledgerId")
	if !ok || !okUser {
		respond.BadRequest(w)
		return
	}
	m, err := a.Runs.Batches().Member(r.Context(), id, user)
	link := ""
	if err == nil {
		link, err = a.Runs.UserLink(r.Context(), m)
	}
	a.record(r, audit.ActionBatchShowLink, outcomeOf(err), map[string]string{"batch": r.PathValue("id"), "user": r.PathValue("ledgerId")})
	w.Header().Set("Cache-Control", "no-store")
	if err != nil {
		writeBatchError(w, r, err)
		return
	}
	respond.JSON(w, http.StatusOK, map[string]string{"url": link})
}

// RenameBatch gives a batch a new name: PATCH {name}.
func (a *API) RenameBatch(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r, "id")
	var body struct {
		Name string `json:"name"`
	}
	if !ok || respond.Decode(w, r, &body) != nil {
		respond.BadRequest(w)
		return
	}
	b, err := a.Runs.Batches().Rename(r.Context(), id, body.Name)
	a.record(r, audit.ActionBatchRename, outcomeOf(err), map[string]string{"batch": r.PathValue("id"), "name": b.Name})
	if err != nil {
		writeBatchError(w, r, err)
		return
	}
	respond.JSON(w, http.StatusOK, b)
}

// DeleteBatch marks an inactive batch deleted (D20) once {confirm} is its
// name.
func (a *API) DeleteBatch(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r, "id")
	var body struct {
		Confirm string `json:"confirm"`
	}
	if !ok || respond.Decode(w, r, &body) != nil {
		respond.BadRequest(w)
		return
	}
	err := a.Runs.Batches().Delete(r.Context(), id, body.Confirm)
	a.record(r, audit.ActionBatchDelete, outcomeOf(err), map[string]string{"batch": r.PathValue("id")})
	if err != nil {
		writeBatchError(w, r, err)
		return
	}
	respond.NoContent(w)
}

// ReactivateBatch makes a batch whose restore failed active again.
func (a *API) ReactivateBatch(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r, "id")
	if !ok {
		respond.BadRequest(w)
		return
	}
	b, err := a.Runs.Batches().Reactivate(r.Context(), id)
	a.record(r, audit.ActionBatchReactivate, outcomeOf(err), map[string]string{"batch": r.PathValue("id")})
	if err != nil {
		writeBatchError(w, r, err)
		return
	}
	respond.JSON(w, http.StatusOK, b)
}
