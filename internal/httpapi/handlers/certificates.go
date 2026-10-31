package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/PdYrust/karvan/internal/audit"
	"github.com/PdYrust/karvan/internal/bridge"
	"github.com/PdYrust/karvan/internal/httpapi/respond"
)

// maxCertBody caps a certificate or inspect body.
const maxCertBody = 256 << 10

// writeCertError answers a domain or certificate failure; the rest are
// site codes.
func writeCertError(w http.ResponseWriter, r *http.Request, err error) {
	var domain *bridge.DomainInvalidError
	var acme *bridge.ACMEInvalidError
	var invalid *bridge.CertInvalidError
	var inUse *bridge.InUseError
	switch {
	case errors.As(err, &domain):
		respond.Error(w, respond.APIError{Status: http.StatusBadRequest, Code: "domain_invalid",
			Extra: map[string]any{"field": domain.Field}})
	case errors.As(err, &acme):
		extra := map[string]any{"field": acme.Field}
		withReason(extra, acme.Reason, nil)
		respond.Error(w, respond.APIError{Status: http.StatusBadRequest, Code: "acme_invalid", Extra: extra})
	case errors.As(err, &invalid):
		extra := map[string]any{"field": invalid.Field}
		withReason(extra, invalid.Reason, nil)
		respond.Error(w, respond.APIError{Status: http.StatusBadRequest, Code: "certificate_invalid", Extra: extra})
	case errors.As(err, &inUse):
		extra := map[string]any{"rules": inUse.Rules}
		if inUse.What == "certificate" {
			extra["domains"] = inUse.Domains
		}
		respond.Error(w, respond.APIError{Status: http.StatusConflict, Code: inUse.What + "_in_use", Extra: extra})
	case errors.Is(err, bridge.ErrManaged):
		respond.Error(w, respond.APIError{Status: http.StatusConflict, Code: "certificate_managed"})
	default:
		writeSiteError(w, r, err)
	}
}

// Certificates lists the stored certificates with their facts and the
// rules that use them. No private key leaves the server.
func (a *API) Certificates(w http.ResponseWriter, r *http.Request) {
	list, err := a.Bridge.Certificates(r.Context())
	if err != nil {
		respond.Internal(w, r, err)
		return
	}
	respond.Lists(w, http.StatusOK, map[string]any{"certificates": list})
}

// SaveCertificate adds (POST) or changes (PATCH /{id}) a certificate: a
// file pair or a pasted one. A change sends the whole certificate; an
// empty key keeps the stored one.
func (a *API) SaveCertificate(w http.ResponseWriter, r *http.Request) {
	var in bridge.CertificateInput
	if err := respond.DecodeLimit(w, r, &in, maxCertBody); err != nil {
		respond.BadRequest(w)
		return
	}
	var id int64
	action, status := audit.ActionCertAdd, http.StatusCreated
	if v := r.PathValue("id"); v != "" {
		var err error
		if id, err = strconv.ParseInt(v, 10, 64); err != nil || id < 1 {
			respond.BadRequest(w)
			return
		}
		action, status = audit.ActionCertSave, http.StatusOK
	}
	c, err := a.Bridge.SaveCertificate(r.Context(), id, in)
	a.record(r, action, outcomeOf(err), map[string]string{"certificate": in.Name})
	if err != nil {
		writeCertError(w, r, err)
		return
	}
	respond.Lists(w, status, c)
}

// DeleteCertificate removes a certificate no rule uses.
func (a *API) DeleteCertificate(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		respond.BadRequest(w)
		return
	}
	err = a.Bridge.DeleteCertificate(r.Context(), id)
	a.record(r, audit.ActionCertDelete, outcomeOf(err), map[string]string{"certificate": r.PathValue("id")})
	if err != nil {
		writeCertError(w, r, err)
		return
	}
	respond.NoContent(w)
}

// ScanCertificates looks for certificates in the known places on this
// server, read only. The body is empty; the places are fixed.
func (a *API) ScanCertificates(w http.ResponseWriter, r *http.Request) {
	res, err := a.Bridge.ScanCertificates(r.Context())
	switch {
	case errors.Is(err, bridge.ErrScanBusy):
		respond.Error(w, respond.APIError{Status: http.StatusConflict, Code: codeBusy})
		return
	case err != nil:
		respond.Internal(w, r, err)
		return
	}
	a.record(r, audit.ActionCertScan, audit.Success, map[string]string{"found": strconv.Itoa(len(res.Found)),
		"skipped": strconv.Itoa(len(res.Skipped))})
	respond.Lists(w, http.StatusOK, res)
}

// InspectCert says what a certificate covers and what is wrong with it,
// before a rule is saved: files, a paste, or a stored one by id.
func (a *API) InspectCert(w http.ResponseWriter, r *http.Request) {
	var in bridge.CertInput
	if err := respond.DecodeLimit(w, r, &in, maxCertBody); err != nil {
		respond.BadRequest(w)
		return
	}
	facts, err := a.Bridge.InspectCert(r.Context(), in)
	if err != nil {
		writeSiteError(w, r, err)
		return
	}
	respond.Lists(w, http.StatusOK, facts)
}
