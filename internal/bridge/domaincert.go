package bridge

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"slices"
	"strings"
	"unicode/utf8"
)

// Ways a domain gets its certificate (D61).
const (
	ChoiceStored = "stored" // a certificates row that covers the name
	ChoiceFile   = "file"   // a file pair on this server
	ChoicePasted = "pasted" // a pasted certificate with its key
	ChoiceNone   = "none"   // plain HTTP, confirmed with plainOk
	ChoiceACME   = "acme"   // Let's Encrypt (D66)
)

// ReasonNotCovered means a domain's certificate does not cover its name.
const ReasonNotCovered = "not_covered"

// CertChoice is how a domain gets its certificate (D61). Only the fields
// of its kind are read; a key goes in and never out.
type CertChoice struct {
	Kind          string `json:"kind"`
	CertificateID int64  `json:"certificateId"`
	CertFile      string `json:"certFile"`
	KeyFile       string `json:"keyFile"`
	CertPEM       string `json:"certPem"`
	KeyPEM        string `json:"keyPem"`
	Name          string `json:"name"` // the new certificates row's name; default the domain's
	PlainOK       bool   `json:"plainOk"`
	Method        string `json:"method"`
	AgreeTOS      bool   `json:"agreeTos"`
}

// ErrInvalidDomain means a domain or its certificate choice is wrong; a
// *DomainInvalidError says which field.
var ErrInvalidDomain = errors.New("invalid domain")

// DomainInvalidError names the wrong field: "name", "certificate" or
// "plainOk".
type DomainInvalidError struct{ Field string }

func (e *DomainInvalidError) Error() string { return fmt.Sprintf("%v: %s", ErrInvalidDomain, e.Field) }

// Is makes errors.Is(err, ErrInvalidDomain) hold.
func (e *DomainInvalidError) Is(target error) bool { return target == ErrInvalidDomain }

// ErrInvalidACME means a Let's Encrypt request is wrong; an
// *ACMEInvalidError says which field and why.
var ErrInvalidACME = errors.New("invalid Let's Encrypt request")

// ACMEInvalidError names the wrong field and the reason, when there is one.
type ACMEInvalidError struct{ Field, Reason string }

func (e *ACMEInvalidError) Error() string { return fmt.Sprintf("%v: %s", ErrInvalidACME, e.Field) }

// Is makes errors.Is(err, ErrInvalidACME) hold.
func (e *ACMEInvalidError) Is(target error) bool { return target == ErrInvalidACME }

// DomainCert is a domain's certificate with its facts.
type DomainCert struct {
	ID     int64     `json:"id"`
	Name   string    `json:"name"`
	Origin string    `json:"origin"`
	Facts  CertFacts `json:"facts"`
}

// DomainPin is a port that keeps serving another certificate for the
// domain (D62).
type DomainPin struct {
	Port        int `json:"port"`
	Certificate Ref `json:"certificate"`
}

// Domain is a domains row: its certificate, the stored certificates that
// cover it, its pins, the rules that hold it and its warnings (D48, D50,
// D61).
type Domain struct {
	ID           int64       `json:"id"`
	Name         string      `json:"name"`
	Certificate  *DomainCert `json:"certificate"`
	Certificates []Ref       `json:"certificates"`
	Pins         []DomainPin `json:"pins"`
	ACME         *DomainACME `json:"acme"`
	Rules        []RuleRef   `json:"rules"`
	Warnings     []string    `json:"warnings"`
	certID       int64
	filled       bool
}

// NormalDomain is a domain name in its stored form, or "" when it is not
// a host name, one leading wildcard or an IP address.
func NormalDomain(name string) string { return normalDomain(name) }

func normalDomain(name string) string {
	name = strings.ToLower(strings.TrimSuffix(strings.TrimSpace(name), "."))
	if len(name) > 253 || !validPattern(name) {
		return ""
	}
	return name
}

// Domains lists every domain by name.
func (b *Bridge) Domains(ctx context.Context) ([]Domain, error) { return b.domains(ctx, 0) }

// domains reads the domain id, or every domain for 0, with its views.
func (b *Bridge) domains(ctx context.Context, id int64) ([]Domain, error) {
	certs, err := b.certificates(ctx)
	if err != nil {
		return nil, err
	}
	rules, err := b.st.rules(ctx)
	if err != nil {
		return nil, err
	}
	out := []Domain{}
	err = b.st.each(ctx, fmt.Sprintf("SELECT id, name, COALESCE(certificate_id, 0), cert_filled FROM domains WHERE %d IN (0, id) "+
		"ORDER BY name", id), func(r *sql.Rows) error {
		var d Domain
		err := r.Scan(&d.ID, &d.Name, &d.certID, &d.filled)
		out = append(out, d)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("read domains: %w", err)
	}
	pins, err := b.st.pins(ctx)
	if err != nil {
		return nil, err
	}
	acme, err := b.st.acmeRows(ctx, id)
	if err != nil {
		return nil, err
	}
	for i := range out {
		fillDomain(&out[i], certs, rules, pins[out[i].ID])
		addACME(&out[i], acme[out[i].ID])
	}
	return out, nil
}

// pins reads every pin by domain id, by port.
func (st *store) pins(ctx context.Context) (map[int64][]DomainPin, error) {
	out := map[int64][]DomainPin{}
	err := st.each(ctx, "SELECT p.domain_id, p.port, c.id, c.name FROM domain_pins p JOIN certificates c ON c.id = p.certificate_id "+
		"ORDER BY p.domain_id, p.port", func(r *sql.Rows) error {
		var id int64
		var p DomainPin
		err := r.Scan(&id, &p.Port, &p.Certificate.ID, &p.Certificate.Name)
		out[id] = append(out[id], p)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("read pins: %w", err)
	}
	return out, nil
}

// fillDomain adds a domain's certificate, the certificates that cover it,
// its rules and its warnings (D50): its certificate's warnings and
// domain_mismatch, no_certificate without one (certificate_not_adopted
// while a rule served from its own certificate keeps it unfilled), and
// cert_conflict with pins.
func fillDomain(d *Domain, certs []Certificate, rules []rule, pins []DomainPin) {
	d.Certificates, d.Pins, d.Warnings = []Ref{}, append([]DomainPin{}, pins...), []string{}
	for _, c := range certs {
		if c.leaf != nil && certCovers(c.leaf, d.Name) {
			d.Certificates = append(d.Certificates, Ref{ID: c.ID, Name: c.Name})
		}
		if c.ID == d.certID {
			f := c.Facts
			f.Warnings = slices.Clone(f.Warnings)
			if c.leaf != nil && !certCovers(c.leaf, d.Name) {
				f.Warnings = append(f.Warnings, WarnDomain)
			}
			d.Certificate = &DomainCert{ID: c.ID, Name: c.Name, Origin: c.Origin, Facts: f}
			d.Warnings = append(d.Warnings, f.Warnings...)
		}
	}
	d.Rules = usedBy(rules, func(r rule) bool { return slices.Contains(r.domains, d.Name) })
	switch {
	case !d.filled:
		d.Warnings = append(d.Warnings, WarnNotAdopted)
	case d.Certificate == nil:
		d.Warnings = append(d.Warnings, WarnNoCertificate)
	}
	if len(pins) > 0 {
		d.Warnings = append(d.Warnings, WarnCertConflict)
	}
	slices.Sort(d.Warnings)
	d.Warnings = slices.Compact(d.Warnings)
}

// Domain reads one domain with its views.
func (b *Bridge) Domain(ctx context.Context, id int64) (Domain, error) { return b.domain(ctx, id) }

func (b *Bridge) domain(ctx context.Context, id int64) (Domain, error) {
	list, err := b.domains(ctx, id)
	if err != nil {
		return Domain{}, err
	}
	if len(list) == 0 {
		return Domain{}, ErrNotFound
	}
	return list[0], nil
}

// AddDomain stores a domain name in its normal form with its certificate
// (D61), in one transaction.
func (b *Bridge) AddDomain(ctx context.Context, name string, ch CertChoice) (Domain, error) {
	if name = normalDomain(name); name == "" {
		return Domain{}, &DomainInvalidError{Field: "name"}
	}
	id, err := b.st.addDomain(ctx, name, ch)
	if err != nil {
		return Domain{}, err
	}
	return b.domain(ctx, id)
}

func (st *store) addDomain(ctx context.Context, name string, ch CertChoice) (int64, error) {
	st.saveMu.Lock()
	defer st.saveMu.Unlock()
	tx, err := st.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback() //nolint:errcheck // a no-op after Commit
	now := st.now().Unix()
	res, err := tx.ExecContext(ctx, "INSERT INTO domains (name, created_at, updated_at) VALUES (?, ?, ?) ON CONFLICT (name) DO NOTHING",
		name, now, now)
	if err != nil {
		return 0, fmt.Errorf("add domain: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return 0, ErrNameTaken
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}
	certID, err := st.choose(ctx, tx, name, ch)
	if err != nil {
		return 0, err
	}
	if _, err := tx.ExecContext(ctx, "UPDATE domains SET certificate_id = ? WHERE id = ?", nullID(certID), id); err != nil {
		return 0, fmt.Errorf("set domain certificate: %w", err)
	}
	return id, tx.Commit()
}

// choose resolves a certificate choice for the domain name inside tx: the
// stored row, the row of the same file pair, or a new row; 0 for none.
// The certificate must cover the name.
func (st *store) choose(ctx context.Context, tx *sql.Tx, name string, ch CertChoice) (int64, error) {
	switch ch.Kind {
	case ChoiceNone:
		if !ch.PlainOK {
			return 0, &DomainInvalidError{Field: "plainOk"}
		}
		return 0, nil
	case ChoiceStored:
		return st.chooseStored(ctx, tx, name, ch.CertificateID)
	case ChoiceFile:
		p, err := checkCertFiles(ch.CertFile, ch.KeyFile)
		if err != nil {
			return 0, err
		}
		if !certCovers(p.leaf, name) {
			return 0, certInvalid("certificate", ReasonNotCovered)
		}
		var id int64
		err = tx.QueryRowContext(ctx, "SELECT id FROM certificates WHERE cert_file = ? AND key_file = ? ORDER BY id LIMIT 1",
			p.certFile, p.keyFile).Scan(&id)
		if !errors.Is(err, sql.ErrNoRows) {
			return id, err // the stored row of this pair, so the scan's added flag keeps matching it
		}
		return st.insertChosen(ctx, tx, name, ch.Name, p)
	case ChoicePasted:
		p, err := checkPasted(ch.CertPEM, ch.KeyPEM, nil)
		if err != nil {
			return 0, err
		}
		if !certCovers(p.leaf, name) {
			return 0, certInvalid("certificate", ReasonNotCovered)
		}
		return st.insertChosen(ctx, tx, name, ch.Name, p)
	case ChoiceACME:
		return 0, &ACMEInvalidError{Field: "kind"} // an order is started by certorder, never chosen here
	}
	return 0, &DomainInvalidError{Field: "certificate"}
}

func (st *store) chooseStored(ctx context.Context, tx *sql.Tx, name string, id int64) (int64, error) {
	r, err := scanCertRow(tx.QueryRowContext(ctx, certRowColumns+" WHERE id = ?", id))
	if errors.Is(err, sql.ErrNoRows) {
		return 0, certInvalid("certificateId", "")
	}
	if err != nil {
		return 0, fmt.Errorf("read certificate: %w", err)
	}
	certPEM := []byte(r.pem)
	if r.certFile != "" {
		certPEM, _ = readSmallFile(r.certFile)
	}
	if leaf := firstCert(certPEM); leaf == nil || !certCovers(leaf, name) {
		return 0, certInvalid("certificate", ReasonNotCovered)
	}
	return id, nil
}

// insertChosen adds the chosen certificate as a new row: named as sent,
// else after the domain, with " 2" and so on when that name is taken.
func (st *store) insertChosen(ctx context.Context, tx *sql.Tx, domain, name string, p certParts) (int64, error) {
	if name != "" {
		n, err := certName(name)
		if err != nil {
			return 0, err
		}
		var taken bool
		if err := tx.QueryRowContext(ctx, "SELECT EXISTS (SELECT 1 FROM certificates WHERE name = ?)", n).Scan(&taken); err != nil {
			return 0, err
		}
		if taken {
			return 0, ErrNameTaken
		}
		return st.insertCert(ctx, tx, n, p)
	}
	n, err := freeCertName(ctx, tx, certNameOf(domain))
	if err != nil {
		return 0, err
	}
	return st.insertCert(ctx, tx, n, p)
}

// insertCert writes a new certificates row; a pasted key is sealed under
// the new id in the same transaction (D49).
func (st *store) insertCert(ctx context.Context, tx *sql.Tx, name string, p certParts) (int64, error) {
	id, err := writeCertificate(ctx, tx, 0, name, p, st.now().Unix())
	if err != nil {
		return 0, err
	}
	return id, st.sealCertKey(ctx, tx, id, p.key)
}

// SetDomainCertificate changes a domain's certificate (D65): its pins go,
// every HTTPS rule that holds it takes it again, and the bridge reloads.
// None is refused while an HTTPS rule holds the domain.
func (b *Bridge) SetDomainCertificate(ctx context.Context, id int64, ch CertChoice) (Domain, error) {
	if err := b.st.setDomainCert(ctx, id, ch); err != nil {
		return Domain{}, err
	}
	if err := b.Reload(ctx); err != nil {
		return Domain{}, err
	}
	return b.domain(ctx, id)
}

func (st *store) setDomainCert(ctx context.Context, id int64, ch CertChoice) error {
	st.saveMu.Lock()
	defer st.saveMu.Unlock()
	tx, err := st.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck // a no-op after Commit
	var name string
	err = tx.QueryRowContext(ctx, "SELECT name FROM domains WHERE id = ?", id).Scan(&name)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	certID, err := st.choose(ctx, tx, name, ch)
	if err != nil {
		return err
	}
	rules, err := st.rules(ctx)
	if err != nil {
		return err
	}
	holders := slices.DeleteFunc(slices.Clone(rules), func(r rule) bool { return !slices.Contains(r.domains, name) || !r.tls })
	if certID == 0 && len(holders) > 0 {
		return &InUseError{What: "domain", Rules: usedBy(holders, func(rule) bool { return true })}
	}
	now := st.now().Unix()
	for _, q := range []struct {
		sql  string
		args []any
	}{
		{"UPDATE domains SET certificate_id = ?, cert_filled = 1, acme_method = '', updated_at = ? WHERE id = ?", []any{nullID(certID), now, id}},
		{"DELETE FROM domain_pins WHERE domain_id = ?", []any{id}},
		{"UPDATE acme_orders SET state = 'cancelled', updated_at = ? WHERE domain_id = ? AND state IN ('starting', 'waiting_dns', " +
			"'validating', 'issuing')", []any{now, id}},
	} {
		if _, err := tx.ExecContext(ctx, q.sql, q.args...); err != nil {
			return fmt.Errorf("change domain certificate: %w", err)
		}
	}
	for _, r := range holders {
		if err := rederive(ctx, tx, r); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// rederive sets the certificate_id of HTTPS rule r again from its domains
// (D63, D65). A rule served from its own certificate keeps it until its
// own save (D54).
func rederive(ctx context.Context, tx *sql.Tx, r rule) error {
	if r.site != nil {
		return nil
	}
	id, err := domainsCert(ctx, tx, r.port, r.domains)
	if err != nil || id == 0 {
		return err
	}
	table := "legacy_sites"
	if r.ref.Kind == RuleRedirect {
		table = "link_redirects"
	}
	if _, err := tx.ExecContext(ctx, "UPDATE "+table+" SET certificate_id = ? WHERE id = ?", id, r.ref.ID); err != nil {
		return fmt.Errorf("change rule certificate: %w", err)
	}
	return nil
}

// querier is a database or a transaction.
type querier interface {
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// domainsCert is the certificate a rule on port with domains (sorted)
// takes (D63): for the first domain that has one, the port's pin of it,
// else its certificate; 0 when none has one.
func domainsCert(ctx context.Context, q querier, port int, domains []string) (int64, error) {
	for _, d := range domains {
		var id int64
		err := q.QueryRowContext(ctx, "SELECT COALESCE(p.certificate_id, d.certificate_id, 0) FROM domains d "+
			"LEFT JOIN domain_pins p ON p.domain_id = d.id AND p.port = ? WHERE d.name = ?", port, d).Scan(&id)
		switch {
		case errors.Is(err, sql.ErrNoRows):
		case err != nil:
			return 0, fmt.Errorf("read domain certificate: %w", err)
		case id != 0:
			return id, nil
		}
	}
	return 0, nil
}

// certNameOf is a default certificate name: the domain's, cut to the
// longest name the table takes.
func certNameOf(domain string) string {
	if utf8.RuneCountInString(domain) <= maxCertName {
		return domain
	}
	return string([]rune(domain)[:maxCertName])
}
