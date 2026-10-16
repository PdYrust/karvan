package marzbandb

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/PdYrust/karvan/internal/panel"
	"github.com/PdYrust/karvan/internal/panel/paneldb"
	"github.com/PdYrust/karvan/internal/panel/paneldb/paneldbtest"
)

func inspect(t *testing.T, tg paneldb.Target) (Schema, error) {
	t.Helper()
	ctx := context.Background()
	r, err := paneldb.OpenReader(ctx, tg)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	var s Schema
	err = r.View(ctx, func(q paneldb.Querier) error {
		var err error
		s, err = Inspect(ctx, q, r.Server())
		return err
	})
	return s, err
}

func TestInspectFixtures(t *testing.T) {
	paneldbtest.Each(t, "testdata", Revision, func(t *testing.T, tg paneldb.Target) {
		s, err := inspect(t, tg)
		if err != nil {
			t.Fatal(err)
		}
		if s.UsersFoldCase {
			t.Error("users.username is case-sensitive at this revision on every backend")
		}
		// Admin usernames use the server default: BINARY on SQLite, a
		// case-insensitive collation on MySQL and MariaDB.
		if want := tg.Dialect == paneldb.MySQL; s.AdminsFoldCase != want {
			t.Errorf("AdminsFoldCase = %v, want %v", s.AdminsFoldCase, want)
		}
		if len(s.UnknownChildren) > 0 {
			t.Errorf("unknown children: %+v", s.UnknownChildren)
		}
	})
}

func sqliteFixture(t *testing.T, extra string) paneldb.Target {
	t.Helper()
	tg := paneldbtest.Load(t, paneldbtest.Fixture{Flavor: "sqlite", Path: "testdata/" + Revision + ".sqlite.sql"})
	db, err := sql.Open("sqlite", tg.Path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec(extra); err != nil {
		t.Fatal(err)
	}
	return tg
}

func TestInspectRefusesOtherRevisions(t *testing.T) {
	_, err := inspect(t, sqliteFixture(t, "UPDATE alembic_version SET version_num = 'e56f1c781e46'"))
	var ve *panel.VersionError
	if !errors.As(err, &ve) || ve.Revision != "e56f1c781e46" {
		t.Fatalf("err = %v", err)
	}
}

func TestInspectFindsUnknownChildren(t *testing.T) {
	s, err := inspect(t, sqliteFixture(t, "CREATE TABLE user_extras (id INTEGER PRIMARY KEY, user_id INTEGER REFERENCES users (id))"))
	if err != nil {
		t.Fatal(err)
	}
	if len(s.UnknownChildren) != 1 || s.UnknownChildren[0].Table != "user_extras" {
		t.Fatalf("UnknownChildren = %+v", s.UnknownChildren)
	}
}

func TestInspectRefusesAMissingColumn(t *testing.T) {
	_, err := inspect(t, sqliteFixture(t, "ALTER TABLE users DROP COLUMN note"))
	var se *panel.SchemaError
	if !errors.As(err, &se) {
		t.Fatalf("err = %v", err)
	}
}
