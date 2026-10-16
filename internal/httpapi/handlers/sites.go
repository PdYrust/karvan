package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/PdYrust/karvan/internal/audit"
	"github.com/PdYrust/karvan/internal/bridge"
	"github.com/PdYrust/karvan/internal/httpapi/respond"
)

// maxTokenUpload caps a token list upload (D64).
const maxTokenUpload = 16 << 20

func writeSiteError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, bridge.ErrNotFound):
		respond.Error(w, respond.APIError{Status: http.StatusNotFound, Code: codeNotFound})
	case errors.Is(err, bridge.ErrPlainHTTP):
		respond.Error(w, respond.APIError{Status: http.StatusConflict, Code: "plain_http_off"})
	case errors.Is(err, bridge.ErrOverlap):
		respond.Error(w, respond.APIError{Status: http.StatusConflict, Code: "site_overlap"})
	case errors.Is(err, bridge.ErrInvalidSite):
		respond.Error(w, respond.APIError{Status: http.StatusBadRequest, Code: "site_invalid", Extra: map[string]any{"problem": err.Error()}})
	default:
		respond.Internal(w, r, err)
	}
}

// Sites lists the legacy sites, and ports that could not listen.
func (a *API) Sites(w http.ResponseWriter, r *http.Request) {
	ss, err := a.Bridge.Sites(r.Context())
	if err != nil {
		respond.Internal(w, r, err)
		return
	}
	respond.JSON(w, http.StatusOK, map[string]any{"sites": orEmpty(ss), "portErrors": a.Bridge.Status()})
}

// SaveSite adds (POST) or changes (PATCH /{id}) a legacy site. A change
// sends the whole site; an empty private key keeps the stored one.
func (a *API) SaveSite(w http.ResponseWriter, r *http.Request) {
	var s bridge.Site
	if err := respond.DecodeLimit(w, r, &s, 256<<10); err != nil {
		respond.BadRequest(w)
		return
	}
	s.ID = 0
	status := http.StatusCreated
	if v := r.PathValue("id"); v != "" {
		id, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			respond.BadRequest(w)
			return
		}
		s.ID, status = id, http.StatusOK
	}
	saved, err := a.Bridge.SaveSite(r.Context(), s)
	a.record(r, audit.ActionSiteSave, outcomeOf(err), map[string]string{"site": s.Name})
	if err != nil {
		writeSiteError(w, r, err)
		return
	}
	respond.JSON(w, status, saved)
}

// DeleteSite removes a legacy site and its verified tokens.
func (a *API) DeleteSite(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		respond.BadRequest(w)
		return
	}
	err = a.Bridge.DeleteSite(r.Context(), id)
	a.record(r, audit.ActionSiteDelete, outcomeOf(err), map[string]string{"site": r.PathValue("id")})
	if err != nil {
		writeSiteError(w, r, err)
		return
	}
	respond.NoContent(w)
}

// UploadTokens takes a list of old links (text or CSV) and stores the
// ones that check out, so their users never wait for a lazy check.
func (a *API) UploadTokens(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		respond.BadRequest(w)
		return
	}
	res, err := a.Bridge.Upload(r.Context(), id, http.MaxBytesReader(w, r.Body, maxTokenUpload))
	a.record(r, audit.ActionSiteTokens, outcomeOf(err), map[string]string{"site": r.PathValue("id"),
		"stored": strconv.Itoa(res.Stored), "found": strconv.Itoa(res.Found)})
	if err != nil {
		writeSiteError(w, r, err)
		return
	}
	respond.JSON(w, http.StatusOK, res)
}

func outcomeOf(err error) audit.Outcome {
	if err != nil {
		return audit.Failure
	}
	return audit.Success
}
