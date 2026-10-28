package bridge

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"sync"
	"time"

	"github.com/PdYrust/karvan/internal/migration/ledger"
	"github.com/PdYrust/karvan/internal/panel/source"
	"github.com/PdYrust/karvan/internal/secret"
)

// ErrNotFound means no such site.
var ErrNotFound = errors.New("no such site")

// ErrOverlap means another rule has the same port, host key and path key,
// so no request could tell the two apart (D52).
var ErrOverlap = errors.New("another rule answers the same port, domain and path")

// ErrNameTaken means another site has this name.
var ErrNameTaken = errors.New("another site has this name")

// ErrPortMixed means the port already serves sites the other way: one
// port is either HTTPS for all its sites or plain HTTP for all of them.
var ErrPortMixed = errors.New("the port has sites with and without a certificate")

// store keeps sites, verified tokens and closed users in Karvan's DB.
type store struct {
	db         *sql.DB
	key        *secret.Key
	now        func() time.Time
	panelPorts func() []int // Karvan's own web panel ports; nil: none known
	// saveMu runs one save of rules, domains, their certificates and pins
	// at a time, so no two can both pass their checks before either is
	// written.
	saveMu sync.Mutex
}

// siteColumns read a site with the certificate it is served with: its
// certificates row once adopted (D54), else its own columns.
const siteColumns = "s.id, s.name, s.panel_id, s.source_kind, s.source_url, s.source_ip, s.source_pin, s.old_path, s.domains, " +
	"s.listen_port, COALESCE(c.cert_file, s.cert_file), COALESCE(c.key_file, s.key_file), COALESCE(c.cert_pem, s.cert_pem), " +
	"COALESCE(c.key_sealed, s.key_sealed) IS NOT NULL, s.lazy, s.plain_http_ok, s.mode, COALESCE(s.certificate_id, 0), " +
	"COALESCE(c.name, '')" +
	" FROM legacy_sites s LEFT JOIN certificates c ON c.id = s.certificate_id"

func scanSite(row interface{ Scan(...any) error }) (Site, error) {
	var s Site
	var domains string
	err := row.Scan(&s.ID, &s.Name, &s.PanelID, &s.SourceKind, &s.SourceURL, &s.SourceIP, &s.SourcePin, &s.OldPath,
		&domains, &s.Port, &s.CertFile, &s.KeyFile, &s.CertPEM, &s.HasKey, &s.Lazy, &s.PlainHTTPOK, &s.Mode, &s.CertificateID,
		&s.certName)
	if err != nil {
		return s, err
	}
	if err := json.Unmarshal([]byte(domains), &s.Domains); err != nil {
		return s, fmt.Errorf("site %d domains: %w", s.ID, err)
	}
	https := s.TLS()
	s.HTTPS = &https
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
	rows, err := st.db.QueryContext(ctx, "SELECT "+siteColumns+" ORDER BY s.id")
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
	s, err := scanSite(st.db.QueryRowContext(ctx, "SELECT "+siteColumns+" WHERE s.id = ?", id))
	if errors.Is(err, sql.ErrNoRows) {
		return s, ErrNotFound
	}
	return s, err
}

// siteKey reads the private key of a site's pasted certificate: from its
// certificates row once adopted, else from the site.
func (st *store) siteKey(ctx context.Context, id int64) ([]byte, error) {
	var own, adopted []byte
	var certID sql.NullInt64
	err := st.db.QueryRowContext(ctx, "SELECT s.key_sealed, s.certificate_id, c.key_sealed FROM legacy_sites s "+
		"LEFT JOIN certificates c ON c.id = s.certificate_id WHERE s.id = ?", id).Scan(&own, &certID, &adopted)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return nil, ErrNotFound
	case err != nil:
		return nil, err
	case certID.Valid && adopted != nil:
		return st.key.Open(adopted, secret.AAD("certificates", "key_sealed", certID.Int64))
	case certID.Valid || own == nil:
		return nil, nil
	}
	return st.key.Open(own, secret.AAD("legacy_sites", "key_sealed", id))
}

// Save creates (ID 0) or updates a site: a rule of batches takes its
// derived fields from them, every domain must be registered, and the
// certificate is derived from the domains and https (D48, D63). The
// site's own certificate columns are cleared in the same transaction; a
// site still served from them is saved only once its domains have
// certificates (ErrNotAdopted, D54).
func (st *store) Save(ctx context.Context, s Site) (Site, error) {
	st.saveMu.Lock()
	defer st.saveMu.Unlock()
	var stored *Site
	if s.ID != 0 {
		old, err := st.Site(ctx, s.ID)
		if err != nil {
			return s, err
		}
		stored = &old
	}
	if err := checkMode(&s, stored); err != nil {
		return s, err
	}
	if s.Mode == ModeBatches {
		if err := st.fromBatches(ctx, &s); err != nil {
			return s, err
		}
	}
	if err := s.normalize(); err != nil {
		return s, err
	}
	in := ruleCert{self: RuleRef{Kind: RulePanel, ID: s.ID}, port: s.Port, domains: s.Domains, https: s.HTTPS}
	if stored != nil {
		in.stored, in.notAdopted = stored.HTTPS, stored.TLS() && stored.CertificateID == 0
	}
	id, err := st.derive(ctx, in)
	if err != nil {
		return s, err
	}
	s.CertificateID = id
	if err := st.pickCertificate(ctx, &s); err != nil {
		return s, err
	}
	if err := st.checkSave(ctx, s); err != nil {
		return s, err
	}
	if err := st.write(ctx, &s); err != nil {
		return s, err
	}
	return st.Site(ctx, s.ID)
}

// write stores a checked site and its batches, and drops the pins no
// rule holds any more, in one transaction.
func (st *store) write(ctx context.Context, s *Site) error {
	domains, _ := json.Marshal(s.Domains)
	now := st.now().Unix()
	tx, err := st.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck // a no-op after Commit
	args := []any{s.Name, s.PanelID, string(s.SourceKind), s.SourceURL, s.SourceIP, s.SourcePin, s.OldPath, string(domains),
		s.Port, s.Lazy, s.PlainHTTPOK, nullID(s.CertificateID), now}
	if s.ID == 0 {
		res, err := tx.ExecContext(ctx, "INSERT INTO legacy_sites (name, panel_id, source_kind, source_url, source_ip, source_pin, "+
			"old_path, domains, listen_port, lazy, plain_http_ok, certificate_id, updated_at, mode, created_at) "+
			"VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)", append(args, s.Mode, now)...)
		if err != nil {
			return fmt.Errorf("add site: %w", err)
		}
		if s.ID, err = res.LastInsertId(); err != nil {
			return err
		}
	} else if err := update(ctx, tx, s.ID, args); err != nil {
		return err
	}
	if err := linkBatches(ctx, tx, *s); err != nil {
		return err
	}
	if err := prunePins(ctx, tx); err != nil {
		return err
	}
	return tx.Commit()
}

// checkSave checks s against its panel, Karvan's own port and every other
// panel and redirect rule (D52). The bridge store writes both rule
// tables, so a request reaches at most one rule.
func (st *store) checkSave(ctx context.Context, s Site) error {
	var n int
	if err := st.db.QueryRowContext(ctx, "SELECT count(*) FROM panels WHERE id = ?", s.PanelID).Scan(&n); err != nil {
		return fmt.Errorf("read panel: %w", err)
	}
	if n == 0 {
		return invalid("panelId")
	}
	if st.panelPorts != nil && slices.Contains(st.panelPorts(), s.Port) {
		return ErrPortTaken
	}
	all, err := st.rules(ctx)
	if err != nil {
		return err
	}
	r := ruleOf(s)
	for _, o := range all {
		if o.ref.Kind == RulePanel && o.ref.ID == s.ID {
			continue
		}
		if err := r.conflict(o); err != nil {
			return err
		}
	}
	return nil
}

// update rewrites a site; its own certificate columns are cleared, since
// the certificate is its certificate_id from now on.
func update(ctx context.Context, tx *sql.Tx, id int64, args []any) error {
	res, err := tx.ExecContext(ctx, "UPDATE legacy_sites SET name = ?, panel_id = ?, source_kind = ?, source_url = ?, "+
		"source_ip = ?, source_pin = ?, old_path = ?, domains = ?, listen_port = ?, lazy = ?, plain_http_ok = ?, "+
		"certificate_id = ?, updated_at = ?, cert_file = '', key_file = '', cert_pem = '', key_sealed = NULL WHERE id = ?",
		append(args, id)...)
	if err != nil {
		return fmt.Errorf("update site: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

// Delete removes a site and its tokens, and the pins no rule holds any
// more.
func (st *store) Delete(ctx context.Context, id int64) error {
	return st.deleteRule(ctx, "DELETE FROM legacy_sites WHERE id = ?", id)
}

// deleteRule runs the delete q of rule id and drops the pins no rule
// holds any more, in one transaction.
func (st *store) deleteRule(ctx context.Context, q string, id int64) error {
	tx, err := st.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck // a no-op after Commit
	res, err := tx.ExecContext(ctx, q, id)
	if err != nil {
		return fmt.Errorf("delete rule: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	if err := prunePins(ctx, tx); err != nil {
		return err
	}
	return tx.Commit()
}
