package paneldb

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"slices"
	"strings"
)

// ReadSpec reads the live shape of tables: every column (PRAGMA on
// SQLite, information_schema elsewhere), the unique indexes other than
// primary keys, and the foreign keys between the tables. A table that
// does not exist has no columns.
func ReadSpec(ctx context.Context, q Querier, d Dialect, tables []string) (Spec, error) {
	s := Spec{Columns: map[string]map[string]Column{}, Unique: []string{}, Foreign: []string{}}
	for _, t := range tables {
		cols, err := readColumns(ctx, q, d, t)
		if err != nil {
			return s, fmt.Errorf("read columns of %s: %w", t, err)
		}
		s.Columns[t] = cols
		uniq, err := readUnique(ctx, q, d, t)
		if err != nil {
			return s, fmt.Errorf("read unique indexes of %s: %w", t, err)
		}
		s.Unique = append(s.Unique, uniq...)
	}
	fks, err := ForeignKeysTo(ctx, q, d, tables...)
	if err != nil {
		return s, err
	}
	for _, fk := range fks {
		if slices.Contains(tables, fk.Table) {
			s.Foreign = append(s.Foreign, foreignName(fk))
		}
	}
	slices.Sort(s.Unique)
	slices.Sort(s.Foreign)
	s.Foreign = slices.Compact(s.Foreign)
	return s, nil
}

// CheckShape compares a database at a revision Karvan does not know with
// known, the shape of a known revision per server flavor, and lists what
// differs (see Diff). inserts are the tables the writer inserts into.
func CheckShape(ctx context.Context, q Querier, srv Server, known map[string]Spec, inserts []string) ([]string, error) {
	k, ok := known[srv.Flavor]
	if !ok {
		return []string{"no known schema on " + srv.Flavor}, nil
	}
	live, err := ReadSpec(ctx, q, srv.Dialect, slices.Sorted(maps.Keys(k.Columns)))
	if err != nil {
		return nil, err
	}
	return Diff(k, live, inserts), nil
}

// ParseSpecs reads a spec file: a JSON object of server flavor to Spec.
func ParseSpecs(b []byte) (map[string]Spec, error) {
	var out map[string]Spec
	return out, json.Unmarshal(b, &out)
}

func readColumns(ctx context.Context, q Querier, d Dialect, table string) (map[string]Column, error) {
	switch d {
	case SQLite:
		return sqliteColumns(ctx, q, table)
	case MySQL:
		return scanColumns(ctx, q, mysqlType, "SELECT COLUMN_NAME, DATA_TYPE, COLUMN_TYPE, COALESCE(CHARACTER_MAXIMUM_LENGTH, 0), "+
			"IS_NULLABLE = 'YES', (COLUMN_DEFAULT IS NOT NULL AND COLUMN_DEFAULT <> 'NULL') OR EXTRA LIKE '%auto_increment%' "+
			"FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = ?", table)
	default:
		return scanColumns(ctx, q, postgresType, "SELECT c.column_name, c.data_type, "+
			"COALESCE((SELECT string_agg(e.enumlabel, ',' ORDER BY e.enumsortorder) FROM pg_enum e "+
			"JOIN pg_type t ON t.oid = e.enumtypid WHERE t.typname = c.udt_name), ''), "+
			"COALESCE(c.character_maximum_length, 0), c.is_nullable = 'YES', c.column_default IS NOT NULL OR c.is_identity = 'YES' "+
			"FROM information_schema.columns c WHERE c.table_schema = current_schema() AND c.table_name = ?", table)
	}
}

// scanColumns reads rows of name, data type, the type's detail, length,
// nullable and has-default, and names each type with norm.
func scanColumns(ctx context.Context, q Querier, norm func(typ, detail string, n int64) string, query, table string) (map[string]Column, error) {
	rows, err := q.QueryContext(ctx, query, table)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]Column{}
	for rows.Next() {
		var name, typ, detail string
		var n int64
		var c Column
		if err := rows.Scan(&name, &typ, &detail, &n, &c.Null, &c.Default); err != nil {
			return nil, err
		}
		c.Type = norm(strings.ToLower(typ), strings.ToLower(detail), n)
		out[name] = c
	}
	return out, rows.Err()
}

func mysqlType(typ, columnType string, n int64) string {
	switch {
	case typ == "varchar" || typ == "char":
		return fmt.Sprintf("%s(%d)", typ, n)
	case typ == "tinyint" && strings.HasPrefix(columnType, "tinyint(1)"):
		return "boolean"
	case typ == "enum":
		v := strings.TrimSuffix(strings.TrimPrefix(columnType, "enum("), ")")
		return "enum(" + strings.ReplaceAll(v, "'", "") + ")"
	}
	return typ
}

func postgresType(typ, labels string, n int64) string {
	switch {
	case typ == "character varying" && n > 0:
		return fmt.Sprintf("varchar(%d)", n)
	case typ == "character varying":
		return "varchar"
	case typ == "character":
		return fmt.Sprintf("char(%d)", n)
	case typ == "user-defined" && labels != "":
		return "enum(" + labels + ")"
	case typ == "timestamp without time zone":
		return "timestamp"
	case typ == "timestamp with time zone":
		return "timestamptz"
	}
	return typ
}

// sqliteColumns reads PRAGMA table_info. A lone INTEGER primary key is
// the rowid, so it fills itself.
func sqliteColumns(ctx context.Context, q Querier, table string) (map[string]Column, error) {
	rows, err := q.QueryContext(ctx, `SELECT name, type, "notnull", dflt_value IS NOT NULL, pk FROM pragma_table_info(?)`, table)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]Column{}
	var rowid string
	keys := 0
	for rows.Next() {
		var name, typ string
		var notNull, def bool
		var pk int
		if err := rows.Scan(&name, &typ, &notNull, &def, &pk); err != nil {
			return nil, err
		}
		typ = strings.ToLower(strings.ReplaceAll(typ, " ", ""))
		if pk > 0 {
			keys++
			if typ == "integer" {
				rowid = name
			}
		}
		out[name] = Column{Type: typ, Null: !notNull && pk == 0, Default: def}
	}
	if keys == 1 && rowid != "" {
		c := out[rowid]
		c.Default = true
		out[rowid] = c
	}
	return out, rows.Err()
}

func readUnique(ctx context.Context, q Querier, d Dialect, table string) ([]string, error) {
	query := map[Dialect]string{
		SQLite: "SELECT l.name, i.name FROM pragma_index_list(?) l JOIN pragma_index_info(l.name) i " +
			"WHERE l.\"unique\" = 1 AND l.origin <> 'pk' ORDER BY l.name, i.seqno",
		MySQL: "SELECT INDEX_NAME, COLUMN_NAME FROM information_schema.STATISTICS WHERE TABLE_SCHEMA = DATABASE() " +
			"AND TABLE_NAME = ? AND NON_UNIQUE = 0 AND INDEX_NAME <> 'PRIMARY' ORDER BY INDEX_NAME, SEQ_IN_INDEX",
		Postgres: "SELECT ic.relname, a.attname FROM pg_index x JOIN pg_class tc ON tc.oid = x.indrelid " +
			"JOIN pg_class ic ON ic.oid = x.indexrelid JOIN pg_namespace n ON n.oid = tc.relnamespace " +
			"JOIN LATERAL unnest(x.indkey::int2[]) WITH ORDINALITY k(attnum, pos) ON true " +
			"JOIN pg_attribute a ON a.attrelid = tc.oid AND a.attnum = k.attnum " +
			"WHERE x.indisunique AND NOT x.indisprimary AND n.nspname = current_schema() AND tc.relname = ? ORDER BY 1, k.pos",
	}[d]
	rows, err := q.QueryContext(ctx, query, table)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var names []string
	cols := map[string][]string{}
	for rows.Next() {
		var index, col string
		if err := rows.Scan(&index, &col); err != nil {
			return nil, err
		}
		if cols[index] == nil {
			names = append(names, index)
		}
		cols[index] = append(cols[index], col)
	}
	out := make([]string, 0, len(names))
	for _, n := range names {
		out = append(out, uniqueName(table, cols[n]))
	}
	return out, rows.Err()
}
