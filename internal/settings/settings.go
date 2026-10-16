// Package settings stores the values Karvan changes at runtime: the listen
// address and the hidden base path.
package settings

import (
	"context"
	"crypto/rand"
	"database/sql"
	"errors"
	"fmt"
	"math/big"
	"net"
	"net/netip"
	"regexp"
	"strconv"

	"github.com/PdYrust/karvan/internal/secret"
)

const (
	keyBind     = "http.bind"
	keyPort     = "http.port"
	keyBasePath = "http.base_path"

	basePathLen = 16
	alphabet    = "abcdefghijkmnopqrstuvwxyzABCDEFGHJKLMNPQRSTUVWXYZ23456789"
)

// Generated ports stay in this range, away from common services.
const (
	MinRandomPort = 20000 // lowest generated port
	MaxRandomPort = 60000 // highest generated port
)

// ErrNotInitialized means `karvan init` has not been run on this database.
var ErrNotInitialized = errors.New("karvan is not initialized; run: karvan init")

var basePathPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{8,64}$`)

// HTTP is where and under which path the web panel is served.
type HTTP struct {
	Bind     string // empty means all interfaces
	Port     int
	BasePath string // without slashes
}

// Addr returns the listen address for net.Listen.
func (h HTTP) Addr() string { return net.JoinHostPort(h.Bind, strconv.Itoa(h.Port)) }

// Validate checks every field.
func (h HTTP) Validate() error {
	if h.Bind != "" {
		if _, err := netip.ParseAddr(h.Bind); err != nil {
			return fmt.Errorf("bind address %q is not an IP address", h.Bind)
		}
	}
	if h.Port < 1 || h.Port > 65535 {
		return fmt.Errorf("port %d is out of range 1-65535", h.Port)
	}
	if !basePathPattern.MatchString(h.BasePath) {
		return fmt.Errorf("path %q must be 8 to 64 letters, digits, '-' or '_'", h.BasePath)
	}
	return nil
}

// Store reads and writes settings.
type Store struct{ db *sql.DB }

// New returns a Store over db.
func New(db *sql.DB) *Store { return &Store{db: db} }

// HTTP returns the current HTTP settings, or ErrNotInitialized.
func (s *Store) HTTP(ctx context.Context) (HTTP, error) {
	vals := map[string]string{}
	rows, err := s.db.QueryContext(ctx, `SELECT key, value FROM settings WHERE key IN (?, ?, ?)`, keyBind, keyPort, keyBasePath)
	if err != nil {
		return HTTP{}, fmt.Errorf("read settings: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var k, v string
		if err := rows.Scan(&k, &v); err != nil {
			return HTTP{}, fmt.Errorf("read settings: %w", err)
		}
		vals[k] = v
	}
	if err := rows.Err(); err != nil {
		return HTTP{}, fmt.Errorf("read settings: %w", err)
	}
	if _, ok := vals[keyPort]; !ok {
		return HTTP{}, ErrNotInitialized
	}
	port, err := strconv.Atoi(vals[keyPort])
	if err != nil {
		return HTTP{}, fmt.Errorf("stored port %q is not a number", vals[keyPort])
	}
	h := HTTP{Bind: vals[keyBind], Port: port, BasePath: vals[keyBasePath]}
	// Fail closed: an empty or damaged path would put the panel at "/".
	if err := h.Validate(); err != nil {
		return HTTP{}, fmt.Errorf("stored settings are invalid (%w); fix them with: karvan reset -path -port", err)
	}
	return h, nil
}

// SetHTTP validates and stores h.
func (s *Store) SetHTTP(ctx context.Context, h HTTP) error {
	if err := h.Validate(); err != nil {
		return err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("save settings: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	for k, v := range map[string]string{keyBind: h.Bind, keyPort: strconv.Itoa(h.Port), keyBasePath: h.BasePath} {
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

// NewBasePath returns a random hidden path (about 93 bits). Look-alike
// characters are left out so it can be read from a terminal and typed.
func NewBasePath() (string, error) { return secret.RandomString(alphabet, basePathLen) }

// RandomPort returns a port in [MinRandomPort, MaxRandomPort].
func RandomPort() (int, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(MaxRandomPort-MinRandomPort+1))
	if err != nil {
		return 0, fmt.Errorf("generate port: %w", err)
	}
	return MinRandomPort + int(n.Int64()), nil
}

const keySafeMode = "migration.safe_mode"

// SafeMode reports whether migrations take a backup and undo everything
// on failure. It is on until the operator turns it off.
func (s *Store) SafeMode(ctx context.Context) (bool, error) {
	var v string
	err := s.db.QueryRowContext(ctx, `SELECT value FROM settings WHERE key = ?`, keySafeMode).Scan(&v)
	if errors.Is(err, sql.ErrNoRows) {
		return true, nil
	}
	if err != nil {
		return true, fmt.Errorf("read settings: %w", err)
	}
	return v != "off", nil
}

// SetSafeMode stores the Safe Mode switch.
func (s *Store) SetSafeMode(ctx context.Context, on bool) error {
	v := "on"
	if !on {
		v = "off"
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO settings (key, value) VALUES (?, ?)
		ON CONFLICT (key) DO UPDATE SET value = excluded.value`, keySafeMode, v)
	if err != nil {
		return fmt.Errorf("save setting %s: %w", keySafeMode, err)
	}
	return nil
}
