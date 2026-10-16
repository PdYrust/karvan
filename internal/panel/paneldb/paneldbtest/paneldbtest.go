// Package paneldbtest gives tests a fresh panel database built from a
// committed schema fixture (scripts/testdb/seed.sh). SQLite always runs;
// MySQL, MariaDB, PostgreSQL and TimescaleDB run with the dbtest build tag
// against the servers from test/db/compose.yml.
package paneldbtest

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-sql-driver/mysql"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"

	"github.com/PdYrust/karvan/internal/panel/paneldb"
	"github.com/PdYrust/karvan/internal/secret"
)

// Fixture is one schema dump, named <revision>.<flavor>.sql.
type Fixture struct {
	Flavor string // sqlite, mysql, mariadb, postgres, timescale
	Path   string
}

// Fixtures lists the dumps for revision in dir that this build can load.
func Fixtures(t *testing.T, dir, revision string) []Fixture {
	t.Helper()
	var out []Fixture
	for _, f := range append([]string{"sqlite"}, serverFlavors...) {
		p := filepath.Join(dir, revision+"."+f+".sql")
		if _, err := os.Stat(p); err == nil {
			out = append(out, Fixture{Flavor: f, Path: p})
		}
	}
	if len(out) == 0 {
		t.Fatalf("no fixtures for %s in %s", revision, dir)
	}
	return out
}

// Each runs fn once per loadable fixture, each on a fresh database.
func Each(t *testing.T, dir, revision string, fn func(t *testing.T, tg paneldb.Target)) {
	t.Helper()
	for _, f := range Fixtures(t, dir, revision) {
		t.Run(f.Flavor, func(t *testing.T) { fn(t, Load(t, f)) })
	}
}

// Load creates a fresh database from f and returns where it is.
func Load(t *testing.T, f Fixture) paneldb.Target {
	t.Helper()
	dump, err := os.ReadFile(f.Path)
	if err != nil {
		t.Fatal(err)
	}
	switch f.Flavor {
	case "sqlite":
		return loadSQLite(t, string(dump))
	case "mysql", "mariadb":
		return loadMySQL(t, f.Flavor, string(dump))
	default:
		return loadPostgres(t, f.Flavor, string(dump))
	}
}

func loadSQLite(t *testing.T, dump string) paneldb.Target {
	p := filepath.Join(t.TempDir(), "panel.sqlite3")
	db, err := sql.Open("sqlite", p)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec("PRAGMA journal_mode=WAL;\n" + dump); err != nil {
		t.Fatalf("load sqlite fixture: %v", err)
	}
	return paneldb.Target{Dialect: paneldb.SQLite, Path: p}
}

// env reads a server DSN; under dbtest a missing one fails the test.
func env(t *testing.T, flavor string) string {
	key := "KARVAN_TEST_" + strings.ToUpper(flavor)
	v := os.Getenv(key)
	if v == "" {
		t.Fatalf("%s is not set; run `make test-db`", key)
	}
	return v
}

func dbName() string {
	b := make([]byte, 6)
	_, _ = rand.Read(b)
	return "t_" + hex.EncodeToString(b)
}

func loadMySQL(t *testing.T, flavor, dump string) paneldb.Target {
	c, err := mysql.ParseDSN(env(t, flavor))
	if err != nil {
		t.Fatal(err)
	}
	name := dbName()
	admin := openMySQL(t, c, "")
	defer admin.Close()
	if _, err := admin.Exec("CREATE DATABASE " + name); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		a := openMySQL(t, c, "")
		defer a.Close()
		_, _ = a.Exec("DROP DATABASE " + name)
	})
	load := openMySQL(t, c, name)
	defer load.Close()
	if _, err := load.Exec(dump); err != nil {
		t.Fatalf("load %s fixture: %v", flavor, err)
	}
	host, port := splitAddr(t, c.Addr)
	return paneldb.Target{Dialect: paneldb.MySQL, Host: host, Port: port, User: c.User,
		Password: secret.NewValue(c.Passwd), Database: name}
}

func openMySQL(t *testing.T, base *mysql.Config, name string) *sql.DB {
	c := base.Clone()
	c.DBName, c.MultiStatements = name, true
	conn, err := mysql.NewConnector(c)
	if err != nil {
		t.Fatal(err)
	}
	return sql.OpenDB(conn)
}

func loadPostgres(t *testing.T, flavor, dump string) paneldb.Target {
	base, err := pgx.ParseConfig(env(t, flavor))
	if err != nil {
		t.Fatal(err)
	}
	name := dbName()
	admin := openPG(base, "")
	defer admin.Close()
	if _, err := admin.Exec("CREATE DATABASE " + name); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		a := openPG(base, "")
		defer a.Close()
		_, _ = a.Exec("DROP DATABASE " + name + " WITH (FORCE)")
	})
	load := openPG(base, name)
	defer load.Close()
	if _, err := load.ExecContext(context.Background(), dump); err != nil {
		t.Fatalf("load %s fixture: %v", flavor, err)
	}
	return paneldb.Target{Dialect: paneldb.Postgres, Host: base.Host, Port: int(base.Port), User: base.User,
		Password: secret.NewValue(base.Password), Database: name}
}

func openPG(base *pgx.ConnConfig, name string) *sql.DB {
	c := base.Copy()
	if name != "" {
		c.Database = name
	}
	// A dump is many statements; only the simple protocol runs them at once.
	c.DefaultQueryExecMode = pgx.QueryExecModeSimpleProtocol
	return stdlib.OpenDB(*c)
}

func splitAddr(t *testing.T, addr string) (string, int) {
	host, p, ok := strings.Cut(addr, ":")
	var port int
	if _, err := fmt.Sscanf(p, "%d", &port); !ok || err != nil {
		t.Fatalf("bad address %q", addr)
	}
	return host, port
}
