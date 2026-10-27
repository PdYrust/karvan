package run

import (
	"context"
	"log/slog"
	"slices"
	"time"

	"github.com/PdYrust/karvan/internal/migration/ledger"
	"github.com/PdYrust/karvan/internal/panel"
	"github.com/PdYrust/karvan/internal/panel/paneldb"
	"github.com/PdYrust/karvan/internal/safemode"
)

// rollBack takes a failed job back the way the Safe Mode setting says
// (owner, 2026-10-16): undo the job's own rows (D35, D36), or put the
// whole backup back. A backup that cannot be put back falls back to the
// undo, so the job's rows still go.
func (s *Service) rollBack(ctx context.Context, j *Job, p *prepared, rep *Report) {
	// The run's own context may be cancelled; the rollback must still finish.
	ctx = context.WithoutCancel(ctx)
	if j.rollbackOrUndo() == RollbackBackup {
		if s.restore(ctx, j, p, rep) {
			return
		}
		s.emit(j, StageRestore, EventLog, map[string]any{"code": "restore_failed_undoing"})
		s.undo(ctx, j, p, rep, false)
		return
	}
	s.undo(ctx, j, p, rep, true)
}

// undo removes exactly this job's rows, newest chunk first, each chunk in
// its own short transaction, and checks nothing of the job is left. Users
// a later batch holds stay (D14): they are listed as left with
// LeftInBatch and do not make the undo incomplete. If the undo failed or
// something else is left, it puts the backup back when orRestore is set.
func (s *Service) undo(ctx context.Context, j *Job, p *prepared, rep *Report, orRestore bool) {
	undoErr := s.stage(j, rep, StageUndo, func() error { return s.compensate(ctx, j, p, rep) })
	left, held, err := s.leftovers(ctx, j)
	rep.Left, rep.LeftReasons = held, nil
	if len(held) > 0 {
		rep.LeftReasons = map[int64]string{}
		for _, id := range held {
			rep.LeftReasons[id] = LeftInBatch
		}
		slog.Warn("rollback left users a later batch holds", "job", j.ID, "users", len(held))
	}
	if undoErr == nil && err == nil && len(left) == 0 {
		rep.State = StateRolledBack
		return
	}
	rep.Left = append(rep.Left, left...)
	s.emit(j, StageUndo, EventLog, map[string]any{"code": "undo_incomplete", "left": len(left)})
	if orRestore {
		s.restore(ctx, j, p, rep)
	}
}

func (s *Service) compensate(ctx context.Context, j *Job, p *prepared, rep *Report) error {
	rows, err := s.ledger.Job(ctx, j.ID, ledger.Done)
	if err != nil {
		return err
	}
	held, err := s.batches.Foreign(ctx, j.ID)
	if err != nil {
		return err
	}
	rows = slices.DeleteFunc(rows, func(e ledger.Entry) bool { return held[e.ID] })
	chunks := map[int][]ledger.Entry{}
	var order []int
	var admin *ledger.Entry
	for i, e := range rows {
		if e.Kind == ledger.KindAdmin {
			admin = &rows[i]
			continue
		}
		if _, ok := chunks[e.Chunk]; !ok {
			order = append(order, e.Chunk)
		}
		chunks[e.Chunk] = append(chunks[e.Chunk], e)
	}
	slices.Sort(order)
	slices.Reverse(order)
	// Progress counts user rows handled: removed, or found gone already.
	total, handled := len(rows), 0
	if admin != nil {
		total--
	}
	s.emit(j, StageUndo, EventProgress, map[string]any{"done": 0, "total": total})
	for _, n := range order {
		if err := s.removeChunk(ctx, p, chunks[n], rep); err != nil {
			return err
		}
		handled += len(chunks[n])
		s.emit(j, StageUndo, EventProgress, map[string]any{"done": handled, "total": total})
	}
	if admin != nil {
		return s.removeAdmin(ctx, p, *admin)
	}
	return nil
}

func (s *Service) removeChunk(ctx context.Context, p *prepared, chunk []ledger.Entry, rep *Report) error {
	rows := make([]panel.Row, len(chunk))
	byID := map[int64]int64{}
	for i, e := range chunk {
		rows[i] = e.Row
		byID[e.Row.ID] = e.ID
	}
	var removed []int64
	err := p.db.Chunk(ctx, func(x paneldb.Execer) error {
		var err error
		removed, _, err = p.writer.RemoveUsers(ctx, x, rows)
		return err
	})
	if err != nil {
		return err
	}
	ids := make([]int64, 0, len(removed))
	for _, r := range removed {
		ids = append(ids, byID[r])
	}
	rep.Counts.Undone += len(ids)
	return s.ledger.Set(ctx, ledger.Compensated, ids)
}

func (s *Service) removeAdmin(ctx context.Context, p *prepared, e ledger.Entry) error {
	var ok bool
	err := p.db.Chunk(ctx, func(x paneldb.Execer) error {
		var err error
		ok, err = p.writer.RemoveAdmin(ctx, x, e.Row)
		return err
	})
	if err != nil || !ok {
		return err
	}
	return s.ledger.Set(ctx, ledger.Compensated, []int64{e.ID})
}

// leftovers lists destination ids the job wrote that are still done:
// left, and held, the ones a later batch holds (D14).
func (s *Service) leftovers(ctx context.Context, j *Job) (left, held []int64, err error) {
	rows, err := s.ledger.Job(ctx, j.ID, ledger.Done, ledger.Pending)
	if err != nil {
		return nil, nil, err
	}
	foreign, err := s.batches.Foreign(ctx, j.ID)
	if err != nil {
		return nil, nil, err
	}
	for _, e := range rows {
		switch {
		case e.Kind != ledger.KindUser:
		case foreign[e.ID]:
			held = append(held, e.Row.ID)
		default:
			left = append(left, e.Row.ID)
		}
	}
	return left, held, nil
}

// restore puts the Safe Mode backup back with the panel stopped, then
// starts the panel again (D36). It reports whether the backup is back.
func (s *Service) restore(ctx context.Context, j *Job, p *prepared, rep *Report) bool {
	if j.backupID == "" {
		s.emit(j, StageRestore, EventLog, map[string]any{"code": "restore_no_backup"})
		return false
	}
	ctl, err := s.control(ctx, j.envPath)
	if err != nil {
		s.emit(j, StageRestore, EventLog, map[string]any{"code": "restore_unavailable", "backup": j.backupID})
		return false
	}
	err = s.stage(j, rep, StageRestore, func() error {
		b, err := s.backupRecord(ctx, j.backupID)
		if err != nil {
			return err
		}
		return restoreStopped(ctx, ctl, func() error {
			return safemode.Restorer{Key: s.Key, Run: s.Tools}.Restore(ctx, p.target, p.db.Server(), b)
		}, s.noter(j, StageRestore))
	})
	if err != nil {
		return false
	}
	// The backup holds none of the job's rows, so its ledger rows are done with.
	if err := s.markRestored(ctx, j.ID); err != nil {
		s.emit(j, StageRestore, EventLog, map[string]any{"code": "ledger_not_updated"})
	}
	rep.State, rep.Left, rep.LeftReasons = StateRolledBack, nil, nil
	rep.NotUndoable = append(rep.NotUndoable, UndoLostPanelWrites)
	return true
}

// Log codes of a restore's steps (B12), so the live view can show the
// panel going offline and coming back.
const (
	codePanelStopped = "panel_stopped"
	codeBackupBack   = "backup_put_back"
	codePanelStarted = "panel_started"
)

// restoreStopped runs put with the panel stopped, and always starts the
// panel again once it was stopped. note hears each step that went well.
func restoreStopped(ctx context.Context, ctl PanelControl, put func() error, note func(code string)) error {
	if err := ctl.Stop(ctx); err != nil {
		return err
	}
	note(codePanelStopped)
	putErr := put()
	if putErr == nil {
		note(codeBackupBack)
	}
	err := ctl.Start(ctx)
	if err == nil {
		note(codePanelStarted)
	}
	if putErr != nil {
		return putErr
	}
	return err
}

// markRestored retires a job's written rows after its backup went back.
func (s *Service) markRestored(ctx context.Context, jobID string) error {
	rows, err := s.ledger.Job(ctx, jobID, ledger.Done, ledger.Pending, ledger.Unknown)
	if err != nil {
		return err
	}
	ids := make([]int64, 0, len(rows))
	for _, e := range rows {
		ids = append(ids, e.ID)
	}
	return s.ledger.Set(ctx, ledger.Compensated, ids)
}

func (s *Service) backupRecord(ctx context.Context, id string) (safemode.Backup, error) {
	b := safemode.Backup{ID: id}
	var created int64
	err := s.db.QueryRowContext(ctx, "SELECT path, size, method, created_at FROM safe_mode_backups WHERE id = ?", id).
		Scan(&b.Path, &b.Size, &b.Method, &created)
	b.CreatedAt = time.UnixMilli(created).UTC()
	return b, err
}
