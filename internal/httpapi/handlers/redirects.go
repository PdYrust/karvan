package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/PdYrust/karvan/internal/audit"
	"github.com/PdYrust/karvan/internal/bridge"
	"github.com/PdYrust/karvan/internal/httpapi/respond"
)

// maxRedirectBody caps a redirect rule body.
const maxRedirectBody = 64 << 10

// writeRedirectError answers a redirect failure; conflicts and not_found
// are the site codes.
func writeRedirectError(w http.ResponseWriter, r *http.Request, err error) {
	var invalid *bridge.RedirectInvalidError
	if !errors.As(err, &invalid) {
		writeSiteError(w, r, err)
		return
	}
	extra := map[string]any{"field": invalid.Field}
	if invalid.Reason != "" {
		extra["reason"] = invalid.Reason
	}
	respond.Error(w, respond.APIError{Status: http.StatusBadRequest, Code: "redirect_invalid", Extra: extra})
}

// Redirects lists the redirect rules with their certificate and warnings.
func (a *API) Redirects(w http.ResponseWriter, r *http.Request) {
	list, err := a.Bridge.Redirects(r.Context())
	if err != nil {
		respond.Internal(w, r, err)
		return
	}
	respond.JSON(w, http.StatusOK, map[string]any{"redirects": list})
}

// SaveRedirect adds (POST) or changes (PATCH /{id}) a redirect rule. A
// change sends the whole rule.
func (a *API) SaveRedirect(w http.ResponseWriter, r *http.Request) {
	var in bridge.Redirect
	if err := respond.DecodeLimit(w, r, &in, maxRedirectBody); err != nil {
		respond.BadRequest(w)
		return
	}
	in.ID = 0
	status := http.StatusCreated
	if v := r.PathValue("id"); v != "" {
		id, err := strconv.ParseInt(v, 10, 64)
		if err != nil || id < 1 {
			respond.BadRequest(w)
			return
		}
		in.ID, status = id, http.StatusOK
	}
	saved, err := a.Bridge.SaveRedirect(r.Context(), in)
	a.record(r, audit.ActionRedirectSave, outcomeOf(err), map[string]string{"redirect": in.Name})
	if err != nil {
		writeRedirectError(w, r, err)
		return
	}
	respond.JSON(w, status, saved)
}

// DeleteRedirect removes a redirect rule.
func (a *API) DeleteRedirect(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		respond.BadRequest(w)
		return
	}
	err = a.Bridge.DeleteRedirect(r.Context(), id)
	a.record(r, audit.ActionRedirectDelete, outcomeOf(err), map[string]string{"redirect": r.PathValue("id")})
	if err != nil {
		writeRedirectError(w, r, err)
		return
	}
	respond.NoContent(w)
}
