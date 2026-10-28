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
// columns, since its adoption failed, and the domains it keeps unfilled
// (D54, D50, D64).
const WarnNotAdopted = "certificate_not_adopted"

// SiteView is a site with its batches, certificate, warnings and numbers.
// Cert is nil for a plain HTTP site and holds the facts of the
// certificate it is served with; Certificate is its derived fallback
// certificates row (D63), nil for plain HTTP and for a site not adopted
// (D54). Warnings are its domains' (D50). Stats is nil when the
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
	warns, err := b.domainWarnings(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]SiteView, 0, len(sites))
	for _, s := range sites {
		out = append(out, b.view(ctx, s, warns))
	}
	return out, nil
}

// View adds a site's batches, certificate facts, warnings and numbers.
func (b *Bridge) View(ctx context.Context, s Site) SiteView {
	warns, err := b.domainWarnings(ctx)
	if err != nil {
		b.Log.Warn("bridge: read domains", "site", s.Name, "err", err)
	}
	return b.view(ctx, s, warns)
}

// domainWarnings is the warnings of every domain, by name (D50).
func (b *Bridge) domainWarnings(ctx context.Context) (map[string][]string, error) {
	list, err := b.Domains(ctx)
	out := make(map[string][]string, len(list))
	for _, d := range list {
		out[d.Name] = d.Warnings
	}
	return out, err
}

// ruleWarnings is the warnings of a rule's domains, and
// certificate_not_adopted for a rule served from its own certificate
// (D50, D54), sorted.
func ruleWarnings(domains []string, warns map[string][]string, notAdopted bool) []string {
	out := []string{}
	if notAdopted {
		out = append(out, WarnNotAdopted)
	}
	for _, d := range domains {
		out = append(out, warns[d]...)
	}
	slices.Sort(out)
	return slices.Compact(out)
}

func (b *Bridge) view(ctx context.Context, s Site, warns map[string][]string) SiteView {
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
	if s.CertificateID != 0 && v.Cert != nil {
		v.Certificate = &SiteCertificate{ID: s.CertificateID, Name: s.certName, Facts: *v.Cert}
	}
	v.Warnings = ruleWarnings(s.Domains, warns, s.TLS() && s.CertificateID == 0)
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
