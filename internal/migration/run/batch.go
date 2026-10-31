package run

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/PdYrust/karvan/internal/migration/batch"
	"github.com/PdYrust/karvan/internal/migration/dest"
)

// oldLinkFacts reads the old link facts of a run before the job
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
// leaves the job done; the next start makes the batch.
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

// endRestore ends a restore's hold on its batches, given the
// restore's error and whether putting the backup back started: rolled
// back when the backup is back, restore_failed once the put-back started,
// as they were when nothing was put back. A failure leaves them restoring
// (the bridge serves none of them) until the next start.
func endRestore(ctx context.Context, h *batch.Restore, err error, started bool) error {
	end := h.Cancel
	switch {
	case err == nil:
		end = h.Done
	case started:
		end = h.Fail
	}
	if err := end(ctx); err != nil {
		slog.Error("batches of a backup restore not updated", "err", err)
		return err
	}
	return nil
}

// Batches is the batch store, for the batch routes.
func (s *Service) Batches() *batch.Store { return s.batches }

// UserLink builds a batch user's subscription link the way the
// destination panel does. Karvan never stores it. A panel
// that now points at another database than the batch's gives
// ErrPanelChanged.
func (s *Service) UserLink(ctx context.Context, m batch.Member) (string, error) {
	env, err := s.envPath(ctx, m.PanelID)
	if err != nil {
		return "", err
	}
	d, err := dest.Open(ctx, env)
	if err != nil {
		return "", err
	}
	if d.Target.String() != m.Dest {
		return "", ErrPanelChanged
	}
	l, err := d.Links(ctx, m.AdminDestID)
	if err != nil {
		return "", err
	}
	return l.Link(m.DestID, m.Username, s.now()), nil
}
