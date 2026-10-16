// Package source is the HTTP client Karvan uses to read an old panel's API:
// outbound address checks, TLS with optional certificate pinning, retries
// with backoff, size caps and error codes the UI can turn into words.
package source

import (
	"errors"
	"fmt"
)

// Error codes. The UI maps each to a human message; response bodies from
// the panel are never shown.
const (
	CodeUnreachable      = "unreachable"
	CodeTimeout          = "timeout"
	CodeForbiddenAddress = "forbidden_address"
	CodeTLS              = "tls_failed"
	CodeTLSUntrusted     = "tls_untrusted"
	CodeRedirect         = "redirect"
	CodeUnauthorized     = "unauthorized"
	CodeForbidden        = "forbidden"
	CodeNotFound         = "not_found"
	CodeRateLimited      = "rate_limited"
	CodeServerError      = "server_error"
	CodeBadResponse      = "bad_response"
	CodeTooLarge         = "too_large"
)

// Error is a failed call to the old panel.
type Error struct {
	Code   string
	Status int // HTTP status, 0 when no response came back
	Err    error
}

func (e *Error) Error() string {
	if e.Status != 0 {
		return fmt.Sprintf("source %s (HTTP %d)", e.Code, e.Status)
	}
	if e.Err != nil {
		return fmt.Sprintf("source %s: %v", e.Code, e.Err)
	}
	return "source " + e.Code
}

func (e *Error) Unwrap() error { return e.Err }

// CodeOf returns the code of err, or "" if it is not a source error.
func CodeOf(err error) string {
	var e *Error
	if errors.As(err, &e) {
		return e.Code
	}
	return ""
}

// BadResponse wraps a decode or validation failure of a panel response.
func BadResponse(format string, args ...any) error {
	return &Error{Code: CodeBadResponse, Err: fmt.Errorf(format, args...)}
}
