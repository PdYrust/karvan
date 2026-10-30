package bridge

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"fmt"
	"time"
)

// verified is a stored token: proven once to belong to a ledger row.
type verified struct {
	SiteID   int64
	LedgerID int64
	Issued   time.Time
	Checked  time.Time
}

// mac names a token without keeping it (D57).
func mac(key []byte, token string) []byte {
	h := hmac.New(sha256.New, key)
	h.Write([]byte(token))
	return h.Sum(nil)
}

// putToken stores a verified token. A user closed meanwhile wins: the
// insert only happens while the user is not closed.
func (st *store) putToken(ctx context.Context, m []byte, v verified) (bool, error) {
	now := st.now().Unix()
	res, err := st.db.ExecContext(ctx, "INSERT INTO legacy_tokens (mac, site_id, ledger_id, issued_at, verified_at, checked_at) "+
		"SELECT ?, ?, ?, ?, ?, ? WHERE NOT EXISTS (SELECT 1 FROM legacy_closed WHERE ledger_id = ?) "+
		"ON CONFLICT (mac) DO UPDATE SET checked_at = excluded.checked_at",
		m, v.SiteID, v.LedgerID, v.Issued.Unix(), now, now, v.LedgerID)
	if err != nil {
		return false, fmt.Errorf("store token: %w", err)
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

func (st *store) checked(ctx context.Context, m []byte) error {
	_, err := st.db.ExecContext(ctx, "UPDATE legacy_tokens SET checked_at = ? WHERE mac = ?", st.now().Unix(), m)
	return err
}

func (st *store) dropToken(ctx context.Context, m []byte) error {
	_, err := st.db.ExecContext(ctx, "DELETE FROM legacy_tokens WHERE mac = ?", m)
	return err
}

// close shuts a user's old links for good: their tokens go and no lazy
// check runs for them again. A user whose batch a restore took meanwhile
// stays as it is (D22); each statement reads the batch state as it writes.
// It reports whether the user was closed now.
func (st *store) close(ctx context.Context, ledgerID int64, reason string) (bool, error) {
	tx, err := st.db.BeginTx(ctx, nil)
	if err != nil {
		return false, err
	}
	defer tx.Rollback() //nolint:errcheck // a no-op after Commit
	const open = "NOT EXISTS (SELECT 1 FROM ledger l JOIN batches b ON b.id = l.batch_id WHERE l.id = ? AND b.state <> 'active')"
	res, err := tx.ExecContext(ctx, "INSERT INTO legacy_closed (ledger_id, reason, closed_at) SELECT ?, ?, ? WHERE "+open+
		" ON CONFLICT (ledger_id) DO NOTHING", ledgerID, reason, st.now().Unix(), ledgerID)
	if err != nil {
		return false, fmt.Errorf("close user: %w", err)
	}
	if _, err := tx.ExecContext(ctx, "DELETE FROM legacy_tokens WHERE ledger_id = ? AND "+open, ledgerID, ledgerID); err != nil {
		return false, fmt.Errorf("close user: %w", err)
	}
	n, _ := res.RowsAffected()
	return n > 0, tx.Commit()
}

func (st *store) closed(ctx context.Context, ledgerID int64) (bool, error) {
	var n int
	err := st.queryRow(ctx, "SELECT count(*) FROM legacy_closed WHERE ledger_id = ?", ledgerID).Scan(&n)
	return n > 0, err
}

// closedSet is every closed ledger id, for the sweep.
func (st *store) closedSet(ctx context.Context) (map[int64]bool, error) {
	rows, err := st.db.QueryContext(ctx, "SELECT ledger_id FROM legacy_closed")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[int64]bool{}
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out[id] = true
	}
	return out, rows.Err()
}
