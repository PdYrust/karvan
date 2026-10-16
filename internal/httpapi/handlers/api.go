// Package handlers implements the JSON API and the web panel's HTML and
// asset routes.
package handlers

import (
	"log/slog"
	"net/http"

	"github.com/PdYrust/karvan/internal/audit"
	"github.com/PdYrust/karvan/internal/auth"
	"github.com/PdYrust/karvan/internal/bridge"
	"github.com/PdYrust/karvan/internal/httpapi/middleware"
	"github.com/PdYrust/karvan/internal/migration/run"
	"github.com/PdYrust/karvan/internal/settings"
)

// Error codes for sign-in and account endpoints.
const (
	codeInvalidCredentials = "invalid_credentials" //nolint:gosec // G101: an error code, not a credential.
	codeTOTPRequired       = "totp_required"
	codeTOTPInvalid        = "totp_invalid"
	codeTOTPNotStarted     = "totp_not_started"
	codeTOTPAlreadyOn      = "totp_already_on"
	codeTOTPNotOn          = "totp_not_on"
	codeLocked             = "locked"
	codeBusy               = "busy"
)

// API holds what the JSON endpoints need.
type API struct {
	Auth    *auth.Service
	Audit   *audit.Log
	Cookies middleware.Cookies
	Version string

	Runs      *run.Service
	Bridge    *bridge.Bridge
	Settings  *settings.Store
	ScanRoots []string
}

// record audits an operator action. A failed audit write is logged, not
// shown: the action itself already happened.
func (a *API) record(r *http.Request, action string, outcome audit.Outcome, detail map[string]string) {
	actor := ""
	if s, ok := middleware.Session(r.Context()); ok {
		if op, err := a.Auth.Operators.ByID(r.Context(), s.Session.OperatorID); err == nil {
			actor = op.Username
		}
	}
	err := a.Audit.Record(r.Context(), audit.Event{Actor: actor, IP: middleware.IP(r.Context()), Action: action, Outcome: outcome, Detail: detail})
	if err != nil {
		slog.WarnContext(r.Context(), "audit", "action", action, "err", err)
	}
}
