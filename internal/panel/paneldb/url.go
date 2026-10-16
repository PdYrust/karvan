// Package paneldb opens a panel's own database the way the panel does,
// from its SQLAlchemy URL, with Karvan's safety settings: read-only handles
// for checks, UTC everywhere, and one connection type per dialect.
package paneldb

import (
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/PdYrust/karvan/internal/secret"
)

// Dialect is the SQL family a panel database speaks.
type Dialect string

// Dialects both panels use. MariaDB is MySQL here; TimescaleDB is Postgres.
const (
	SQLite   Dialect = "sqlite"
	MySQL    Dialect = "mysql"
	Postgres Dialect = "postgres"
)

// Errors from ParseURL. Callers report them as discovery findings.
var (
	ErrBadURL       = errors.New("database URL cannot be parsed")
	ErrUnsupported  = errors.New("database type is not supported")
	ErrRelativePath = errors.New("SQLite path is relative to the panel's working directory")
	ErrNoDatabase   = errors.New("database URL names no database")
)

// Target is where a panel database lives.
type Target struct {
	Dialect  Dialect
	Path     string // SQLite file
	Host     string
	Port     int
	Socket   string // MySQL unix socket
	User     string
	Password secret.Value
	Database string
}

// String is safe to log: it never includes the password.
func (t Target) String() string {
	switch {
	case t.Dialect == SQLite:
		return "sqlite://" + t.Path
	case t.Socket != "":
		return fmt.Sprintf("%s://%s@unix(%s)/%s", t.Dialect, t.User, t.Socket, t.Database)
	default:
		return fmt.Sprintf("%s://%s@%s:%d/%s", t.Dialect, t.User, t.Host, t.Port, t.Database)
	}
}

// ParseURL reads a SQLAlchemy URL (dialect+driver://user:pass@host:port/db).
func ParseURL(raw string) (Target, error) {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Scheme == "" {
		return Target{}, ErrBadURL
	}
	scheme, _, _ := strings.Cut(u.Scheme, "+")
	switch scheme {
	case "sqlite":
		return sqliteTarget(u)
	case "mysql", "mariadb":
		return serverTarget(u, MySQL, 3306)
	case "postgresql", "postgres":
		return serverTarget(u, Postgres, 5432)
	}
	return Target{}, fmt.Errorf("%w: %s", ErrUnsupported, scheme)
}

func sqliteTarget(u *url.URL) (Target, error) {
	// sqlite:///rel is relative, sqlite:////abs is absolute.
	p := strings.TrimPrefix(u.Path, "/")
	if !strings.HasPrefix(p, "/") {
		return Target{}, ErrRelativePath
	}
	return Target{Dialect: SQLite, Path: p}, nil
}

func serverTarget(u *url.URL, d Dialect, defaultPort int) (Target, error) {
	t := Target{Dialect: d, Host: u.Hostname(), Port: defaultPort, Database: strings.TrimPrefix(u.Path, "/")}
	if t.Database == "" {
		return Target{}, ErrNoDatabase
	}
	if p := u.Port(); p != "" {
		n, err := strconv.Atoi(p)
		if err != nil || n <= 0 || n > 65535 {
			return Target{}, ErrBadURL
		}
		t.Port = n
	}
	if u.User != nil {
		t.User = u.User.Username()
		if pw, ok := u.User.Password(); ok {
			t.Password = secret.NewValue(pw)
		}
	}
	if s := u.Query().Get("unix_socket"); d == MySQL && s != "" {
		t.Socket, t.Host, t.Port = s, "", 0
	}
	return t, nil
}
