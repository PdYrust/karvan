package bridge

import (
	"context"
	"fmt"
	"time"
)

// statsTimeout bounds opening a destination panel for a site's numbers,
// so a panel that does not answer cannot hold the list.
const statsTimeout = 5 * time.Second

// SiteStats counts a site's moved users: the old panel's users moved into
// its destination, those whose old link was proven, and those closed for
// good. The rest are pending: their first old link request checks them.
type SiteStats struct {
	Moved    int `json:"moved"`
	Verified int `json:"verified"`
	Closed   int `json:"closed"`
}

// SiteView is a site with its certificate facts and numbers. Cert is nil
// for a plain HTTP site; Stats is nil when the destination cannot be read.
type SiteView struct {
	Site
	Cert  *CertFacts `json:"cert,omitempty"`
	Stats *SiteStats `json:"stats,omitempty"`
}

// Views lists every site with its certificate facts and numbers.
func (b *Bridge) Views(ctx context.Context) ([]SiteView, error) {
	sites, err := b.st.Sites(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]SiteView, 0, len(sites))
	for _, s := range sites {
		out = append(out, b.View(ctx, s))
	}
	return out, nil
}

// View adds a site's certificate facts and numbers.
func (b *Bridge) View(ctx context.Context, s Site) SiteView {
	v := SiteView{Site: s, Cert: b.siteCert(ctx, s)}
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
	const served = "l.kind = 'user' AND l.source_origin = ? AND l.dest = ? AND l.state IN ('done', 'adopted')"
	err = b.DB.QueryRowContext(ctx, "SELECT "+
		"(SELECT count(*) FROM ledger l WHERE "+served+"), "+
		"(SELECT count(DISTINCT t.ledger_id) FROM legacy_tokens t JOIN ledger l ON l.id = t.ledger_id WHERE "+served+"), "+
		"(SELECT count(*) FROM legacy_closed c JOIN ledger l ON l.id = c.ledger_id WHERE "+served+")",
		s.origin, d.Key(), s.origin, d.Key(), s.origin, d.Key()).Scan(&st.Moved, &st.Verified, &st.Closed)
	if err != nil {
		return st, fmt.Errorf("site numbers: %w", err)
	}
	return st, nil
}
