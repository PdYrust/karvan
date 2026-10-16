package middleware

import (
	"net/http"
	"time"
)

// Cookie names are short and generic so they do not name the product.
// Over HTTPS the __Host- prefix makes the browser refuse any cookie of that
// name that is not Secure, host-only and Path=/.
const (
	sessionCookie = "s"
	preCSRFCookie = "c"
	hostPrefix    = "__Host-"
)

// Cookies writes and reads Karvan's two cookies.
type Cookies struct {
	// Path is "/<base>/", used over plain HTTP to keep cookies off other paths.
	Path string
}

func (c Cookies) name(r *http.Request, base string) string {
	if IsHTTPS(r) {
		return hostPrefix + base
	}
	return base
}

func (c Cookies) cookie(r *http.Request, base, value string, maxAge int) *http.Cookie {
	ck := &http.Cookie{ //nolint:gosec // G124: Secure is set whenever the request is HTTPS; plain HTTP mode cannot use it.
		Name:     c.name(r, base),
		Value:    value,
		Path:     c.Path,
		MaxAge:   maxAge,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Secure:   IsHTTPS(r),
	}
	if ck.Secure {
		ck.Path = "/"
	}
	if maxAge < 0 {
		ck.Expires = time.Unix(0, 0)
	}
	return ck
}

func (c Cookies) read(r *http.Request, base string) string {
	ck, err := r.Cookie(c.name(r, base))
	if err != nil {
		return ""
	}
	return ck.Value
}

// SetSession stores the session token in a browser-session cookie; the
// server enforces the real timeouts.
func (c Cookies) SetSession(w http.ResponseWriter, r *http.Request, token string) {
	http.SetCookie(w, c.cookie(r, sessionCookie, token, 0))
}

// ClearSession removes the session cookie.
func (c Cookies) ClearSession(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, c.cookie(r, sessionCookie, "", -1))
}

// SessionToken returns the session cookie value, or "".
func (c Cookies) SessionToken(r *http.Request) string { return c.read(r, sessionCookie) }

// SetPreCSRF stores the token that guards the sign-in form.
func (c Cookies) SetPreCSRF(w http.ResponseWriter, r *http.Request, token string) {
	http.SetCookie(w, c.cookie(r, preCSRFCookie, token, 0))
}

// ClearPreCSRF removes the sign-in form token once a session exists.
func (c Cookies) ClearPreCSRF(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, c.cookie(r, preCSRFCookie, "", -1))
}

// PreCSRF returns the sign-in form token, or "".
func (c Cookies) PreCSRF(r *http.Request) string { return c.read(r, preCSRFCookie) }
