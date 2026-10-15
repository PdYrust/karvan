// Package middleware holds the HTTP layers every request passes through.
package middleware

import (
	"net/http"
	"strconv"
)

var notFoundBody = []byte("Not Found\n")

// NotFound writes the one 404 Karvan ever sends. Every unknown path, wrong
// method and missing asset gets these exact bytes, so a scanner learns
// nothing, including that this is Karvan or that it is written in Go.
// Headers set by earlier layers are dropped for the same reason.
func NotFound(w http.ResponseWriter, _ *http.Request) {
	h := w.Header()
	clear(h)
	h.Set("Content-Type", "text/plain; charset=utf-8")
	h.Set("Content-Length", strconv.Itoa(len(notFoundBody)))
	h.Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(http.StatusNotFound)
	_, _ = w.Write(notFoundBody)
}

// Router wraps a ServeMux so that requests without a matching pattern,
// including a known path with the wrong method, get NotFound instead of the
// mux's own 404 or 405.
type Router struct {
	*http.ServeMux
}

// NewRouter returns an empty Router.
func NewRouter() *Router { return &Router{ServeMux: http.NewServeMux()} }

func (rt *Router) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !rt.matches(r) {
		NotFound(w, r)
		return
	}
	rt.ServeMux.ServeHTTP(w, r)
}

func (rt *Router) matches(r *http.Request) bool {
	_, pattern := rt.Handler(r)
	return pattern != ""
}

// Guard answers NotFound for requests the router has no route for, before
// any layer in between (sessions, CSRF) gets a chance to answer differently.
func (rt *Router) Guard(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !rt.matches(r) {
			NotFound(w, r)
			return
		}
		next.ServeHTTP(w, r)
	})
}
