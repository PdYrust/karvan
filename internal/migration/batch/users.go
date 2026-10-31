package batch

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"time"
)

// User is one user of a batch as the API shows it. Old tokens never
// leave the server: Karvan keeps only an HMAC of each.
type User struct {
	ID             int64  `json:"id"` // the ledger row, for the new link
	SourceUsername string `json:"sourceUsername"`
	Username       string `json:"username"`
	Admin          string `json:"admin"`
	DestID         int64  `json:"destId"`
	// Joined is how the user came into the batch: moved by its run,
	// adopted (already on the destination), or carried from an earlier
	// run of the same old panel that made no batch.
	Joined   string `json:"joined"`
	OldLinks struct {
		Verified    int        `json:"verified"`
		LastChecked *time.Time `json:"lastChecked"`
	} `json:"oldLinks"`
	Closed bool `json:"closed"`
}

// Users lists a batch's users by ledger id, a page at a time. q, when
// set, keeps users whose old or new username contains it, in any case.
func (s *Store) Users(ctx context.Context, id int64, q, after string, limit int) ([]User, string, error) {
	var from int64
	if after != "" {
		n, err := strconv.ParseInt(after, 10, 64)
		if err != nil || n < 0 {
			return nil, "", ErrBadCursor
		}
		from = n
	}
	if _, err := s.Get(ctx, id); err != nil {
		return nil, "", err
	}
	limit = min(max(limit, 1), MaxPage)
	rows, err := s.db.QueryContext(ctx, `SELECT l.id, l.source_username, l.dest_username,
		COALESCE(NULLIF(a.dest_username, ''), j.source_admin), l.dest_id,
		CASE WHEN l.job_id <> b.job_id THEN 'carried' WHEN l.state = 'adopted' THEN 'adopted' ELSE 'moved' END,
		(SELECT count(*) FROM legacy_tokens t WHERE t.ledger_id = l.id),
		(SELECT max(t.checked_at) FROM legacy_tokens t WHERE t.ledger_id = l.id),
		EXISTS (SELECT 1 FROM legacy_closed c WHERE c.ledger_id = l.id)
		FROM ledger l JOIN batches b ON b.id = l.batch_id `+adminOf+`
		WHERE l.batch_id = ? AND l.id > ? AND (? = '' OR instr(lower(l.source_username), lower(?)) > 0
			OR instr(lower(l.dest_username), lower(?)) > 0)
		ORDER BY l.id LIMIT ?`, id, from, q, q, q, limit+1)
	if err != nil {
		return nil, "", fmt.Errorf("read the users of batch %d: %w", id, err)
	}
	defer rows.Close()
	out := []User{}
	for rows.Next() {
		var u User
		var checked sql.NullInt64
		if err := rows.Scan(&u.ID, &u.SourceUsername, &u.Username, &u.Admin, &u.DestID, &u.Joined,
			&u.OldLinks.Verified, &checked, &u.Closed); err != nil {
			return nil, "", fmt.Errorf("read the users of batch %d: %w", id, err)
		}
		if checked.Valid {
			at := time.Unix(checked.Int64, 0).UTC()
			u.OldLinks.LastChecked = &at
		}
		out = append(out, u)
	}
	if err := rows.Err(); err != nil {
		return nil, "", fmt.Errorf("read the users of batch %d: %w", id, err)
	}
	next := ""
	if len(out) > limit {
		out = out[:limit]
		next = strconv.FormatInt(out[limit-1].ID, 10)
	}
	return out, next, nil
}

// Member is what building a user's new link needs: the destination panel
// and database the batch moved it to, the user there, and its admin's id
// there (a PasarGuard admin can have its own link domain).
type Member struct {
	PanelID     int64
	Dest        string
	DestID      int64
	Username    string
	AdminDestID int64
}

// Member reads a user of a batch that is not deleted; ErrNotFound when
// the ledger row is not a user of that batch.
func (s *Store) Member(ctx context.Context, batchID, ledgerID int64) (Member, error) {
	var m Member
	err := s.db.QueryRowContext(ctx, `SELECT b.panel_id, b.dest, l.dest_id, l.dest_username, COALESCE(a.dest_id, 0)
		FROM ledger l JOIN batches b ON b.id = l.batch_id `+adminOf+`
		WHERE l.id = ? AND l.batch_id = ? AND l.kind = 'user' AND b.state <> 'deleted'`, ledgerID, batchID).
		Scan(&m.PanelID, &m.Dest, &m.DestID, &m.Username, &m.AdminDestID)
	if errors.Is(err, sql.ErrNoRows) {
		return m, ErrNotFound
	}
	if err != nil {
		return m, fmt.Errorf("read user %d of batch %d: %w", ledgerID, batchID, err)
	}
	return m, nil
}
