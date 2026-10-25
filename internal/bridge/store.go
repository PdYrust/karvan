package bridge

import (
	"context"
	"crypto/tls"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/PdYrust/karvan/internal/migration/ledger"
	"github.com/PdYrust/karvan/internal/panel/source"
	"github.com/PdYrust/karvan/internal/secret"
)

// ErrNotFound means no such site.
var ErrNotFound = errors.New("no such site")

// ErrOverlap means another site already answers the same requests.
var ErrOverlap = errors.New("another site answers the same port, domain and path")

// ErrNameTaken means another site has this name.
var ErrNameTaken = errors.New("another site has this name")

// ErrPortMixed means the port already serves sites the other way: one
// port is either HTTPS for all its sites or plain HTTP for all of them.
var ErrPortMixed = errors.New("the port has sites with and without a certificate")

// store keeps sites, verified tokens and closed users in Karvan's DB.
type store struct {
	db  *sql.DB
	key *secret.Key
	now func() time.Time
}

const siteColumns = "id, name, panel_id, source_kind, source_url, source_ip, source_pin, old_path, domains, " +
	"listen_port, cert_file, key_file, cert_pem, key_sealed IS NOT NULL, lazy, plain_http_ok"

func scanSite(row interface{ Scan(...any) error }) (Site, error) {
	var s Site
	var domains string
	err := row.Scan(&s.ID, &s.Name, &s.PanelID, &s.SourceKind, &s.SourceURL, &s.SourceIP, &s.SourcePin, &s.OldPath,
		&domains, &s.Port, &s.CertFile, &s.KeyFile, &s.CertPEM, &s.HasKey, &s.Lazy, &s.PlainHTTPOK)
	if err != nil {
		return s, err
	}
	if err := json.Unmarshal([]byte(domains), &s.Domains); err != nil {
		return s, fmt.Errorf("site %d domains: %w", s.ID, err)
	}
	return s, s.setOrigin()
}

func (s *Site) setOrigin() error {
	u, err := source.ParseBase(s.SourceURL)
	if err != nil {
		return err
	}
	s.origin = ledger.Origin(u)
	return nil
}

// Sites lists every site.
func (st *store) Sites(ctx context.Context) ([]Site, error) {
	rows, err := st.db.QueryContext(ctx, "SELECT "+siteColumns+" FROM legacy_sites ORDER BY id")
	if err != nil {
		return nil, fmt.Errorf("read sites: %w", err)
	}
	defer rows.Close()
	var out []Site
	for rows.Next() {
		s, err := scanSite(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// Site reads one site.
func (st *store) Site(ctx context.Context, id int64) (Site, error) {
	s, err := scanSite(st.db.QueryRowContext(ctx, "SELECT "+siteColumns+" FROM legacy_sites WHERE id = ?", id))
	if errors.Is(err, sql.ErrNoRows) {
		return s, ErrNotFound
	}
	return s, err
}

// key reads a site's private key, for an uploaded certificate.
func (st *store) siteKey(ctx context.Context, id int64) ([]byte, error) {
	var sealed []byte
	err := st.db.QueryRowContext(ctx, "SELECT key_sealed FROM legacy_sites WHERE id = ?", id).Scan(&sealed)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if sealed == nil {
		return nil, nil
	}
	return st.key.Open(sealed, secret.AAD("legacy_sites", "key_sealed", id))
}

// Save creates (ID 0) or updates a site. An empty KeyPEM on update keeps
// the stored key.
func (st *store) Save(ctx context.Context, s Site) (Site, error) {
	if err := s.normalize(); err != nil {
		return s, err
	}
	if err := st.checkSave(ctx, s); err != nil {
		return s, err
	}
	domains, _ := json.Marshal(s.Domains)
	now := st.now().Unix()
	tx, err := st.db.BeginTx(ctx, nil)
	if err != nil {
		return s, err
	}
	defer tx.Rollback() //nolint:errcheck // a no-op after Commit
	args := []any{s.Name, s.PanelID, string(s.SourceKind), s.SourceURL, s.SourceIP, s.SourcePin, s.OldPath, string(domains),
		s.Port, s.CertFile, s.KeyFile, s.CertPEM, s.Lazy, s.PlainHTTPOK, now}
	if s.ID == 0 {
		res, err := tx.ExecContext(ctx, "INSERT INTO legacy_sites (name, panel_id, source_kind, source_url, source_ip, source_pin, "+
			"old_path, domains, listen_port, cert_file, key_file, cert_pem, lazy, plain_http_ok, updated_at, created_at) "+
			"VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)", append(args, now)...)
		if err != nil {
			return s, fmt.Errorf("add site: %w", err)
		}
		if s.ID, err = res.LastInsertId(); err != nil {
			return s, err
		}
	} else if err := update(ctx, tx, s.ID, args); err != nil {
		return s, err
	}
	if err := st.sealKey(ctx, tx, &s); err != nil {
		return s, err
	}
	if err := tx.Commit(); err != nil {
		return s, err
	}
	return st.Site(ctx, s.ID)
}

// checkSave checks s against the other sites, its panel and its pasted key.
func (st *store) checkSave(ctx context.Context, s Site) error {
	var n int
	if err := st.db.QueryRowContext(ctx, "SELECT count(*) FROM panels WHERE id = ?", s.PanelID).Scan(&n); err != nil {
		return fmt.Errorf("read panel: %w", err)
	}
	if n == 0 {
		return invalid("panelId")
	}
	all, err := st.Sites(ctx)
	if err != nil {
		return err
	}
	for _, o := range all {
		switch {
		case o.ID == s.ID:
		case o.Name == s.Name:
			return ErrNameTaken
		case o.Port == s.Port && o.TLS() != s.TLS():
			return fmt.Errorf("%w: %s", ErrPortMixed, o.Name)
		case s.overlaps(o):
			return fmt.Errorf("%w: %s", ErrOverlap, o.Name)
		}
	}
	return st.checkPastedKey(ctx, s)
}

// checkPastedKey makes sure a pasted certificate parses and has its key:
// the one sent, or the one stored for the site.
func (st *store) checkPastedKey(ctx context.Context, s Site) error {
	if s.CertPEM == "" {
		return nil
	}
	if firstCert([]byte(s.CertPEM)) == nil {
		return invalid("certPem")
	}
	key := []byte(s.KeyPEM)
	if len(key) == 0 && s.ID != 0 {
		stored, err := st.siteKey(ctx, s.ID)
		if err != nil && !errors.Is(err, ErrNotFound) {
			return err
		}
		key = stored
	}
	if len(key) == 0 {
		return invalid("keyPem")
	}
	if _, err := tls.X509KeyPair([]byte(s.CertPEM), key); err != nil {
		return invalid("keyPem")
	}
	return nil
}

func update(ctx context.Context, tx *sql.Tx, id int64, args []any) error {
	res, err := tx.ExecContext(ctx, "UPDATE legacy_sites SET name = ?, panel_id = ?, source_kind = ?, source_url = ?, "+
		"source_ip = ?, source_pin = ?, old_path = ?, domains = ?, listen_port = ?, cert_file = ?, key_file = ?, cert_pem = ?, "+
		"lazy = ?, plain_http_ok = ?, updated_at = ? WHERE id = ?", append(args, id)...)
	if err != nil {
		return fmt.Errorf("update site: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

// sealKey stores a pasted private key sealed, and drops the stored key
// when the site no longer has a pasted certificate.
func (st *store) sealKey(ctx context.Context, tx *sql.Tx, s *Site) error {
	switch {
	case s.CertPEM == "":
		_, err := tx.ExecContext(ctx, "UPDATE legacy_sites SET key_sealed = NULL WHERE id = ?", s.ID)
		return err
	case s.KeyPEM == "":
		return nil
	}
	sealed, err := st.key.Seal([]byte(s.KeyPEM), secret.AAD("legacy_sites", "key_sealed", s.ID))
	if err != nil {
		return err
	}
	s.KeyPEM = ""
	_, err = tx.ExecContext(ctx, "UPDATE legacy_sites SET key_sealed = ? WHERE id = ?", sealed, s.ID)
	return err
}

// Delete removes a site and its tokens.
func (st *store) Delete(ctx context.Context, id int64) error {
	res, err := st.db.ExecContext(ctx, "DELETE FROM legacy_sites WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("delete site: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}
