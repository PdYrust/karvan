package handlers

import (
	"net/http"
	"path/filepath"
	"strings"
	"sync"

	"github.com/PdYrust/karvan/internal/audit"
	"github.com/PdYrust/karvan/internal/discovery"
	"github.com/PdYrust/karvan/internal/httpapi/respond"
	"github.com/PdYrust/karvan/internal/panelctl"
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
	all := make([]panelJSON, len(rows))
	found := make([]discovery.Panel, len(rows))
	var wg sync.WaitGroup
	for i, p := range rows {
		wg.Go(func() {
			found[i] = discovery.Inspect(r.Context(), p.EnvPath)
			all[i] = panelJSON{ID: p.ID, Panel: found[i], Control: a.controlOf(r, p.EnvPath, fresh)}
		})
	}
	wg.Wait()
	copies := discovery.Copies(found, runningDirs(r))
	out := make([]panelJSON, 0, len(all))
	for _, p := range all {
		if orig, ok := copies[p.EnvPath]; ok {
			used, err := a.Runs.PanelInUse(r.Context(), p.ID)
			if err != nil {
				respond.Internal(w, r, err)
				return
			}
			if !used {
				continue // a scan removes it
			}
			p.Findings = append(p.Findings, discovery.Finding{Code: discovery.PanelCopy, Detail: orig})
		}
		p.Ready = p.Panel.Ready()
		out = append(out, p)
	}
	respond.JSON(w, http.StatusOK, map[string]any{"panels": out})
}

// runningDirs are the folders of the running compose projects; nil when
// Docker cannot be reached.
func runningDirs(r *http.Request) map[string]bool {
	dirs, err := panelctl.RunningDirs(r.Context(), "")
	if err != nil {
		return nil
	}
	return dirs
}

// dropCopies removes the saved panels that are copies from Karvan's list,
// its own record only, and returns the ones kept because a rule, a batch
// or a job still uses them, with a finding that asks to remove them.
func (a *API) dropCopies(r *http.Request, copies map[string]string) ([]panelJSON, error) {
	rows, err := a.Runs.Panels(r.Context())
	if err != nil {
		return nil, err
	}
	var kept []panelJSON
	for _, row := range rows {
		orig, ok := copies[row.EnvPath]
		if !ok {
			continue
		}
		gone, err := a.Runs.ForgetPanel(r.Context(), row.ID)
		if err != nil {
			return nil, err
		}
		if gone {
			a.record(r, audit.ActionPanelCopyRemoved, audit.Success, map[string]string{"panel": row.Name, "dir": filepath.Dir(row.EnvPath), "copyOf": orig})
			continue
		}
		p := discovery.Inspect(r.Context(), row.EnvPath)
		p.Findings = append(p.Findings, discovery.Finding{Code: discovery.PanelCopy, Detail: orig})
		kept = append(kept, panelJSON{ID: row.ID, Panel: p, Ready: false})
	}
	return kept, nil
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
	found, copies := discovery.Scan(r.Context(), a.ScanRoots, b.Extra, runningDirs(r))
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
	kept, err := a.dropCopies(r, copies)
	if err != nil {
		respond.Internal(w, r, err)
		return
	}
	out = append(out, kept...)
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
