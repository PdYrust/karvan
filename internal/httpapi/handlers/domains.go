package handlers

import (
	"net/http"
	"strconv"

	"github.com/PdYrust/karvan/internal/audit"
	"github.com/PdYrust/karvan/internal/bridge"
	"github.com/PdYrust/karvan/internal/httpapi/respond"
)

// Domains lists the domains with the certificates that cover them, the
// rules that hold them and their warnings (D48, D50).
func (a *API) Domains(w http.ResponseWriter, r *http.Request) {
	list, err := a.Bridge.Domains(r.Context())
	if err != nil {
		respond.Internal(w, r, err)
		return
	}
	respond.JSON(w, http.StatusOK, map[string]any{"domains": list})
}

// AddDomain stores a domain name (a host, one leading wildcard or an IP)
// with its certificate, chosen one of the D61 ways.
func (a *API) AddDomain(w http.ResponseWriter, r *http.Request) {
	var b struct {
		Name        string            `json:"name"`
		Certificate bridge.CertChoice `json:"certificate"`
	}
	if err := respond.DecodeLimit(w, r, &b, maxCertBody); err != nil {
		respond.BadRequest(w)
		return
	}
	d, err := a.Bridge.AddDomain(r.Context(), b.Name, b.Certificate)
	a.record(r, audit.ActionDomainAdd, outcomeOf(err), map[string]string{"domain": b.Name, "way": b.Certificate.Kind})
	if err != nil {
		writeCertError(w, r, err)
		return
	}
	respond.JSON(w, http.StatusCreated, d)
}

// SetDomainCertificate changes a domain's certificate (D65): its pins go
// and the HTTPS rules that hold it serve the new one.
func (a *API) SetDomainCertificate(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		respond.BadRequest(w)
		return
	}
	var b struct {
		Certificate bridge.CertChoice `json:"certificate"`
	}
	if err := respond.DecodeLimit(w, r, &b, maxCertBody); err != nil {
		respond.BadRequest(w)
		return
	}
	d, err := a.Bridge.SetDomainCertificate(r.Context(), id, b.Certificate)
	a.record(r, audit.ActionDomainCert, outcomeOf(err), map[string]string{"domain": r.PathValue("id"), "way": b.Certificate.Kind})
	if err != nil {
		writeCertError(w, r, err)
		return
	}
	respond.JSON(w, http.StatusOK, d)
}

// DeleteDomain removes a domain no rule holds.
func (a *API) DeleteDomain(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		respond.BadRequest(w)
		return
	}
	err = a.Bridge.DeleteDomain(r.Context(), id)
	a.record(r, audit.ActionDomainDelete, outcomeOf(err), map[string]string{"domain": r.PathValue("id")})
	if err != nil {
		writeCertError(w, r, err)
		return
	}
	respond.NoContent(w)
}
