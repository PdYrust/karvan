package run

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/PdYrust/karvan/internal/migration/batch"
)

// oldLinkFacts reads the old link facts of a run (D7) before the job
// drops its login and snapshot. The caller holds mu.
func oldLinkFacts(j *Job) batch.Facts {
	f := batch.Facts{SourcePin: strings.ToLower(strings.ReplaceAll(j.login.PinSHA256, ":", ""))}
	if j.snapshot != nil {
		links := make([]string, len(j.snapshot.Users))
		for i, u := range j.snapshot.Users {
			links[i] = u.SubscriptionURL
		}
		f.OldPath, f.OldHosts, f.OldPort = batch.OldLinks(j.origin, links)
	}
	if j.client != nil {
		if a := j.client.Addr(); a.IsValid() {
			f.SourceIP = a.Unmap().String()
		}
	}
	return f
}

// ensureBatch makes the batch of a run that just ended done. A failure
// leaves the job done; the next start makes the batch (D5).
func (s *Service) ensureBatch(j *Job, dest string, f batch.Facts) {
	if _, err := s.batches.Ensure(context.Background(), j.ID, dest, f); err != nil {
		slog.Warn("batch not stored", "job", j.ID, "err", err)
		s.emit(j, "", EventLog, map[string]any{"code": "batch_not_stored"})
	}
}

// recoverBatches makes the batch of every done job that has none.
func (s *Service) recoverBatches(ctx context.Context) error {
	ids, err := s.batches.Missing(ctx)
	if err != nil {
		return err
	}
	var errs []error
	for _, id := range ids {
		if _, err := s.batches.Ensure(ctx, id, "", batch.Facts{}); err != nil {
			errs = append(errs, fmt.Errorf("batch of job %s: %w", id, err))
		}
	}
	return errors.Join(errs...)
}
