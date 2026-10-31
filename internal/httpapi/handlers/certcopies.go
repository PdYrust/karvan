package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/PdYrust/karvan/internal/audit"
	"github.com/PdYrust/karvan/internal/bridge"
	"github.com/PdYrust/karvan/internal/httpapi/respond"
)

// maxCopyBody caps a copy target body.
const maxCopyBody = 1 << 10

// CopyCertificate copies a certificate's fullchain.pem and privkey.pem
// to Let's Encrypt live or a panel's certificate folder, and keeps it
// copied there after each change. It never touches a panel's ENV.
func (a *API) CopyCertificate(w http.ResponseWriter, r *http.Request) {
	id, ok := domainID(w, r)
	if !ok {
		return
	}
	var t bridge.CopyTarget
	if err := respond.DecodeLimit(w, r, &t, maxCopyBody); err != nil {
		respond.BadRequest(w)
		return
	}
	c, err := a.Bridge.CopyCertificate(r.Context(), id, t)
	detail := map[string]string{"certificate": r.PathValue("id"), "target": t.Target}
	if t.Target == bridge.CopyPanel {
		detail["panel"] = strconv.FormatInt(t.PanelID, 10)
	}
	if c.Dir != "" {
		detail["dir"] = c.Dir
	}
	var refused *bridge.CopyError
	if errors.As(err, &refused) {
		detail["reason"], detail["path"] = refused.Reason, refused.Path
	}
	a.record(r, audit.ActionCertCopy, outcomeOf(err), detail)
	switch {
	case errors.Is(err, bridge.ErrCopyTarget):
		respond.BadRequest(w)
	case refused != nil:
		respond.Error(w, respond.APIError{Status: http.StatusConflict, Code: "copy_refused",
			Extra: map[string]any{"reason": refused.Reason, "path": refused.Path}})
	case err != nil:
		writeSiteError(w, r, err)
	default:
		respond.Lists(w, http.StatusOK, c)
	}
}

// StopCertificateCopy forgets one copy of a certificate. The files stay.
func (a *API) StopCertificateCopy(w http.ResponseWriter, r *http.Request) {
	id, ok := domainID(w, r)
	if !ok {
		return
	}
	copyID, err := strconv.ParseInt(r.PathValue("copyId"), 10, 64)
	if err != nil || copyID < 1 {
		respond.BadRequest(w)
		return
	}
	c, err := a.Bridge.StopCopy(r.Context(), id, copyID)
	a.record(r, audit.ActionCertCopyStop, outcomeOf(err), map[string]string{"certificate": r.PathValue("id"), "dir": c.Dir})
	if err != nil {
		writeSiteError(w, r, err)
		return
	}
	respond.NoContent(w)
}
