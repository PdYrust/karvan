package discovery

import (
	"maps"
	"testing"

	"github.com/PdYrust/karvan/internal/panel"
)

func folder(dir, identity string) Panel {
	return Panel{Dir: dir, EnvPath: dir + "/.env", Kind: panel.PasarGuard, Identity: identity}
}

// The owner's server: the update script's backups and the real folder on
// one MySQL database.
func TestCopies(t *testing.T) {
	db := "mysql:127.0.0.1:3306/pasarguard"
	ps := []Panel{
		folder("/opt/pasarguard", db),
		folder("/opt/pasarguard.backup.20261028193827", db),
		folder("/opt/pasarguard.backup.20261028193928", db),
		folder("/opt/marzban.bak", "sqlite:1:2"),
		folder("/opt/marzban", "sqlite:1:3"),
		folder("/opt/panel-old", ""),
		folder("/opt/x-backup-2", ""),
	}
	want := map[string]string{
		"/opt/pasarguard.backup.20261028193827/.env": "/opt/pasarguard",
		"/opt/pasarguard.backup.20261028193928/.env": "/opt/pasarguard",
		"/opt/marzban.bak/.env":                      "",
		"/opt/x-backup-2/.env":                       "",
	}
	if got := Copies(ps, nil); !maps.Equal(got, want) {
		t.Fatalf("Copies = %v", got)
	}
}

// The folder a running container uses is the real one, whatever the
// names say; with no container, the one named after the panel kind.
func TestCopiesPreferTheRunningFolder(t *testing.T) {
	db := "mysql:127.0.0.1:3306/pasarguard"
	ps := []Panel{folder("/opt/pasarguard", db), folder("/opt/pg-new", db), folder("/opt/a-panel", db)}
	got := Copies(ps, map[string]bool{"/opt/pg-new": true})
	want := map[string]string{"/opt/pasarguard/.env": "/opt/pg-new", "/opt/a-panel/.env": "/opt/pg-new"}
	if !maps.Equal(got, want) {
		t.Fatalf("running: %v", got)
	}
	got = Copies(ps, nil)
	want = map[string]string{"/opt/pg-new/.env": "/opt/pasarguard", "/opt/a-panel/.env": "/opt/pasarguard"}
	if !maps.Equal(got, want) {
		t.Fatalf("by kind: %v", got)
	}
	if got := Copies([]Panel{folder("/opt/one", db), folder("/opt/two", "other"), folder("/opt/up.bak", "")},
		map[string]bool{"/opt/up.bak": true}); len(got) != 0 {
		t.Fatalf("separate databases: %v", got)
	}
}
