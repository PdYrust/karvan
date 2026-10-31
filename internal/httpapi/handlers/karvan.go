package handlers

import (
	"errors"
	"log/slog"
	"net"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/PdYrust/karvan/internal/audit"
	"github.com/PdYrust/karvan/internal/bridge"
	"github.com/PdYrust/karvan/internal/httpapi/address"
	"github.com/PdYrust/karvan/internal/httpapi/respond"
	"github.com/PdYrust/karvan/internal/paneltls"
	"github.com/PdYrust/karvan/internal/settings"
)

// Error codes of the Karvan domain.
const (
	codeKarvanInvalid = "karvan_invalid"
	codeKarvanShared  = "karvan_shared"
	codeDomainKarvan  = "domain_karvan"
)

// karvanJSON is the domain enabled for Karvan (0 for none), the plain
// HTTP choice for it while it has no certificate, and whether it is the
// only way in.
type karvanJSON struct {
	DomainID int64 `json:"domainId"`
	Plain    bool  `json:"plain"`
	// Shared is a panel on a bridge HTTPS port (ADR 0011): its domain
	// stays until the panel has a port of its own.
	Shared bool `json:"shared"`
	// Only is "Only open Karvan on its domain".
	Only bool `json:"only"`
	// Access is who the panel opens for now, and Reason why the IP opened
	// again in a fallback.
	Access paneltls.Access `json:"access"`
	Reason string          `json:"reason,omitempty"`
	// EndsSoon is the end of the domain's certificate when it is within
	// 14 days and nothing renews it by itself.
	EndsSoon *time.Time `json:"endsSoon,omitempty"`
}

func (a *API) karvanView(r *http.Request) karvanJSON {
	p, st := a.PanelTLS.Panel(), a.Address.Status()
	access, reason := a.PanelTLS.Access(r.Context())
	return karvanJSON{DomainID: p.DomainID, Plain: p.Plain, Shared: st.Current.Shared || (st.Pending != nil && st.Pending.Shared),
		Only: !p.OpenIP, Access: access, Reason: reason, EndsSoon: a.karvanEndsSoon(r, p.DomainID)}
}

// karvanEndsSoon is the end of the Karvan domain's certificate when it
// ends within 14 days and is not a Let's Encrypt one that Karvan renews.
func (a *API) karvanEndsSoon(r *http.Request, id int64) *time.Time {
	if id < 1 || a.Bridge == nil {
		return nil
	}
	d, err := a.Bridge.Domain(r.Context(), id)
	if err != nil || d.Certificate == nil || !slices.Contains(d.Certificate.Facts.Warnings, bridge.WarnExpiresSoon) {
		return nil
	}
	if d.Certificate.Origin == "acme" && d.ACME != nil && d.ACME.NextRenewal != nil {
		return nil
	}
	end := d.Certificate.Facts.NotAfter
	return &end
}

// KarvanDomain returns the domain enabled for Karvan, the plain HTTP
// choice and who the panel opens for.
func (a *API) KarvanDomain(w http.ResponseWriter, r *http.Request) {
	respond.JSON(w, http.StatusOK, a.karvanView(r))
}

type putKarvanBody struct {
	// DomainID is the domain to enable, 0 to stop; absent keeps it.
	DomainID *int64 `json:"domainId"`
	// Plain is the choice for the enabled domain without a certificate:
	// plain HTTP (true) or Karvan's own certificate; absent keeps it.
	Plain *bool `json:"plain"`
	// Only is "Only open Karvan on its domain"; absent keeps it.
	Only *bool `json:"only"`
}

// karvanError is a refused change.
type karvanError struct{ respond.APIError }

func (e *karvanError) Error() string { return e.Code }

func karvanInvalid(reason string) *karvanError {
	return &karvanError{respond.APIError{Status: http.StatusBadRequest, Code: codeKarvanInvalid,
		Extra: map[string]any{"field": "domainId", "reason": reason}}}
}

// PutKarvanDomain enables a domain for Karvan, enabling another switches
// to it and 0 stops, or sets the plain HTTP choice or "Only open Karvan on
// its domain". It is no address change: until the operator signs in
// through the domain, the server's IP keeps opening the panel with
// Karvan's own certificate, so the change applies to the next handshake
// and request without a restart or a confirm (ADR 0010). Each change is
// audited.
func (a *API) PutKarvanDomain(w http.ResponseWriter, r *http.Request) {
	var b putKarvanBody
	if err := respond.Decode(w, r, &b); err != nil || !b.valid() {
		respond.BadRequest(w)
		return
	}
	err := a.PanelTLS.Change(r.Context(), func(cur settings.TLS) (settings.TLS, error) { return a.changeKarvan(r, b, cur) })
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
	respond.JSON(w, http.StatusOK, a.karvanView(r))
}

// valid reports whether b asks for a change, with no negative domain.
func (b putKarvanBody) valid() bool {
	return (b.DomainID != nil || b.Plain != nil || b.Only != nil) && (b.DomainID == nil || *b.DomainID >= 0)
}

// changeKarvan checks, stores and audits the change b asks for from cur,
// and returns the new value. A new domain, or the switch turned, waits
// again for a sign-in through the domain.
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
	if b.Only != nil {
		next.OpenIP = !*b.Only
	}
	if next == cur {
		return cur, nil
	}
	if next.DomainID != cur.DomainID || next.OpenIP != cur.OpenIP {
		next.Armed = false
	}
	if err := a.Settings.SetKarvan(r.Context(), next); err != nil {
		return cur, err
	}
	if next.DomainID != cur.DomainID {
		a.record(r, audit.ActionHTTPS, audit.Success, map[string]string{"domain": strconv.FormatInt(next.DomainID, 10),
			"previous": strconv.FormatInt(cur.DomainID, 10)})
	}
	if next.Plain != cur.Plain {
		a.record(r, audit.ActionHTTPS, audit.Success, map[string]string{"plain": strconv.FormatBool(next.Plain)})
	}
	if next.OpenIP != cur.OpenIP {
		a.record(r, audit.ActionHTTPS, audit.Success, map[string]string{"only": strconv.FormatBool(!next.OpenIP)})
	}
	return next, nil
}

// checkKarvanDomain refuses a change while the panel is on a shared port,
// and a domain that is not registered or is an IP address: a browser
// sends no server name for an IP, so it always gets Karvan's own
// certificate.
func (a *API) checkKarvanDomain(r *http.Request, id int64) error {
	if a.karvanView(r).Shared {
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

// armKarvan makes the Karvan domain the only way in once the operator
// loads the panel through it, while "Only open Karvan on its domain" is
// on: the request's host and, over TLS, its server name must both be the
// domain's.
func (a *API) armKarvan(r *http.Request) {
	if a.PanelTLS == nil {
		return
	}
	names := []string{address.RequestHost(r)}
	if r.TLS != nil {
		names = append(names, strings.ToLower(r.TLS.ServerName))
	}
	armed, err := a.PanelTLS.Arm(r.Context(), names...)
	switch {
	case err != nil:
		slog.WarnContext(r.Context(), "arm the Karvan domain", "err", err)
	case armed:
		a.record(r, audit.ActionKarvanOnly, audit.Success,
			map[string]string{"domain": strconv.FormatInt(a.PanelTLS.Panel().DomainID, 10)})
	}
}
