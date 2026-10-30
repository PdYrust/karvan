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
// the batch went inactive (D21).
type scoped struct {
	ledger.Entry
	held    bool // the row's batch is inactive or deleted (D22)
	ofRule  bool // the row's batch is one of the rule's batches
	inScope bool // the rule serves the row (D23)
}

// holds reports whether r is in a held batch of rule s: one of its
// batches, or, for a manual rule, any batch of its old panel and
// destination. Such a row answers 404 before the panel is asked (D22).
func (s Site) holds(r scoped) bool { return r.held && (s.Mode != ModeBatches || r.ofRule) }

// The parts of the scope queries, over ledger l left joined with its
// batch b. A row whose batch_id names no batch counts as held.
const (
	fromLedger = " FROM ledger l LEFT JOIN batches b ON b.id = l.batch_id"
	heldSQL    = "(l.batch_id IS NOT NULL AND b.state IS NOT 'active')"
	ofRuleSQL  = "EXISTS (SELECT 1 FROM legacy_site_batches sb WHERE sb.site_id = ? AND sb.batch_id = l.batch_id)"
	// scopeSQL is the condition for the rows a rule serves (D23): moved
	// or adopted users of its old panel on its destination, in one of its
	// batches while that batch is active (a rule of batches), or in an
	// active batch or none (a manual rule). Its arguments are scopeArgs.
	scopeSQL = "(l.kind = 'user' AND l.source_origin = ? AND l.dest = ? AND l.state IN ('done', 'adopted') AND " +
		"CASE WHEN ? = '" + ModeBatches + "' THEN b.state IS 'active' AND " + ofRuleSQL +
		" ELSE l.batch_id IS NULL OR b.state IS 'active' END)"
)

// The queries of token and served.
const (
	tokenSQL = "SELECT " + ledger.Columns + ", t.site_id, t.issued_at, t.checked_at, " + heldSQL + ", " + scopeSQL +
		" FROM legacy_tokens t JOIN ledger l ON l.id = t.ledger_id LEFT JOIN batches b ON b.id = l.batch_id WHERE t.mac = ?"
	servedSQL = "SELECT " + ledger.Columns + fromLedger + " WHERE " + scopeSQL + " ORDER BY l.id"
)

func scopeArgs(s Site, dest string) []any { return []any{s.origin, dest, s.Mode, s.ID} }

// token reads a stored token with its ledger row and batch, for rule s
// on destination dest, in one query (D21).
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
	namedSQL = "SELECT " + ledger.Columns + ", " + heldSQL + ", " + ofRuleSQL + ", " + scopeSQL + fromLedger +
		" WHERE l.kind = 'user' AND l.source_origin = ? AND l.dest = ? AND l.state IN ('done', 'adopted') AND "
	namedByName = namedSQL + "l.source_username = ? ORDER BY l.id"
	namedByKey  = namedSQL + "l.source_key = ? ORDER BY l.id"
)

// named reads every moved or adopted user of rule s's old panel on dest
// that a token's payload can name: by id on a PasarGuard old panel when
// the payload has one, else by username. Only the old panel's answer
// tells them apart (D24).
func (st *store) named(ctx context.Context, s Site, dest string, p subtoken.Payload) ([]scoped, error) {
	q, key := namedByName, p.Username
	if s.SourceKind == panel.PasarGuard && p.UserID > 0 {
		q, key = namedByKey, strconv.FormatInt(p.UserID, 10)
	}
	if key == "" {
		return nil, nil
	}
	args := append(append([]any{s.ID}, scopeArgs(s, dest)...), s.origin, dest, key)
	rows, err := st.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("read ledger: %w", err)
	}
	defer rows.Close()
	var out []scoped
	for rows.Next() {
		var r scoped
		if r.Entry, err = ledger.ScanEntry(rows, &r.held, &r.ofRule, &r.inScope); err != nil {
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

// ruleHeld reports whether s is a rule of batches none of which is
// active: it answers 404 to every request, with no pass-through (D22).
func (st *store) ruleHeld(ctx context.Context, s Site) (bool, error) {
	if s.Mode != ModeBatches {
		return false, nil
	}
	var n int
	err := st.db.QueryRowContext(ctx, "SELECT count(*) FROM legacy_site_batches sb JOIN batches b ON b.id = sb.batch_id "+
		"WHERE sb.site_id = ? AND b.state = 'active'", s.ID).Scan(&n)
	if err != nil {
		return false, fmt.Errorf("read rule batches: %w", err)
	}
	return n == 0, nil
}
