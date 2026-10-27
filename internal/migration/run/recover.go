package run

import (
	"context"
	"database/sql"
	"encoding/json"
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

// Rollback removes what a failed or interrupted job wrote, when the
// operator asks (D39), in the background (B5). It works from Karvan's
// records alone; the source login is not needed. It returns once the
// destination is locked; the undo and restore stages, the undo's
// progress and a done event follow on the job's stream. The job's report
// is stored again with the undo counted.
func (s *Service) Rollback(ctx context.Context, jobID string) error {
	row, err := s.rollbackRow(ctx, jobID)
	if err != nil {
		return err
	}
	p, err := s.openPrepared(ctx, row.panel, true)
	if err != nil {
		return err
	}
	// Another rollback may have ended between the read and the lock.
	if row, err = s.rollbackRow(ctx, jobID); err != nil {
		p.release()
		return err
	}
	j := &Job{ID: jobID, PanelID: row.panel, origin: row.origin, backupID: row.backup, safeMode: row.safe, rollback: row.rollback}
	rep, err := s.rollbackReport(j, row, p)
	if err == nil {
		j.envPath, err = s.envPath(ctx, row.panel)
	}
	if err != nil {
		p.release()
		return err
	}
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		s.rollbackJob(j, p, rep)
	}()
	return nil
}

// rollbackRow is what a rollback reads from a job's row.
type rollbackRow struct {
	panel                                 int64
	kind, origin, admin, backup, rollback string
	safe                                  bool
}

// rollbackRow reads a job that can be rolled back: a failed or an
// interrupted one.
func (s *Service) rollbackRow(ctx context.Context, jobID string) (rollbackRow, error) {
	var r rollbackRow
	var state string
	err := s.db.QueryRowContext(ctx, "SELECT panel_id, state, source_kind, source_origin, source_admin, backup_id, safe_mode, rollback FROM jobs WHERE id = ?", jobID).
		Scan(&r.panel, &state, &r.kind, &r.origin, &r.admin, &r.backup, &r.safe, &r.rollback)
	if errors.Is(err, sql.ErrNoRows) {
		return r, fmt.Errorf("%w: %s", ErrNoJob, jobID)
	}
	if err != nil {
		return r, fmt.Errorf("read job %s: %w", jobID, err)
	}
	if state != StateInterrupted && state != StateFailed {
		return r, fmt.Errorf("%w: job %s is %s", ErrNoRollback, jobID, state)
	}
	return r, nil
}

// rollbackReport is the job's stored report, ready to count the undo, or
// a new one for a job that has none (an interrupted run).
func (s *Service) rollbackReport(j *Job, row rollbackRow, p *prepared) (*Report, error) {
	rep, err := s.storedReport(j.ID)
	if err != nil {
		return nil, err
	}
	if rep == nil {
		rep = &Report{Schema: ReportSchema, JobID: j.ID, SafeMode: j.safeMode, BackupID: j.backupID, StartedAt: s.now()}
		rep.Source.Kind, rep.Source.Origin, rep.Source.Admin = row.kind, row.origin, row.admin
		rep.Destination.DB = p.dest
	}
	rep.State, rep.Left, rep.Rollback = "", nil, j.rollbackOrUndo()
	return rep, nil
}

// rollbackJob runs a rollback on request, stores the job's new state and
// report, lets the destination go and sends done.
func (s *Service) rollbackJob(j *Job, p *prepared, rep *Report) {
	ctx := context.Background()
	err := s.settle(ctx, j, p)
	if err == nil {
		if j.safeMode {
			s.rollBack(ctx, j, p, rep)
		} else {
			s.undo(ctx, j, p, rep, false)
		}
	}
	if rep.State == "" {
		rep.State = StateFailed
	}
	if err != nil {
		rep.Error = errorCode(err)
		s.emit(j, StageUndo, EventLog, map[string]any{"code": rep.Error})
	}
	rep.EndedAt = s.now()
	if err := s.storeRollback(ctx, rep); err != nil {
		s.emit(j, "", EventLog, map[string]any{"code": "report_not_stored"})
	}
	// The lock goes before done, so a client that acts on done finds the
	// destination free.
	p.release()
	s.emit(j, "", EventDone, map[string]any{"state": rep.State})
}

// storeRollback stores a rollback's state and report, and puts them in
// the job's memory when the job is there, under its lock, so a later save
// of the job keeps them.
func (s *Service) storeRollback(ctx context.Context, rep *Report) error {
	if j, err := s.job(rep.JobID); err == nil && j.lockIdle() {
		defer j.mu.Unlock()
		j.state, j.report = rep.State, rep
	}
	report, err := json.Marshal(rep.WithoutLinks())
	if err != nil {
		return err
	}
	counts, err := json.Marshal(rep.Counts)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, "UPDATE jobs SET state = ?, report = ?, counts = ?, ended_at = ?, updated_at = ? WHERE id = ?",
		rep.State, string(report), string(counts), rep.EndedAt.UnixMilli(), s.now().UnixMilli(), rep.JobID)
	if err != nil {
		return fmt.Errorf("store the rollback of %s: %w", rep.JobID, err)
	}
	return nil
}

// Cancel stops a running job between chunks; Safe Mode then undoes it.
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
