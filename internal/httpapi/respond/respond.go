// Package respond writes the API's JSON responses and decodes request bodies.
// Every error has the shape {"error":{"code":"...", ...}}; the web panel turns
// codes into human messages, so the server never sends prose.
package respond

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
)

// Error codes shared by middleware and handlers.
const (
	CodeBadRequest   = "bad_request"
	CodeUnauthorized = "unauthorized"
	CodeCSRF         = "csrf"
	CodeInternal     = "internal"
)

// MaxBodyBytes caps request bodies; every request Karvan accepts is small.
const MaxBodyBytes = 16 << 10

// APIError is an error body. Extra holds fields such as retryAfter.
type APIError struct {
	Status int
	Code   string
	Extra  map[string]any
}

// JSON writes v with status.
func JSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if v == nil {
		return
	}
	if err := json.NewEncoder(w).Encode(v); err != nil {
		slog.Debug("write response", "err", err)
	}
}

// NoContent writes 204.
func NoContent(w http.ResponseWriter) { w.WriteHeader(http.StatusNoContent) }

// Error writes e.
func Error(w http.ResponseWriter, e APIError) {
	body := map[string]any{"code": e.Code}
	for k, v := range e.Extra {
		body[k] = v
	}
	JSON(w, e.Status, map[string]any{"error": body})
}

// Internal logs err and writes a generic 500 that reveals nothing.
func Internal(w http.ResponseWriter, r *http.Request, err error) {
	slog.ErrorContext(r.Context(), "request failed", "method", r.Method, "err", err)
	Error(w, APIError{Status: http.StatusInternalServerError, Code: CodeInternal})
}

// Decode reads a JSON body into dst. It refuses unknown fields, trailing
// data, a wrong content type and bodies over MaxBodyBytes.
func Decode(w http.ResponseWriter, r *http.Request, dst any) error {
	if ct := r.Header.Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
		return errors.New("content type must be application/json")
	}
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, MaxBodyBytes))
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return fmt.Errorf("decode body: %w", err)
	}
	if _, err := dec.Token(); !errors.Is(err, io.EOF) {
		return errors.New("decode body: unexpected data after the JSON object")
	}
	return nil
}

// BadRequest writes a 400 for a body that failed Decode or validation.
func BadRequest(w http.ResponseWriter) {
	Error(w, APIError{Status: http.StatusBadRequest, Code: CodeBadRequest})
}
