package handlers

import (
	"errors"
	"io"
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
	case errors.Is(err, bridge.ErrNameTaken):
		respond.Error(w, respond.APIError{Status: http.StatusConflict, Code: codeNameTaken})
	case errors.Is(err, bridge.ErrPortMixed):
		respond.Error(w, respond.APIError{Status: http.StatusConflict, Code: "port_mixed"})
	case errors.Is(err, bridge.ErrCertConflict):
		respond.Error(w, respond.APIError{Status: http.StatusConflict, Code: "cert_conflict"})
	case errors.Is(err, bridge.ErrPortTaken):
		respond.Error(w, respond.APIError{Status: http.StatusConflict, Code: "port_taken"})
	case errors.Is(err, bridge.ErrUploadBusy):
		respond.Error(w, respond.APIError{Status: http.StatusConflict, Code: codeBusy})
	case errors.Is(err, bridge.ErrNoCert):
		respond.BadRequest(w)
	case errors.Is(err, bridge.ErrNotAdopted):
		respond.Error(w, respond.APIError{Status: http.StatusConflict, Code: "certificate_not_adopted"})
	case errors.Is(err, bridge.ErrInvalidSite):
		respond.Error(w, respond.APIError{Status: http.StatusBadRequest, Code: "site_invalid", Extra: siteInvalid(err)})
	default:
		respond.Internal(w, r, err)
	}
}

// Sites lists the legacy sites with their certificate facts and numbers,
// and ports that could not listen.
func (a *API) Sites(w http.ResponseWriter, r *http.Request) {
	vs, err := a.Bridge.Views(r.Context())
	if err != nil {
		respond.Internal(w, r, err)
		return
	}
	respond.JSON(w, http.StatusOK, map[string]any{"sites": vs, "portErrors": a.Bridge.Status()})
}

// siteInvalid is the field a site_invalid answer names, and its reason
// when there is one.
func siteInvalid(err error) map[string]any {
	extra := map[string]any{"field": bridge.FieldOf(err)}
	var e *bridge.InvalidError
	if errors.As(err, &e) && e.Reason != "" {
		extra["reason"] = e.Reason
	}
	return extra
}

// SaveSite adds (POST) or changes (PATCH /{id}) a panel rule. A change
// sends the whole rule. The certificate is certificateId; certificate
// files and PEM are not part of the body (they are certificates rows).
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
	respond.JSON(w, status, a.Bridge.View(r.Context(), saved))
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

// UploadTokens takes a list of old links (text or CSV) and checks it in
// the background; the ones that check out are stored, so their users
// never wait for a lazy check. UploadStatus follows the checks.
func (a *API) UploadTokens(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		respond.BadRequest(w)
		return
	}
	list, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxTokenUpload))
	var tooLarge *http.MaxBytesError
	switch {
	case errors.As(err, &tooLarge):
		respond.Error(w, respond.APIError{Status: http.StatusRequestEntityTooLarge, Code: "too_large"})
		return
	case err != nil:
		respond.BadRequest(w)
		return
	}
	p, err := a.Bridge.StartUpload(id, list)
	a.record(r, audit.ActionSiteTokens, outcomeOf(err), map[string]string{"site": r.PathValue("id"),
		"found": strconv.Itoa(p.Found)})
	if err != nil {
		writeSiteError(w, r, err)
		return
	}
	respond.JSON(w, http.StatusAccepted, p)
}

// UploadStatus is the progress of a site's latest token list.
func (a *API) UploadStatus(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		respond.BadRequest(w)
		return
	}
	p, err := a.Bridge.UploadStatus(id)
	if err != nil {
		writeSiteError(w, r, err)
		return
	}
	respond.JSON(w, http.StatusOK, p)
}

func outcomeOf(err error) audit.Outcome {
	if err != nil {
		return audit.Failure
	}
	return audit.Success
}
