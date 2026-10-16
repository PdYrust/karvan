// Package handlers implements the JSON API and the web panel's HTML and
// asset routes.
package handlers

import (
	"github.com/PdYrust/karvan/internal/audit"
	"github.com/PdYrust/karvan/internal/auth"
	"github.com/PdYrust/karvan/internal/httpapi/middleware"
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
}
