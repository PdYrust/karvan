package paneldb

import (
	"errors"
	"testing"
)

func TestParseURL(t *testing.T) {
	cases := []struct {
		in   string
		want Target
	}{
		{"sqlite:////var/lib/marzban/db.sqlite3", Target{Dialect: SQLite, Path: "/var/lib/marzban/db.sqlite3"}},
		{"sqlite+aiosqlite:////var/lib/pasarguard/db.sqlite3", Target{Dialect: SQLite, Path: "/var/lib/pasarguard/db.sqlite3"}},
		{"mysql+pymysql://marzban:p%40ss@127.0.0.1:3306/marzban", Target{Dialect: MySQL, Host: "127.0.0.1", Port: 3306, User: "marzban", Database: "marzban"}},
		{"mysql+asyncmy://u:pw@db/pasarguard?charset=utf8mb4", Target{Dialect: MySQL, Host: "db", Port: 3306, User: "u", Database: "pasarguard"}},
		{"mariadb+asyncmy://u:pw@127.0.0.1:3307/p", Target{Dialect: MySQL, Host: "127.0.0.1", Port: 3307, User: "u", Database: "p"}},
		{"mysql+pymysql://u:pw@localhost/m?unix_socket=/run/mysqld/mysqld.sock", Target{Dialect: MySQL, Socket: "/run/mysqld/mysqld.sock", User: "u", Database: "m"}},
		{"postgresql+asyncpg://pg:pw@127.0.0.1:6432/pasarguard", Target{Dialect: Postgres, Host: "127.0.0.1", Port: 6432, User: "pg", Database: "pasarguard"}},
		{"postgresql://pg@localhost/p", Target{Dialect: Postgres, Host: "localhost", Port: 5432, User: "pg", Database: "p"}},
	}
	for _, c := range cases {
		got, err := ParseURL(c.in)
		if err != nil {
			t.Errorf("ParseURL(%q): %v", c.in, err)
			continue
		}
		pw := got.Password
		got.Password = c.want.Password
		if got != c.want {
			t.Errorf("ParseURL(%q) = %+v, want %+v", c.in, got, c.want)
		}
		if c.in == cases[2].in && pw.Reveal() != "p@ss" {
			t.Errorf("password not decoded: %q", pw.Reveal())
		}
	}
}

func TestParseURLRefusesWhatCannotBeFound(t *testing.T) {
	cases := map[string]error{
		"sqlite:///db.sqlite3":        ErrRelativePath,
		"sqlite://":                   ErrRelativePath,
		"mssql://u:p@h/db":            ErrUnsupported,
		"postgresql+asyncpg://u:p@h/": ErrNoDatabase,
		"::not a url":                 ErrBadURL,
	}
	for in, want := range cases {
		if _, err := ParseURL(in); !errors.Is(err, want) {
			t.Errorf("ParseURL(%q) = %v, want %v", in, err, want)
		}
	}
}

func TestTargetNeverPrintsThePassword(t *testing.T) {
	tg, err := ParseURL("mysql+pymysql://u:hunter2@h/db")
	if err != nil {
		t.Fatal(err)
	}
	if s := tg.String(); s != "mysql://u@h:3306/db" {
		t.Errorf("String() = %q", s)
	}
}
