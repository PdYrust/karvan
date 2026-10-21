package handlers

import (
	"io"
	"net/http"

	"github.com/PdYrust/karvan/internal/httpapi/respond"
	"github.com/PdYrust/karvan/internal/migration/report"
)

// CheckReport is the pre-flight report (B7): what a run of the current
// plan would do, as JSON (karvan.check/v1), before anything runs.
func (a *API) CheckReport(w http.ResponseWriter, r *http.Request) {
	rep, err := a.Runs.CheckReport(r.Context(), r.PathValue("id"))
	if err != nil {
		writeRunError(w, r, err)
		return
	}
	respond.JSON(w, http.StatusOK, rep)
}

// CheckReportCSV is the pre-flight report, one row per user.
func (a *API) CheckReportCSV(w http.ResponseWriter, r *http.Request) {
	rep, err := a.Runs.CheckReport(r.Context(), r.PathValue("id"))
	if err != nil {
		writeRunError(w, r, err)
		return
	}
	a.csv(w, r, "karvan-check-"+rep.JobID+".csv", func(out io.Writer) error { return report.WriteCheck(out, rep) })
}
