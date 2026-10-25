package handlers

import (
	"net/http"
	"regexp"
	"strconv"
	"time"

	"github.com/PdYrust/karvan/internal/audit"
	"github.com/PdYrust/karvan/internal/httpapi/respond"
)

const (
	auditPage    = 50
	auditPageMax = 200
)

var actionPattern = regexp.MustCompile(`^[a-z_]{1,32}(\.[a-z_]{1,32})?$`)

type auditEventJSON struct {
	ID      int64             `json:"id"`
	At      time.Time         `json:"at"`
	Actor   string            `json:"actor"`
	IP      string            `json:"ip"`
	Action  string            `json:"action"`
	Outcome audit.Outcome     `json:"outcome"`
	Detail  map[string]string `json:"detail"`
}

type auditPageJSON struct {
	Events []auditEventJSON `json:"events"`
	// Next is the before value of the next page; null on the last one.
	Next *int64 `json:"next"`
}

// auditFilter reads ?before, ?action, ?outcome and ?limit.
func auditFilter(r *http.Request) (audit.Filter, bool) {
	q := r.URL.Query()
	f := audit.Filter{Action: q.Get("action"), Outcome: audit.Outcome(q.Get("outcome")), Limit: auditPage}
	if f.Action != "" && !actionPattern.MatchString(f.Action) {
		return f, false
	}
	if f.Outcome != "" && f.Outcome != audit.Success && f.Outcome != audit.Failure {
		return f, false
	}
	if v := q.Get("before"); v != "" {
		n, err := strconv.ParseInt(v, 10, 64)
		if err != nil || n < 1 {
			return f, false
		}
		f.Before = n
	}
	if v := q.Get("limit"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 || n > auditPageMax {
			return f, false
		}
		f.Limit = n
	}
	return f, true
}

// AuditLog lists audit events, newest first, a page at a time. Details
// never hold secrets: the audit log is written without them.
func (a *API) AuditLog(w http.ResponseWriter, r *http.Request) {
	f, ok := auditFilter(r)
	if !ok {
		respond.BadRequest(w)
		return
	}
	limit := f.Limit
	f.Limit++ // one more tells whether a next page exists
	events, err := a.Audit.List(r.Context(), f)
	if err != nil {
		respond.Internal(w, r, err)
		return
	}
	out := auditPageJSON{Events: make([]auditEventJSON, 0, min(len(events), limit))}
	if len(events) > limit {
		events = events[:limit]
		next := events[limit-1].ID
		out.Next = &next
	}
	for _, e := range events {
		detail := e.Detail
		if detail == nil {
			detail = map[string]string{}
		}
		out.Events = append(out.Events, auditEventJSON{ID: e.ID, At: e.At.UTC(), Actor: e.Actor, IP: e.IP,
			Action: e.Action, Outcome: e.Outcome, Detail: detail})
	}
	respond.JSON(w, http.StatusOK, out)
}
