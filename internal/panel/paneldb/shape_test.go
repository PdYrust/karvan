package paneldb

import (
	"slices"
	"testing"
)

func knownSpec() Spec {
	return Spec{
		Columns: map[string]map[string]Column{
			"users": {
				"id":       {Type: "integer", Default: true},
				"username": {Type: "varchar(34)"},
				"status":   {Type: "enum(active,disabled)", Default: true},
				"traffic":  {Type: "integer"},
				"note":     {Type: "varchar(500)", Null: true},
			},
			"admins": {"id": {Type: "integer", Default: true}, "username": {Type: "varchar(34)"}},
		},
		Unique:  []string{"users(username)"},
		Foreign: []string{"users.admin_id -> admins"},
	}
}

// liveSpec is knownSpec as the database reports it: every column of
// each table, not only the listed ones.
func liveSpec() Spec {
	s := knownSpec()
	s.Columns["users"]["admin_id"] = Column{Type: "integer", Null: true}
	s.Columns["admins"]["created_at"] = Column{Type: "datetime"}
	return s
}

func diff(t *testing.T, live Spec) []string {
	t.Helper()
	return Diff(knownSpec(), live, []string{"users"})
}

func TestDiffSameShape(t *testing.T) {
	if d := diff(t, liveSpec()); d != nil {
		t.Fatalf("diff = %q", d)
	}
}

func TestDiffMissingColumn(t *testing.T) {
	live := liveSpec()
	delete(live.Columns["users"], "note")
	if d := diff(t, live); !slices.Equal(d, []string{"users.note missing"}) {
		t.Fatalf("diff = %q", d)
	}
}

func TestDiffMissingTable(t *testing.T) {
	live := liveSpec()
	delete(live.Columns, "admins")
	if d := diff(t, live); !slices.Equal(d, []string{"table admins missing"}) {
		t.Fatalf("diff = %q", d)
	}
}

func TestDiffTypeChange(t *testing.T) {
	live := liveSpec()
	live.Columns["users"]["username"] = Column{Type: "integer"}
	live.Columns["users"]["traffic"] = Column{Type: "smallint"}
	want := []string{"users.traffic type integer, now smallint", "users.username type varchar(34), now integer"}
	if d := diff(t, live); !slices.Equal(d, want) {
		t.Fatalf("diff = %q", d)
	}
}

func TestDiffAcceptsWiderTypes(t *testing.T) {
	live := liveSpec()
	live.Columns["users"]["username"] = Column{Type: "varchar(64)"}
	live.Columns["users"]["traffic"] = Column{Type: "bigint"}
	live.Columns["users"]["note"] = Column{Type: "text", Null: true}
	live.Columns["users"]["status"] = Column{Type: "enum(active,disabled,limited)", Default: true}
	if d := diff(t, live); d != nil {
		t.Fatalf("diff = %q", d)
	}
}

func TestDiffNarrowerTypes(t *testing.T) {
	live := liveSpec()
	live.Columns["users"]["username"] = Column{Type: "varchar(16)"}
	live.Columns["users"]["status"] = Column{Type: "enum(active)", Default: true}
	want := []string{"users.status type enum(active,disabled), now enum(active)", "users.username type varchar(34), now varchar(16)"}
	if d := diff(t, live); !slices.Equal(d, want) {
		t.Fatalf("diff = %q", d)
	}
}

func TestDiffNullabilityAndDefault(t *testing.T) {
	live := liveSpec()
	live.Columns["users"]["note"] = Column{Type: "varchar(500)"}
	live.Columns["users"]["traffic"] = Column{Type: "integer", Null: true}
	live.Columns["users"]["status"] = Column{Type: "enum(active,disabled)"}
	want := []string{"users.note nullable, now not null", "users.status default dropped", "users.traffic not null, now nullable"}
	if d := diff(t, live); !slices.Equal(d, want) {
		t.Fatalf("diff = %q", d)
	}
}

func TestDiffNewNotNullColumnWithoutDefault(t *testing.T) {
	live := liveSpec()
	live.Columns["users"]["group_id"] = Column{Type: "integer"}
	if d := diff(t, live); !slices.Equal(d, []string{"users.group_id is new, not null and has no default"}) {
		t.Fatalf("diff = %q", d)
	}
}

// A new column the insert can leave out is fine, and so is a not null
// one on a table Karvan only reads.
func TestDiffAcceptsNewNullableColumns(t *testing.T) {
	live := liveSpec()
	live.Columns["users"]["tag"] = Column{Type: "varchar(8)", Null: true}
	live.Columns["users"]["kind"] = Column{Type: "integer", Default: true}
	live.Columns["admins"]["role"] = Column{Type: "integer"}
	if d := diff(t, live); d != nil {
		t.Fatalf("diff = %q", d)
	}
}

func TestDiffIgnoresNewTables(t *testing.T) {
	live := liveSpec()
	live.Columns["user_extras"] = map[string]Column{"id": {Type: "integer"}, "user_id": {Type: "integer"}}
	live.Unique = append(live.Unique, "user_extras(user_id)")
	live.Foreign = append(live.Foreign, "user_extras.user_id -> users")
	if d := diff(t, live); d != nil {
		t.Fatalf("diff = %q", d)
	}
}

func TestDiffMissingConstraints(t *testing.T) {
	live := liveSpec()
	live.Unique, live.Foreign = nil, nil
	want := []string{"unique users(username) missing", "foreign key users.admin_id -> admins missing"}
	if d := diff(t, live); !slices.Equal(d, want) {
		t.Fatalf("diff = %q", d)
	}
}

func TestOnlyKeepsListedColumns(t *testing.T) {
	live := liveSpec()
	live.Unique = append(live.Unique, "admins(created_at)")
	got := live.Only(map[string][]string{"users": {"id", "username"}, "admins": {"id", "username"}})
	if len(got.Columns["users"]) != 2 || len(got.Columns["admins"]) != 2 {
		t.Errorf("columns = %+v", got.Columns)
	}
	if !slices.Equal(got.Unique, []string{"users(username)"}) || len(got.Foreign) != 0 {
		t.Errorf("unique = %q, foreign = %q", got.Unique, got.Foreign)
	}
}
