package handlers

import (
	"net/http"
	"path/filepath"
	"strings"
	"sync"

	"github.com/PdYrust/karvan/internal/audit"
	"github.com/PdYrust/karvan/internal/discovery"
	"github.com/PdYrust/karvan/internal/httpapi/respond"
)

// panelJSON is a destination panel on this server, as discovery sees it
// now. It carries no secret: the DB password stays in the panel's ENV.
type panelJSON struct {
	ID int64 `json:"id"`
	discovery.Panel
	Ready bool `json:"ready"`
	// Control says whether Karvan can stop and start the panel to put a
	// whole backup back, and if not, why.
	Control *controlJSON `json:"control,omitempty"`
}

type controlJSON struct {
	Available bool   `json:"available"`
	Reason    string `json:"reason,omitempty"`
}

func (a *API) controlOf(r *http.Request, envPath string, fresh bool) *controlJSON {
	if err := a.Runs.PanelControlStatus(r.Context(), envPath, fresh); err != nil {
		return &controlJSON{Reason: controlReason(err)}
	}
	return &controlJSON{Available: true}
}

// Panels lists the registered panels with a fresh look at each, all at
// once. Panel control comes from a probe up to a minute old unless the
// request says ?fresh=1.
func (a *API) Panels(w http.ResponseWriter, r *http.Request) {
	rows, err := a.Runs.Panels(r.Context())
	if err != nil {
		respond.Internal(w, r, err)
		return
	}
	fresh := r.URL.Query().Get("fresh") == "1"
	out := make([]panelJSON, len(rows))
	var wg sync.WaitGroup
	for i, p := range rows {
		wg.Go(func() {
			found := discovery.Inspect(r.Context(), p.EnvPath)
			out[i] = panelJSON{ID: p.ID, Panel: found, Ready: found.Ready(), Control: a.controlOf(r, p.EnvPath, fresh)}
		})
	}
	wg.Wait()
	respond.JSON(w, http.StatusOK, map[string]any{"panels": out})
}

// ScanPanels looks for panels under the scan roots, plus ENV files the
// operator names, and registers every one it recognizes (D53).
func (a *API) ScanPanels(w http.ResponseWriter, r *http.Request) {
	var b struct {
		Extra []string `json:"extra"`
	}
	if r.ContentLength != 0 {
		if err := respond.Decode(w, r, &b); err != nil || !validExtra(b.Extra) {
			respond.BadRequest(w)
			return
		}
	}
	found := discovery.Scan(r.Context(), a.ScanRoots, b.Extra)
	out := []panelJSON{}
	for _, p := range found {
		id := int64(0)
		if p.Kind != "" {
			var err error
			if id, err = a.Runs.RegisterPanel(r.Context(), p.EnvPath, p.Name, string(p.Kind)); err != nil {
				respond.Internal(w, r, err)
				return
			}
		}
		out = append(out, panelJSON{ID: id, Panel: p, Ready: p.Ready()})
	}
	a.record(r, audit.ActionPanelsScan, audit.Success, map[string]string{"found": strings.Join(names(found), ",")})
	respond.JSON(w, http.StatusOK, map[string]any{"panels": out})
}

// validExtra accepts a few absolute paths to files named .env, and
// nothing else, so the scan cannot be pointed at other files.
func validExtra(paths []string) bool {
	if len(paths) > 20 {
		return false
	}
	for _, p := range paths {
		if !filepath.IsAbs(p) || filepath.Clean(p) != p || filepath.Base(p) != ".env" || len(p) > 1024 {
			return false
		}
	}
	return true
}

func names(ps []discovery.Panel) []string {
	out := make([]string, 0, len(ps))
	for _, p := range ps {
		out = append(out, p.Name)
	}
	return out
}
