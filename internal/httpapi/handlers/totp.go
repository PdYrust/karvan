package handlers

import (
	"net/http"

	"github.com/PdYrust/karvan/internal/httpapi/middleware"
	"github.com/PdYrust/karvan/internal/httpapi/respond"
	"github.com/PdYrust/karvan/internal/secret"
)

type totpSetupJSON struct {
	Secret string `json:"secret"`
	URI    string `json:"uri"`
}

// BeginTOTP starts two-step setup and returns the secret to scan.
func (a *API) BeginTOTP(w http.ResponseWriter, r *http.Request) {
	s, _ := middleware.Session(r.Context())
	setup, err := a.Auth.BeginTOTP(r.Context(), s.Session.OperatorID)
	if err != nil {
		respond.Internal(w, r, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	respond.JSON(w, http.StatusOK, totpSetupJSON{Secret: setup.Secret, URI: setup.URI})
}

type codeBody struct {
	Code string `json:"code"`
}

// ConfirmTOTP turns two-step sign-in on.
func (a *API) ConfirmTOTP(w http.ResponseWriter, r *http.Request) {
	var body codeBody
	if err := respond.Decode(w, r, &body); err != nil || len(body.Code) != 6 {
		respond.BadRequest(w)
		return
	}
	s, _ := middleware.Session(r.Context())
	if err := a.Auth.ConfirmTOTP(r.Context(), s.Session.OperatorID, body.Code, middleware.IP(r.Context())); err != nil {
		writeAuthError(w, r, err, http.StatusForbidden)
		return
	}
	respond.NoContent(w)
}

type disableTOTPBody struct {
	Password string `json:"password"`
	Code     string `json:"code"`
}

// DisableTOTP turns two-step sign-in off after checking password and code.
func (a *API) DisableTOTP(w http.ResponseWriter, r *http.Request) {
	var body disableTOTPBody
	if err := respond.Decode(w, r, &body); err != nil || body.Password == "" || len(body.Password) > 1024 || len(body.Code) != 6 {
		respond.BadRequest(w)
		return
	}
	s, _ := middleware.Session(r.Context())
	err := a.Auth.DisableTOTP(r.Context(), s.Session.OperatorID, secret.NewValue(body.Password), body.Code, middleware.IP(r.Context()))
	if err != nil {
		writeAuthError(w, r, err, http.StatusForbidden)
		return
	}
	respond.NoContent(w)
}
