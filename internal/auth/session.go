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

// Create starts a session and returns its token.
func (s *Sessions) Create(ctx context.Context, operatorID int64, ip, userAgent string) (string, Session, error) {
	token, err := secret.NewToken()
	if err != nil {
		return "", Session{}, err
	}
	if len(userAgent) > maxUserAgent {
		userAgent = userAgent[:maxUserAgent]
	}
	now := s.now()
	sess := Session{OperatorID: operatorID, CreatedAt: now, LastSeenAt: now, ExpiresAt: now.Add(AbsoluteTimeout), IP: ip, UserAgent: userAgent}
	_, err = s.db.ExecContext(ctx, `INSERT INTO sessions (token_hash, operator_id, created_at, last_seen_at, expires_at, ip, user_agent)
		VALUES (?, ?, ?, ?, ?, ?, ?)`, hashToken(token), operatorID, now.Unix(), now.Unix(), sess.ExpiresAt.Unix(), ip, userAgent)
	if err != nil {
		return "", Session{}, fmt.Errorf("create session: %w", err)
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
