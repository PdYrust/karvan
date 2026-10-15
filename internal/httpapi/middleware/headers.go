package middleware

import "net/http"

// ContentSecurityPolicy allows only Karvan's own files. No inline scripts,
// no third-party origins, no framing.
const ContentSecurityPolicy = "default-src 'none'; script-src 'self'; style-src 'self'; img-src 'self' data: blob:; " +
	"font-src 'self'; connect-src 'self'; manifest-src 'self'; worker-src 'self'; base-uri 'self'; " +
	"form-action 'self'; frame-ancestors 'none'"

// SecurityHeaders sets the browser protections on every response it wraps.
// Referrer-Policy no-referrer matters most here: it keeps the hidden path
// out of the Referer header of any link the operator follows.
func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("Content-Security-Policy", ContentSecurityPolicy)
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("Referrer-Policy", "no-referrer")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Cross-Origin-Opener-Policy", "same-origin")
		h.Set("Cross-Origin-Resource-Policy", "same-origin")
		h.Set("Permissions-Policy", "camera=(), microphone=(), geolocation=(), payment=(), usb=()")
		if r.TLS != nil {
			h.Set("Strict-Transport-Security", "max-age=31536000")
		}
		next.ServeHTTP(w, r)
	})
}
