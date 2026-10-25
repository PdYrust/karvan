package run

import (
	"context"
	"fmt"
	"time"
)

// EventsKept is how long the events of an ended job are kept (B17).
const EventsKept = 90 * 24 * time.Hour

// TrimEvents drops the events of jobs that ended, and were last changed,
// more than EventsKept ago, and returns how many it dropped. Reports stay
// in jobs. Each job keeps its last event, so its next event (a later
// restore) numbers after it and a reader's Last-Event-ID never repeats.
// Interrupted jobs are left whole, since they can still be rolled back.
func (s *Service) TrimEvents(ctx context.Context) (int64, error) {
	cutoff := s.now().Add(-EventsKept).UnixMilli()
	res, err := s.db.ExecContext(ctx, `DELETE FROM job_events
		WHERE job_id IN (SELECT id FROM jobs WHERE state IN ('done', 'failed', 'rolled_back')
			AND ended_at > 0 AND ended_at < ? AND updated_at < ?)
		AND seq < (SELECT MAX(e.seq) FROM job_events e WHERE e.job_id = job_events.job_id)`, cutoff, cutoff)
	if err != nil {
		return 0, fmt.Errorf("trim job events: %w", err)
	}
	return res.RowsAffected()
}
