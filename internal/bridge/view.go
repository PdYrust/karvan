package bridge

import (
	"context"
	"fmt"
	"slices"
	"time"
)

// statsTimeout bounds opening a destination panel for a site's numbers,
// so a panel that does not answer cannot hold the list.
const statsTimeout = 5 * time.Second

// SiteStats counts the users in a site's scope (D23), those whose old
// link was proven, and those closed for good. The rest are pending: their
// first old link request checks them.
type SiteStats struct {
	Moved    int `json:"moved"`
	Verified int `json:"verified"`
	Closed   int `json:"closed"`
}

// WarnNotAdopted flags a site whose certificate is still in its own
// columns: its adoption failed (D54, D50).
const WarnNotAdopted = "certificate_not_adopted"

// SiteView is a site with its batches, certificate, warnings and numbers.
// Cert is nil for a plain HTTP site and holds the facts of the
// certificate it is served with; Certificate is its certificates row, nil
// for plain HTTP and for a site not adopted (D54). Stats is nil when the
// destination cannot be read.
type SiteView struct {
	Site
	Batches     []BatchRef       `json:"batches"`
	Certificate *SiteCertificate `json:"certificate"`
	Cert        *CertFacts       `json:"cert,omitempty"`
	Warnings    []string         `json:"warnings"`
	Stats       *SiteStats       `json:"stats,omitempty"`
}

// SiteCertificate is the certificates row a site is served with.
type SiteCertificate struct {
	ID    int64     `json:"id"`
	Name  string    `json:"name"`
	Facts CertFacts `json:"facts"`
}

// Views lists every site with its certificate facts and numbers.
func (b *Bridge) Views(ctx context.Context) ([]SiteView, error) {
	sites, err := b.st.Sites(ctx)
	if err != nil {
		return nil, err
	}
	rules, err := b.st.rules(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]SiteView, 0, len(sites))
	for _, s := range sites {
		out = append(out, b.view(ctx, s, rules))
	}
	return out, nil
}

// View adds a site's batches, certificate facts, warnings and numbers.
func (b *Bridge) View(ctx context.Context, s Site) SiteView {
	rules, err := b.st.rules(ctx)
	if err != nil {
		b.Log.Warn("bridge: read rules", "site", s.Name, "err", err)
	}
	return b.view(ctx, s, rules)
}

func (b *Bridge) view(ctx context.Context, s Site, rules []rule) SiteView {
	v := SiteView{Site: s, Batches: []BatchRef{}, Cert: b.siteCert(ctx, s), Warnings: []string{}}
	if bs, err := b.st.siteBatches(ctx, s.ID); err != nil {
		b.Log.Warn("bridge: read rule batches", "site", s.Name, "err", err)
	} else {
		v.Batches = bs
	}
	v.BatchIDs = nil
	for _, r := range v.Batches {
		v.BatchIDs = append(v.BatchIDs, r.ID)
	}
	switch {
	case s.CertificateID != 0 && v.Cert != nil:
		v.Certificate = &SiteCertificate{ID: s.CertificateID, Name: s.certName, Facts: *v.Cert}
	case s.TLS():
		v.Warnings = append(v.Warnings, WarnNotAdopted)
	}
	if v.Cert != nil && slices.Contains(v.Cert.Warnings, WarnDomain) {
		v.Warnings = append(v.Warnings, WarnDomain)
	}
	self := ruleOf(s)
	if slices.ContainsFunc(rules, func(o rule) bool {
		return (o.ref.Kind != RulePanel || o.ref.ID != s.ID) && self.tls && o.tls && self.certClash(o)
	}) {
		v.Warnings = append(v.Warnings, WarnCertConflict)
	}
	st, err := b.stats(ctx, s)
	if err != nil {
		b.Log.Warn("bridge: site numbers", "site", s.Name, "err", err)
		return v
	}
	v.Stats = &st
	return v
}

func (b *Bridge) stats(ctx context.Context, s Site) (SiteStats, error) {
	var st SiteStats
	if s.origin == "" {
		if err := s.setOrigin(); err != nil {
			return st, err
		}
	}
	dctx, cancel := context.WithTimeout(ctx, statsTimeout)
	defer cancel()
	d, err := b.dest(dctx, s.PanelID)
	if err != nil {
		return st, err
	}
	err = b.DB.QueryRowContext(ctx, "SELECT count(*), "+
		"count(*) FILTER (WHERE EXISTS (SELECT 1 FROM legacy_tokens t WHERE t.ledger_id = l.id)), "+
		"count(*) FILTER (WHERE EXISTS (SELECT 1 FROM legacy_closed c WHERE c.ledger_id = l.id))"+fromLedger+
		" WHERE "+scopeSQL, scopeArgs(s, d.Key())...).Scan(&st.Moved, &st.Verified, &st.Closed)
	if err != nil {
		return st, fmt.Errorf("site numbers: %w", err)
	}
	return st, nil
}
