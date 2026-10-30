package paneldb

import (
	"fmt"
	"maps"
	"slices"
	"strconv"
	"strings"
)

// Spec is the shape of the tables a writer uses: each column's type,
// nullability and default, and the unique indexes and foreign keys among
// those columns. A known revision's Spec is read from its fixture; a
// database at a revision Karvan does not know is compared against it.
type Spec struct {
	Columns map[string]map[string]Column `json:"columns"` // table, then column
	Unique  []string                     `json:"unique"`  // "users(username)"
	Foreign []string                     `json:"foreign"` // "proxies.user_id -> users"
}

// Column is one column's shape. Type is normalized per dialect:
// "varchar(34)", "bigint", "enum(a,b)", "timestamp".
type Column struct {
	Type    string
	Null    bool
	Default bool // a server default, an auto increment or an identity
}

// MarshalText writes c as "varchar(34) null default", one line per column
// in a spec file.
func (c Column) MarshalText() ([]byte, error) {
	s := c.Type
	if c.Null {
		s += " null"
	}
	if c.Default {
		s += " default"
	}
	return []byte(s), nil
}

// UnmarshalText reads what MarshalText writes.
func (c *Column) UnmarshalText(b []byte) error {
	s := string(b)
	s, c.Default = strings.CutSuffix(s, " default")
	s, c.Null = strings.CutSuffix(s, " null")
	c.Type = s
	return nil
}

// Only keeps the columns in want, and the unique indexes and foreign keys
// made of them only.
func (s Spec) Only(want map[string][]string) Spec {
	out := Spec{Columns: map[string]map[string]Column{}, Unique: []string{}, Foreign: []string{}}
	for t, cols := range want {
		out.Columns[t] = map[string]Column{}
		for _, c := range cols {
			if col, ok := s.Columns[t][c]; ok {
				out.Columns[t][c] = col
			}
		}
	}
	for _, u := range s.Unique {
		t, cols := splitUnique(u)
		if !slices.ContainsFunc(cols, func(c string) bool { return !slices.Contains(want[t], c) }) {
			out.Unique = append(out.Unique, u)
		}
	}
	for _, f := range s.Foreign {
		t, c, _ := splitForeign(f)
		if slices.Contains(want[t], c) {
			out.Foreign = append(out.Foreign, f)
		}
	}
	return out
}

// Diff lists how live differs from the known Spec k in ways the writer
// cannot take: a missing table, column, unique index or foreign key; a
// column whose type narrowed or changed; a nullability change; a default
// the writer relied on that is gone. On the tables in inserts, a new
// column that is not null and has no default is a difference too, since
// an insert that leaves it out would fail. Extra tables, extra nullable
// columns and extra columns with a default are not.
func Diff(k, live Spec, inserts []string) []string {
	var out []string
	for _, t := range slices.Sorted(maps.Keys(k.Columns)) {
		have := live.Columns[t]
		if len(have) == 0 {
			out = append(out, "table "+t+" missing")
			continue
		}
		for _, c := range slices.Sorted(maps.Keys(k.Columns[t])) {
			col, ok := have[c]
			out = append(out, columnDiff(t+"."+c, k.Columns[t][c], col, ok)...)
		}
		if slices.Contains(inserts, t) {
			for _, c := range slices.Sorted(maps.Keys(have)) {
				if _, known := k.Columns[t][c]; !known && !have[c].Null && !have[c].Default {
					out = append(out, t+"."+c+" is new, not null and has no default")
				}
			}
		}
	}
	for _, u := range k.Unique {
		if !slices.Contains(live.Unique, u) {
			out = append(out, "unique "+u+" missing")
		}
	}
	for _, f := range k.Foreign {
		if !slices.Contains(live.Foreign, f) {
			out = append(out, "foreign key "+f+" missing")
		}
	}
	return out
}

func columnDiff(name string, k, live Column, found bool) []string {
	if !found {
		return []string{name + " missing"}
	}
	var out []string
	if !compatibleType(k.Type, live.Type) {
		out = append(out, fmt.Sprintf("%s type %s, now %s", name, k.Type, live.Type))
	}
	switch {
	case k.Null && !live.Null:
		out = append(out, name+" nullable, now not null")
	case !k.Null && live.Null:
		out = append(out, name+" not null, now nullable")
	}
	if k.Default && !live.Default {
		out = append(out, name+" default dropped")
	}
	return out
}

// intRank orders the integer types by width.
var intRank = map[string]int{"tinyint": 1, "smallint": 2, "mediumint": 3, "int": 4, "integer": 4, "bigint": 5}

// compatibleType reports whether a column of type live takes every value
// a column of type k took: the same type, a longer string, a wider
// integer, or an enum with every old value.
func compatibleType(k, live string) bool {
	if k == live {
		return true
	}
	kb, kn := splitType(k)
	lb, ln := splitType(live)
	switch {
	case intRank[kb] > 0 && intRank[lb] > 0 && kn == "" && ln == "":
		return intRank[lb] >= intRank[kb]
	case kb == "enum" && lb == "enum":
		have := strings.Split(ln, ",")
		return !slices.ContainsFunc(strings.Split(kn, ","), func(v string) bool { return !slices.Contains(have, v) })
	case kb == "varchar" || kb == "char":
		return stringFits(kb, kn, lb, ln)
	}
	return false
}

// stringFits reports whether a string column of type lb(ln) holds every
// value of kb(kn): text, an unbounded varchar or the same type, as long.
func stringFits(kb, kn, lb, ln string) bool {
	if lb == "text" || lb == "varchar" && ln == "" {
		return true
	}
	kl, err1 := strconv.Atoi(kn)
	ll, err2 := strconv.Atoi(ln)
	return lb == kb && err1 == nil && err2 == nil && ll >= kl
}

// splitType splits "varchar(34)" into "varchar" and "34".
func splitType(t string) (base, arg string) {
	base, arg, ok := strings.Cut(t, "(")
	if !ok {
		return t, ""
	}
	return base, strings.TrimSuffix(arg, ")")
}

func uniqueName(table string, cols []string) string {
	return table + "(" + strings.Join(cols, ",") + ")"
}

func splitUnique(u string) (string, []string) {
	t, cols := splitType(u)
	return t, strings.Split(cols, ",")
}

func foreignName(fk ForeignKey) string { return fk.Table + "." + fk.Column + " -> " + fk.RefTable }

func splitForeign(f string) (table, column, ref string) {
	tc, ref, _ := strings.Cut(f, " -> ")
	table, column, _ = strings.Cut(tc, ".")
	return table, column, ref
}
