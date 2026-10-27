package hoststats

import (
	"context"
	"time"
)

// LiveWindow is how far back the live view goes: the samples kept in memory.
const LiveWindow = keep * Interval

// backfillBounds is the span of saved minutes that belongs before the
// samples in memory: from the minute the live window starts in, up to (not
// including) the minute of the oldest sample, or of now when there is none.
// That minute and the later ones are the samples' own, so the two never
// overlap.
func backfillBounds(oldest *time.Time, now time.Time) (from, to int64) {
	from = now.Add(-LiveWindow).Unix() / 60 * 60
	end := now
	if oldest != nil {
		end = *oldest
	}
	return from, end.Unix() / 60 * 60
}

// Backfill returns the saved minutes of the live window that come before
// the samples in memory, oldest first, so the live view is whole right
// after Karvan starts. A minute that was never saved stays missing.
func (s *Sampler) Backfill(ctx context.Context, now time.Time) ([]Minute, error) {
	if s.Store == nil {
		return []Minute{}, nil
	}
	var oldest *time.Time
	s.mu.Lock()
	if len(s.ring) > 0 {
		at := s.ring[0].At
		oldest = &at
	}
	s.mu.Unlock()
	from, to := backfillBounds(oldest, now)
	if to <= from {
		return []Minute{}, nil
	}
	return s.Store.Range(ctx, from, to, 60)
}
