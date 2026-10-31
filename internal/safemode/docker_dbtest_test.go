//go:build dbtest

package safemode

import (
	"context"
	"os/exec"
	"slices"
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

// restoring reports whether new sessions on tg start in TimescaleDB's
// restore mode, in which its background workers stay stopped.
func restoring(t *testing.T, tg paneldb.Target) bool {
	t.Helper()
	out, err := (Restorer{Run: DefaultRunner}).psql(context.Background(), tg,
		"SELECT count(*) FROM pg_db_role_setting s JOIN pg_database d ON d.oid = s.setdatabase WHERE d.datname = current_database() AND 'timescaledb.restoring=on' = ANY (s.setconfig)")
	if err != nil {
		t.Fatal(err)
	}
	return out != "0"
}

// TimescaleDB's background workers write their own rows into
// _timescaledb_catalog.metadata, racing the dump's COPY of the same rows.
// Every put-back, including a second one into the same database, must run
// with the workers stopped and start them again afterwards.
func TestTimescaleRestoresTwice(t *testing.T) {
	ctx := context.Background()
	tg := paneldbtest.Load(t, paneldbtest.Fixture{Flavor: "timescale", Path: "../panel/pasarguarddb/testdata/" + pasarguarddb.Revision + ".timescale.sql"})
	k, srv := key(t), server(t, tg)
	before := paneldbtest.Checksum(t, tg)
	b, err := Taker{Dir: t.TempDir(), Key: k, Run: DefaultRunner, Now: time.Now}.Take(ctx, tg, srv)
	if err != nil {
		t.Fatal(err)
	}
	stopped := 0
	run := func(ctx context.Context, name string, args ...string) *exec.Cmd {
		if name == "psql" && !slices.Contains(args, "-c") && restoring(t, tg) {
			stopped++
		}
		return DefaultRunner(ctx, name, args...)
	}
	for i := range 2 {
		scribble(t, tg)
		if err := (Restorer{Key: k, Run: run}).Restore(ctx, tg, srv, b); err != nil {
			t.Fatalf("restore %d: %v", i+1, err)
		}
		if stopped != i+1 {
			t.Fatalf("restore %d ran with the TimescaleDB workers running", i+1)
		}
		if restoring(t, tg) {
			t.Fatalf("restore %d left the database in restore mode", i+1)
		}
		if paneldbtest.Checksum(t, tg) != before {
			t.Fatalf("restore %d did not bring the database back", i+1)
		}
	}
}
