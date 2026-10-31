package handlers

import (
	"net/http"
	"os"

	"github.com/PdYrust/karvan/internal/httpapi/respond"
)

// InstallSettings says how Karvan runs on this server. It always runs as
// root when installed; a development run reads false.
func (a *API) InstallSettings(w http.ResponseWriter, _ *http.Request) {
	respond.JSON(w, http.StatusOK, struct {
		RunsAsRoot bool `json:"runsAsRoot"`
	}{os.Geteuid() == 0})
}
