package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/PdYrust/karvan/internal/audit"
	"github.com/PdYrust/karvan/internal/bridge"
	"github.com/PdYrust/karvan/internal/certorder"
	"github.com/PdYrust/karvan/internal/httpapi/respond"
	"github.com/PdYrust/karvan/internal/settings"
)

// maxACMEBody caps the small Let's Encrypt request bodies.
const maxACMEBody = 4 << 10

// writeACMEError answers a Let's Encrypt failure (section 4), else as a
// domain or certificate save.
func writeACMEError(w http.ResponseWriter, r *http.Request, err error) {
	var notFound *certorder.DNSNotFoundError
	code := ""
	switch {
	case errors.Is(err, certorder.ErrBusy):
		code = "acme_busy"
	case errors.Is(err, certorder.ErrState):
		code = "acme_state"
	case errors.Is(err, bridge.ErrPortDenied):
		code = certorder.CodePortDenied
	case errors.Is(err, bridge.ErrPortBusy):
		code = certorder.CodePortBusy
	case errors.Is(err, bridge.ErrPortTLS):
		code = certorder.CodePortTLS
	case errors.As(err, &notFound):
		respond.Error(w, respond.APIError{Status: http.StatusConflict, Code: "acme_dns_not_found",
			Extra: map[string]any{"seen": notFound.Seen}})
		return
	default:
		writeCertError(w, r, err)
		return
	}
	respond.Error(w, respond.APIError{Status: http.StatusConflict, Code: code})
}

// domainID reads the {id} of a domain route.
func domainID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id < 1 {
		respond.BadRequest(w)
		return 0, false
	}
	return id, true
}

// ACME is a domain's latest Let's Encrypt order and its renewal; the page
// polls it every 2 seconds while an order runs (D69).
func (a *API) ACME(w http.ResponseWriter, r *http.Request) {
	id, ok := domainID(w, r)
	if !ok {
		return
	}
	st, err := a.Bridge.ACME(r.Context(), id)
	if err != nil {
		writeACMEError(w, r, err)
		return
	}
	respond.Lists(w, http.StatusOK, st)
}

// StartACME starts an order for a domain now, also to renew ahead of time.
func (a *API) StartACME(w http.ResponseWriter, r *http.Request) {
	id, ok := domainID(w, r)
	if !ok {
		return
	}
	var b struct {
		Method   string `json:"method"`
		AgreeTOS bool   `json:"agreeTos"`
	}
	if err := respond.DecodeLimit(w, r, &b, maxACMEBody); err != nil {
		respond.BadRequest(w)
		return
	}
	o, err := a.Orders.Start(r.Context(), id, b.Method, b.AgreeTOS)
	a.record(r, audit.ActionACMEOrder, outcomeOf(err), map[string]string{"domain": r.PathValue("id"), "method": b.Method})
	if err != nil {
		writeACMEError(w, r, err)
		return
	}
	respond.Lists(w, http.StatusAccepted, o)
}

// CheckACME finishes a DNS-01 order once its TXT record is there; force
// asks Let's Encrypt without looking the record up first (D68).
func (a *API) CheckACME(w http.ResponseWriter, r *http.Request) {
	id, ok := domainID(w, r)
	if !ok {
		return
	}
	var b struct {
		Force bool `json:"force"`
	}
	if err := respond.DecodeLimit(w, r, &b, maxACMEBody); err != nil {
		respond.BadRequest(w)
		return
	}
	o, err := a.Orders.Check(r.Context(), id, b.Force)
	a.record(r, audit.ActionACMECheck, outcomeOf(err), map[string]string{"domain": r.PathValue("id"), "force": strconv.FormatBool(b.Force)})
	if err != nil {
		writeACMEError(w, r, err)
		return
	}
	respond.Lists(w, http.StatusAccepted, o)
}

// CancelACME cancels a domain's running order.
func (a *API) CancelACME(w http.ResponseWriter, r *http.Request) {
	id, ok := domainID(w, r)
	if !ok {
		return
	}
	err := a.Orders.Cancel(r.Context(), id)
	a.record(r, audit.ActionACMECancel, outcomeOf(err), map[string]string{"domain": r.PathValue("id")})
	if err != nil {
		writeACMEError(w, r, err)
		return
	}
	respond.NoContent(w)
}

// ACMESettings reads the Let's Encrypt settings (D73).
func (a *API) ACMESettings(w http.ResponseWriter, r *http.Request) {
	v, err := a.Orders.GetSettings(r.Context())
	if err != nil {
		respond.Internal(w, r, err)
		return
	}
	respond.Lists(w, http.StatusOK, v)
}

// PutACMESettings stores the Let's Encrypt settings; turning keepPort80
// on opens the challenge port first.
func (a *API) PutACMESettings(w http.ResponseWriter, r *http.Request) {
	var b settings.ACME
	if err := respond.DecodeLimit(w, r, &b, maxACMEBody); err != nil {
		respond.BadRequest(w)
		return
	}
	err := a.Orders.SetSettings(r.Context(), b)
	a.record(r, audit.ActionACMESettings, outcomeOf(err), map[string]string{"staging": strconv.FormatBool(b.Staging),
		"keepPort80": strconv.FormatBool(b.KeepPort80)})
	if err != nil {
		writeACMEError(w, r, err)
		return
	}
	a.ACMESettings(w, r)
}
