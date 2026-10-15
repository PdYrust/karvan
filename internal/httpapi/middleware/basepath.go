package middleware

import (
	"crypto/subtle"
	"net/http"
	"path"
	"strings"
)

// BasePath serves next only under "/<base>/", with the prefix removed.
// Anything else is NotFound, including unclean or oddly encoded paths that
// the standard mux would otherwise answer with a redirect.
func BasePath(base string, next http.Handler) http.Handler {
	prefix := "/" + base
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := r.URL.Path
		if r.URL.RawPath != "" || !hasPrefix(p, prefix) {
			NotFound(w, r)
			return
		}
		rest := p[len(prefix):]
		if rest == "" {
			if r.Method != http.MethodGet && r.Method != http.MethodHead {
				NotFound(w, r)
				return
			}
			target := prefix + "/"
			if r.URL.RawQuery != "" {
				target += "?" + r.URL.RawQuery
			}
			http.Redirect(w, r, target, http.StatusPermanentRedirect) //nolint:gosec // G710: the target path is our own fixed prefix; only the query is copied.
			return
		}
		if rest[0] != '/' || !isClean(rest) {
			NotFound(w, r)
			return
		}
		r2 := new(http.Request)
		*r2 = *r
		u := *r.URL
		u.Path = rest
		r2.URL = &u
		next.ServeHTTP(w, r2)
	})
}

// hasPrefix compares in constant time over the prefix, so response timing
// does not reveal how much of a guessed path was right.
func hasPrefix(p, prefix string) bool {
	if len(p) < len(prefix) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(p[:len(prefix)]), []byte(prefix)) == 1
}

func isClean(p string) bool {
	c := path.Clean(p)
	return c == p || (strings.HasSuffix(p, "/") && c+"/" == p)
}
