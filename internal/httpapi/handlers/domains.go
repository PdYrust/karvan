package handlers

import (
	"errors"
	"net/http"

	"github.com/PdYrust/karvan/internal/audit"
	"github.com/PdYrust/karvan/internal/bridge"
	"github.com/PdYrust/karvan/internal/httpapi/respond"
	"github.com/PdYrust/karvan/internal/settings"
)

// Domains lists the domains with the certificates that cover them, the
// rules that hold them and their warnings.
func (a *API) Domains(w http.ResponseWriter, r *http.Request) {
	list, err := a.Bridge.Domains(r.Context())
	if err != nil {
		respond.Internal(w, r, err)
		return
	}
	respond.Lists(w, http.StatusOK, map[string]any{"domains": list})
}

// AddDomain stores a domain name (a host, one leading wildcard or an IP)
// with its certificate, chosen one of the CertChoice ways.
func (a *API) AddDomain(w http.ResponseWriter, r *http.Request) {
	var b struct {
		Name        string            `json:"name"`
		Certificate bridge.CertChoice `json:"certificate"`
	}
	if err := respond.DecodeLimit(w, r, &b, maxCertBody); err != nil {
		respond.BadRequest(w)
		return
	}
	var d bridge.Domain
	var err error
	if ch := b.Certificate; ch.Kind == bridge.ChoiceACME {
		d, err = a.Orders.AddDomain(r.Context(), b.Name, ch.Method, ch.AgreeTOS) // plain until the order is done
	} else {
		d, err = a.Bridge.AddDomain(r.Context(), b.Name, ch)
	}
	a.record(r, audit.ActionDomainAdd, outcomeOf(err), map[string]string{"domain": b.Name, "way": b.Certificate.Kind})
	if err != nil {
		writeACMEError(w, r, err)
		return
	}
	respond.Lists(w, http.StatusCreated, d)
}

// SetDomainCertificate changes a domain's certificate: its pins go
// and the HTTPS rules that hold it serve the new one. The acme way starts
// an order and keeps the current certificate until it is done; any other
// way cancels a running order.
func (a *API) SetDomainCertificate(w http.ResponseWriter, r *http.Request) {
	id, ok := domainID(w, r)
	if !ok {
		return
	}
	var b struct {
		Certificate bridge.CertChoice `json:"certificate"`
	}
	if err := respond.DecodeLimit(w, r, &b, maxCertBody); err != nil {
		respond.BadRequest(w)
		return
	}
	ch, status := b.Certificate, http.StatusOK
	var d bridge.Domain
	var err error
	if ch.Kind == bridge.ChoiceACME {
		if _, err = a.Orders.Start(r.Context(), id, ch.Method, ch.AgreeTOS); err == nil {
			d, err = a.Bridge.Domain(r.Context(), id)
		}
		status = http.StatusAccepted
	} else if d, err = a.Bridge.SetDomainCertificate(r.Context(), id, ch); err == nil {
		a.Orders.Forget(id)
	}
	a.record(r, audit.ActionDomainCert, outcomeOf(err), map[string]string{"domain": r.PathValue("id"), "way": ch.Kind})
	if err != nil {
		writeACMEError(w, r, err)
		return
	}
	respond.Lists(w, status, d)
}

// errDomainKarvan means the domain to delete is the one enabled for Karvan.
var errDomainKarvan = errors.New("the domain is enabled for Karvan")

// DeleteDomain removes a domain no rule holds and Karvan does not use; its
// orders go with it.
func (a *API) DeleteDomain(w http.ResponseWriter, r *http.Request) {
	id, ok := domainID(w, r)
	if !ok {
		return
	}
	del := func() error {
		err := a.Bridge.DeleteDomain(r.Context(), id)
		if err == nil {
			a.Orders.Forget(id)
		}
		return err
	}
	var err error
	if a.PanelTLS == nil {
		err = del()
	} else {
		err = a.PanelTLS.Change(func(cur settings.TLS) (settings.TLS, error) {
			if cur.DomainID == id {
				return cur, errDomainKarvan
			}
			return cur, del()
		})
	}
	a.record(r, audit.ActionDomainDelete, outcomeOf(err), map[string]string{"domain": r.PathValue("id")})
	switch {
	case errors.Is(err, errDomainKarvan):
		respond.Error(w, respond.APIError{Status: http.StatusConflict, Code: codeDomainKarvan})
	case err != nil:
		writeCertError(w, r, err)
	default:
		respond.NoContent(w)
	}
}
