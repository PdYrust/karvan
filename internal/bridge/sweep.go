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
		if _, row, ok, err := b.st.token(ctx, q.mac, q.site, q.dest.Key()); err != nil || !ok || row.held {
			return // a held batch's tokens stay as they are (D22)
		}
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
// and closes the old links of those it revoked or deleted. Users in an
// inactive or deleted batch are skipped (D22). It returns how many users
// it closed.
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
	seen := map[int64]bool{} // ledger ids, since rules can share users
	for _, s := range sites {
		d, err := b.dest(ctx, s.PanelID)
		if err != nil {
			b.Log.Warn("bridge: sweep", "site", s.Name, "err", err)
			continue
		}
		entries, err := b.st.served(ctx, s, d.Key())
		if err != nil {
			return n, err
		}
		var open []ledger.Entry
		for _, e := range entries {
			if !closed[e.ID] && !seen[e.ID] {
				seen[e.ID] = true
				open = append(open, e)
			}
		}
		got, err := b.sweepOne(ctx, d, open)
		n += got
		if err != nil {
			return n, err
		}
	}
	return n, nil
}

func (b *Bridge) sweepOne(ctx context.Context, d Destination, open []ledger.Entry) (int, error) {
	if len(open) == 0 {
		return 0, nil
	}
	ids := make([]int64, len(open))
	for i, e := range open {
		ids[i] = e.Row.ID
	}
	rows, err := d.Rows(ctx, ids)
	if err != nil {
		return 0, err
	}
	n := 0
	for _, e := range open {
		r, found := rows[e.Row.ID]
		if b.apply(ctx, e, judge(e, r, found, time.Time{})) {
			n++
		}
	}
	return n, nil
}

// Wait waits for background checks to end, token lists included.
func (b *Bridge) Wait() {
	b.wg.Wait()
	b.upWG.Wait()
}
