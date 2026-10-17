package middleware

import (
	"context"
	"errors"
	"net/http"

	"github.com/PdYrust/karvan/internal/auth"
	"github.com/PdYrust/karvan/internal/httpapi/respond"
)

// SessionInfo is the signed-in session of a request.
type SessionInfo struct {
	Token   string
	Session auth.Session
}

// BackgroundHeader marks a read the panel makes by itself, such as live
// charts polling. It does not count as the operator being active.
const BackgroundHeader = "X-Karvan-Background"

// LoadSession attaches the session named by the cookie, if it is live.
// A dead session's cookie is cleared.
func LoadSession(sessions *auth.Sessions, cookies Cookies, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := cookies.SessionToken(r)
		if token == "" {
			next.ServeHTTP(w, r)
			return
		}
		lookup := sessions.Lookup
		if r.Method == http.MethodGet && r.Header.Get(BackgroundHeader) == "1" {
			lookup = sessions.Peek
		}
		sess, err := lookup(r.Context(), token)
		if errors.Is(err, auth.ErrNoSession) {
			cookies.ClearSession(w, r)
			next.ServeHTTP(w, r)
			return
		}
		if err != nil {
			respond.Internal(w, r, err)
			return
		}
		ctx := context.WithValue(r.Context(), sessionKey, SessionInfo{Token: token, Session: sess})
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// Session returns the request's session, if any.
func Session(ctx context.Context) (SessionInfo, bool) {
	s, ok := ctx.Value(sessionKey).(SessionInfo)
	return s, ok
}

// RequireSession answers 401 unless the request is signed in.
func RequireSession(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, ok := Session(r.Context()); !ok {
			respond.Error(w, respond.APIError{Status: http.StatusUnauthorized, Code: respond.CodeUnauthorized})
			return
		}
		next.ServeHTTP(w, r)
	})
}
