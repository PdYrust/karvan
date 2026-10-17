package run

import (
	"context"
	"errors"
	"fmt"

	"github.com/PdYrust/karvan/internal/migration/dest"
	"github.com/PdYrust/karvan/internal/panel/paneldb"
)

// Recover runs once at start: jobs that were running are interrupted.
// Their pending ledger rows are settled (read only on the destination);
// nothing is written or removed by itself (D39).
func (s *Service) Recover(ctx context.Context) error {
	jobs, err := s.stuckJobs(ctx)
	if err != nil {
		return err
	}
	var errs []error
	for _, sj := range jobs {
		if _, err := s.db.ExecContext(ctx, "UPDATE jobs SET state = ?, updated_at = ? WHERE id = ?", StateInterrupted, s.now().UnixMilli(), sj.id); err != nil {
			errs = append(errs, err)
			continue
		}
		p, err := s.openPrepared(ctx, sj.panel, false)
		if err != nil {
			errs = append(errs, fmt.Errorf("settle job %s: %w", sj.id, err))
			continue
		}
		errs = append(errs, s.settle(ctx, &Job{ID: sj.id, origin: sj.origin}, p))
		p.release()
	}
	return errors.Join(errs...)
}

type stuck struct {
	id     string
	panel  int64
	origin string
}

func (s *Service) stuckJobs(ctx context.Context) ([]stuck, error) {
	rows, err := s.db.QueryContext(ctx, "SELECT id, panel_id, source_origin FROM jobs WHERE state = ?", StateRunning)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []stuck
	for rows.Next() {
		var j stuck
		if err := rows.Scan(&j.id, &j.panel, &j.origin); err != nil {
			return nil, err
		}
		out = append(out, j)
	}
	return out, rows.Err()
}

// openPrepared opens a destination for settling or undoing outside a run.
// lock takes the destination lock as a run would.
func (s *Service) openPrepared(ctx context.Context, panelID int64, lock bool) (*prepared, error) {
	env, err := s.envPath(ctx, panelID)
	if err != nil {
		return nil, err
	}
	d, err := dest.Open(ctx, env)
	if err != nil {
		return nil, err
	}
	facts, err := d.Facts(ctx, nil)
	if err != nil {
		return nil, err
	}
	db, err := paneldb.OpenWriter(ctx, d.Target)
	if err != nil {
		return nil, err
	}
	unlock := func() {}
	if lock {
		if unlock, err = db.Lock(ctx, d.Target); err != nil {
			_ = db.Close()
			return nil, err
		}
	}
	return &prepared{db: db, target: d.Target, dest: d.Target.String(), unlock: unlock,
		writer: dest.NewWriter(facts.Kind, db.Server().Dialect, facts)}, nil
}

// Rollback removes what an interrupted job wrote, when the operator asks
// (D39). It works from Karvan's records alone; the source login is not
// needed.
func (s *Service) Rollback(ctx context.Context, jobID string) (*Report, error) {
	var panelID int64
	var state, origin, backup, rollback string
	var safe bool
	err := s.db.QueryRowContext(ctx, "SELECT panel_id, state, source_origin, backup_id, safe_mode, rollback FROM jobs WHERE id = ?", jobID).
		Scan(&panelID, &state, &origin, &backup, &safe, &rollback)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrNoJob, jobID)
	}
	if state != StateInterrupted && state != StateFailed {
		return nil, fmt.Errorf("%w: job %s is %s", ErrNoRollback, jobID, state)
	}
	p, err := s.openPrepared(ctx, panelID, true)
	if err != nil {
		return nil, err
	}
	defer p.release()
	j := &Job{ID: jobID, PanelID: panelID, origin: origin, backupID: backup, safeMode: safe, rollback: rollback}
	if j.envPath, err = s.envPath(ctx, panelID); err != nil {
		return nil, err
	}
	if err := s.settle(ctx, j, p); err != nil {
		return nil, err
	}
	rep := &Report{Schema: ReportSchema, JobID: jobID, SafeMode: safe, Rollback: j.rollbackOrUndo(), BackupID: backup, StartedAt: s.now()}
	if safe {
		s.rollBack(ctx, j, p, rep)
	} else {
		s.undo(ctx, j, p, rep, false)
	}
	if rep.State == "" {
		rep.State = StateFailed
	}
	rep.EndedAt = s.now()
	_, err = s.db.ExecContext(ctx, "UPDATE jobs SET state = ?, updated_at = ? WHERE id = ?", rep.State, s.now().UnixMilli(), jobID)
	return rep, err
}

// Cancel stops a running job between batches; Safe Mode then undoes it.
func (s *Service) Cancel(jobID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if c, ok := s.cancels[jobID]; ok {
		c()
		return nil
	}
	if _, ok := s.jobs[jobID]; !ok {
		return ErrNoJob
	}
	return nil
}
