package handlers

import (
	"errors"
	"log/slog"
	"net/http"
	"slices"
	"strconv"
	"time"

	"github.com/PdYrust/karvan/internal/audit"
	"github.com/PdYrust/karvan/internal/bridge"
	"github.com/PdYrust/karvan/internal/httpapi/address"
	"github.com/PdYrust/karvan/internal/httpapi/respond"
	"github.com/PdYrust/karvan/internal/settings"
)

// Error codes of the HTTPS change (D10).
const (
	codeHTTPSInvalid = "https_invalid"
	codePortBusy     = "port_busy"
	codePortMixed    = "port_mixed"
)

type pendingHTTPSJSON struct {
	TLS       string    `json:"tls"`
	DomainID  int64     `json:"domainId"`
	Shared    bool      `json:"shared"`
	Port      int       `json:"port"`
	ExpiresAt time.Time `json:"expiresAt"`
	FromNew   bool      `json:"fromNew"`
}

type httpsJSON struct {
	TLS      string `json:"tls"`
	DomainID int64  `json:"domainId"`
	Shared   bool   `json:"shared"`
	Port     int    `json:"port"`
	// Fingerprint is the self-signed certificate's, when the current or
	// the waiting address uses it.
	Fingerprint string            `json:"fingerprint,omitempty"`
	Pending     *pendingHTTPSJSON `json:"pending"`
}

func modeOf(t settings.TLS) string {
	if t.Mode == "" {
		return settings.TLSOff
	}
	return t.Mode
}

// GetHTTPS returns how the panel is served and the change waiting, if any.
func (a *API) GetHTTPS(w http.ResponseWriter, r *http.Request) {
	st := a.Address.Status()
	c := st.Current
	out := httpsJSON{TLS: modeOf(c.TLS), DomainID: c.TLS.DomainID, Shared: c.TLS.Shared, Port: c.Port}
	self := c.TLS.Mode == settings.TLSSelf
	if p := st.Pending; p != nil {
		out.Pending = &pendingHTTPSJSON{TLS: modeOf(p.TLS), DomainID: p.TLS.DomainID, Shared: p.TLS.Shared, Port: p.Port,
			ExpiresAt: p.Expires.UTC(), FromNew: address.From(r.Context()) == p.Site}
		self = self || p.TLS.Mode == settings.TLSSelf
	}
	if self {
		out.Fingerprint = a.fingerprint(r)
	}
	respond.JSON(w, http.StatusOK, out)
}

func (a *API) fingerprint(r *http.Request) string {
	if a.PanelTLS == nil {
		return ""
	}
	fp, err := a.PanelTLS.Fingerprint(r.Context())
	if err != nil {
		slog.WarnContext(r.Context(), "self-signed fingerprint", "err", err)
	}
	return fp
}

type putHTTPSBody struct {
	reauthBody
	TLS      string `json:"tls"`
	DomainID int64  `json:"domainId"`
	// Port is the new port; 0 keeps it.
	Port   int  `json:"port"`
	Shared bool `json:"shared"`
}

type newHTTPSJSON struct {
	TLS         string    `json:"tls"`
	Port        int       `json:"port"`
	ExpiresAt   time.Time `json:"expiresAt"`
	Fingerprint string    `json:"fingerprint,omitempty"`
}

// PutHTTPS serves the panel in a new mode or on a new port next to the
// current address until it is confirmed from there (ADR 0009, ADR 0010).
// It is confirmed like an address change.
func (a *API) PutHTTPS(w http.ResponseWriter, r *http.Request) {
	var body putHTTPSBody
	if err := respond.Decode(w, r, &body); err != nil || !body.valid() || body.Port < 0 || body.Port > 65535 {
		respond.BadRequest(w)
		return
	}
	t, port, bad, err := a.checkHTTPS(r, body)
	switch {
	case err != nil:
		respond.Internal(w, r, err)
		return
	case bad != nil:
		respond.Error(w, *bad)
		return
	}
	if _, err := a.Auth.Reauthenticate(r.Context(), body.reauth(r), audit.ActionHTTPS); err != nil {
		writeAuthError(w, r, err, http.StatusForbidden)
		return
	}
	p, err := a.Address.Start(address.Change{Port: body.Port, TLS: &t})
	switch {
	case errors.Is(err, address.ErrNoChange):
		respond.BadRequest(w)
		return
	case errors.Is(err, address.ErrPortUnavailable):
		a.record(r, audit.ActionHTTPS, audit.Failure, map[string]string{"reason": codePortBusy, "port": strconv.Itoa(port)})
		respond.Error(w, respond.APIError{Status: http.StatusConflict, Code: codePortBusy})
		return
	case err != nil:
		respond.Internal(w, r, err)
		return
	}
	a.record(r, audit.ActionHTTPS, audit.Success, map[string]string{"tls": modeOf(t), "domain": strconv.FormatInt(t.DomainID, 10),
		"port": strconv.Itoa(p.Port), "shared": strconv.FormatBool(t.Shared)})
	out := newHTTPSJSON{TLS: modeOf(t), Port: p.Port, ExpiresAt: p.Expires.UTC()}
	if t.Mode == settings.TLSSelf {
		out.Fingerprint = a.fingerprint(r)
	}
	w.Header().Set("Cache-Control", "no-store")
	respond.JSON(w, http.StatusOK, out)
}

func httpsInvalid(field, reason string) *respond.APIError {
	return &respond.APIError{Status: http.StatusBadRequest, Code: codeHTTPSInvalid, Extra: map[string]any{"field": field, "reason": reason}}
}

var portMixed = &respond.APIError{Status: http.StatusConflict, Code: codePortMixed}

// checkHTTPS applies D10 to body: the mode, the domain and its
// certificate, and a shared port. It returns the new mode and the port it
// is served on.
func (a *API) checkHTTPS(r *http.Request, body putHTTPSBody) (settings.TLS, int, *respond.APIError, error) {
	t := settings.TLS{Mode: body.TLS, DomainID: body.DomainID, Shared: body.Shared}
	port := body.Port
	if port == 0 {
		port = a.Address.Status().Current.Port
	}
	switch body.TLS {
	case settings.TLSOff, settings.TLSSelf:
		if body.Shared {
			return t, port, portMixed, nil
		}
		return t.Normal(), port, nil, nil
	case settings.TLSDomain:
	default:
		return t, port, httpsInvalid("tls", "unknown"), nil
	}
	if body.DomainID < 1 || a.Bridge == nil {
		return t, port, httpsInvalid("domainId", "unknown"), nil
	}
	d, err := a.Bridge.Domain(r.Context(), body.DomainID)
	switch {
	case errors.Is(err, bridge.ErrNotFound):
		return t, port, httpsInvalid("domainId", "unknown"), nil
	case err != nil:
		return t, port, nil, err
	case d.Certificate == nil || slices.Contains(d.Certificate.Facts.Warnings, bridge.WarnDomain):
		return t, port, httpsInvalid("domainId", "no_certificate"), nil
	}
	if body.Shared {
		ok, err := a.Bridge.ServesHTTPS(r.Context(), port, d.Name)
		if err != nil || !ok {
			return t, port, portMixed, err
		}
	}
	return t, port, nil, nil
}
