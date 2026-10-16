package handlers

import (
	"errors"
	"math"
	"net/http"
	"strconv"

	"github.com/PdYrust/karvan/internal/auth"
	"github.com/PdYrust/karvan/internal/httpapi/middleware"
	"github.com/PdYrust/karvan/internal/httpapi/respond"
	"github.com/PdYrust/karvan/internal/secret"
)

type operatorJSON struct {
	Username    string `json:"username"`
	TOTPEnabled bool   `json:"totpEnabled"`
}

type sessionJSON struct {
	Authenticated bool          `json:"authenticated"`
	Operator      *operatorJSON `json:"operator,omitempty"`
	CSRFToken     string        `json:"csrfToken"`
}

// GetSession reports whether the browser is signed in. Before sign-in it
// hands out the token that guards the sign-in form.
func (a *API) GetSession(w http.ResponseWriter, r *http.Request) {
	if s, ok := middleware.Session(r.Context()); ok {
		op, err := a.Auth.Operators.ByID(r.Context(), s.Session.OperatorID)
		if err != nil {
			respond.Internal(w, r, err)
			return
		}
		respond.JSON(w, http.StatusOK, sessionJSON{
			Authenticated: true,
			Operator:      &operatorJSON{Username: op.Username, TOTPEnabled: op.TOTPEnabled},
			CSRFToken:     auth.CSRFToken(s.Token),
		})
		return
	}
	token, err := secret.NewToken()
	if err != nil {
		respond.Internal(w, r, err)
		return
	}
	a.Cookies.SetPreCSRF(w, r, token)
	respond.JSON(w, http.StatusOK, sessionJSON{CSRFToken: token})
}

type loginBody struct {
	Username string `json:"username"`
	Password string `json:"password"`
	TOTPCode string `json:"totpCode"`
}

func (b loginBody) valid() bool {
	return b.Username != "" && len(b.Username) <= 64 && b.Password != "" && len(b.Password) <= 1024 && len(b.TOTPCode) <= 6
}

// CreateSession signs in.
func (a *API) CreateSession(w http.ResponseWriter, r *http.Request) {
	var body loginBody
	if err := respond.Decode(w, r, &body); err != nil || !body.valid() {
		respond.BadRequest(w)
		return
	}
	res, err := a.Auth.Login(r.Context(), auth.LoginRequest{
		Username:  body.Username,
		Password:  secret.NewValue(body.Password),
		TOTPCode:  body.TOTPCode,
		IP:        middleware.IP(r.Context()),
		UserAgent: r.UserAgent(),
	})
	if err != nil {
		writeAuthError(w, r, err, http.StatusUnauthorized)
		return
	}
	if old, ok := middleware.Session(r.Context()); ok {
		if err := a.Auth.Sessions.Delete(r.Context(), old.Token); err != nil {
			respond.Internal(w, r, err)
			return
		}
	}
	a.Cookies.SetSession(w, r, res.Token)
	a.Cookies.ClearPreCSRF(w, r)
	respond.JSON(w, http.StatusOK, sessionJSON{
		Authenticated: true,
		Operator:      &operatorJSON{Username: res.Operator.Username, TOTPEnabled: res.Operator.TOTPEnabled},
		CSRFToken:     auth.CSRFToken(res.Token),
	})
}

// DeleteSession signs out.
func (a *API) DeleteSession(w http.ResponseWriter, r *http.Request) {
	s, _ := middleware.Session(r.Context())
	op, err := a.Auth.Operators.ByID(r.Context(), s.Session.OperatorID)
	if err != nil {
		respond.Internal(w, r, err)
		return
	}
	if err := a.Auth.Logout(r.Context(), s.Token, op.Username, middleware.IP(r.Context())); err != nil {
		respond.Internal(w, r, err)
		return
	}
	a.Cookies.ClearSession(w, r)
	respond.NoContent(w)
}

// writeAuthError maps auth errors to API errors. badStatus is used for wrong
// credentials: 401 at sign-in, 403 when an already signed-in operator
// confirms an action, so the panel does not mistake it for an ended session.
func writeAuthError(w http.ResponseWriter, r *http.Request, err error, badStatus int) {
	var locked *auth.LockedError
	switch {
	case errors.As(err, &locked):
		secs := int(math.Ceil(locked.RetryAfter.Seconds()))
		w.Header().Set("Retry-After", strconv.Itoa(secs))
		respond.Error(w, respond.APIError{Status: http.StatusTooManyRequests, Code: codeLocked, Extra: map[string]any{"retryAfter": secs}})
	case errors.Is(err, auth.ErrBusy):
		w.Header().Set("Retry-After", "5")
		respond.Error(w, respond.APIError{Status: http.StatusServiceUnavailable, Code: codeBusy})
	case errors.Is(err, auth.ErrInvalidCredentials):
		respond.Error(w, respond.APIError{Status: badStatus, Code: codeInvalidCredentials})
	case errors.Is(err, auth.ErrTOTPRequired):
		respond.Error(w, respond.APIError{Status: badStatus, Code: codeTOTPRequired})
	case errors.Is(err, auth.ErrTOTPInvalid):
		respond.Error(w, respond.APIError{Status: badStatus, Code: codeTOTPInvalid})
	case errors.Is(err, auth.ErrNoPendingTOTP):
		respond.Error(w, respond.APIError{Status: http.StatusConflict, Code: codeTOTPNotStarted})
	case errors.Is(err, auth.ErrTOTPAlreadyActive):
		respond.Error(w, respond.APIError{Status: http.StatusConflict, Code: codeTOTPAlreadyOn})
	case errors.Is(err, auth.ErrTOTPNotEnabled):
		respond.Error(w, respond.APIError{Status: http.StatusConflict, Code: codeTOTPNotOn})
	default:
		respond.Internal(w, r, err)
	}
}
