package bridge

import (
	"context"
	"crypto/x509"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
)

// Kinds of rule (D47) and of certificate (D49).
const (
	RulePanel      = "panel"    // a legacy_sites row
	RuleRedirect   = "redirect" // a link_redirects row
	CertKindFile   = "file"     // a file pair on this server, read on use
	CertKindPasted = "pasted"   // a pasted certificate, its key sealed
)

// Domain warnings beyond the certificate ones (D50).
const (
	WarnNoCertificate = "no_certificate" // no stored certificate covers the domain
	WarnCertConflict  = "cert_conflict"  // rules on one port that can meet one name use different certificates (D53)
)

// ErrInUse means a rule uses the domain or certificate; an *InUseError
// names the rules.
var ErrInUse = errors.New("in use by a rule")

// InUseError names what is in use ("domain" or "certificate") and the
// rules that use it.
type InUseError struct {
	What  string
	Rules []RuleRef
}

func (e *InUseError) Error() string { return fmt.Sprintf("%s %v", e.What, ErrInUse) }

// Is makes errors.Is(err, ErrInUse) hold.
func (e *InUseError) Is(target error) bool { return target == ErrInUse }

// Ref names a stored row.
type Ref struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

// RuleRef is a panel rule or a redirect rule.
type RuleRef struct {
	Kind string `json:"kind"`
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

// CertRule is a rule that uses a certificate, with its domain_mismatch.
type CertRule struct {
	RuleRef
	Warnings []string `json:"warnings"`
}

// Domain is a domains row with the stored certificates that cover it,
// the rules that hold it and its warnings (D48, D50).
type Domain struct {
	ID           int64     `json:"id"`
	Name         string    `json:"name"`
	Certificates []Ref     `json:"certificates"`
	Rules        []RuleRef `json:"rules"`
	Warnings     []string  `json:"warnings"`
}

// Certificate is a certificates row with its facts and the rules that use
// it. The private key never leaves the store (D49).
type Certificate struct {
	ID       int64      `json:"id"`
	Name     string     `json:"name"`
	Kind     string     `json:"kind"`
	CertFile string     `json:"certFile,omitempty"`
	KeyFile  string     `json:"keyFile,omitempty"`
	CertPEM  string     `json:"certPem,omitempty"`
	HasKey   bool       `json:"hasKey"`
	Facts    CertFacts  `json:"facts"`
	Rules    []CertRule `json:"rules"`
	leaf     *x509.Certificate
}

// rule is a panel or redirect rule as the certificate pages see it.
type rule struct {
	ref     RuleRef
	domains []string
	port    int
	certID  int64 // its certificates row, or 0
	site    *Site // a panel rule still served from its own columns (D54)
}

// certKey tells apart the certificates rules on one port are served with.
func (r rule) certKey() string {
	switch {
	case r.certID != 0:
		return fmt.Sprintf("c%d", r.certID)
	case r.site != nil:
		return fmt.Sprintf("s%d", r.ref.ID)
	}
	return ""
}

// rules reads every panel and redirect rule.
func (st *store) rules(ctx context.Context) ([]rule, error) {
	sites, err := st.Sites(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]rule, 0, len(sites))
	for _, s := range sites {
		r := rule{ref: RuleRef{Kind: RulePanel, ID: s.ID, Name: s.Name}, domains: s.Domains, port: s.Port, certID: s.CertificateID}
		if s.TLS() && s.CertificateID == 0 {
			r.site = &s
		}
		out = append(out, r)
	}
	rows, err := st.db.QueryContext(ctx, "SELECT id, name, domains, listen_port, COALESCE(certificate_id, 0) FROM link_redirects ORDER BY id")
	if err != nil {
		return nil, fmt.Errorf("read redirects: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		r := rule{ref: RuleRef{Kind: RuleRedirect}}
		var domains string
		if err := rows.Scan(&r.ref.ID, &r.ref.Name, &domains, &r.port, &r.certID); err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(domains), &r.domains); err != nil {
			return nil, fmt.Errorf("redirect %d domains: %w", r.ref.ID, err)
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// usedBy lists the rules that pass keep.
func usedBy(rules []rule, keep func(rule) bool) []RuleRef {
	out := []RuleRef{}
	for _, r := range rules {
		if keep(r) {
			out = append(out, r.ref)
		}
	}
	return out
}

// canMeet reports whether two domain patterns can match one name: equal
// patterns, or a wildcard and a name it covers (D53).
func canMeet(a, b string) bool {
	wa, wb := strings.HasPrefix(a, "*."), strings.HasPrefix(b, "*.")
	return a == b || wa && !wb && matches(a, b) || wb && !wa && matches(b, a)
}

// Domains lists every domain by name.
func (b *Bridge) Domains(ctx context.Context) ([]Domain, error) {
	certs, err := b.certificates(ctx)
	if err != nil {
		return nil, err
	}
	rules, err := b.st.rules(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := b.DB.QueryContext(ctx, "SELECT id, name FROM domains ORDER BY name")
	if err != nil {
		return nil, fmt.Errorf("read domains: %w", err)
	}
	defer rows.Close()
	out := []Domain{}
	for rows.Next() {
		var d Domain
		if err := rows.Scan(&d.ID, &d.Name); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for i := range out {
		b.fillDomain(ctx, &out[i], certs, rules)
	}
	return out, nil
}

// fillDomain adds a domain's certificates, rules and warnings.
func (b *Bridge) fillDomain(ctx context.Context, d *Domain, certs []Certificate, rules []rule) {
	d.Certificates = []Ref{}
	byID := map[int64]Certificate{}
	for _, c := range certs {
		byID[c.ID] = c
		if c.leaf != nil && certCovers(c.leaf, d.Name) {
			d.Certificates = append(d.Certificates, Ref{ID: c.ID, Name: c.Name})
		}
	}
	holds := func(r rule) bool { return slices.Contains(r.domains, d.Name) }
	d.Rules = usedBy(rules, holds)
	var warn []string
	for _, r := range rules {
		switch {
		case !holds(r):
		case r.certID != 0:
			c := byID[r.certID]
			warn = append(warn, c.Facts.Warnings...)
			if c.leaf != nil && !certCovers(c.leaf, d.Name) {
				warn = append(warn, WarnDomain)
			}
		case r.site != nil:
			s := *r.site
			s.Domains = []string{d.Name}
			warn = append(warn, b.siteCert(ctx, s).Warnings...)
		}
	}
	if len(d.Certificates) == 0 {
		warn = append(warn, WarnNoCertificate)
	}
	if certConflict(d.Name, rules) {
		warn = append(warn, WarnCertConflict)
	}
	slices.Sort(warn)
	d.Warnings = append([]string{}, slices.Compact(warn)...)
}

// certConflict reports whether rules on one port that can meet name are
// served with different certificates (D53).
func certConflict(name string, rules []rule) bool {
	seen := map[int]string{}
	for _, r := range rules {
		if !slices.ContainsFunc(r.domains, func(p string) bool { return canMeet(p, name) }) {
			continue
		}
		k := r.certKey()
		if prev, ok := seen[r.port]; ok && prev != k {
			return true
		}
		seen[r.port] = k
	}
	return false
}

// AddDomain stores a domain name in its normal form (D48).
func (b *Bridge) AddDomain(ctx context.Context, name string) (Domain, error) {
	name = strings.ToLower(strings.TrimSuffix(strings.TrimSpace(name), "."))
	if len(name) > 253 || !validPattern(name) {
		return Domain{}, ErrInvalidDomain
	}
	now := b.Now().Unix()
	res, err := b.DB.ExecContext(ctx, "INSERT INTO domains (name, created_at, updated_at) VALUES (?, ?, ?) "+
		"ON CONFLICT (name) DO NOTHING", name, now, now)
	if err != nil {
		return Domain{}, fmt.Errorf("add domain: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return Domain{}, ErrNameTaken
	}
	id, err := res.LastInsertId()
	if err != nil {
		return Domain{}, err
	}
	list, err := b.Domains(ctx)
	if err != nil {
		return Domain{}, err
	}
	for _, d := range list {
		if d.ID == id {
			return d, nil
		}
	}
	return Domain{}, ErrNotFound
}

// ErrInvalidDomain means a domain name is not a host name, one leading
// wildcard or an IP address.
var ErrInvalidDomain = errors.New("invalid domain")

// DeleteDomain removes a domain no rule holds, in one statement so a rule
// saved at the same time cannot be left without its row.
func (b *Bridge) DeleteDomain(ctx context.Context, id int64) error {
	res, err := b.DB.ExecContext(ctx, "DELETE FROM domains WHERE id = ? "+
		"AND NOT EXISTS (SELECT 1 FROM legacy_sites s, json_each(s.domains) j WHERE j.value = domains.name) "+
		"AND NOT EXISTS (SELECT 1 FROM link_redirects r, json_each(r.domains) j WHERE j.value = domains.name)", id)
	if err != nil {
		return fmt.Errorf("delete domain: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 1 {
		return nil
	}
	var name string
	err = b.DB.QueryRowContext(ctx, "SELECT name FROM domains WHERE id = ?", id).Scan(&name)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	rules, err := b.st.rules(ctx)
	if err != nil {
		return err
	}
	return &InUseError{What: "domain", Rules: usedBy(rules, func(r rule) bool { return slices.Contains(r.domains, name) })}
}
