package auth

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/PdYrust/karvan/internal/secret"
)

// ErrNotFound means no operator matched.
var ErrNotFound = errors.New("operator not found")

// ErrUsernameTaken means another operator already has the username.
var ErrUsernameTaken = errors.New("username is already taken")

// Operator is a person who signs in to Karvan.
type Operator struct {
	ID           int64
	Username     string
	PasswordHash string
	TOTPEnabled  bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// Operators stores operator accounts. TOTP secrets are sealed with key.
type Operators struct {
	db  *sql.DB
	key *secret.Key
	now func() time.Time
}

// NewOperators returns an Operators store.
func NewOperators(db *sql.DB, key *secret.Key) *Operators {
	return &Operators{db: db, key: key, now: time.Now}
}

const operatorColumns = `id, username, password_hash, totp_secret IS NOT NULL, created_at, updated_at`

func scanOperator(row interface{ Scan(...any) error }) (Operator, error) {
	var o Operator
	var created, updated int64
	err := row.Scan(&o.ID, &o.Username, &o.PasswordHash, &o.TOTPEnabled, &created, &updated)
	if errors.Is(err, sql.ErrNoRows) {
		return Operator{}, ErrNotFound
	}
	if err != nil {
		return Operator{}, fmt.Errorf("scan operator: %w", err)
	}
	o.CreatedAt, o.UpdatedAt = time.Unix(created, 0), time.Unix(updated, 0)
	return o, nil
}

// Create adds an operator. username must already be normalized.
func (s *Operators) Create(ctx context.Context, username, passwordHash string) (Operator, error) {
	now := s.now().Unix()
	row := s.db.QueryRowContext(ctx, `INSERT INTO operators (username, password_hash, created_at, updated_at)
		VALUES (?, ?, ?, ?) RETURNING `+operatorColumns, username, passwordHash, now, now)
	o, err := scanOperator(row)
	if err != nil && isUniqueViolation(err) {
		return Operator{}, ErrUsernameTaken
	}
	return o, err
}

// ByUsername finds an operator by normalized username.
func (s *Operators) ByUsername(ctx context.Context, username string) (Operator, error) {
	return scanOperator(s.db.QueryRowContext(ctx, `SELECT `+operatorColumns+` FROM operators WHERE username = ?`, username))
}

// ByID finds an operator by id.
func (s *Operators) ByID(ctx context.Context, id int64) (Operator, error) {
	return scanOperator(s.db.QueryRowContext(ctx, `SELECT `+operatorColumns+` FROM operators WHERE id = ?`, id))
}

// First returns the oldest operator. Version 1 has exactly one.
func (s *Operators) First(ctx context.Context) (Operator, error) {
	return scanOperator(s.db.QueryRowContext(ctx, `SELECT `+operatorColumns+` FROM operators ORDER BY id LIMIT 1`))
}

// SetUsername renames an operator. username must already be normalized.
func (s *Operators) SetUsername(ctx context.Context, id int64, username string) error {
	err := s.update(ctx, id, `username = ?`, username)
	if err != nil && isUniqueViolation(err) {
		return ErrUsernameTaken
	}
	return err
}

// SetPasswordHash replaces an operator's password hash.
func (s *Operators) SetPasswordHash(ctx context.Context, id int64, hash string) error {
	return s.update(ctx, id, `password_hash = ?`, hash)
}

// setLogin replaces the username and the password hash in tx.
func (s *Operators) setLogin(ctx context.Context, tx *sql.Tx, id int64, username, hash string) error {
	res, err := tx.ExecContext(ctx, `UPDATE operators SET username = ?, password_hash = ?, updated_at = ? WHERE id = ?`,
		username, hash, s.now().Unix(), id)
	if err != nil {
		if isUniqueViolation(err) {
			return ErrUsernameTaken
		}
		return fmt.Errorf("update login of operator %d: %w", id, err)
	}
	if n, err := res.RowsAffected(); err != nil || n == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Operators) update(ctx context.Context, id int64, set string, args ...any) error {
	args = append(args, s.now().Unix(), id)
	res, err := s.db.ExecContext(ctx, `UPDATE operators SET `+set+`, updated_at = ? WHERE id = ?`, args...) //nolint:gosec // G202: set is a constant fragment from this package.
	if err != nil {
		return fmt.Errorf("update operator %d: %w", id, err)
	}
	if n, err := res.RowsAffected(); err != nil || n == 0 {
		return ErrNotFound
	}
	return nil
}

func isUniqueViolation(err error) bool {
	return strings.Contains(err.Error(), "UNIQUE constraint failed")
}
