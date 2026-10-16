//go:build dbtest

package safemode

import (
	"context"
	"os/exec"
	"testing"
	"time"

	"github.com/PdYrust/karvan/internal/panel/marzbandb"
	"github.com/PdYrust/karvan/internal/panel/paneldb"
	"github.com/PdYrust/karvan/internal/panel/paneldb/paneldbtest"
	"github.com/PdYrust/karvan/internal/panel/pasarguarddb"
)

// containerRunner runs the MySQL and MariaDB client tools inside the test
// containers, since the test host has none. Inside, the server listens on
// 3306; the password comes through the environment like on a real host.
func containerRunner(port int) Runner {
	container := map[int]string{33061: "karvan-testdb-mysql-1", 33062: "karvan-testdb-mariadb-1"}[port]
	return func(ctx context.Context, name string, args ...string) *exec.Cmd {
		for i := range args {
			if args[i] == "-P" && i+1 < len(args) {
				args[i+1] = "3306"
			}
		}
		full := append([]string{"exec", "-i", "-e", "MYSQL_PWD", container, name}, args...)
		return exec.CommandContext(ctx, "docker", full...)
	}
}

func mysqlRoundTrip(t *testing.T, tg paneldb.Target) {
	ctx := context.Background()
	k, srv := key(t), server(t, tg)
	run := containerRunner(tg.Port)
	if err := Probe(ctx, srv, run); err != nil {
		t.Fatalf("probe: %v", err)
	}
	before := paneldbtest.Checksum(t, tg)
	b, err := Taker{Dir: t.TempDir(), Key: k, Run: run, Now: time.Now}.Take(ctx, tg, srv)
	if err != nil {
		t.Fatal(err)
	}
	scribble(t, tg)
	if err := (Restorer{Key: k, Run: run}).Restore(ctx, tg, srv, b); err != nil {
		t.Fatal(err)
	}
	if paneldbtest.Checksum(t, tg) != before {
		t.Fatal("restore did not bring the database back")
	}
}

func TestMySQLBackupAndRestore(t *testing.T) {
	for _, dir := range []struct{ path, rev string }{
		{"../panel/marzbandb/testdata", marzbandb.Revision},
		{"../panel/pasarguarddb/testdata", pasarguarddb.Revision},
	} {
		paneldbtest.Each(t, dir.path, dir.rev, func(t *testing.T, tg paneldb.Target) {
			if tg.Dialect == paneldb.MySQL {
				mysqlRoundTrip(t, tg)
			}
		})
	}
}
