package paneldb

import (
	"context"
	"fmt"
	"regexp"
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
		c = strings.ToLower(c)
		return !(c == "" || c == "binary" || strings.HasSuffix(c, "_bin") || strings.HasSuffix(c, "_cs")), nil
	}
	return false, nil
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
		rows, err := q.QueryContext(ctx, `SELECT "from", "table", on_delete FROM pragma_foreign_key_list(?) ORDER BY "from"`, t)
		if err != nil {
			return nil, fmt.Errorf("read foreign keys of %s: %w", t, err)
		}
		for rows.Next() {
			fk := ForeignKey{Table: t}
			if err := rows.Scan(&fk.Column, &fk.RefTable, &fk.OnDelete); err != nil {
				rows.Close()
				return nil, err
			}
			if want[fk.RefTable] {
				out = append(out, fk)
			}
		}
		err = rows.Err()
		if cerr := rows.Close(); err == nil {
			err = cerr
		}
		if err != nil {
			return nil, err
		}
	}
	return out, nil
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
