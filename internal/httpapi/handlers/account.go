package handlers

import (
	"net/http"
	"regexp"
	"strconv"
	"time"

	"github.com/PdYrust/karvan/internal/audit"
	"github.com/PdYrust/karvan/internal/auth"
	"github.com/PdYrust/karvan/internal/httpapi/middleware"
	"github.com/PdYrust/karvan/internal/httpapi/respond"
	"github.com/PdYrust/karvan/internal/secret"
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

// reauthBody is the password, and the code when two-step is on, typed
// again for an account change.
type reauthBody struct {
	Password string `json:"password"`
	Code     string `json:"code"`
}

func (b reauthBody) valid() bool {
	return b.Password != "" && len(b.Password) <= 1024 && (b.Code == "" || len(b.Code) == 6)
}

func (b reauthBody) reauth(r *http.Request) auth.Reauth {
	s, _ := middleware.Session(r.Context())
	return auth.Reauth{OperatorID: s.Session.OperatorID, Password: secret.NewValue(b.Password), Code: b.Code, IP: middleware.IP(r.Context())}
}

type newLoginJSON struct {
	Username  string `json:"username"`
	Password  string `json:"password"`
	CSRFToken string `json:"csrfToken"`
}

// RenewLogin gives the operator a new random username and password. They
// are in this response only, which is never stored; every other session
// ends and this browser gets a new one.
func (a *API) RenewLogin(w http.ResponseWriter, r *http.Request) {
	var body reauthBody
	if err := respond.Decode(w, r, &body); err != nil || !body.valid() {
		respond.BadRequest(w)
		return
	}
	res, err := a.Auth.RenewLogin(r.Context(), body.reauth(r), r.UserAgent())
	if err != nil {
		writeAuthError(w, r, err, http.StatusForbidden)
		return
	}
	a.Cookies.SetSession(w, r, res.Token)
	w.Header().Set("Cache-Control", "no-store")
	respond.JSON(w, http.StatusOK, newLoginJSON{Username: res.Username, Password: res.Password.Reveal(), CSRFToken: auth.CSRFToken(res.Token)})
}
