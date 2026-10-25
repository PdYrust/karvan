package auth

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"github.com/PdYrust/karvan/internal/secret"
)

// Session lifetimes.
const (
	IdleTimeout     = 30 * time.Minute
	AbsoluteTimeout = 12 * time.Hour

	// last_seen_at is only rewritten after this long, to keep reads cheap.
	touchInterval = time.Minute
	maxUserAgent  = 256
)

// ErrNoSession means the token is unknown or the session has ended.
var ErrNoSession = errors.New("no valid session")

// Session is a signed-in browser.
type Session struct {
	// ID names the session in lists. It is derived from the token hash
	// one way, so neither the token nor its hash leaves the server.
	ID         string
	OperatorID int64
	CreatedAt  time.Time
	LastSeenAt time.Time
	ExpiresAt  time.Time
	IP         string
	UserAgent  string
}

// Sessions stores sessions by the SHA-256 of their token; the token itself
// only exists in the browser cookie.
type Sessions struct {
	db  *sql.DB
	now func() time.Time
}

// NewSessions returns a Sessions store.
func NewSessions(db *sql.DB) *Sessions { return &Sessions{db: db, now: time.Now} }

func hashToken(token string) []byte {
	sum := sha256.Sum256([]byte(token))
	return sum[:]
}

// sessionID is the opaque id of the session stored under tokenHash.
func sessionID(tokenHash []byte) string {
	sum := sha256.Sum256(append([]byte("session-id/v1:"), tokenHash...))
	return base64.RawURLEncoding.EncodeToString(sum[:12])
}

type execer interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

// Create starts a session and returns its token.
func (s *Sessions) Create(ctx context.Context, operatorID int64, ip, userAgent string) (string, Session, error) {
	return s.insert(ctx, s.db, operatorID, ip, userAgent)
}

func (s *Sessions) insert(ctx context.Context, db execer, operatorID int64, ip, userAgent string) (string, Session, error) {
	token, err := secret.NewToken()
	if err != nil {
		return "", Session{}, err
	}
	if len(userAgent) > maxUserAgent {
		userAgent = userAgent[:maxUserAgent]
	}
	now, h := s.now(), hashToken(token)
	sess := Session{ID: sessionID(h), OperatorID: operatorID, CreatedAt: now, LastSeenAt: now, ExpiresAt: now.Add(AbsoluteTimeout), IP: ip, UserAgent: userAgent}
	_, err = db.ExecContext(ctx, `INSERT INTO sessions (token_hash, operator_id, created_at, last_seen_at, expires_at, ip, user_agent)
		VALUES (?, ?, ?, ?, ?, ?, ?)`, h, operatorID, now.Unix(), now.Unix(), sess.ExpiresAt.Unix(), ip, userAgent)
	if err != nil {
		return "", Session{}, fmt.Errorf("create session: %w", err)
	}
	return token, sess, nil
}

// Replace runs change, ends every session of the operator and starts a new
// one, all in one transaction: either all of it happens or none does.
func (s *Sessions) Replace(ctx context.Context, operatorID int64, ip, userAgent string, change func(*sql.Tx) error) (string, Session, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return "", Session{}, fmt.Errorf("replace sessions: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	if err := change(tx); err != nil {
		return "", Session{}, err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM sessions WHERE operator_id = ?`, operatorID); err != nil {
		return "", Session{}, fmt.Errorf("end sessions of operator %d: %w", operatorID, err)
	}
	token, sess, err := s.insert(ctx, tx, operatorID, ip, userAgent)
	if err != nil {
		return "", Session{}, err
	}
	if err := tx.Commit(); err != nil {
		return "", Session{}, fmt.Errorf("replace sessions: %w", err)
	}
	return token, sess, nil
}

// Lookup returns the live session for token and refreshes its idle timer.
func (s *Sessions) Lookup(ctx context.Context, token string) (Session, error) {
	return s.lookup(ctx, token, true)
}

// Peek returns the live session for token without refreshing its idle
// timer, for polling the operator did not ask for: a dashboard left open
// still signs out after IdleTimeout.
func (s *Sessions) Peek(ctx context.Context, token string) (Session, error) {
	return s.lookup(ctx, token, false)
}

func (s *Sessions) lookup(ctx context.Context, token string, touch bool) (Session, error) {
	if token == "" {
		return Session{}, ErrNoSession
	}
	h := hashToken(token)
	var sess Session
	var created, seen, expires int64
	err := s.db.QueryRowContext(ctx, `SELECT operator_id, created_at, last_seen_at, expires_at, ip, user_agent
		FROM sessions WHERE token_hash = ?`, h).Scan(&sess.OperatorID, &created, &seen, &expires, &sess.IP, &sess.UserAgent)
	if errors.Is(err, sql.ErrNoRows) {
		return Session{}, ErrNoSession
	}
	if err != nil {
		return Session{}, fmt.Errorf("look up session: %w", err)
	}
	sess.ID = sessionID(h)
	sess.CreatedAt, sess.LastSeenAt, sess.ExpiresAt = time.Unix(created, 0), time.Unix(seen, 0), time.Unix(expires, 0)

	now := s.now()
	if !now.Before(sess.ExpiresAt) || now.Sub(sess.LastSeenAt) >= IdleTimeout {
		if _, err := s.db.ExecContext(ctx, `DELETE FROM sessions WHERE token_hash = ?`, h); err != nil {
			return Session{}, fmt.Errorf("end expired session: %w", err)
		}
		return Session{}, ErrNoSession
	}
	if touch && now.Sub(sess.LastSeenAt) >= touchInterval {
		if _, err := s.db.ExecContext(ctx, `UPDATE sessions SET last_seen_at = ? WHERE token_hash = ?`, now.Unix(), h); err != nil {
			return Session{}, fmt.Errorf("touch session: %w", err)
		}
		sess.LastSeenAt = now
	}
	return sess, nil
}

// Delete ends the session for token. Ending an unknown session is not an error.
func (s *Sessions) Delete(ctx context.Context, token string) error {
	if _, err := s.db.ExecContext(ctx, `DELETE FROM sessions WHERE token_hash = ?`, hashToken(token)); err != nil {
		return fmt.Errorf("end session: %w", err)
	}
	return nil
}

// DeleteForOperator ends every session of an operator, for example after a
// password change.
func (s *Sessions) DeleteForOperator(ctx context.Context, operatorID int64) error {
	if _, err := s.db.ExecContext(ctx, `DELETE FROM sessions WHERE operator_id = ?`, operatorID); err != nil {
		return fmt.Errorf("end sessions of operator %d: %w", operatorID, err)
	}
	return nil
}

// List returns the operator's live sessions, the most recently used first.
func (s *Sessions) List(ctx context.Context, operatorID int64) ([]Session, error) {
	now := s.now()
	rows, err := s.db.QueryContext(ctx, `SELECT token_hash, created_at, last_seen_at, expires_at, ip, user_agent FROM sessions
		WHERE operator_id = ? AND expires_at > ? AND last_seen_at > ? ORDER BY last_seen_at DESC, created_at DESC`,
		operatorID, now.Unix(), now.Add(-IdleTimeout).Unix())
	if err != nil {
		return nil, fmt.Errorf("list sessions: %w", err)
	}
	defer rows.Close()
	var out []Session
	for rows.Next() {
		var h []byte
		var created, seen, expires int64
		sess := Session{OperatorID: operatorID}
		if err := rows.Scan(&h, &created, &seen, &expires, &sess.IP, &sess.UserAgent); err != nil {
			return nil, fmt.Errorf("list sessions: %w", err)
		}
		sess.ID = sessionID(h)
		sess.CreatedAt, sess.LastSeenAt, sess.ExpiresAt = time.Unix(created, 0), time.Unix(seen, 0), time.Unix(expires, 0)
		out = append(out, sess)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list sessions: %w", err)
	}
	return out, nil
}

// DeleteOthers ends every session of the operator except the one for
// keepToken, and returns how many it ended.
func (s *Sessions) DeleteOthers(ctx context.Context, operatorID int64, keepToken string) (int64, error) {
	res, err := s.db.ExecContext(ctx, `DELETE FROM sessions WHERE operator_id = ? AND token_hash != ?`, operatorID, hashToken(keepToken))
	if err != nil {
		return 0, fmt.Errorf("end other sessions of operator %d: %w", operatorID, err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("end other sessions of operator %d: %w", operatorID, err)
	}
	return n, nil
}

// DeleteByID ends the operator's session with the opaque id. It reports
// false when the operator has no such session.
func (s *Sessions) DeleteByID(ctx context.Context, operatorID int64, id string) (bool, error) {
	h, err := s.hashByID(ctx, operatorID, id)
	if err != nil || h == nil {
		return false, err
	}
	res, err := s.db.ExecContext(ctx, `DELETE FROM sessions WHERE token_hash = ? AND operator_id = ?`, h, operatorID)
	if err != nil {
		return false, fmt.Errorf("end session: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("end session: %w", err)
	}
	return n == 1, nil
}

// hashByID finds the token hash behind an id. Ids are not stored, so the
// operator's few sessions are read and each id is derived again.
func (s *Sessions) hashByID(ctx context.Context, operatorID int64, id string) ([]byte, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT token_hash FROM sessions WHERE operator_id = ?`, operatorID)
	if err != nil {
		return nil, fmt.Errorf("find session: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var h []byte
		if err := rows.Scan(&h); err != nil {
			return nil, fmt.Errorf("find session: %w", err)
		}
		if subtle.ConstantTimeCompare([]byte(sessionID(h)), []byte(id)) == 1 {
			return h, nil
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("find session: %w", err)
	}
	return nil, nil
}

// PurgeExpired removes sessions past either timeout.
func (s *Sessions) PurgeExpired(ctx context.Context) error {
	now := s.now()
	_, err := s.db.ExecContext(ctx, `DELETE FROM sessions WHERE expires_at <= ? OR last_seen_at <= ?`,
		now.Unix(), now.Add(-IdleTimeout).Unix())
	if err != nil {
		return fmt.Errorf("purge sessions: %w", err)
	}
	return nil
}

// CSRFToken derives the anti-CSRF token for a session token. Deriving it
// means nothing extra is stored, and it changes whenever the session does.
func CSRFToken(sessionToken string) string {
	mac := hmac.New(sha256.New, []byte(sessionToken))
	mac.Write([]byte("csrf/v1"))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

// ValidCSRF reports whether got is the CSRF token for sessionToken.
func ValidCSRF(sessionToken, got string) bool {
	if sessionToken == "" || got == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(CSRFToken(sessionToken)), []byte(got)) == 1
}
