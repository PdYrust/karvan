package settings

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/mail"
)

const (
	keyACMEStaging = "acme.staging"
	keyACMEEmail   = "acme.email"
	keyACMEKeep80  = "acme.keepPort80"
)

// maxACMEEmail caps the Let's Encrypt contact address.
const maxACMEEmail = 254

// ErrACMEEmail means the Let's Encrypt contact is not one plain address.
var ErrACMEEmail = errors.New("the Let's Encrypt contact must be one email address")

// ACME are the Let's Encrypt settings. All are off or empty until
// the operator changes them.
type ACME struct {
	Staging    bool   `json:"staging"`    // new orders use the Let's Encrypt test server
	Email      string `json:"email"`      // optional contact for the account
	KeepPort80 bool   `json:"keepPort80"` // the challenge port stays open between orders
}

// Validate checks the contact: empty, or one address without a display
// name, at most 254 bytes.
func (a ACME) Validate() error {
	if a.Email == "" {
		return nil
	}
	addr, err := mail.ParseAddress(a.Email)
	if len(a.Email) > maxACMEEmail || err != nil || addr.Name != "" || addr.Address != a.Email {
		return ErrACMEEmail
	}
	return nil
}

// ACME reads the Let's Encrypt settings.
func (s *Store) ACME(ctx context.Context) (ACME, error) {
	vals := map[string]string{}
	for _, key := range []string{keyACMEStaging, keyACMEEmail, keyACMEKeep80} {
		var v string
		err := s.db.QueryRowContext(ctx, `SELECT value FROM settings WHERE key = ?`, key).Scan(&v)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return ACME{}, fmt.Errorf("read settings: %w", err)
		}
		vals[key] = v
	}
	return ACME{Staging: vals[keyACMEStaging] == "on", Email: vals[keyACMEEmail], KeepPort80: vals[keyACMEKeep80] == "on"}, nil
}

// SetACME validates and stores the Let's Encrypt settings.
func (s *Store) SetACME(ctx context.Context, a ACME) error {
	if err := a.Validate(); err != nil {
		return err
	}
	onOff := map[bool]string{true: "on", false: "off"}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("save settings: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	for k, v := range map[string]string{keyACMEStaging: onOff[a.Staging], keyACMEEmail: a.Email, keyACMEKeep80: onOff[a.KeepPort80]} {
		if _, err := tx.ExecContext(ctx, `INSERT INTO settings (key, value) VALUES (?, ?)
			ON CONFLICT (key) DO UPDATE SET value = excluded.value`, k, v); err != nil {
			return fmt.Errorf("save setting %s: %w", k, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("save settings: %w", err)
	}
	return nil
}
