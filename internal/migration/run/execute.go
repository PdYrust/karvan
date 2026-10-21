package run

import (
	"context"
	"errors"
	"fmt"
	"maps"

	"golang.org/x/crypto/bcrypt"

	"github.com/PdYrust/karvan/internal/migration/dest"
	"github.com/PdYrust/karvan/internal/panel"
	"github.com/PdYrust/karvan/internal/panel/paneldb"
	"github.com/PdYrust/karvan/internal/panel/pasarguard"
	"github.com/PdYrust/karvan/internal/safemode"
	"github.com/PdYrust/karvan/internal/secret"
)

// Options are the operator's last choices (step 5).
type Options struct {
	Version       int
	AdminPassword secret.Value // empty: generate one (owner decision 1)
	Reload        string
	SafeMode      bool
	// Rollback is what Safe Mode does on failure: RollbackUndo (the
	// default) or RollbackBackup.
	Rollback string
}

// What Safe Mode does on failure (owner, 2026-10-16).
const (
	RollbackUndo   = "undo"   // remove the job's rows; the backup only if some are left
	RollbackBackup = "backup" // put the whole backup back
)

// errStale means the source or the destination changed since the plan
// was approved (D29); the job's changes say how.
var errStale = errors.New("the outcome changed since the plan was approved")

// bcryptCost matches PasarGuard (app/models/admin.py:26).
const bcryptCost = 12

// Run starts the job in the background. It returns once the run is
// accepted; progress comes as events.
func (s *Service) Run(ctx context.Context, jobID string, o Options) error {
	j, err := s.job(jobID)
	if err != nil {
		return err
	}
	if !j.lockIdle() {
		return ErrBusy
	}
	defer j.mu.Unlock()
	if j.state == StateRunning || j.checking.Load() {
		return ErrBusy
	}
	if err := s.canRun(j, o); err != nil {
		return err
	}
	if _, ok := s.Reloaders[o.Reload]; !ok {
		return fmt.Errorf("%w %q", ErrBadReload, o.Reload)
	}
	if o.SafeMode && o.Rollback == RollbackBackup {
		// Checked now, not on failure: a run that could not put its backup
		// back must not start.
		if _, err := s.control(ctx, j.envPath); err != nil {
			return err
		}
	}
	before := maps.Clone(j.passwords)
	password, err := s.adminPassword(j, o.AdminPassword)
	if err != nil {
		return err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password.Reveal()), bcryptCost)
	if err != nil {
		j.passwords = before
		return err
	}
	state, safe, rollback := j.state, j.safeMode, j.rollback
	j.state, j.safeMode, j.rollback, j.lastUse = StateRunning, o.SafeMode, o.Rollback, s.now()
	if err := s.save(ctx, j); err != nil {
		j.state, j.safeMode, j.rollback, j.passwords = state, safe, rollback, before
		return err
	}
	s.start(j, string(hash), o)
	return nil
}

// start launches the run goroutine. The caller holds j.mu; the goroutine
// takes it once the caller lets go and keeps it to the end.
func (s *Service) start(j *Job, hash string, o Options) {
	runCtx, cancel := context.WithCancel(context.Background())
	s.mu.Lock()
	s.cancels[j.ID] = cancel
	s.mu.Unlock()
	j.frozen.Store(&frozenView{result: j.result, plan: j.plan, version: j.version})
	j.running.Store(true)
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		defer func() {
			cancel()
			s.mu.Lock()
			delete(s.cancels, j.ID)
			s.mu.Unlock()
		}()
		s.execute(runCtx, j, hash, o)
	}()
}

func (s *Service) canRun(j *Job, o Options) error {
	switch {
	case j.result == nil:
		return ErrNotChecked
	case j.source == nil:
		return ErrNoLogin
	case o.Version != j.version:
		return ErrStaleVer
	case j.result.Blockers > 0 || !j.result.Stable:
		return ErrBlocked
	}
	return nil
}

// Where the moved admin's password comes from.
const (
	PasswordKept      = "kept"      // the reseller's own, from the old panel's login
	PasswordTyped     = "typed"     // the operator typed one
	PasswordGenerated = "generated" // in passwords.csv, once
)

// adminPassword uses a typed password, else the reseller's own one from
// the old panel's login, else generates one. A generated one is kept
// until the operator downloads it once (D22).
func (s *Service) adminPassword(j *Job, typed secret.Value) (secret.Value, error) {
	name := j.result.Admin.Username
	if j.result.Admin.Merge != nil {
		j.passwordFrom = ""
		return secret.NewValue("unused-for-a-merged-admin"), nil
	}
	if !typed.IsZero() {
		code := pasarguard.CheckPassword(typed.Reveal(), name)
		if j.facts.Kind == panel.Marzban && len(typed.Reveal()) >= 12 && code != pasarguard.PasswordLong {
			code = "" // Marzban has no policy; Karvan asks for 12 characters
		}
		if code != "" {
			return secret.Value{}, &WeakPasswordError{Rule: code}
		}
		j.passwordFrom = PasswordTyped
		return typed, nil
	}
	// The reseller signed in to the old panel with this password, so the
	// moved admin keeps it (owner, 2026-10-16). bcrypt reads 72 bytes.
	if l := j.login; l.APIKey.IsZero() && !l.Password.IsZero() && len(l.Password.Reveal()) <= 72 && l.Username == j.snapshot.Admin.Username {
		j.passwordFrom = PasswordKept
		return l.Password, nil
	}
	j.passwordFrom = PasswordGenerated
	p, err := pasarguard.GeneratePassword(name)
	if err != nil {
		return secret.Value{}, err
	}
	j.passwords[name] = p
	return secret.NewValue(p), nil
}

// execute runs the stages. Any failure after the first write goes to undo
// when Safe Mode is on.
func (s *Service) execute(ctx context.Context, j *Job, hash string, o Options) {
	start := s.now()
	j.mu.Lock()
	defer j.mu.Unlock()
	defer j.running.Store(false)
	rep := newReport(j, start)
	j.report = rep
	w, err := s.prepare(ctx, j, rep)
	if err == nil {
		defer w.release()
		err = s.write(ctx, j, w, hash, rep)
		if err == nil {
			err = s.verify(ctx, j, w, rep)
		}
		if err != nil && w.wrote && j.safeMode {
			s.rollBack(ctx, j, w, rep)
		}
	}
	switch {
	case err == nil:
		s.links(ctx, j, w, rep)
		s.reload(ctx, j, rep, o.Reload)
		j.state = StateDone
	case rep.State == StateRolledBack:
		j.state = StateRolledBack
	default:
		j.state = StateFailed
	}
	if err != nil {
		rep.Error = errorCode(err)
		s.emit(j, StageExecute, EventLog, map[string]any{"code": rep.Error})
	}
	rep.State, rep.EndedAt, rep.BackupID = j.state, s.now(), j.backupID
	j.wipe()
	_ = s.save(context.Background(), j)
	// Not running any more before "done" goes out: a client that asks for
	// the report on "done" then waits for mu instead of getting ErrBusy.
	j.running.Store(false)
	s.emit(j, "", EventDone, map[string]any{"state": j.state})
}

// links adds each moved user's new subscription link to the report. A
// failure here only leaves the links out; the move itself is done.
func (s *Service) links(ctx context.Context, j *Job, p *prepared, rep *Report) {
	l, err := p.panel.Links(ctx, p.adminID)
	if err != nil {
		s.emit(j, StageVerify, EventLog, map[string]any{"code": "links_unavailable"})
		return
	}
	for i := range rep.Users {
		if u := &rep.Users[i]; u.Moved {
			u.NewLink = l.Link(u.DestID, u.Username, s.now())
		}
	}
}

// prepared holds what the write stages share.
type prepared struct {
	panel   *dest.Panel
	db      *paneldb.Writer
	target  paneldb.Target
	writer  dest.Writer
	dest    string
	unlock  func()
	wrote   bool
	adminID int64
	created bool // the job created the admin (not a merge)
}

func (p *prepared) release() {
	p.unlock()
	_ = p.db.Close()
}

// prepare re-checks (D29), locks the destination (D30) and backs it up
// (D42).
func (s *Service) prepare(ctx context.Context, j *Job, rep *Report) (*prepared, error) {
	if err := s.stage(j, rep, StageCheck, func() error { return s.recheck(ctx, j) }); err != nil {
		return nil, err
	}
	d, err := dest.Open(ctx, j.envPath)
	if err != nil {
		return nil, err
	}
	rep.Destination.DB, rep.Destination.Name = d.Target.String(), d.Found.Name
	db, err := paneldb.OpenWriter(ctx, d.Target)
	if err != nil {
		return nil, err
	}
	unlock, err := db.Lock(ctx, d.Target)
	if err != nil {
		_ = db.Close()
		return nil, err
	}
	p := &prepared{db: db, target: d.Target, dest: d.Target.String(), unlock: unlock, panel: d,
		writer: dest.NewWriter(j.facts.Kind, db.Server().Dialect, j.facts)}
	if err := s.settle(ctx, j, p); err != nil {
		p.release()
		return nil, err
	}
	if j.safeMode {
		if err := s.stage(j, rep, StageBackup, func() error { return s.backup(ctx, j, p) }); err != nil {
			p.release()
			return nil, err
		}
	}
	return p, nil
}

// recheck reads everything again and refuses a plan whose outcome moved.
func (s *Service) recheck(ctx context.Context, j *Job) error {
	f := j.readFrom()
	src, err := readSource(ctx, f)
	if err != nil {
		return err
	}
	r, err := s.readDest(ctx, f, src)
	if err != nil {
		return err
	}
	changes, err := s.install(j, r)
	if err != nil {
		return err
	}
	if changes.Changed() {
		return errStale
	}
	return nil
}

func (s *Service) stage(j *Job, rep *Report, name string, fn func() error) error {
	start := s.now()
	s.emit(j, name, EventStage, map[string]any{"state": "start"})
	err := fn()
	state := "done"
	if err != nil {
		state = "failed"
	}
	rep.stage(name, start, err == nil, s.now())
	s.emit(j, name, EventStage, map[string]any{"state": state})
	return err
}

func errorCode(err error) string {
	var verify *VerifyError
	var tool *safemode.ToolError
	var space *safemode.SpaceError
	switch {
	case errors.Is(err, errStale):
		return "plan_stale"
	case errors.As(err, &verify):
		return "verify_failed"
	case errors.As(err, &tool):
		return "backup_tool_missing"
	case errors.As(err, &space):
		return "backup_no_space"
	case errors.Is(err, paneldb.ErrLocked):
		return "destination_locked"
	case errors.Is(err, context.Canceled):
		return "cancelled"
	}
	return "run_failed"
}
