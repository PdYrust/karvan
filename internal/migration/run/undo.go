package run

import (
	"context"
	"slices"
	"time"

	"github.com/PdYrust/karvan/internal/migration/ledger"
	"github.com/PdYrust/karvan/internal/panel"
	"github.com/PdYrust/karvan/internal/panel/paneldb"
	"github.com/PdYrust/karvan/internal/safemode"
)

// undo is the Safe Mode rollback (D35, D36): remove exactly this job's
// rows, newest batch first, each batch in its own short transaction; check
// that nothing of the job is left; if the undo failed or something is
// left, restore the backup with the panel stopped.
func (s *Service) undo(ctx context.Context, j *Job, p *prepared, rep *Report) {
	// The run's own context may be cancelled; the undo must still finish.
	ctx = context.WithoutCancel(ctx)
	undoErr := s.stage(j, rep, StageUndo, func() error { return s.compensate(ctx, j, p, rep) })
	left, err := s.leftovers(ctx, j)
	if undoErr == nil && err == nil && len(left) == 0 {
		rep.State = StateRolledBack
		return
	}
	rep.Left = left
	s.emit(j, StageUndo, EventLog, map[string]any{"code": "undo_incomplete", "left": len(left)})
	s.restore(ctx, j, p, rep)
}

func (s *Service) compensate(ctx context.Context, j *Job, p *prepared, rep *Report) error {
	rows, err := s.ledger.Job(ctx, j.ID, ledger.Done)
	if err != nil {
		return err
	}
	batches := map[int][]ledger.Entry{}
	var order []int
	var admin *ledger.Entry
	for i, e := range rows {
		if e.Kind == ledger.KindAdmin {
			admin = &rows[i]
			continue
		}
		if _, ok := batches[e.Batch]; !ok {
			order = append(order, e.Batch)
		}
		batches[e.Batch] = append(batches[e.Batch], e)
	}
	slices.Sort(order)
	slices.Reverse(order)
	for _, n := range order {
		if err := s.removeBatch(ctx, p, batches[n], rep); err != nil {
			return err
		}
	}
	if admin != nil {
		return s.removeAdmin(ctx, p, *admin)
	}
	return nil
}

func (s *Service) removeBatch(ctx context.Context, p *prepared, batch []ledger.Entry, rep *Report) error {
	rows := make([]panel.Row, len(batch))
	byID := map[int64]int64{}
	for i, e := range batch {
		rows[i] = e.Row
		byID[e.Row.ID] = e.ID
	}
	var removed []int64
	err := p.db.Batch(ctx, func(x paneldb.Execer) error {
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
	err := p.db.Batch(ctx, func(x paneldb.Execer) error {
		var err error
		ok, err = p.writer.RemoveAdmin(ctx, x, e.Row)
		return err
	})
	if err != nil || !ok {
		return err
	}
	return s.ledger.Set(ctx, ledger.Compensated, []int64{e.ID})
}

// leftovers lists destination ids the job wrote that are still done.
func (s *Service) leftovers(ctx context.Context, j *Job) ([]int64, error) {
	rows, err := s.ledger.Job(ctx, j.ID, ledger.Done, ledger.Pending)
	if err != nil {
		return nil, err
	}
	var out []int64
	for _, e := range rows {
		if e.Kind == ledger.KindUser {
			out = append(out, e.Row.ID)
		}
	}
	return out, nil
}

// restore puts the Safe Mode backup back with the panel stopped. Without
// the stop and start permission it only tells the operator how (D36).
func (s *Service) restore(ctx context.Context, j *Job, p *prepared, rep *Report) {
	var ctl PanelControl
	if s.Control != nil {
		ctl = s.Control(j.envPath)
	}
	if ctl == nil || j.backupID == "" {
		s.emit(j, StageRestore, EventLog, map[string]any{"code": "restore_manual", "backup": j.backupID})
		return
	}
	err := s.stage(j, rep, StageRestore, func() error {
		b, err := s.backupRecord(ctx, j.backupID)
		if err != nil {
			return err
		}
		if err := ctl.Stop(ctx); err != nil {
			return err
		}
		restoreErr := safemode.Restorer{Key: s.Key, Run: s.Tools}.Restore(ctx, p.target, p.db.Server(), b)
		if err := ctl.Start(ctx); err != nil && restoreErr == nil {
			return err
		}
		return restoreErr
	})
	if err == nil {
		rep.State = StateRolledBack
		rep.NotUndoable = append(rep.NotUndoable, "restore_lost_panel_writes")
	}
}

func (s *Service) backupRecord(ctx context.Context, id string) (safemode.Backup, error) {
	b := safemode.Backup{ID: id}
	var created int64
	err := s.db.QueryRowContext(ctx, "SELECT path, size, method, created_at FROM safe_mode_backups WHERE id = ?", id).
		Scan(&b.Path, &b.Size, &b.Method, &created)
	b.CreatedAt = time.UnixMilli(created).UTC()
	return b, err
}
