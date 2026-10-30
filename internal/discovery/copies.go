package discovery

import (
	"fmt"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
)

// PanelCopy is the finding of a saved panel that is a copy of another
// folder but is kept because a rule, a batch or a job still uses it; the
// detail is the main panel's folder.
const PanelCopy = "panel_copy"

// backupName matches the folder names update scripts and people give
// copies: pasarguard.backup.20261028193827, marzban.bak, marzban-bak-2.
var backupName = regexp.MustCompile(`(?i)[._-](backup|bak)([._-].*)?$`)

// Copies finds the panels in ps that are copies and maps each one's ENV
// path to the folder of the panel it copies ("" when only its name says
// so). A copy is a folder with a backup-style name that no running
// container uses, or one of several
// folders on the same database that is not the main one: the folder a
// running container uses (running holds their folders, from the label
// com.docker.compose.project.working_dir), else the one named after the
// panel kind, else the first by path.
func Copies(ps []Panel, running map[string]bool) map[string]string {
	out := map[string]string{}
	groups := map[string][]Panel{}
	for i, p := range ps {
		key := p.Identity
		if key == "" {
			key = fmt.Sprint("none:", i) // no database to compare: on its own
		}
		groups[key] = append(groups[key], p)
	}
	for _, g := range groups {
		var named []Panel
		for _, p := range g {
			if running[filepath.Clean(p.Dir)] || !backupName.MatchString(filepath.Base(p.Dir)) {
				named = append(named, p)
			}
		}
		main := Panel{}
		if len(named) > 0 {
			main = realOf(named, running)
		}
		for _, p := range g {
			if p.EnvPath != main.EnvPath {
				out[p.EnvPath] = main.Dir
			}
		}
	}
	return out
}

// realOf picks the main panel among folders on one database.
func realOf(g []Panel, running map[string]bool) Panel {
	g = slices.Clone(g)
	slices.SortFunc(g, func(a, b Panel) int { return strings.Compare(a.EnvPath, b.EnvPath) })
	for _, p := range g {
		if running[filepath.Clean(p.Dir)] {
			return p
		}
	}
	for _, p := range g {
		if p.Kind != "" && strings.EqualFold(filepath.Base(p.Dir), string(p.Kind)) {
			return p
		}
	}
	return g[0]
}
