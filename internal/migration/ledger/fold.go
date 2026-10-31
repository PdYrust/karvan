package ledger

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
)

// liveStates are the states the ledger_live index covers.
const liveStates = "('pending', 'done', 'adopted', 'unknown')"

// FoldOrigins runs at start: an old panel name stored before Origin
// folded every way of typing an address is rewritten to today's name, in
// one transaction, on jobs, ledger rows, batches and the rules made from
// batches, whose address is that name. A name whose live ledger rows
// would clash with live rows already under the new name keeps its
// spelling; kept lists those names. Names Origin already gives stay as
// they are, so a second run changes nothing.
func FoldOrigins(ctx context.Context, db *sql.DB) (kept []string, err error) {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback() //nolint:errcheck // a no-op after Commit
	names, err := storedOrigins(ctx, tx)
	if err != nil {
		return nil, err
	}
	for _, old := range names {
		u, err := url.Parse(old)
		if err != nil || u.Scheme != "http" && u.Scheme != "https" || u.Hostname() == "" {
			continue // not a name Origin made
		}
		name := Origin(u)
		if name == old {
			continue
		}
		var clash bool
		if err := tx.QueryRowContext(ctx, `SELECT EXISTS (SELECT 1 FROM ledger a JOIN ledger b ON b.kind = a.kind
			AND b.source_key = a.source_key AND b.source_created = a.source_created AND b.dest = a.dest
			WHERE a.source_origin = ? AND b.source_origin = ? AND a.state IN `+liveStates+` AND b.state IN `+liveStates+`)`,
			old, name).Scan(&clash); err != nil {
			return nil, fmt.Errorf("check old panel name %s: %w", old, err)
		}
		if clash {
			kept = append(kept, old)
			continue
		}
		for _, q := range []string{
			"UPDATE jobs SET source_origin = ? WHERE source_origin = ?",
			"UPDATE ledger SET source_origin = ? WHERE source_origin = ?",
			"UPDATE batches SET source_origin = ? WHERE source_origin = ?",
			"UPDATE legacy_sites SET source_url = ? WHERE mode = 'batches' AND source_url = ?",
		} {
			if _, err := tx.ExecContext(ctx, q, name, old); err != nil {
				return nil, fmt.Errorf("rename old panel %s: %w", old, err)
			}
		}
	}
	return kept, tx.Commit()
}

// storedOrigins lists every old panel name the store holds, sorted.
func storedOrigins(ctx context.Context, tx *sql.Tx) ([]string, error) {
	rows, err := tx.QueryContext(ctx, `SELECT source_origin FROM jobs UNION SELECT source_origin FROM ledger
		UNION SELECT source_origin FROM batches ORDER BY 1`)
	if err != nil {
		return nil, fmt.Errorf("read old panel names: %w", err)
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
