package bridge

import (
	"context"
	"crypto/x509"
	"database/sql"
	"errors"
	"fmt"
	"slices"
)

// OriginACME marks a certificate Karvan got from Let's Encrypt.
const OriginACME = "acme"

// Kinds of rule and of certificate.
const (
	RulePanel      = "panel"    // a legacy_sites row
	RuleRedirect   = "redirect" // a link_redirects row
	CertKindFile   = "file"     // a file pair on this server, read on use
	CertKindPasted = "pasted"   // a pasted certificate, its key sealed
)

// Domain warnings beyond the certificate ones.
const (
	WarnNoCertificate = "no_certificate" // the domain has no certificate
	WarnCertConflict  = "cert_conflict"  // a port keeps serving another certificate for the domain through a pin
)

// ErrInUse means a rule uses the domain or certificate; an *InUseError
// names the rules.
var ErrInUse = errors.New("in use by a rule")

// InUseError names what is in use ("domain" or "certificate"), the rules
// that use it and, for a certificate, the domains whose certificate or
// pin it is.
type InUseError struct {
	What    string
	Rules   []RuleRef
	Domains []Ref
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

// CertDomain is a domain whose certificate it is, with its domain_mismatch.
type CertDomain struct {
	Ref
	Warnings []string `json:"warnings"`
}

// CertRule is a rule that uses a certificate, with its domain_mismatch.
type CertRule struct {
	RuleRef
	Warnings []string `json:"warnings"`
}

// Certificate is a certificates row with its facts and the rules that use
// it. The private key never leaves the store.
type Certificate struct {
	ID       int64        `json:"id"`
	Name     string       `json:"name"`
	Kind     string       `json:"kind"`
	Origin   string       `json:"origin"` // "acme" for a Let's Encrypt certificate, else ""
	CertFile string       `json:"certFile,omitempty"`
	KeyFile  string       `json:"keyFile,omitempty"`
	CertPEM  string       `json:"certPem,omitempty"`
	HasKey   bool         `json:"hasKey"`
	Facts    CertFacts    `json:"facts"`
	Domains  []CertDomain `json:"domains"`
	Rules    []CertRule   `json:"rules"`
	// Found is a file pair in one of the places the certificate scan reads.
	Found  bool       `json:"found"`
	Copies []CertCopy `json:"copies"`
	leaf   *x509.Certificate
}

// rule is a panel or redirect rule as the router and the certificate
// pages see it.
type rule struct {
	ref      RuleRef
	domains  []string
	port     int
	certID   int64 // its certificates row, or 0
	site     *Site // a panel rule still served from its own columns
	tls      bool
	path     string    // its path without the outer slashes: a panel rule's old path, a redirect's from path
	exact    bool      // the path must match exactly; else it is a prefix
	redirect *Redirect // the redirect rule it is; nil for a panel rule
}

// ruleOf is a panel rule as a rule.
func ruleOf(s Site) rule {
	r := rule{ref: RuleRef{Kind: RulePanel, ID: s.ID, Name: s.Name}, domains: s.Domains, port: s.Port, certID: s.CertificateID,
		tls: s.TLS(), path: s.OldPath}
	if s.TLS() && s.CertificateID == 0 {
		r.site = &s
	}
	return r
}

// rules reads every panel and redirect rule.
func (st *store) rules(ctx context.Context) ([]rule, error) {
	sites, err := st.Sites(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]rule, 0, len(sites))
	for _, s := range sites {
		out = append(out, ruleOf(s))
	}
	redirects, err := st.redirectRules(ctx)
	return append(out, redirects...), err
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

// DeleteDomain removes a domain no rule holds, in one statement so a rule
// saved at the same time cannot be left without its row.
func (b *Bridge) DeleteDomain(ctx context.Context, id int64) error {
	b.st.saveMu.Lock() // a rule save checks its domains are registered before it writes
	defer b.st.saveMu.Unlock()
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
