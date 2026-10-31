package bridge

import (
	"bytes"
	"context"
	"crypto/tls"
	"database/sql"
	"errors"
	"fmt"
	"strconv"

	"github.com/PdYrust/karvan/internal/secret"
)

// maxCertName is the longest certificate name the table takes.
const maxCertName = 64

// adoptCertificates moves the certificate of every site that still has
// its own into a certificates row, one transaction per site. A site
// that fails keeps its own columns, is served from them as before and is
// tried again at the next start.
func (b *Bridge) adoptCertificates(ctx context.Context) {
	list, err := b.st.toAdopt(ctx)
	if err != nil {
		b.Log.Warn("bridge: read sites to adopt", "err", err)
		return
	}
	for _, s := range list {
		b.adoptSite(ctx, s.ID, s.Name)
	}
}

// toAdopt lists the sites still served from their own certificate columns.
func (st *store) toAdopt(ctx context.Context) ([]Site, error) {
	rows, err := st.db.QueryContext(ctx, "SELECT id, name FROM legacy_sites "+
		"WHERE certificate_id IS NULL AND (cert_file <> '' OR cert_pem <> '') ORDER BY id")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Site
	for rows.Next() {
		var s Site
		if err := rows.Scan(&s.ID, &s.Name); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// adoptSite adopts one site's certificate, logging a failure.
func (b *Bridge) adoptSite(ctx context.Context, id int64, name string) {
	if err := b.st.adopt(ctx, id); err != nil {
		b.Log.Warn("bridge: certificate not adopted", "site", name, "err", err)
	}
}

// ownCert is a site's own certificate, as stored before the certificates
// table.
type ownCert struct {
	name, certFile, keyFile, certPEM string
	sealed                           []byte
}

// adopt moves site id's own certificate into a certificates row: the row
// of the same file pair, or a new one. The pair is checked and a pasted
// key must open under its new seal before the site's own columns are
// cleared. A site with nothing to adopt is left alone.
func (st *store) adopt(ctx context.Context, id int64) error {
	tx, err := st.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck // a no-op after Commit
	var c ownCert
	err = tx.QueryRowContext(ctx, "SELECT name, cert_file, key_file, cert_pem, key_sealed FROM legacy_sites "+
		"WHERE id = ? AND certificate_id IS NULL", id).Scan(&c.name, &c.certFile, &c.keyFile, &c.certPEM, &c.sealed)
	if errors.Is(err, sql.ErrNoRows) || err == nil && c.certFile == "" && c.certPEM == "" {
		return nil
	}
	if err != nil {
		return err
	}
	var certID int64
	if c.certFile != "" {
		certID, err = st.adoptFiles(ctx, tx, c)
	} else {
		certID, err = st.adoptPasted(ctx, tx, id, c)
	}
	if err != nil {
		return err
	}
	res, err := tx.ExecContext(ctx, "UPDATE legacy_sites SET certificate_id = ?, cert_file = '', key_file = '', cert_pem = '', "+
		"key_sealed = NULL WHERE id = ? AND certificate_id IS NULL", certID, id)
	if err != nil {
		return fmt.Errorf("point site at certificate: %w", err)
	}
	if n, _ := res.RowsAffected(); n != 1 {
		return errors.New("site changed during adoption")
	}
	return tx.Commit()
}

// adoptFiles checks a file pair and returns the certificate row that has
// it, made when there is none.
func (st *store) adoptFiles(ctx context.Context, tx *sql.Tx, c ownCert) (int64, error) {
	certPEM, err := readSmallFile(c.certFile)
	if err != nil {
		return 0, err
	}
	keyPEM, err := readSmallFile(c.keyFile)
	if err != nil {
		return 0, err
	}
	if _, err := tls.X509KeyPair(certPEM, keyPEM); err != nil {
		return 0, fmt.Errorf("certificate files: %w", err)
	}
	var id int64
	err = tx.QueryRowContext(ctx, "SELECT id FROM certificates WHERE cert_file = ? AND key_file = ? ORDER BY id LIMIT 1",
		c.certFile, c.keyFile).Scan(&id)
	if err == nil || !errors.Is(err, sql.ErrNoRows) {
		return id, err
	}
	return st.addCertificate(ctx, tx, c.name, c.certFile, c.keyFile, "")
}

// adoptPasted opens a pasted key under the site, checks the pair, and
// seals the key again under a new certificate row.
func (st *store) adoptPasted(ctx context.Context, tx *sql.Tx, siteID int64, c ownCert) (int64, error) {
	if c.sealed == nil {
		return 0, errors.New("pasted certificate without a key")
	}
	key, err := st.key.Open(c.sealed, secret.AAD("legacy_sites", "key_sealed", siteID))
	if err != nil {
		return 0, fmt.Errorf("open site key: %w", err)
	}
	if _, err := tls.X509KeyPair([]byte(c.certPEM), key); err != nil {
		return 0, fmt.Errorf("pasted certificate: %w", err)
	}
	id, err := st.addCertificate(ctx, tx, c.name, "", "", c.certPEM)
	if err != nil {
		return 0, err
	}
	aad := secret.AAD("certificates", "key_sealed", id)
	sealed, err := st.key.Seal(key, aad)
	if err != nil {
		return 0, err
	}
	if back, err := st.key.Open(sealed, aad); err != nil || !bytes.Equal(back, key) {
		return 0, errors.New("sealed key does not open")
	}
	if _, err := tx.ExecContext(ctx, "UPDATE certificates SET key_sealed = ? WHERE id = ?", sealed, id); err != nil {
		return 0, fmt.Errorf("store certificate key: %w", err)
	}
	return id, nil
}

// addCertificate inserts a certificate named after base: base itself, or
// base with " 2", " 3" and so on when the name is taken.
func (st *store) addCertificate(ctx context.Context, tx *sql.Tx, base, certFile, keyFile, certPEM string) (int64, error) {
	name, err := freeCertName(ctx, tx, base)
	if err != nil {
		return 0, err
	}
	now := st.now().Unix()
	res, err := tx.ExecContext(ctx, "INSERT INTO certificates (name, cert_file, key_file, cert_pem, created_at, updated_at) "+
		"VALUES (?, ?, ?, ?, ?, ?)", name, certFile, keyFile, certPEM, now, now)
	if err != nil {
		return 0, fmt.Errorf("add certificate: %w", err)
	}
	return res.LastInsertId()
}

func freeCertName(ctx context.Context, tx *sql.Tx, base string) (string, error) {
	for n := 1; ; n++ {
		name := base
		if n > 1 {
			suffix := " " + strconv.Itoa(n)
			r := []rune(base)
			name = string(r[:min(len(r), maxCertName-len(suffix))]) + suffix
		}
		var taken bool
		err := tx.QueryRowContext(ctx, "SELECT EXISTS (SELECT 1 FROM certificates WHERE name = ?)", name).Scan(&taken)
		if err != nil || !taken {
			return name, err
		}
	}
}
