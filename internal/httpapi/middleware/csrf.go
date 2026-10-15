package middleware

import (
	"crypto/subtle"
	"net/http"

	"github.com/PdYrust/karvan/internal/auth"
	"github.com/PdYrust/karvan/internal/httpapi/respond"
)

// CSRFHeader carries the anti-CSRF token on every state-changing request.
const CSRFHeader = "X-CSRF-Token"

// CSRF guards every request that is not GET, HEAD or OPTIONS with two
// independent checks: the browser must say the request came from this
// origin, and the request must carry the token for its session (or, before
// sign-in, the token from the pre-session cookie).
func CSRF(cookies Cookies, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet, http.MethodHead, http.MethodOptions:
			next.ServeHTTP(w, r)
			return
		}
		if !sameOrigin(r) || !validToken(r, cookies) {
			respond.Error(w, respond.APIError{Status: http.StatusForbidden, Code: respond.CodeCSRF})
			return
		}
		next.ServeHTTP(w, r)
	})
}

func sameOrigin(r *http.Request) bool {
	if site := r.Header.Get("Sec-Fetch-Site"); site != "" {
		return site == "same-origin"
	}
	origin := r.Header.Get("Origin")
	if origin == "" {
		return false
	}
	scheme := "http://"
	if r.TLS != nil {
		scheme = "https://"
	}
	return origin == scheme+r.Host
}

func validToken(r *http.Request, cookies Cookies) bool {
	got := r.Header.Get(CSRFHeader)
	if s, ok := Session(r.Context()); ok {
		return auth.ValidCSRF(s.Token, got)
	}
	want := cookies.PreCSRF(r)
	return want != "" && subtle.ConstantTimeCompare([]byte(want), []byte(got)) == 1
}
