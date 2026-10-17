package handlers

import (
	"net/http"
	"strconv"

	"github.com/PdYrust/karvan/internal/audit"
	"github.com/PdYrust/karvan/internal/httpapi/respond"
	"github.com/PdYrust/karvan/internal/settings"
)

// safeModeOff is what the operator types to turn Safe Mode off.
const safeModeOff = "turn-off-safe-mode"

type safeModeJSON struct {
	Enabled  bool   `json:"enabled"`
	Rollback string `json:"rollback"` // "undo" or "backup"
}

func (a *API) safeMode(r *http.Request) (safeModeJSON, error) {
	on, err := a.Settings.SafeMode(r.Context())
	if err != nil {
		return safeModeJSON{}, err
	}
	rb, err := a.Settings.Rollback(r.Context())
	return safeModeJSON{Enabled: on, Rollback: rb}, err
}

// GetSafeMode reports whether runs take a backup, and what they do on
// failure.
func (a *API) GetSafeMode(w http.ResponseWriter, r *http.Request) {
	sm, err := a.safeMode(r)
	if err != nil {
		respond.Internal(w, r, err)
		return
	}
	respond.JSON(w, http.StatusOK, sm)
}

// PutSafeMode turns Safe Mode on, or off with the typed confirmation, and
// sets what it does on failure. Every change is audited.
func (a *API) PutSafeMode(w http.ResponseWriter, r *http.Request) {
	var b struct {
		Enabled  bool   `json:"enabled"`
		Confirm  string `json:"confirm"`
		Rollback string `json:"rollback"` // empty: unchanged
	}
	if err := respond.Decode(w, r, &b); err != nil || (b.Rollback != "" && b.Rollback != settings.RollbackUndo && b.Rollback != settings.RollbackBackup) {
		respond.BadRequest(w)
		return
	}
	before, err := a.safeMode(r)
	if err != nil {
		respond.Internal(w, r, err)
		return
	}
	if !b.Enabled && before.Enabled && b.Confirm != safeModeOff {
		respond.Error(w, respond.APIError{Status: http.StatusBadRequest, Code: codeConfirm, Extra: map[string]any{"confirm": safeModeOff}})
		return
	}
	if err := a.Settings.SetSafeMode(r.Context(), b.Enabled); err != nil {
		respond.Internal(w, r, err)
		return
	}
	if b.Enabled != before.Enabled {
		action := audit.ActionSafeModeOn
		if !b.Enabled {
			action = audit.ActionSafeModeOff
		}
		a.record(r, action, audit.Success, nil)
	}
	if b.Rollback != "" && b.Rollback != before.Rollback {
		if err := a.Settings.SetRollback(r.Context(), b.Rollback); err != nil {
			respond.Internal(w, r, err)
			return
		}
		a.record(r, audit.ActionSafeModeRollback, audit.Success, map[string]string{"rollback": b.Rollback})
	}
	a.GetSafeMode(w, r)
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

// RestoreBackup puts a Safe Mode backup back: Karvan stops the panel,
// restores and starts it again. The operator types the backup id to
// confirm, because this also takes back everything written since.
func (a *API) RestoreBackup(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var b struct {
		Confirm string `json:"confirm"`
	}
	if err := respond.Decode(w, r, &b); err != nil {
		respond.BadRequest(w)
		return
	}
	if b.Confirm != id {
		respond.Error(w, respond.APIError{Status: http.StatusBadRequest, Code: codeConfirm, Extra: map[string]any{"confirm": id}})
		return
	}
	err := a.Runs.RestoreBackup(r.Context(), id, false)
	a.record(r, audit.ActionBackupRestoreWeb, outcomeOf(err), map[string]string{"backup": id})
	if err != nil {
		writeRunError(w, r, err)
		return
	}
	respond.NoContent(w)
}
