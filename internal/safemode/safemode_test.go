package safemode

import (
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/PdYrust/karvan/internal/panel/marzbandb"
	"github.com/PdYrust/karvan/internal/panel/paneldb"
	"github.com/PdYrust/karvan/internal/panel/paneldb/paneldbtest"
	"github.com/PdYrust/karvan/internal/panel/pasarguarddb"
	"github.com/PdYrust/karvan/internal/secret"
)

func key(t *testing.T) *secret.Key {
	t.Helper()
	p := filepath.Join(t.TempDir(), "key")
	if err := secret.CreateKeyFile(p); err != nil {
		t.Fatal(err)
	}
	k, err := secret.LoadKeyFile(p)
	if err != nil {
		t.Fatal(err)
	}
	return k
}

func server(t *testing.T, tg paneldb.Target) paneldb.Server {
	t.Helper()
	r, err := paneldb.OpenReader(context.Background(), tg)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	return r.Server()
}

func scribble(t *testing.T, tg paneldb.Target) {
	t.Helper()
	ctx := context.Background()
	w, err := paneldb.OpenWriter(ctx, tg)
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	err = w.Chunk(ctx, func(x paneldb.Execer) error {
		_, err := x.ExecContext(ctx, "INSERT INTO jwt (id, secret_key) VALUES (?, ?)", 99, "scribble")
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
}

// roundTrip backs up, changes the database, restores, and compares.
func roundTrip(t *testing.T, tg paneldb.Target) {
	t.Helper()
	ctx := context.Background()
	k, srv := key(t), server(t, tg)
	if err := Probe(ctx, srv, DefaultRunner); err != nil {
		t.Fatalf("probe: %v", err)
	}
	before := paneldbtest.Checksum(t, tg)
	b, err := Taker{Dir: filepath.Join(t.TempDir(), "safe-mode"), Key: k, Run: DefaultRunner, Now: time.Now}.Take(ctx, tg, srv)
	if err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(b.Path)
	if err != nil || info.Mode().Perm() != 0o600 || b.Size == 0 {
		t.Fatalf("backup file: %v %v", info, err)
	}
	raw, _ := os.ReadFile(b.Path)
	if strings.Contains(string(raw), "CREATE TABLE") {
		t.Fatal("the backup is not encrypted")
	}
	scribble(t, tg)
	if paneldbtest.Checksum(t, tg) == before {
		t.Fatal("the change did not land")
	}
	if err := (Restorer{Key: k, Run: DefaultRunner}).Restore(ctx, tg, srv, b); err != nil {
		t.Fatal(err)
	}
	if paneldbtest.Checksum(t, tg) != before {
		t.Fatal("restore did not bring the database back")
	}
}

func TestBackupAndRestore(t *testing.T) {
	t.Run("marzban", func(t *testing.T) {
		paneldbtest.Each(t, "../panel/marzbandb/testdata", marzbandb.Revision, func(t *testing.T, tg paneldb.Target) {
			if tg.Dialect == paneldb.MySQL {
				return // MySQL and MariaDB run through their containers in TestMySQLBackupAndRestore
			}
			roundTrip(t, tg)
		})
	})
	t.Run("pasarguard", func(t *testing.T) {
		paneldbtest.Each(t, "../panel/pasarguarddb/testdata", pasarguarddb.Revision, func(t *testing.T, tg paneldb.Target) {
			if tg.Dialect == paneldb.MySQL {
				return
			}
			roundTrip(t, tg)
		})
	})
}

func fakeVersion(out string) Runner {
	return func(ctx context.Context, _ string, _ ...string) *exec.Cmd {
		return exec.CommandContext(ctx, "echo", out)
	}
}

func TestDumpToolsAreProbed(t *testing.T) {
	ctx := context.Background()
	missing := func(ctx context.Context, _ string, _ ...string) *exec.Cmd {
		return exec.CommandContext(ctx, "karvan-no-such-tool")
	}
	var te *ToolError
	if err := Probe(ctx, paneldb.Server{Dialect: paneldb.MySQL, Flavor: "mariadb", Version: "11.4.13-MariaDB"}, missing); !errors.As(err, &te) || te.Package != "mariadb-client" {
		t.Fatalf("missing tool: %v", err)
	}
	pg := paneldb.Server{Dialect: paneldb.Postgres, Flavor: "postgres", Version: "17.11"}
	if err := Probe(ctx, pg, fakeVersion("pg_dump (PostgreSQL) 16.4")); !errors.As(err, &te) || te.Package != "postgresql-client-17" {
		t.Fatalf("old pg_dump: %v", err)
	}
	if err := Probe(ctx, pg, fakeVersion("pg_dump (PostgreSQL) 17.2")); err != nil {
		t.Fatalf("current pg_dump: %v", err)
	}
}

func TestCheckSpace(t *testing.T) {
	var se *SpaceError
	if err := CheckSpace(1<<30, 1<<30); !errors.As(err, &se) {
		t.Fatal("1 GiB free is not enough for a 1 GiB database")
	}
	if err := CheckSpace(1<<30, 2<<30); err != nil {
		t.Fatal(err)
	}
	if free, err := Free(t.TempDir()); err != nil || free <= 0 {
		t.Fatal(free, err)
	}
}

func TestAFailedBackupLeavesNothing(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "safe-mode")
	tk := Taker{Dir: dir, Key: key(t), Run: DefaultRunner, Now: time.Now}
	_, err := tk.Take(context.Background(), paneldb.Target{Dialect: paneldb.SQLite, Path: "/nonexistent/db.sqlite3"}, paneldb.Server{Dialect: paneldb.SQLite})
	if err == nil {
		t.Fatal("backup of a missing database succeeded")
	}
	if entries, _ := os.ReadDir(dir); len(entries) != 0 {
		t.Fatalf("left behind: %v", entries)
	}
}

func TestAnotherBackupFileDoesNotOpen(t *testing.T) {
	k := key(t)
	tg := paneldbtest.Load(t, paneldbtest.Fixture{Flavor: "sqlite", Path: "../panel/marzbandb/testdata/" + marzbandb.Revision + ".sqlite.sql"})
	tk := Taker{Dir: filepath.Join(t.TempDir(), "sm"), Key: k, Run: DefaultRunner, Now: time.Now}
	b, err := tk.Take(context.Background(), tg, paneldb.Server{Dialect: paneldb.SQLite})
	if err != nil {
		t.Fatal(err)
	}
	b.ID = "someone-else"
	r, err := Open(k, b)
	if err == nil {
		_, err = io.ReadAll(r)
		r.Close()
	}
	if !errors.Is(err, secret.ErrStream) {
		t.Fatalf("a backup opened under another id: %v", err)
	}
}
