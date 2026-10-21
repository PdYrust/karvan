package handlers

import (
	"net/http"
	"time"

	"github.com/PdYrust/karvan/internal/audit"
	"github.com/PdYrust/karvan/internal/httpapi/middleware"
	"github.com/PdYrust/karvan/internal/httpapi/respond"
	"github.com/PdYrust/karvan/internal/migration/run"
)

const overviewActivityLimit = 8

type signInJSON struct {
	At time.Time `json:"at"`
	IP string    `json:"ip"`
}

type activityJSON struct {
	ID      int64         `json:"id"`
	At      time.Time     `json:"at"`
	Actor   string        `json:"actor"`
	Action  string        `json:"action"`
	Outcome audit.Outcome `json:"outcome"`
	IP      string        `json:"ip"`
}

type overviewJSON struct {
	Version  string `json:"version"`
	SafeMode bool   `json:"safeMode"`
	Security struct {
		TOTPEnabled    bool        `json:"totpEnabled"`
		PreviousSignIn *signInJSON `json:"previousSignIn"`
	} `json:"security"`
	// Migrations sums up every job: how many are in each state, and the
	// users they moved and left out.
	Migrations run.Totals     `json:"migrations"`
	Activity   []activityJSON `json:"activity"`
}

// Overview returns the data for the first page after sign-in. The panels
// themselves come from GET /panels, which inspects each one.
func (a *API) Overview(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	s, _ := middleware.Session(ctx)
	op, err := a.Auth.Operators.ByID(ctx, s.Session.OperatorID)
	if err != nil {
		respond.Internal(w, r, err)
		return
	}
	prev, err := a.Audit.List(ctx, audit.Filter{Actor: op.Username, Action: audit.ActionLogin, Outcome: audit.Success, Limit: 1, SkipNewest: 1})
	if err != nil {
		respond.Internal(w, r, err)
		return
	}
	recent, err := a.Audit.List(ctx, audit.Filter{Limit: overviewActivityLimit})
	if err != nil {
		respond.Internal(w, r, err)
		return
	}

	safe := true
	if a.Settings != nil {
		if safe, err = a.Settings.SafeMode(ctx); err != nil {
			respond.Internal(w, r, err)
			return
		}
	}
	totals := run.Totals{Jobs: map[string]int64{}}
	if a.Runs != nil {
		if totals, err = a.Runs.Totals(ctx); err != nil {
			respond.Internal(w, r, err)
			return
		}
	}
	out := overviewJSON{Version: a.Version, SafeMode: safe, Migrations: totals, Activity: []activityJSON{}}
	out.Security.TOTPEnabled = op.TOTPEnabled
	if len(prev) == 1 {
		out.Security.PreviousSignIn = &signInJSON{At: prev[0].At.UTC(), IP: prev[0].IP}
	}
	for _, e := range recent {
		out.Activity = append(out.Activity, activityJSON{ID: e.ID, At: e.At.UTC(), Actor: e.Actor, Action: e.Action, Outcome: e.Outcome, IP: e.IP})
	}
	respond.JSON(w, http.StatusOK, out)
}
