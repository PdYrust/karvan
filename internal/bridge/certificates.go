package bridge

import (
	"cmp"
	"context"
	"crypto/tls"
	"crypto/x509"
	"database/sql"
	"errors"
	"fmt"
	"path/filepath"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/PdYrust/karvan/internal/secret"
)

// Limits of a certificate as sent.
const (
	maxCertPaste = 64 << 10 // a pasted certificate or key
	maxCertPath  = 4096
)

// ErrInvalidCert means a certificate as sent is wrong; a
// *CertInvalidError says which field and why.
var ErrInvalidCert = errors.New("invalid certificate")

// CertInvalidError names the wrong field by its JSON key and the reason:
// "format", "unreadable", "no_cert" or "key_mismatch".
type CertInvalidError struct{ Field, Reason string }

func (e *CertInvalidError) Error() string {
	return fmt.Sprintf("%v: %s %s", ErrInvalidCert, e.Field, e.Reason)
}

// Is makes errors.Is(err, ErrInvalidCert) hold.
func (e *CertInvalidError) Is(target error) bool { return target == ErrInvalidCert }

func certInvalid(field, reason string) error { return &CertInvalidError{Field: field, Reason: reason} }

// CertificateInput is a certificate as sent: a file pair on this server
// or a pasted certificate with its key. The key goes in and never out.
type CertificateInput struct {
	Name     string `json:"name"`
	CertFile string `json:"certFile"`
	KeyFile  string `json:"keyFile"`
	CertPEM  string `json:"certPem"`
	KeyPEM   string `json:"keyPem"`
}

// certRow is a certificates row as stored.
type certRow struct {
	id                                   int64
	name, certFile, keyFile, pem, origin string
	sealed                               []byte
}

func scanCertRow(row interface{ Scan(...any) error }) (certRow, error) {
	var r certRow
	err := row.Scan(&r.id, &r.name, &r.certFile, &r.keyFile, &r.pem, &r.sealed, &r.origin)
	return r, err
}

const certRowColumns = "SELECT id, name, cert_file, key_file, cert_pem, key_sealed, origin FROM certificates"

// ErrManaged means a Let's Encrypt certificate was to be changed by hand;
// Karvan renews it in place.
var ErrManaged = errors.New("the certificate is renewed by Karvan")

// material reads a certificate and its key: the files, or the pasted
// certificate with its key opened. What cannot be read is nil.
func (b *Bridge) material(r certRow) (certPEM, keyPEM []byte) {
	if r.certFile != "" {
		certPEM, _ = readSmallFile(r.certFile)
		keyPEM, _ = readSmallFile(r.keyFile)
		return certPEM, keyPEM
	}
	if r.sealed == nil {
		return []byte(r.pem), nil
	}
	key, err := b.Key.Open(r.sealed, secret.AAD("certificates", "key_sealed", r.id))
	if err != nil {
		b.Log.Warn("bridge: read certificate key", "certificate", r.name, "err", err)
	}
	return []byte(r.pem), key
}

// certificates reads every certificate by name, with its facts.
func (b *Bridge) certificates(ctx context.Context) ([]Certificate, error) {
	rows, err := b.DB.QueryContext(ctx, certRowColumns+" ORDER BY name, id")
	if err != nil {
		return nil, fmt.Errorf("read certificates: %w", err)
	}
	defer rows.Close()
	var list []certRow
	for rows.Next() {
		r, err := scanCertRow(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, r)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	out := make([]Certificate, 0, len(list))
	for _, r := range list {
		certPEM, keyPEM := b.material(r)
		c := Certificate{ID: r.id, Name: r.name, Kind: CertKindFile, Origin: r.origin, CertFile: r.certFile, KeyFile: r.keyFile,
			HasKey: len(keyPEM) > 0, Facts: inspect(certPEM, keyPEM, nil, b.Now()), Domains: []CertDomain{}, Rules: []CertRule{},
			leaf: firstCert(certPEM)}
		if r.certFile == "" {
			c.Kind, c.CertPEM = CertKindPasted, r.pem
		}
		out = append(out, c)
	}
	return out, nil
}

// Certificates lists every certificate by name, with its facts, the
// domains whose certificate it is and the rules that use it; each domain
// and rule carries its domain_mismatch.
func (b *Bridge) Certificates(ctx context.Context) ([]Certificate, error) {
	certs, err := b.certificates(ctx)
	if err != nil {
		return nil, err
	}
	rules, err := b.st.rules(ctx)
	if err != nil {
		return nil, err
	}
	byCert := map[int64][]Ref{}
	err = b.st.each(ctx, "SELECT certificate_id, id, name FROM domains WHERE certificate_id IS NOT NULL ORDER BY name",
		func(r *sql.Rows) error {
			var id int64
			var d Ref
			err := r.Scan(&id, &d.ID, &d.Name)
			byCert[id] = append(byCert[id], d)
			return err
		})
	if err != nil {
		return nil, fmt.Errorf("read domain certificates: %w", err)
	}
	for i := range certs {
		c := &certs[i]
		for _, d := range byCert[c.ID] {
			w := []string{}
			if c.leaf != nil && !certCovers(c.leaf, d.Name) {
				w = append(w, WarnDomain)
			}
			c.Domains = append(c.Domains, CertDomain{Ref: d, Warnings: w})
		}
		for _, r := range rules {
			if r.certID != c.ID {
				continue
			}
			w := []string{}
			if c.leaf != nil && slices.ContainsFunc(r.domains, func(d string) bool { return !certCovers(c.leaf, d) }) {
				w = append(w, WarnDomain)
			}
			c.Rules = append(c.Rules, CertRule{RuleRef: r.ref, Warnings: w})
		}
	}
	return certs, nil
}

// SaveCertificate adds (id 0) or changes a certificate, then reloads the
// rules so the ones using it serve the change.
func (b *Bridge) SaveCertificate(ctx context.Context, id int64, in CertificateInput) (Certificate, error) {
	id, err := b.st.saveCertificate(ctx, id, in)
	if err != nil {
		return Certificate{}, err
	}
	if err := b.Reload(ctx); err != nil {
		return Certificate{}, err
	}
	list, err := b.Certificates(ctx)
	if err != nil {
		return Certificate{}, err
	}
	for _, c := range list {
		if c.ID == id {
			return c, nil
		}
	}
	return Certificate{}, ErrNotFound
}

// DeleteCertificate removes a certificate no domain, pin or rule uses, in
// one statement.
func (b *Bridge) DeleteCertificate(ctx context.Context, id int64) error {
	b.st.saveMu.Lock() // a domain or rule save may be about to point at it
	defer b.st.saveMu.Unlock()
	res, err := b.DB.ExecContext(ctx, "DELETE FROM certificates WHERE id = ? "+
		"AND NOT EXISTS (SELECT 1 FROM legacy_sites WHERE certificate_id = certificates.id) "+
		"AND NOT EXISTS (SELECT 1 FROM link_redirects WHERE certificate_id = certificates.id) "+
		"AND NOT EXISTS (SELECT 1 FROM domains WHERE certificate_id = certificates.id) "+
		"AND NOT EXISTS (SELECT 1 FROM domain_pins WHERE certificate_id = certificates.id)", id)
	if err != nil {
		return fmt.Errorf("delete certificate: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 1 {
		return nil
	}
	var found bool
	if err := b.DB.QueryRowContext(ctx, "SELECT EXISTS (SELECT 1 FROM certificates WHERE id = ?)", id).Scan(&found); err != nil {
		return err
	}
	if !found {
		return ErrNotFound
	}
	rules, err := b.st.rules(ctx)
	if err != nil {
		return err
	}
	domains := []Ref{}
	err = b.st.each(ctx, fmt.Sprintf("SELECT id, name FROM domains WHERE certificate_id = %d OR id IN "+
		"(SELECT domain_id FROM domain_pins WHERE certificate_id = %d) ORDER BY name", id, id), func(r *sql.Rows) error {
		var d Ref
		err := r.Scan(&d.ID, &d.Name)
		domains = append(domains, d)
		return err
	})
	if err != nil {
		return fmt.Errorf("read domains of certificate: %w", err)
	}
	return &InUseError{What: "certificate", Rules: usedBy(rules, func(r rule) bool { return r.certID == id }), Domains: domains}
}

// certParts is a checked certificate: a file pair, or a pasted one with
// the key to seal (nil: keep the stored key), and its first certificate.
type certParts struct {
	certFile, keyFile, certPEM string
	key                        []byte
	leaf                       *x509.Certificate
}

// saveCertificate checks in and writes it; a pasted key is sealed under
// the row's own id in the same transaction.
func (st *store) saveCertificate(ctx context.Context, id int64, in CertificateInput) (int64, error) {
	name, err := certName(in.Name)
	if err != nil {
		return 0, err
	}
	tx, err := st.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback() //nolint:errcheck // a no-op after Commit
	var stored []byte
	if id != 0 {
		if stored, err = st.storedKey(ctx, tx, id, in.CertPEM != "" && in.KeyPEM == ""); err != nil {
			return 0, err
		}
	}
	p, err := checkCertParts(in, stored)
	if err != nil {
		return 0, err
	}
	var taken bool
	if err := tx.QueryRowContext(ctx, "SELECT EXISTS (SELECT 1 FROM certificates WHERE name = ? AND id <> ?)", name, id).
		Scan(&taken); err != nil || taken {
		return 0, cmp.Or(err, ErrNameTaken)
	}
	if id, err = writeCertificate(ctx, tx, id, name, p, st.now().Unix()); err != nil {
		return 0, err
	}
	if err := st.sealCertKey(ctx, tx, id, p.key); err != nil {
		return 0, err
	}
	return id, tx.Commit()
}

// sealCertKey seals key under certificates row id and stores it; nil
// keeps the stored key.
func (st *store) sealCertKey(ctx context.Context, tx *sql.Tx, id int64, key []byte) error {
	if key == nil {
		return nil
	}
	sealed, err := st.key.Seal(key, secret.AAD("certificates", "key_sealed", id))
	if err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, "UPDATE certificates SET key_sealed = ? WHERE id = ?", sealed, id); err != nil {
		return fmt.Errorf("store certificate key: %w", err)
	}
	return nil
}

// storedKey reads the key of certificate id when need is set and it is
// pasted, so a change without a key keeps it; ErrNotFound for no such row.
func (st *store) storedKey(ctx context.Context, tx *sql.Tx, id int64, need bool) ([]byte, error) {
	r, err := scanCertRow(tx.QueryRowContext(ctx, certRowColumns+" WHERE id = ?", id))
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return nil, ErrNotFound
	case err != nil:
		return nil, err
	case r.origin == OriginACME:
		return nil, ErrManaged
	case !need || r.pem == "" || r.sealed == nil:
		return nil, nil
	}
	key, err := st.key.Open(r.sealed, secret.AAD("certificates", "key_sealed", id))
	if err != nil {
		return nil, fmt.Errorf("open certificate key: %w", err)
	}
	return key, nil
}

// writeCertificate inserts (id 0) or updates the row; a file pair drops
// any stored key.
func writeCertificate(ctx context.Context, tx *sql.Tx, id int64, name string, p certParts, now int64) (int64, error) {
	if id == 0 {
		res, err := tx.ExecContext(ctx, "INSERT INTO certificates (name, cert_file, key_file, cert_pem, created_at, updated_at) "+
			"VALUES (?, ?, ?, ?, ?, ?)", name, p.certFile, p.keyFile, p.certPEM, now, now)
		if err != nil {
			return 0, fmt.Errorf("add certificate: %w", err)
		}
		return res.LastInsertId()
	}
	_, err := tx.ExecContext(ctx, "UPDATE certificates SET name = ?, cert_file = ?, key_file = ?, cert_pem = ?, updated_at = ?, "+
		"key_sealed = CASE WHEN ? = '' THEN NULL ELSE key_sealed END WHERE id = ?", name, p.certFile, p.keyFile, p.certPEM, now,
		p.certPEM, id)
	if err != nil {
		return 0, fmt.Errorf("change certificate: %w", err)
	}
	return id, nil
}

// certName trims a certificate name and checks it: 1 to 64 characters,
// no control characters.
func certName(s string) (string, error) {
	s = strings.TrimSpace(s)
	if n := utf8.RuneCountInString(s); n < 1 || n > maxCertName || !utf8.ValidString(s) ||
		strings.ContainsFunc(s, unicode.IsControl) {
		return "", certInvalid("name", "format")
	}
	return s, nil
}

// checkCertParts checks a file pair or a pasted certificate; stored is
// the key a pasted certificate keeps when none is sent.
func checkCertParts(in CertificateInput, stored []byte) (certParts, error) {
	files := in.CertFile != "" || in.KeyFile != ""
	switch {
	case files && (in.CertPEM != "" || in.KeyPEM != ""):
		return certParts{}, certInvalid("certPem", "format") // files or a paste, not both
	case files:
		return checkCertFiles(in.CertFile, in.KeyFile)
	}
	return checkPasted(in.CertPEM, in.KeyPEM, stored)
}

func checkCertFiles(certFile, keyFile string) (certParts, error) {
	for _, f := range [...]struct{ field, path string }{{"certFile", certFile}, {"keyFile", keyFile}} {
		if f.path == "" || len(f.path) > maxCertPath || strings.ContainsRune(f.path, 0) || !filepath.IsAbs(f.path) ||
			filepath.Clean(f.path) != f.path {
			return certParts{}, certInvalid(f.field, "format")
		}
	}
	certPEM, err := readSmallFile(certFile)
	if err != nil {
		return certParts{}, certInvalid("certFile", "unreadable")
	}
	if firstCert(certPEM) == nil {
		return certParts{}, certInvalid("certFile", "no_cert")
	}
	keyPEM, err := readSmallFile(keyFile)
	if err != nil {
		return certParts{}, certInvalid("keyFile", "unreadable")
	}
	if _, err := tls.X509KeyPair(certPEM, keyPEM); err != nil {
		return certParts{}, certInvalid("keyFile", "key_mismatch")
	}
	return certParts{certFile: certFile, keyFile: keyFile, leaf: firstCert(certPEM)}, nil
}

func checkPasted(certPEM, keyPEM string, stored []byte) (certParts, error) {
	switch {
	case len(certPEM) > maxCertPaste:
		return certParts{}, certInvalid("certPem", "format")
	case len(keyPEM) > maxCertPaste:
		return certParts{}, certInvalid("keyPem", "format")
	case firstCert([]byte(certPEM)) == nil:
		return certParts{}, certInvalid("certPem", "no_cert")
	}
	p := certParts{certPEM: certPEM, leaf: firstCert([]byte(certPEM))}
	key := stored
	if keyPEM != "" {
		key, p.key = []byte(keyPEM), []byte(keyPEM)
	}
	if len(key) == 0 {
		return certParts{}, certInvalid("keyPem", "format")
	}
	if _, err := tls.X509KeyPair([]byte(certPEM), key); err != nil {
		return certParts{}, certInvalid("keyPem", "key_mismatch")
	}
	return p, nil
}
