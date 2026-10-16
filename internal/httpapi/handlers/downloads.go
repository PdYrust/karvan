package handlers

import (
	"bytes"
	"io"
	"net/http"

	"github.com/PdYrust/karvan/internal/audit"
	"github.com/PdYrust/karvan/internal/httpapi/respond"
	"github.com/PdYrust/karvan/internal/migration/report"
	"github.com/PdYrust/karvan/internal/migration/run"
)

// Report is the JSON report of a run (D52), without the new links.
func (a *API) Report(w http.ResponseWriter, r *http.Request) {
	rep, ok := a.report(w, r)
	if !ok {
		return
	}
	respond.JSON(w, http.StatusOK, rep.WithoutLinks())
}

// ReportCSV is one row per user.
func (a *API) ReportCSV(w http.ResponseWriter, r *http.Request) {
	rep, ok := a.report(w, r)
	if !ok {
		return
	}
	a.csv(w, r, "karvan-report-"+rep.JobID+".csv", func(out io.Writer) error { return report.WriteUsers(out, rep) })
}

// LinksCSV is every moved user's new subscription link, minted again
// when the job is from before a restart.
func (a *API) LinksCSV(w http.ResponseWriter, r *http.Request) {
	rep, err := a.Runs.WithLinks(r.Context(), r.PathValue("id"))
	if err == nil && rep == nil {
		err = run.ErrNotFound
	}
	if err != nil {
		writeRunError(w, r, err)
		return
	}
	a.csv(w, r, "karvan-links-"+rep.JobID+".csv", func(out io.Writer) error { return report.WriteLinks(out, rep) })
}

// PasswordsCSV hands out the generated admin passwords once. A second
// download finds nothing: Karvan never stores them.
func (a *API) PasswordsCSV(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	pw, err := a.Runs.TakePasswords(id)
	if err != nil {
		writeRunError(w, r, err)
		return
	}
	if len(pw) == 0 {
		respond.Error(w, respond.APIError{Status: http.StatusGone, Code: codeNothingLeft})
		return
	}
	a.record(r, audit.ActionPasswordsTaken, audit.Success, map[string]string{"job": id})
	a.csv(w, r, "karvan-passwords-"+id+".csv", func(out io.Writer) error { return report.WritePasswords(out, pw) })
}

func (a *API) report(w http.ResponseWriter, r *http.Request) (*run.Report, bool) {
	rep, err := a.Runs.Report(r.PathValue("id"))
	if err == nil && rep == nil {
		err = run.ErrNotFound
	}
	if err != nil {
		writeRunError(w, r, err)
		return nil, false
	}
	return rep, true
}

// csv builds the file first, so a failure is still a clean error answer.
func (a *API) csv(w http.ResponseWriter, r *http.Request, name string, write func(io.Writer) error) {
	var b bytes.Buffer
	if err := write(&b); err != nil {
		respond.Internal(w, r, err)
		return
	}
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="`+name+`"`)
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(b.Bytes())
}
