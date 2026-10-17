package handlers

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/PdYrust/karvan/internal/dbstats"
	"github.com/PdYrust/karvan/internal/hoststats"
	"github.com/PdYrust/karvan/internal/httpapi/respond"
)

// rangeSpec is how far back a range goes and how wide its buckets are, so
// every range comes back as roughly 300 points.
type rangeSpec struct {
	span time.Duration
	step int64
}

const liveRange = "live"

var analyticsRanges = map[string]rangeSpec{
	liveRange: {time.Hour, 60}, // databases; the server's live range is its 5 s samples
	"1h":      {time.Hour, 60},
	"24h":     {24 * time.Hour, 300},
	"7d":      {7 * 24 * time.Hour, 1800},
	"30d":     {30 * 24 * time.Hour, 7200},
}

func readRange(r *http.Request, def string) (string, rangeSpec, bool) {
	name := r.URL.Query().Get("range")
	if name == "" {
		name = def
	}
	spec, ok := analyticsRanges[name]
	return name, spec, ok
}

type serverJSON struct {
	Available bool               `json:"available"`
	Range     string             `json:"range"`
	Step      int64              `json:"step"`
	Seq       int64              `json:"seq"`
	Facts     hoststats.Facts    `json:"facts"`
	Disks     []hoststats.Disk   `json:"disks"`
	Now       *hoststats.Sample  `json:"now"`
	Samples   []hoststats.Sample `json:"samples,omitempty"`
	Points    []hoststats.Minute `json:"points,omitempty"`
}

// ServerAnalytics is this server's CPU, memory, disks and network: live
// samples (after a seq, for polling) or minutes over a range.
func (a *API) ServerAnalytics(w http.ResponseWriter, r *http.Request) {
	name, spec, ok := readRange(r, liveRange)
	if !ok || a.Host == nil {
		respond.BadRequest(w)
		return
	}
	now := time.Now()
	out := serverJSON{Range: name, Facts: a.HostReader.Facts(now), Disks: a.HostReader.Disks()}
	if latest, has, _ := a.Host.Latest(); has {
		out.Available, out.Now, out.Seq = true, &latest, latest.Seq
	}
	if name == liveRange {
		after, _ := strconv.ParseInt(r.URL.Query().Get("after"), 10, 64)
		out.Step, out.Samples = int64(hoststats.Interval/time.Second), a.Host.Since(after)
		respond.JSON(w, http.StatusOK, out)
		return
	}
	points, err := a.HostStore.Range(r.Context(), now.Add(-spec.span).Unix(), now.Unix()+1, spec.step)
	if err != nil {
		respond.Internal(w, r, err)
		return
	}
	out.Step, out.Points = spec.step, points
	respond.JSON(w, http.StatusOK, out)
}

type panelStatsJSON struct {
	dbstats.Snapshot
	// Pending: registered, but not read yet (Karvan just started).
	Pending bool             `json:"pending,omitempty"`
	Points  []dbstats.Minute `json:"points"`
}

// PanelsAnalytics is every registered panel's database: its last reading
// and minutes over a range.
func (a *API) PanelsAnalytics(w http.ResponseWriter, r *http.Request) {
	name, spec, ok := readRange(r, "24h")
	if !ok || a.DBStats == nil {
		respond.BadRequest(w)
		return
	}
	rows, err := a.Runs.Panels(r.Context())
	if err != nil {
		respond.Internal(w, r, err)
		return
	}
	now := time.Now()
	out := make([]panelStatsJSON, 0, len(rows))
	for _, p := range rows {
		item, err := a.panelStats(r, p.ID, p.Name, spec, now)
		if err != nil {
			respond.Internal(w, r, err)
			return
		}
		out = append(out, item)
	}
	respond.JSON(w, http.StatusOK, map[string]any{"range": name, "step": spec.step, "panels": out})
}

func (a *API) panelStats(r *http.Request, id int64, name string, spec rangeSpec, now time.Time) (panelStatsJSON, error) {
	snap, ok := a.DBStats.Snapshot(id)
	item := panelStatsJSON{Snapshot: snap, Pending: !ok}
	if !ok {
		item.PanelID, item.Name = id, name
	}
	points, err := a.DBStats.Store.Range(r.Context(), id, now.Add(-spec.span).Unix(), now.Unix()+1, spec.step)
	item.Points = points
	return item, err
}

// minDetailSpan is the shortest history the detail reads: hourly traffic
// over one hour would be a single bar.
const minDetailSpan = 24 * time.Hour

// PanelAnalytics is one panel's database in full: its reading, minutes,
// largest tables, hourly traffic, sign-ups and subscription apps.
func (a *API) PanelAnalytics(w http.ResponseWriter, r *http.Request) {
	name, spec, ok := readRange(r, "24h")
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if !ok || err != nil || a.DBStats == nil {
		respond.BadRequest(w)
		return
	}
	rows, err := a.Runs.Panels(r.Context())
	if err != nil {
		respond.Internal(w, r, err)
		return
	}
	var panelName string
	for _, p := range rows {
		if p.ID == id {
			panelName = p.Name
		}
	}
	if panelName == "" {
		respond.Error(w, respond.APIError{Status: http.StatusNotFound, Code: codeNotFound})
		return
	}
	now := time.Now()
	item, err := a.panelStats(r, id, panelName, spec, now)
	if err != nil {
		respond.Internal(w, r, err)
		return
	}
	out := map[string]any{"range": name, "step": spec.step, "panel": item}
	detail, err := a.DBStats.Detail(r.Context(), id, name, now.Add(-max(spec.span, minDetailSpan)))
	switch {
	case err == nil:
		out["detail"] = detail
	case errors.Is(err, dbstats.ErrNoPanel):
		out["detailProblem"] = "pending"
	default:
		out["detailProblem"] = "db_unreachable"
	}
	respond.JSON(w, http.StatusOK, out)
}
