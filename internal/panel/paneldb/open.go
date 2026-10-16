package paneldb

import (
	"context"
	"database/sql"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	_ "modernc.org/sqlite" // registers the "sqlite" driver
)

// sqliteBusy is how long Karvan waits for the panel to release SQLite's
// single writer lock. The panel's own default is 5 s (pysqlite).
const sqliteBusy = 5000

// Server describes the database server behind a handle.
type Server struct {
	Dialect Dialect
	Flavor  string // "sqlite", "mysql", "mariadb" or "postgres"
	Version string
}

type mode int

const (
	readOnly mode = iota
	readWrite
)

// open connects to t, detects the server, and returns a pool whose every
// connection has the session settings for m.
func open(ctx context.Context, t Target, m mode) (*sql.DB, Server, error) {
	switch t.Dialect {
	case SQLite:
		return openSQLite(ctx, t, m)
	case MySQL:
		return openMySQL(ctx, t, m)
	case Postgres:
		return openPostgres(ctx, t, m)
	}
	return nil, Server{}, ErrUnsupported
}

func openSQLite(ctx context.Context, t Target, m mode) (*sql.DB, Server, error) {
	q := url.Values{}
	q.Add("_pragma", fmt.Sprintf("busy_timeout(%d)", sqliteBusy))
	if m == readOnly {
		q.Set("mode", "ro")
		q.Set("_query_only", "1")
	} else {
		q.Set("_txlock", "immediate")
		q.Add("_pragma", "foreign_keys(1)")
	}
	dsn := (&url.URL{Scheme: "file", Path: t.Path, RawQuery: q.Encode()}).String()
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, Server{}, err
	}
	var v string
	if err := db.QueryRowContext(ctx, "SELECT sqlite_version()").Scan(&v); err != nil {
		_ = db.Close()
		return nil, Server{}, err
	}
	return db, Server{Dialect: SQLite, Flavor: "sqlite", Version: v}, nil
}

func mysqlConfig(t Target) *mysql.Config {
	c := mysql.NewConfig()
	c.User, c.Passwd, c.DBName = t.User, t.Password.Reveal(), t.Database
	if t.Socket != "" {
		c.Net, c.Addr = "unix", t.Socket
	} else {
		c.Net, c.Addr = "tcp", net.JoinHostPort(t.Host, strconv.Itoa(t.Port))
	}
	c.Loc, c.ParseTime = time.UTC, true
	c.Timeout = 10 * time.Second
	c.Params = map[string]string{"time_zone": "'+00:00'"}
	return c
}

func openMySQL(ctx context.Context, t Target, m mode) (*sql.DB, Server, error) {
	c := mysqlConfig(t)
	srv, err := probeMySQL(ctx, c)
	if err != nil {
		return nil, Server{}, err
	}
	if m == readOnly {
		c.Params[readOnlyVar(srv)] = "1"
	}
	conn, err := mysql.NewConnector(c)
	if err != nil {
		return nil, Server{}, err
	}
	db := sql.OpenDB(conn)
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, Server{}, err
	}
	return db, srv, nil
}

func probeMySQL(ctx context.Context, c *mysql.Config) (Server, error) {
	conn, err := mysql.NewConnector(c)
	if err != nil {
		return Server{}, err
	}
	db := sql.OpenDB(conn)
	defer db.Close()
	var v string
	if err := db.QueryRowContext(ctx, "SELECT VERSION()").Scan(&v); err != nil {
		return Server{}, err
	}
	flavor := "mysql"
	if strings.Contains(strings.ToLower(v), "mariadb") {
		flavor = "mariadb"
	}
	return Server{Dialect: MySQL, Flavor: flavor, Version: v}, nil
}

// readOnlyVar is the session variable that makes every transaction read
// only. MySQL 8 removed tx_read_only; MariaDB added transaction_read_only
// in 11.1 and still has tx_read_only.
func readOnlyVar(s Server) string {
	if s.Flavor == "mariadb" {
		return "tx_read_only"
	}
	return "transaction_read_only"
}

func openPostgres(ctx context.Context, t Target, m mode) (*sql.DB, Server, error) {
	c, err := pgx.ParseConfig("")
	if err != nil {
		return nil, Server{}, err
	}
	c.Host, c.Port, c.User, c.Database = t.Host, uint16(t.Port), t.User, t.Database //nolint:gosec // port is checked to 1..65535 in ParseURL
	c.Password = t.Password.Reveal()
	c.ConnectTimeout = 10 * time.Second
	c.RuntimeParams["TimeZone"] = "UTC"
	if m == readOnly {
		c.RuntimeParams["default_transaction_read_only"] = "on"
	}
	// Unnamed statements only, so a pooler in front of the database works.
	c.DefaultQueryExecMode = pgx.QueryExecModeExec
	db := stdlib.OpenDB(*c)
	var v string
	if err := db.QueryRowContext(ctx, "SHOW server_version").Scan(&v); err != nil {
		_ = db.Close()
		return nil, Server{}, err
	}
	return db, Server{Dialect: Postgres, Flavor: "postgres", Version: v}, nil
}
