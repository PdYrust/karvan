package bridge

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
)

// redirectColumns read a redirect with the certificates row it uses.
const redirectColumns = "SELECT r.id, r.name, r.domains, r.listen_port, r.from_path, r.match, r.to_url, " +
	"COALESCE(r.certificate_id, 0), COALESCE(c.name, ''), COALESCE(c.cert_file, ''), COALESCE(c.key_file, ''), " +
	"COALESCE(c.cert_pem, ''), c.key_sealed FROM link_redirects r LEFT JOIN certificates c ON c.id = r.certificate_id"

func scanRedirect(row interface{ Scan(...any) error }) (Redirect, error) {
	var d Redirect
	var domains string
	err := row.Scan(&d.ID, &d.Name, &domains, &d.Port, &d.FromPath, &d.Match, &d.ToURL, &d.CertificateID, &d.cert.name,
		&d.cert.certFile, &d.cert.keyFile, &d.cert.pem, &d.cert.sealed)
	if err != nil {
		return d, err
	}
	if err := json.Unmarshal([]byte(domains), &d.Domains); err != nil {
		return d, fmt.Errorf("redirect %d domains: %w", d.ID, err)
	}
	d.cert.id = d.CertificateID
	https := d.CertificateID != 0
	d.HTTPS = &https
	if u, ok := parseTarget(d.ToURL); ok {
		d.target = u
	}
	return d, nil
}

// redirects reads every redirect rule by id.
func (st *store) redirects(ctx context.Context) ([]Redirect, error) {
	rows, err := st.db.QueryContext(ctx, redirectColumns+" ORDER BY r.id")
	if err != nil {
		return nil, fmt.Errorf("read redirects: %w", err)
	}
	defer rows.Close()
	var out []Redirect
	for rows.Next() {
		d, err := scanRedirect(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

// redirectRules reads every redirect rule by id, as the router sees them.
func (st *store) redirectRules(ctx context.Context) ([]rule, error) {
	list, err := st.redirects(ctx)
	out := make([]rule, len(list))
	for i := range list {
		out[i] = list[i].rule()
	}
	return out, err
}

func (st *store) redirect(ctx context.Context, id int64) (Redirect, error) {
	d, err := scanRedirect(st.db.QueryRowContext(ctx, redirectColumns+" WHERE r.id = ?", id))
	if errors.Is(err, sql.ErrNoRows) {
		return d, ErrNotFound
	}
	return d, err
}

// saveRedirect creates (ID 0) or changes a redirect: the fields, its
// registered domains and derived certificate, Karvan's own
// port, every other rule and a loop, in that order; then it is
// written.
func (st *store) saveRedirect(ctx context.Context, d Redirect) (Redirect, error) {
	st.saveMu.Lock()
	defer st.saveMu.Unlock()
	in := ruleCert{self: RuleRef{Kind: RuleRedirect, ID: d.ID}, https: d.HTTPS}
	storedFrom := ""
	if d.ID != 0 {
		stored, err := st.redirect(ctx, d.ID)
		if err != nil {
			return d, err
		}
		in.stored, storedFrom = stored.HTTPS, stored.FromPath
	}
	if err := d.normalize(); err != nil {
		return d, err
	}
	if d.FromPath != storedFrom && reservedPath(d.FromPath) { // a stored one stays editable
		return d, &RedirectInvalidError{Field: "fromPath", Reason: ReasonReserved}
	}
	in.port, in.domains = d.Port, d.Domains
	id, err := st.derive(ctx, in)
	var ie *InvalidError
	if errors.As(err, &ie) {
		return d, &RedirectInvalidError{Field: ie.Field, Reason: ie.Reason, Names: ie.Names}
	}
	if err != nil {
		return d, err
	}
	d.CertificateID = id
	if err := st.checkRedirect(ctx, d); err != nil {
		return d, err
	}
	if err := st.writeRedirect(ctx, &d); err != nil {
		return d, err
	}
	return st.redirect(ctx, d.ID)
}

func (st *store) checkRedirect(ctx context.Context, d Redirect) error {
	if st.panelPorts != nil && slices.Contains(st.panelPorts(), d.Port) {
		return ErrPortTaken
	}
	if err := st.checkChallenge(d.Port, d.CertificateID != 0); err != nil {
		return err
	}
	all, err := st.rules(ctx)
	if err != nil {
		return err
	}
	self := d.rule()
	others := []rule{}
	for _, o := range all {
		if o.ref.Kind == RuleRedirect && o.ref.ID == d.ID {
			continue
		}
		if err := self.conflict(o); err != nil {
			return err
		}
		others = append(others, o)
	}
	if loops(d.target, append(others, self)) {
		return &RedirectInvalidError{Field: "toUrl", Reason: ReasonLoop}
	}
	return nil
}

// writeRedirect stores a checked redirect, and drops the pins no rule
// holds any more, in one transaction.
func (st *store) writeRedirect(ctx context.Context, d *Redirect) error {
	domains, _ := json.Marshal(d.Domains)
	now := st.now().Unix()
	certID := nullID(d.CertificateID)
	tx, err := st.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck // a no-op after Commit
	args := []any{d.Name, string(domains), d.Port, d.FromPath, d.Match, d.ToURL, certID, now}
	if d.ID == 0 {
		res, err := tx.ExecContext(ctx, "INSERT INTO link_redirects (name, domains, listen_port, from_path, match, to_url, "+
			"certificate_id, updated_at, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)", append(args, now)...)
		if err != nil {
			return fmt.Errorf("add redirect: %w", err)
		}
		if d.ID, err = res.LastInsertId(); err != nil {
			return err
		}
	} else {
		res, err := tx.ExecContext(ctx, "UPDATE link_redirects SET name = ?, domains = ?, listen_port = ?, from_path = ?, "+
			"match = ?, to_url = ?, certificate_id = ?, updated_at = ? WHERE id = ?", append(args, d.ID)...)
		if err != nil {
			return fmt.Errorf("change redirect: %w", err)
		}
		if n, _ := res.RowsAffected(); n == 0 {
			return ErrNotFound
		}
	}
	if err := prunePins(ctx, tx); err != nil {
		return err
	}
	return tx.Commit()
}

// Redirects lists every redirect rule with its certificate and warnings.
func (b *Bridge) Redirects(ctx context.Context) ([]RedirectView, error) {
	list, err := b.st.redirects(ctx)
	if err != nil {
		return nil, err
	}
	warns, err := b.domainWarnings(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]RedirectView, 0, len(list))
	for _, d := range list {
		out = append(out, b.redirectView(d, warns))
	}
	return out, nil
}

// SaveRedirect adds or changes a redirect rule and starts serving it.
func (b *Bridge) SaveRedirect(ctx context.Context, d Redirect) (RedirectView, error) {
	saved, err := b.st.saveRedirect(ctx, d)
	if err != nil {
		return RedirectView{}, err
	}
	if err := b.Reload(ctx); err != nil {
		return RedirectView{}, err
	}
	warns, err := b.domainWarnings(ctx)
	if err != nil {
		return RedirectView{}, err
	}
	return b.redirectView(saved, warns), nil
}

// DeleteRedirect removes a redirect rule and stops serving it.
func (b *Bridge) DeleteRedirect(ctx context.Context, id int64) error {
	if err := b.st.deleteRule(ctx, "DELETE FROM link_redirects WHERE id = ?", id); err != nil {
		return err
	}
	return b.Reload(ctx)
}

// redirectView is a redirect with its fallback certificate and its
// domains' warnings.
func (b *Bridge) redirectView(d Redirect, warns map[string][]string) RedirectView {
	v := RedirectView{Redirect: d, Warnings: ruleWarnings(d.Domains, warns, false)}
	if d.CertificateID != 0 {
		certPEM, keyPEM := b.material(d.cert)
		v.Certificate = &SiteCertificate{ID: d.CertificateID, Name: d.cert.name, Facts: inspect(certPEM, keyPEM, d.Domains, b.Now())}
	}
	return v
}
