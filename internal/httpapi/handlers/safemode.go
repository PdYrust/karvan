package handlers

import (
	"net/http"
	"strconv"

	"github.com/PdYrust/karvan/internal/audit"
	"github.com/PdYrust/karvan/internal/httpapi/respond"
)

// safeModeOff is what the operator types to turn Safe Mode off.
const safeModeOff = "turn-off-safe-mode"

// GetSafeMode reports whether runs take a backup and undo on failure.
func (a *API) GetSafeMode(w http.ResponseWriter, r *http.Request) {
	on, err := a.Settings.SafeMode(r.Context())
	if err != nil {
		respond.Internal(w, r, err)
		return
	}
	respond.JSON(w, http.StatusOK, map[string]bool{"enabled": on})
}

// PutSafeMode turns Safe Mode on, or off with the typed confirmation.
// Both are audited.
func (a *API) PutSafeMode(w http.ResponseWriter, r *http.Request) {
	var b struct {
		Enabled bool   `json:"enabled"`
		Confirm string `json:"confirm"`
	}
	if err := respond.Decode(w, r, &b); err != nil {
		respond.BadRequest(w)
		return
	}
	if !b.Enabled && b.Confirm != safeModeOff {
		respond.Error(w, respond.APIError{Status: http.StatusBadRequest, Code: codeConfirm, Extra: map[string]any{"confirm": safeModeOff}})
		return
	}
	if err := a.Settings.SetSafeMode(r.Context(), b.Enabled); err != nil {
		respond.Internal(w, r, err)
		return
	}
	action := audit.ActionSafeModeOn
	if !b.Enabled {
		action = audit.ActionSafeModeOff
	}
	a.record(r, action, audit.Success, nil)
	respond.JSON(w, http.StatusOK, map[string]bool{"enabled": b.Enabled})
}

// Backups lists Safe Mode backups.
func (a *API) Backups(w http.ResponseWriter, r *http.Request) {
	bs, err := a.Runs.Backups(r.Context())
	if err != nil {
		respond.Internal(w, r, err)
		return
	}
	respond.JSON(w, http.StatusOK, map[string]any{"backups": orEmpty(bs)})
}

// DeleteBackup removes a Safe Mode backup.
func (a *API) DeleteBackup(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	err := a.Runs.DeleteBackup(r.Context(), id)
	outcome := audit.Success
	if err != nil {
		outcome = audit.Failure
	}
	a.record(r, audit.ActionBackupDelete, outcome, map[string]string{"backup": id, "ok": strconv.FormatBool(err == nil)})
	if err != nil {
		writeRunError(w, r, err)
		return
	}
	respond.NoContent(w)
}
