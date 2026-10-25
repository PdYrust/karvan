package run

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"

	"github.com/PdYrust/karvan/internal/panel/paneldb"
	"github.com/PdYrust/karvan/internal/safemode"
)

// Restoring is what a backup restore on request answers (B12): the job
// whose stream follows the restore, and its last event before it.
type Restoring struct {
	JobID string `json:"jobId"`
	After int64  `json:"after"`
}

// restoreTask is a restore ready to run: the destination is locked.
type restoreTask struct {
	backup safemode.Backup
	job    *Job
	prior  string       // the job's state before the restore
	ctl    PanelControl // nil: the caller stopped the panel
	w      *paneldb.Writer
	target paneldb.Target
	unlock func()
	rep    *Report
}

func (t *restoreTask) release() {
	t.unlock()
	_ = t.w.Close()
}

// StartRestore puts a Safe Mode backup back in the background (B12):
// Karvan stops the panel, restores and starts it again. It returns once
// the destination is locked; the restore stage, its steps (panel_stopped,
// backup_put_back, panel_started) and done follow on the job's stream, and
// the job's report is stored again.
func (s *Service) StartRestore(ctx context.Context, id string) (Restoring, error) {
	t, err := s.prepareRestore(ctx, id, false)
	if err != nil {
		return Restoring{}, err
	}
	after, err := s.events.last(ctx, t.job.ID)
	if err != nil {
		t.release()
		return Restoring{}, err
	}
	bg := context.WithoutCancel(ctx)
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		_ = s.runRestore(bg, t)
	}()
	return Restoring{JobID: t.job.ID, After: after}, nil
}

// RestoreBackup puts a Safe Mode backup back and returns when it is done
// (the command line). With stopped false, Karvan stops the panel,
// restores and starts it again; with stopped true the caller has stopped
// it. The destination's lock is held throughout, so no job writes
// meanwhile.
func (s *Service) RestoreBackup(ctx context.Context, id string, stopped bool) error {
	t, err := s.prepareRestore(ctx, id, stopped)
	if err != nil {
		return err
	}
	return s.runRestore(ctx, t)
}

// prepareRestore finds the backup and its panel, checks Karvan can stop
// the panel, locks the destination and readies the job's report.
func (s *Service) prepareRestore(ctx context.Context, id string, stopped bool) (*restoreTask, error) {
	b, jobID, env, d, err := s.backupTarget(ctx, id)
	if err != nil {
		return nil, err
	}
	t := &restoreTask{backup: b, job: &Job{ID: jobID}, target: d.Target}
	if !stopped {
		if t.ctl, err = s.control(ctx, env); err != nil {
			return nil, err
		}
	}
	if t.w, err = paneldb.OpenWriter(ctx, d.Target); err != nil {
		return nil, err
	}
	if t.unlock, err = t.w.Lock(ctx, d.Target); err != nil {
		_ = t.w.Close()
		return nil, err
	}
	if t.rep, t.prior, err = s.restoreReport(ctx, jobID, d.Target.String()); err != nil {
		t.release()
		return nil, err
	}
	return t, nil
}

// restoreReport is the job's stored report, ready for the restore, or a
// new one for a job that has none (an interrupted run), and the job's
// state now.
func (s *Service) restoreReport(ctx context.Context, jobID, db string) (*Report, string, error) {
	var state, kind, origin, admin, backup, rollback string
	var safe bool
	err := s.db.QueryRowContext(ctx, "SELECT state, source_kind, source_origin, source_admin, backup_id, safe_mode, rollback FROM jobs WHERE id = ?", jobID).
		Scan(&state, &kind, &origin, &admin, &backup, &safe, &rollback)
	if err != nil {
		return nil, "", fmt.Errorf("read job %s: %w", jobID, err)
	}
	rep, err := s.storedReport(jobID)
	if err != nil {
		return nil, "", err
	}
	if rep == nil {
		j := &Job{rollback: rollback}
		rep = &Report{Schema: ReportSchema, JobID: jobID, SafeMode: safe, Rollback: j.rollbackOrUndo(), BackupID: backup, StartedAt: s.now()}
		rep.Source.Kind, rep.Source.Origin, rep.Source.Admin = kind, origin, admin
		rep.Destination.DB = db
	}
	rep.Error = ""
	return rep, state, nil
}

// runRestore puts the backup back, stores the job's new state and report,
// lets the destination go and sends done. A restore that failed leaves
// the job's state as it was and says why on the stream and in the report.
func (s *Service) runRestore(ctx context.Context, t *restoreTask) error {
	j, rep := t.job, t.rep
	note := s.noter(j, StageRestore)
	err := s.stage(j, rep, StageRestore, func() error {
		put := func() error {
			return safemode.Restorer{Key: s.Key, Run: s.Tools}.Restore(ctx, t.target, t.w.Server(), t.backup)
		}
		var err error
		if t.ctl != nil {
			err = restoreStopped(ctx, t.ctl, put, note)
		} else if err = put(); err == nil {
			note(codeBackupBack)
		}
		if err != nil {
			// The failed stage reads its reason from the log line before it.
			note(restoreCode(err))
		}
		return err
	})
	var ledgerErr error
	if err == nil {
		// The backup holds none of the job's rows, so its ledger rows are done with.
		if ledgerErr = s.markRestored(ctx, j.ID); ledgerErr != nil {
			note("ledger_not_updated")
		}
		rep.State, rep.Left = StateRolledBack, nil
		if !slices.Contains(rep.NotUndoable, UndoLostPanelWrites) {
			rep.NotUndoable = append(rep.NotUndoable, UndoLostPanelWrites)
		}
	} else {
		rep.State, rep.Error = t.prior, restoreCode(err)
		slog.Warn("backup restore failed", "job", j.ID, "backup", t.backup.ID, "err", err)
	}
	rep.EndedAt = s.now()
	storeErr := s.storeRollback(ctx, rep)
	if storeErr != nil {
		note("report_not_stored")
	}
	// The lock goes before done, so a client that acts on done finds the
	// destination free.
	t.release()
	s.emit(j, "", EventDone, map[string]any{"state": rep.State})
	return errors.Join(err, ledgerErr, storeErr)
}

// restoreCode names why a restore failed; a reason without its own code
// is restore_failed, since nothing was written for a migration.
func restoreCode(err error) string {
	if code := errorCode(err); code != "run_failed" {
		return code
	}
	return "restore_failed"
}

// noter logs a step of a stage on the job's stream.
func (s *Service) noter(j *Job, stage string) func(code string) {
	return func(code string) { s.emit(j, stage, EventLog, map[string]any{"code": code}) }
}
