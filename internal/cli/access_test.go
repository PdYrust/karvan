package cli

import (
	"bytes"
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/PdYrust/karvan/internal/settings"
	"github.com/PdYrust/karvan/internal/store"
)

func doAccess(dir string, args ...string) (int, string, string) {
	var out, errOut bytes.Buffer
	args = append(append([]string{"access"}, args...), "-data-dir", filepath.Join(dir, "data"), "-key-file", filepath.Join(dir, "secret.key"))
	code := Run(context.Background(), args, IO{Stdout: &out, Stderr: &errOut})
	return code, out.String(), errOut.String()
}

// karvanTLS opens the store of dir, runs fn on its settings and returns
// the TLS settings after it.
func karvanTLS(t *testing.T, dir string, fn func(*settings.Store) error) settings.TLS {
	t.Helper()
	ctx := context.Background()
	db, err := store.Open(ctx, filepath.Join(dir, "data"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	s := settings.New(db)
	if err := fn(s); err != nil {
		t.Fatal(err)
	}
	h, err := s.HTTP(ctx)
	if err != nil {
		t.Fatal(err)
	}
	return h.TLS
}

// karvan access open-ip, the way back in, turns "Only open Karvan on its
// domain" off and is audited; run again it changes nothing.
func TestAccessOpenIP(t *testing.T) {
	dir := t.TempDir()
	if code, _, errOut := doInit(t, dir); code != 0 {
		t.Fatalf("init: %d %s", code, errOut)
	}
	karvanTLS(t, dir, func(s *settings.Store) error {
		return s.SetKarvan(context.Background(), settings.TLS{DomainID: 3, Armed: true})
	})
	code, out, errOut := doAccess(dir, "open-ip")
	if code != 0 || !strings.Contains(out, "IP opens the panel again") {
		t.Fatalf("open-ip: %d %q %s", code, out, errOut)
	}
	none := func(*settings.Store) error { return nil }
	if got := karvanTLS(t, dir, none); got != (settings.TLS{DomainID: 3, OpenIP: true}) {
		t.Fatalf("stored %+v", got)
	}
	if code, out, _ := doAccess(dir, "open-ip"); code != 0 || !strings.Contains(out, "already") {
		t.Fatalf("again: %d %q", code, out)
	}
	db, err := store.Open(context.Background(), filepath.Join(dir, "data"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var n int
	if err := db.QueryRow("SELECT count(*) FROM audit_events WHERE action = 'cli.access_open_ip' AND actor = 'cli'").Scan(&n); err != nil || n != 1 {
		t.Fatalf("audited %d times, %v", n, err)
	}
	for _, bad := range [][]string{{}, {"close"}} {
		if code, _, _ := doAccess(dir, bad...); code != 2 {
			t.Errorf("access %v: exit %d, want 2", bad, code)
		}
	}
}
