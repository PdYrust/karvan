package handlers

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/PdYrust/karvan/internal/audit"
	"github.com/PdYrust/karvan/internal/httpapi/address"
	"github.com/PdYrust/karvan/internal/httpapi/respond"
)

// Error codes of the address change.
const (
	codePortUnavailable = "port_unavailable"
	codeNoAddressChange = "no_address_change"
	codeNotNewAddress   = "not_new_address"
)

type pendingAddressJSON struct {
	Port        int       `json:"port"`
	PathChanged bool      `json:"pathChanged"`
	ExpiresAt   time.Time `json:"expiresAt"`
	// FromNew is true when this request came through the new address,
	// where the change can be confirmed.
	FromNew bool `json:"fromNew"`
}

type addressJSON struct {
	Port    int                 `json:"port"`
	Pending *pendingAddressJSON `json:"pending"`
}

// GetAddress returns the panel's port and the change waiting, if any. The
// hidden paths are not in it: the new one was shown once, when it was made.
func (a *API) GetAddress(w http.ResponseWriter, r *http.Request) {
	st := a.Address.Status()
	out := addressJSON{Port: st.Current.Port}
	if p := st.Pending; p != nil {
		out.Pending = &pendingAddressJSON{Port: p.Port, PathChanged: p.BasePath != st.Current.BasePath,
			ExpiresAt: p.Expires.UTC(), FromNew: address.From(r.Context()) == p.Site}
	}
	respond.JSON(w, http.StatusOK, out)
}

type startAddressBody struct {
	reauthBody
	NewPath bool `json:"newPath"`
	// Port is the new port: 0 keeps it, -1 picks a random one.
	Port int `json:"port"`
}

func (b startAddressBody) valid(current int) bool {
	port := b.Port == -1 || b.Port == 0 || (b.Port >= 1 && b.Port <= 65535)
	change := b.NewPath || (b.Port != 0 && b.Port != current)
	return b.reauthBody.valid() && port && change
}

type newAddressJSON struct {
	Port      int       `json:"port"`
	Path      string    `json:"path"`
	ExpiresAt time.Time `json:"expiresAt"`
}

// StartAddress serves a new hidden path or port next to the current
// address until it is confirmed from there, for address.TTL. The new path
// is only in this response, which is never stored, and is not audited.
func (a *API) StartAddress(w http.ResponseWriter, r *http.Request) {
	var body startAddressBody
	if err := respond.Decode(w, r, &body); err != nil || !body.valid(a.Address.Status().Current.Port) {
		respond.BadRequest(w)
		return
	}
	if _, err := a.Auth.Reauthenticate(r.Context(), body.reauth(r), audit.ActionAddressStart); err != nil {
		writeAuthError(w, r, err, http.StatusForbidden)
		return
	}
	p, err := a.Address.Start(address.Change{NewPath: body.NewPath, Port: body.Port})
	switch {
	case errors.Is(err, address.ErrNoChange):
		respond.BadRequest(w)
		return
	case errors.Is(err, address.ErrPortUnavailable):
		a.record(r, audit.ActionAddressStart, audit.Failure, map[string]string{"reason": codePortUnavailable, "port": strconv.Itoa(body.Port)})
		respond.Error(w, respond.APIError{Status: http.StatusConflict, Code: codePortUnavailable})
		return
	case err != nil:
		respond.Internal(w, r, err)
		return
	}
	a.record(r, audit.ActionAddressStart, audit.Success, addressDetail(p.Port, body.NewPath))
	w.Header().Set("Cache-Control", "no-store")
	respond.JSON(w, http.StatusOK, newAddressJSON{Port: p.Port, Path: p.BasePath, ExpiresAt: p.Expires.UTC()})
}

// ConfirmAddress saves the new address. It must come through the new
// address, which proves the operator can reach it; the old one then stops.
func (a *API) ConfirmAddress(w http.ResponseWriter, r *http.Request) {
	via := address.From(r.Context())
	old, err := a.Address.Confirm(r.Context(), via)
	switch {
	case errors.Is(err, address.ErrNoPending):
		respond.Error(w, respond.APIError{Status: http.StatusConflict, Code: codeNoAddressChange})
		return
	case errors.Is(err, address.ErrNotNew):
		a.record(r, audit.ActionAddressChange, audit.Failure, map[string]string{"reason": codeNotNewAddress})
		respond.Error(w, respond.APIError{Status: http.StatusConflict, Code: codeNotNewAddress})
		return
	case err != nil:
		respond.Internal(w, r, err)
		return
	}
	a.record(r, audit.ActionAddressChange, audit.Success, addressDetail(via.Port, via.BasePath != old.BasePath))
	respond.NoContent(w)
}

// addressDetail records the port, which is not secret, and whether the
// path changed, never the path.
func addressDetail(port int, newPath bool) map[string]string {
	path := "same"
	if newPath {
		path = "new"
	}
	return map[string]string{"port": strconv.Itoa(port), "path": path}
}
