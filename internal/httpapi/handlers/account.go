package handlers

import (
	"net/http"
	"regexp"
	"strconv"
	"time"

	"github.com/PdYrust/karvan/internal/audit"
	"github.com/PdYrust/karvan/internal/httpapi/middleware"
	"github.com/PdYrust/karvan/internal/httpapi/respond"
)

var sessionIDPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{16}$`)

type accountSessionJSON struct {
	ID         string    `json:"id"`
	IP         string    `json:"ip"`
	UserAgent  string    `json:"userAgent"`
	CreatedAt  time.Time `json:"createdAt"`
	LastSeenAt time.Time `json:"lastSeenAt"`
	Current    bool      `json:"current"`
}

// AccountSessions lists the operator's signed-in browsers, the most
// recently used first.
func (a *API) AccountSessions(w http.ResponseWriter, r *http.Request) {
	s, _ := middleware.Session(r.Context())
	list, err := a.Auth.Sessions.List(r.Context(), s.Session.OperatorID)
	if err != nil {
		respond.Internal(w, r, err)
		return
	}
	out := make([]accountSessionJSON, 0, len(list))
	for _, x := range list {
		out = append(out, accountSessionJSON{ID: x.ID, IP: x.IP, UserAgent: x.UserAgent,
			CreatedAt: x.CreatedAt.UTC(), LastSeenAt: x.LastSeenAt.UTC(), Current: x.ID == s.Session.ID})
	}
	respond.JSON(w, http.StatusOK, map[string]any{"sessions": out})
}

// EndOtherSessions signs out every browser but this one. Ending sessions
// only takes access away, so it does not ask for the password.
func (a *API) EndOtherSessions(w http.ResponseWriter, r *http.Request) {
	s, _ := middleware.Session(r.Context())
	n, err := a.Auth.Sessions.DeleteOthers(r.Context(), s.Session.OperatorID, s.Token)
	if err != nil {
		respond.Internal(w, r, err)
		return
	}
	a.record(r, audit.ActionSessionsEnd, audit.Success, map[string]string{"ended": strconv.FormatInt(n, 10), "which": "others"})
	respond.JSON(w, http.StatusOK, map[string]int64{"ended": n})
}

// EndSession signs out one browser. Ending this browser's own session
// signs it out too.
func (a *API) EndSession(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !sessionIDPattern.MatchString(id) {
		respond.BadRequest(w)
		return
	}
	s, _ := middleware.Session(r.Context())
	ok, err := a.Auth.Sessions.DeleteByID(r.Context(), s.Session.OperatorID, id)
	if err != nil {
		respond.Internal(w, r, err)
		return
	}
	if !ok {
		respond.Error(w, respond.APIError{Status: http.StatusNotFound, Code: codeNotFound})
		return
	}
	a.record(r, audit.ActionSessionsEnd, audit.Success, map[string]string{"ended": "1", "which": "one"})
	if id == s.Session.ID {
		a.Cookies.ClearSession(w, r)
	}
	respond.NoContent(w)
}
