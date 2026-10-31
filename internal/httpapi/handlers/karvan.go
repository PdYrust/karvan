package handlers

import (
	"errors"
	"net"
	"net/http"
	"strconv"

	"github.com/PdYrust/karvan/internal/audit"
	"github.com/PdYrust/karvan/internal/bridge"
	"github.com/PdYrust/karvan/internal/httpapi/respond"
	"github.com/PdYrust/karvan/internal/settings"
)

// Error codes of the Karvan domain.
const (
	codeKarvanInvalid = "karvan_invalid"
	codeKarvanShared  = "karvan_shared"
	codeDomainKarvan  = "domain_karvan"
)

// karvanJSON is the domain enabled for Karvan (0 for none) and the plain
// HTTP choice for it while it has no certificate.
type karvanJSON struct {
	DomainID int64 `json:"domainId"`
	Plain    bool  `json:"plain"`
	// Shared is a panel on a bridge HTTPS port (ADR 0011): its domain
	// stays until the panel has a port of its own.
	Shared bool `json:"shared"`
}

func (a *API) karvanView() karvanJSON {
	p, st := a.PanelTLS.Panel(), a.Address.Status()
	return karvanJSON{DomainID: p.DomainID, Plain: p.Plain, Shared: st.Current.Shared || (st.Pending != nil && st.Pending.Shared)}
}

// KarvanDomain returns the domain enabled for Karvan and the plain HTTP
// choice.
func (a *API) KarvanDomain(w http.ResponseWriter, _ *http.Request) {
	respond.JSON(w, http.StatusOK, a.karvanView())
}

type putKarvanBody struct {
	// DomainID is the domain to enable, 0 to stop; absent keeps it.
	DomainID *int64 `json:"domainId"`
	// Plain is the choice for the enabled domain without a certificate:
	// plain HTTP (true) or Karvan's own certificate; absent keeps it.
	Plain *bool `json:"plain"`
}

// karvanError is a refused change.
type karvanError struct{ respond.APIError }

func (e *karvanError) Error() string { return e.Code }

func karvanInvalid(reason string) *karvanError {
	return &karvanError{respond.APIError{Status: http.StatusBadRequest, Code: codeKarvanInvalid,
		Extra: map[string]any{"field": "domainId", "reason": reason}}}
}

// PutKarvanDomain enables a domain for Karvan, enabling another switches
// to it and 0 stops, or sets the plain HTTP choice. It is no address
// change: a handshake for any other name, the server's IP included, keeps
// Karvan's own certificate, so the change applies to the next handshake
// and request without a restart or a confirm (ADR 0010). Each change is
// audited.
func (a *API) PutKarvanDomain(w http.ResponseWriter, r *http.Request) {
	var b putKarvanBody
	if err := respond.Decode(w, r, &b); err != nil || !b.valid() {
		respond.BadRequest(w)
		return
	}
	err := a.PanelTLS.Change(func(cur settings.TLS) (settings.TLS, error) { return a.changeKarvan(r, b, cur) })
	var refused *karvanError
	switch {
	case errors.As(err, &refused):
		a.record(r, audit.ActionHTTPS, audit.Failure, map[string]string{"reason": refused.Code})
		respond.Error(w, refused.APIError)
		return
	case err != nil:
		respond.Internal(w, r, err)
		return
	}
	respond.JSON(w, http.StatusOK, a.karvanView())
}

// valid reports whether b asks for a change, with no negative domain.
func (b putKarvanBody) valid() bool {
	return (b.DomainID != nil || b.Plain != nil) && (b.DomainID == nil || *b.DomainID >= 0)
}

// changeKarvan checks, stores and audits the change b asks for from cur,
// and returns the new value.
func (a *API) changeKarvan(r *http.Request, b putKarvanBody, cur settings.TLS) (settings.TLS, error) {
	next := cur
	if b.DomainID != nil && *b.DomainID != cur.DomainID {
		if err := a.checkKarvanDomain(r, *b.DomainID); err != nil {
			return cur, err
		}
		next.DomainID = *b.DomainID
	}
	if b.Plain != nil {
		next.Plain = *b.Plain
	}
	if next == cur {
		return cur, nil
	}
	if err := a.Settings.SetKarvan(r.Context(), next.DomainID, next.Plain); err != nil {
		return cur, err
	}
	if next.DomainID != cur.DomainID {
		a.record(r, audit.ActionHTTPS, audit.Success, map[string]string{"domain": strconv.FormatInt(next.DomainID, 10),
			"previous": strconv.FormatInt(cur.DomainID, 10)})
	}
	if next.Plain != cur.Plain {
		a.record(r, audit.ActionHTTPS, audit.Success, map[string]string{"plain": strconv.FormatBool(next.Plain)})
	}
	return next, nil
}

// checkKarvanDomain refuses a change while the panel is on a shared port,
// and a domain that is not registered or is an IP address: a browser
// sends no server name for an IP, so it always gets Karvan's own
// certificate.
func (a *API) checkKarvanDomain(r *http.Request, id int64) error {
	if a.karvanView().Shared {
		return &karvanError{respond.APIError{Status: http.StatusConflict, Code: codeKarvanShared}}
	}
	if id == 0 {
		return nil
	}
	if a.Bridge == nil {
		return karvanInvalid("unknown")
	}
	d, err := a.Bridge.Domain(r.Context(), id)
	switch {
	case errors.Is(err, bridge.ErrNotFound):
		return karvanInvalid("unknown")
	case err != nil:
		return err
	case net.ParseIP(d.Name) != nil:
		return karvanInvalid("ip")
	}
	return nil
}
