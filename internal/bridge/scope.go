package bridge

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/PdYrust/karvan/internal/migration/ledger"
	"github.com/PdYrust/karvan/internal/panel"
	"github.com/PdYrust/karvan/internal/panel/subtoken"
)

// scoped is a ledger row with what its batch means for one rule, read
// on every request: a cached state could serve a user for a moment after
// the batch went inactive.
type scoped struct {
	ledger.Entry
	held    bool // the row's batch is inactive or deleted
	inScope bool // the rule serves the row
}

// isHeld reports whether r, a user of a rule's old panel and destination,
// is in an inactive or deleted batch. Such a row answers 404 before the
// panel is asked, on a rule of batches and on a manual rule alike.
func isHeld(r scoped) bool { return r.held }

// The parts of the scope queries, over ledger l left joined with its
// batch b. A row whose batch_id names no batch counts as held.
const (
	fromLedger = " FROM ledger l LEFT JOIN batches b ON b.id = l.batch_id"
	heldSQL    = "(l.batch_id IS NOT NULL AND b.state IS NOT 'active')"
	// scopeSQL is the condition for the rows a rule serves: moved
	// or adopted users of its old panel on its destination, in an active
	// batch of any run of that old panel (a rule of batches), or in an
	// active batch or none (a manual rule). Its arguments are scopeArgs.
	scopeSQL = "(l.kind = 'user' AND l.source_origin = ? AND l.dest = ? AND l.state IN ('done', 'adopted') AND " +
		"CASE WHEN ? = '" + ModeBatches + "' THEN b.state IS 'active' ELSE l.batch_id IS NULL OR b.state IS 'active' END)"
)

// The queries of token and served.
const (
	tokenSQL = "SELECT " + ledger.Columns + ", t.site_id, t.issued_at, t.checked_at, " + heldSQL + ", " + scopeSQL +
		" FROM legacy_tokens t JOIN ledger l ON l.id = t.ledger_id LEFT JOIN batches b ON b.id = l.batch_id WHERE t.mac = ?"
	servedSQL = "SELECT " + ledger.Columns + fromLedger + " WHERE " + scopeSQL + " ORDER BY l.id"
)

func scopeArgs(s Site, dest string) []any { return []any{s.origin, dest, s.Mode} }

// token reads a stored token with its ledger row and batch, for rule s
// on destination dest, in one query.
func (st *store) token(ctx context.Context, m []byte, s Site, dest string) (verified, scoped, bool, error) {
	row := st.queryRow(ctx, tokenSQL, append(scopeArgs(s, dest), m)...)
	var v verified
	var r scoped
	var issued, checked int64
	e, err := ledger.ScanEntry(row, &v.SiteID, &issued, &checked, &r.held, &r.inScope)
	if errors.Is(err, sql.ErrNoRows) {
		return v, r, false, nil
	}
	if err != nil {
		return v, r, false, fmt.Errorf("read token: %w", err)
	}
	r.Entry, v.LedgerID = e, e.ID
	v.Issued, v.Checked = time.Unix(issued, 0).UTC(), time.Unix(checked, 0).UTC()
	return v, r, true, nil
}

// The rows a payload names, by source username or by source key.
const (
	namedSQL = "SELECT " + ledger.Columns + ", " + heldSQL + ", " + scopeSQL + fromLedger +
		" WHERE l.kind = 'user' AND l.source_origin = ? AND l.dest = ? AND l.state IN ('done', 'adopted') AND "
	namedByName = namedSQL + "l.source_username = ? ORDER BY l.id"
	namedByKey  = namedSQL + "l.source_key = ? ORDER BY l.id"
)

// named reads every moved or adopted user of rule s's old panel on dest
// that a token's payload can name: by id on a PasarGuard old panel when
// the payload has one, else by username. Only the old panel's answer
// tells them apart.
func (st *store) named(ctx context.Context, s Site, dest string, p subtoken.Payload) ([]scoped, error) {
	q, key := namedByName, p.Username
	if s.SourceKind == panel.PasarGuard && p.UserID > 0 {
		q, key = namedByKey, strconv.FormatInt(p.UserID, 10)
	}
	if key == "" {
		return nil, nil
	}
	args := append(scopeArgs(s, dest), s.origin, dest, key)
	rows, err := st.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("read ledger: %w", err)
	}
	defer rows.Close()
	var out []scoped
	for rows.Next() {
		var r scoped
		if r.Entry, err = ledger.ScanEntry(rows, &r.held, &r.inScope); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// served lists the rows rule s serves on dest, for the sweep.
func (st *store) served(ctx context.Context, s Site, dest string) ([]ledger.Entry, error) {
	rows, err := st.db.QueryContext(ctx, servedSQL, scopeArgs(s, dest)...)
	if err != nil {
		return nil, fmt.Errorf("read ledger: %w", err)
	}
	defer rows.Close()
	var out []ledger.Entry
	for rows.Next() {
		e, err := ledger.ScanEntry(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// ruleHeld reports whether s is a rule of batches whose old panel has no
// active batch on dest: it answers 404 to every request, with no
// pass-through.
func (st *store) ruleHeld(ctx context.Context, s Site, dest string) (bool, error) {
	if s.Mode != ModeBatches {
		return false, nil
	}
	var active bool
	err := st.db.QueryRowContext(ctx, "SELECT EXISTS (SELECT 1 FROM batches WHERE source_origin = ? AND dest = ? AND state = 'active')",
		s.origin, dest).Scan(&active)
	if err != nil {
		return false, fmt.Errorf("read rule batches: %w", err)
	}
	return !active, nil
}
