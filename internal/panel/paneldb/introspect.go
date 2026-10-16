package paneldb

import (
	"context"
	"fmt"
	"regexp"
	"slices"
	"strings"
)

// Columns lists the columns of table.
func Columns(ctx context.Context, q Querier, d Dialect, table string) ([]string, error) {
	query := map[Dialect]string{
		SQLite:   "SELECT name FROM pragma_table_info(?) ORDER BY cid",
		MySQL:    "SELECT COLUMN_NAME FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = ? ORDER BY ORDINAL_POSITION",
		Postgres: "SELECT column_name FROM information_schema.columns WHERE table_schema = current_schema() AND table_name = ? ORDER BY ordinal_position",
	}[d]
	return queryStrings(ctx, q, query, table)
}

// MissingColumns compares the live schema with want (table to columns)
// and describes the first gap, or returns "" when every column is there.
func MissingColumns(ctx context.Context, q Querier, d Dialect, want map[string][]string) (string, error) {
	tables := make([]string, 0, len(want))
	for t := range want {
		tables = append(tables, t)
	}
	slices.Sort(tables)
	for _, t := range tables {
		have, err := Columns(ctx, q, d, t)
		if err != nil {
			return "", fmt.Errorf("read columns of %s: %w", t, err)
		}
		var missing []string
		for _, c := range want[t] {
			if !slices.Contains(have, c) {
				missing = append(missing, c)
			}
		}
		if len(missing) > 0 {
			return t + " lacks " + strings.Join(missing, ", "), nil
		}
	}
	return "", nil
}

// UnknownKeys returns the keys in fks that are not in known (matched on
// table, column and referenced table).
func UnknownKeys(fks, known []ForeignKey) []ForeignKey {
	var out []ForeignKey
	for _, fk := range fks {
		found := slices.ContainsFunc(known, func(k ForeignKey) bool {
			return k.Table == fk.Table && k.Column == fk.Column && k.RefTable == fk.RefTable
		})
		if !found {
			out = append(out, fk)
		}
	}
	return out
}

// CaseInsensitive reports whether table.column compares text without case,
// read from the live schema: SQLite DDL (COLLATE NOCASE), the MySQL column
// collation (anything but _bin, _cs or binary folds case), PostgreSQL
// (deterministic collations never fold).
func CaseInsensitive(ctx context.Context, q Querier, d Dialect, table, column string) (bool, error) {
	switch d {
	case SQLite:
		var ddl string
		err := q.QueryRowContext(ctx, "SELECT sql FROM sqlite_master WHERE type = 'table' AND name = ?", table).Scan(&ddl)
		if err != nil {
			return false, fmt.Errorf("read %s DDL: %w", table, err)
		}
		return sqliteColumnNoCase(ddl, column), nil
	case MySQL:
		var c string
		err := q.QueryRowContext(ctx, "SELECT COALESCE(COLLATION_NAME, '') FROM information_schema.COLUMNS "+
			"WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = ? AND COLUMN_NAME = ?", table, column).Scan(&c)
		if err != nil {
			return false, fmt.Errorf("read %s.%s collation: %w", table, column, err)
		}
		return foldsCase(strings.ToLower(c)), nil
	}
	return false, nil
}

// foldsCase reports whether a MySQL or MariaDB collation ignores case.
func foldsCase(c string) bool {
	sensitive := c == "" || c == "binary" || strings.HasSuffix(c, "_bin") || strings.HasSuffix(c, "_cs")
	return !sensitive
}

func sqliteColumnNoCase(ddl, column string) bool {
	re := regexp.MustCompile(`(?is)["\x60]?\b` + regexp.QuoteMeta(column) + `\b["\x60]?\s+[^,]*?\bCOLLATE\s+["']?NOCASE`)
	return re.MatchString(ddl)
}

// ForeignKey is one column that references another table.
type ForeignKey struct {
	Table, Column, RefTable string
	OnDelete                string // CASCADE, SET NULL, NO ACTION, RESTRICT...
}

// ForeignKeysTo lists every foreign key that points at one of refs.
func ForeignKeysTo(ctx context.Context, q Querier, d Dialect, refs ...string) ([]ForeignKey, error) {
	if d == SQLite {
		return sqliteForeignKeys(ctx, q, refs)
	}
	in := strings.TrimSuffix(strings.Repeat("?, ", len(refs)), ", ")
	query := map[Dialect]string{
		MySQL: "SELECT k.TABLE_NAME, k.COLUMN_NAME, k.REFERENCED_TABLE_NAME, r.DELETE_RULE " +
			"FROM information_schema.KEY_COLUMN_USAGE k JOIN information_schema.REFERENTIAL_CONSTRAINTS r " +
			"ON r.CONSTRAINT_SCHEMA = k.CONSTRAINT_SCHEMA AND r.CONSTRAINT_NAME = k.CONSTRAINT_NAME " +
			"WHERE k.TABLE_SCHEMA = DATABASE() AND k.REFERENCED_TABLE_NAME IN (" + in + ") ORDER BY 1, 2",
		Postgres: "SELECT cl.relname, a.attname, rf.relname, CASE c.confdeltype WHEN 'c' THEN 'CASCADE' " +
			"WHEN 'n' THEN 'SET NULL' WHEN 'd' THEN 'SET DEFAULT' WHEN 'r' THEN 'RESTRICT' ELSE 'NO ACTION' END " +
			"FROM pg_constraint c JOIN pg_class cl ON cl.oid = c.conrelid JOIN pg_class rf ON rf.oid = c.confrelid " +
			"JOIN pg_namespace n ON n.oid = cl.relnamespace " +
			"JOIN pg_attribute a ON a.attrelid = c.conrelid AND a.attnum = c.conkey[1] " +
			"WHERE c.contype = 'f' AND n.nspname = current_schema() AND rf.relname IN (" + in + ") ORDER BY 1, 2",
	}[d]
	args := make([]any, len(refs))
	for i, r := range refs {
		args[i] = r
	}
	rows, err := q.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("read foreign keys: %w", err)
	}
	defer rows.Close()
	var out []ForeignKey
	for rows.Next() {
		var fk ForeignKey
		if err := rows.Scan(&fk.Table, &fk.Column, &fk.RefTable, &fk.OnDelete); err != nil {
			return nil, err
		}
		out = append(out, fk)
	}
	return out, rows.Err()
}

func sqliteForeignKeys(ctx context.Context, q Querier, refs []string) ([]ForeignKey, error) {
	tables, err := queryStrings(ctx, q, "SELECT name FROM sqlite_master WHERE type = 'table' AND name NOT LIKE 'sqlite_%' ORDER BY name")
	if err != nil {
		return nil, err
	}
	want := map[string]bool{}
	for _, r := range refs {
		want[r] = true
	}
	var out []ForeignKey
	for _, t := range tables {
		fks, err := sqliteTableForeignKeys(ctx, q, t)
		if err != nil {
			return nil, err
		}
		for _, fk := range fks {
			if want[fk.RefTable] {
				out = append(out, fk)
			}
		}
	}
	return out, nil
}

func sqliteTableForeignKeys(ctx context.Context, q Querier, table string) ([]ForeignKey, error) {
	rows, err := q.QueryContext(ctx, `SELECT "from", "table", on_delete FROM pragma_foreign_key_list(?) ORDER BY "from"`, table)
	if err != nil {
		return nil, fmt.Errorf("read foreign keys of %s: %w", table, err)
	}
	defer rows.Close()
	var out []ForeignKey
	for rows.Next() {
		fk := ForeignKey{Table: table}
		if err := rows.Scan(&fk.Column, &fk.RefTable, &fk.OnDelete); err != nil {
			return nil, err
		}
		out = append(out, fk)
	}
	return out, rows.Err()
}

func queryStrings(ctx context.Context, q Querier, query string, args ...any) ([]string, error) {
	rows, err := q.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}
