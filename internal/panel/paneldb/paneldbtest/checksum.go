package paneldbtest

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/PdYrust/karvan/internal/panel/paneldb"
)

// Checksum hashes every row of every table in tg, in a stable order, so a
// test can prove a dry run changed nothing.
func Checksum(t *testing.T, tg paneldb.Target) string {
	t.Helper()
	ctx := context.Background()
	r, err := paneldb.OpenReader(ctx, tg)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	h := sha256.New()
	err = r.View(ctx, func(q paneldb.Querier) error {
		tables, err := tableNames(ctx, q, tg.Dialect)
		if err != nil {
			return err
		}
		for _, name := range tables {
			rows, err := dumpTable(ctx, q, tg.Dialect, name)
			if err != nil {
				return err
			}
			fmt.Fprintf(h, "table %s %d\n", name, len(rows))
			for _, row := range rows {
				fmt.Fprintln(h, row)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return hex.EncodeToString(h.Sum(nil))
}

func tableNames(ctx context.Context, q paneldb.Querier, d paneldb.Dialect) ([]string, error) {
	query := map[paneldb.Dialect]string{
		paneldb.SQLite:   "SELECT name FROM sqlite_master WHERE type = 'table' AND name NOT LIKE 'sqlite_%' ORDER BY name",
		paneldb.MySQL:    "SELECT TABLE_NAME FROM information_schema.TABLES WHERE TABLE_SCHEMA = DATABASE() ORDER BY 1",
		paneldb.Postgres: "SELECT table_name FROM information_schema.tables WHERE table_schema = current_schema() ORDER BY 1",
	}[d]
	rows, err := q.QueryContext(ctx, query)
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

func dumpTable(ctx context.Context, q paneldb.Querier, d paneldb.Dialect, table string) ([]string, error) {
	rows, err := q.QueryContext(ctx, "SELECT * FROM "+paneldb.Ident(d, table))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	cols, err := rows.Columns()
	if err != nil {
		return nil, err
	}
	var out []string
	for rows.Next() {
		vals := make([]sql.NullString, len(cols))
		ptrs := make([]any, len(cols))
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return nil, err
		}
		parts := make([]string, len(vals))
		for i, v := range vals {
			parts[i] = fmt.Sprintf("%v:%q", v.Valid, v.String)
		}
		out = append(out, strings.Join(parts, ","))
	}
	slices.Sort(out)
	return out, rows.Err()
}
