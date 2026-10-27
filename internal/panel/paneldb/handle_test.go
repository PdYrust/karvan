package paneldb

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"
)

func newSQLite(t *testing.T) Target {
	t.Helper()
	p := filepath.Join(t.TempDir(), "panel.sqlite3")
	db, err := sql.Open("sqlite", p)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	for _, q := range []string{
		"PRAGMA journal_mode=WAL",
		"CREATE TABLE admins (id INTEGER PRIMARY KEY, username TEXT UNIQUE)",
		"CREATE TABLE users (id INTEGER PRIMARY KEY, admin_id INTEGER REFERENCES admins (id))",
		"INSERT INTO admins (username) VALUES ('a')",
	} {
		if _, err := db.Exec(q); err != nil {
			t.Fatal(err)
		}
	}
	return Target{Dialect: SQLite, Path: p}
}

func TestReaderCannotWriteEvenFromInside(t *testing.T) {
	ctx := context.Background()
	r, err := OpenReader(ctx, newSQLite(t))
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	if r.Server().Flavor != "sqlite" || r.Server().Version == "" {
		t.Fatalf("server = %+v", r.Server())
	}
	if _, err := r.db.ExecContext(ctx, "INSERT INTO admins (username) VALUES ('b')"); err == nil {
		t.Fatal("a read-only pool accepted a write")
	}
	err = r.View(ctx, func(q Querier) error {
		if _, ok := q.(Execer); ok {
			t.Error("View handed out something that can write")
		}
		var n int
		return q.QueryRowContext(ctx, "SELECT count(*) FROM admins WHERE username = ?", "a").Scan(&n)
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestWriterChunkCommitsOrRollsBack(t *testing.T) {
	ctx := context.Background()
	tg := newSQLite(t)
	w, err := OpenWriter(ctx, tg)
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	err = w.Chunk(ctx, func(x Execer) error {
		_, err := x.ExecContext(ctx, "INSERT INTO users (admin_id) VALUES (?)", 1)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	err = w.Chunk(ctx, func(x Execer) error {
		if _, err := x.ExecContext(ctx, "INSERT INTO users (admin_id) VALUES (?)", 1); err != nil {
			return err
		}
		_, err := x.ExecContext(ctx, "INSERT INTO users (admin_id) VALUES (?)", 99)
		return err
	})
	if err == nil {
		t.Fatal("foreign keys are not enforced on the writer")
	}
	var n int
	if err := w.db.QueryRowContext(ctx, "SELECT count(*) FROM users").Scan(&n); err != nil || n != 1 {
		t.Fatalf("users = %d, %v; the failed chunk must leave nothing", n, err)
	}
}

func TestRebind(t *testing.T) {
	if got := Rebind("SELECT a FROM t WHERE b = ? AND c IN (?, ?)"); got != "SELECT a FROM t WHERE b = $1 AND c IN ($2, $3)" {
		t.Errorf("Rebind = %q", got)
	}
}
