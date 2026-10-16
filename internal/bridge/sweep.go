package bridge

import (
	"context"
	"time"

	"github.com/PdYrust/karvan/internal/migration/ledger"
	"github.com/PdYrust/karvan/internal/panel/source"
)

// SweepEvery is how often Run closes the old links of users the new
// panel revoked or deleted, before anyone asks (owner rule).
const SweepEvery = time.Hour

// recheck asks the old panel again, in the background, about a token
// verified more than a day ago. A revoke there drops the token; an old
// panel that does not answer keeps it (D60, D62).
func (b *Bridge) recheck(q request, e ledger.Entry) {
	if plainHTTPRefused(q.site) {
		return
	}
	done, ok := b.lim.acquireRecheck(q.site.ID)
	if !ok {
		return // the next request tries again
	}
	b.wg.Add(1)
	go func() {
		defer b.wg.Done()
		defer done()
		ctx, cancel := context.WithTimeout(context.Background(), lazyTimeout)
		defer cancel()
		o, err := b.old(q.site)
		if err != nil {
			return
		}
		u, err := o.Owner(ctx, q.token)
		switch {
		case source.CodeOf(err) == source.CodeNotFound, err == nil && !sameUser(q.site.SourceKind, u, e):
			_ = b.st.dropToken(ctx, q.mac)
		case err == nil:
			_ = b.st.checked(ctx, q.mac)
		}
	}()
}

// Sweep checks every served user of every site against the new panel
// and closes the old links of those it revoked or deleted. It returns
// how many users it closed.
func (b *Bridge) Sweep(ctx context.Context) (int, error) {
	sites, err := b.st.Sites(ctx)
	if err != nil {
		return 0, err
	}
	closed, err := b.st.closedSet(ctx)
	if err != nil {
		return 0, err
	}
	n := 0
	seen := map[string]bool{}
	for _, s := range sites {
		d, err := b.dest(ctx, s.PanelID)
		if err != nil {
			b.Log.Warn("bridge: sweep", "site", s.Name, "err", err)
			continue
		}
		if seen[s.origin+"\x00"+d.Key()] {
			continue
		}
		seen[s.origin+"\x00"+d.Key()] = true
		got, err := b.sweepOne(ctx, s, d, closed)
		n += got
		if err != nil {
			return n, err
		}
	}
	return n, nil
}

func (b *Bridge) sweepOne(ctx context.Context, s Site, d Destination, closed map[int64]bool) (int, error) {
	entries, err := b.Ledger.Served(ctx, s.origin, d.Key())
	if err != nil {
		return 0, err
	}
	var open []ledger.Entry
	var ids []int64
	for _, e := range entries {
		if !closed[e.ID] {
			open = append(open, e)
			ids = append(ids, e.Row.ID)
		}
	}
	if len(ids) == 0 {
		return 0, nil
	}
	rows, err := d.Rows(ctx, ids)
	if err != nil {
		return 0, err
	}
	n := 0
	for _, e := range open {
		r, found := rows[e.Row.ID]
		if v := judge(e, r, found, time.Time{}); v.close != "" {
			b.apply(ctx, e, v)
			n++
		}
	}
	return n, nil
}

// Wait waits for background checks to end.
func (b *Bridge) Wait() { b.wg.Wait() }
