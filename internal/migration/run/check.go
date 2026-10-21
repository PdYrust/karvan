package run

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"log/slog"
	"maps"
	"time"

	"github.com/PdYrust/karvan/internal/migration/dest"
	"github.com/PdYrust/karvan/internal/migration/engine"
	"github.com/PdYrust/karvan/internal/migration/ledger"
	"github.com/PdYrust/karvan/internal/migration/plan"
	"github.com/PdYrust/karvan/internal/panel"
	"github.com/PdYrust/karvan/internal/panel/source"
)

// Create starts a job for a login and a destination panel. Nothing is
// read yet.
func (s *Service) Create(ctx context.Context, l Login, panelID int64, sourceOpts source.Options) (*Job, error) {
	env, err := s.envPath(ctx, panelID)
	if err != nil {
		return nil, err
	}
	src, kind, origin, _, err := openSource(ctx, l, sourceOpts)
	if err != nil {
		return nil, err
	}
	b := make([]byte, 12)
	_, _ = rand.Read(b)
	j := &Job{ID: hex.EncodeToString(b), PanelID: panelID, state: StateCreated, envPath: env, login: l, source: src,
		kind: kind, origin: origin, generated: map[string]panel.Credentials{}, lastUse: s.now(), safeMode: true,
		plan: plan.Plan{Now: s.now().UTC().Truncate(time.Second)}, passwords: map[string]string{}}
	if err := s.save(ctx, j); err != nil {
		return nil, err
	}
	s.mu.Lock()
	s.jobs[j.ID] = j
	s.mu.Unlock()
	return j, nil
}

// Compare reads both panels again in the background and builds the plan
// over them (steps 1 and 2, B2). It returns once the read is under way.
// Progress and the outcome come as events: the read stage with a progress
// event for each page of users, then the build stage. Cancel and
// Shutdown stop it.
func (s *Service) Compare(jobID string) error {
	ctx, cancel := context.WithCancel(context.Background())
	j, f, err := s.beginCheck(jobID, cancel)
	if err != nil {
		cancel()
		return err
	}
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		defer cancel()
		_, _ = s.check(ctx, j, f)
	}()
	return nil
}

// Check is Compare that waits for the outcome and returns the new build.
func (s *Service) Check(ctx context.Context, jobID string) (*engine.Result, error) {
	j, f, err := s.beginCheck(jobID, nil)
	if err != nil {
		return nil, err
	}
	return s.check(ctx, j, f)
}

// beginCheck marks the job as checking and copies what the read needs. A
// cancel, when given, is what Cancel and Shutdown call to stop the read.
func (s *Service) beginCheck(jobID string, cancel context.CancelFunc) (*Job, readFrom, error) {
	j, err := s.job(jobID)
	if err != nil {
		return nil, readFrom{}, err
	}
	if !j.lockIdle() {
		return nil, readFrom{}, ErrBusy
	}
	defer j.mu.Unlock()
	switch {
	case j.checking.Load():
		return nil, readFrom{}, ErrBusy
	case j.source == nil:
		return nil, readFrom{}, ErrNoLogin
	}
	j.checking.Store(true)
	j.lastUse = s.now()
	if cancel != nil {
		s.mu.Lock()
		s.cancels[j.ID] = cancel
		s.mu.Unlock()
	}
	return j, j.readFrom(), nil
}

// check reads the old panel (the read stage), then the destination, and
// builds the plan over both (the build stage). The old panel is read
// without the job's lock; the new build is put in place under it, and
// only when all of it worked.
func (s *Service) check(ctx context.Context, j *Job, f readFrom) (*engine.Result, error) {
	s.emit(j, StageRead, EventStage, map[string]any{"state": "start"})
	src, err := readSource(source.WithProgress(ctx, s.readProgress(j)), f)
	if err != nil {
		j.mu.Lock()
		defer j.mu.Unlock()
		return nil, s.endCheck(j, StageRead, err)
	}
	s.emit(j, StageRead, EventStage, map[string]any{"state": "done"})
	s.emit(j, StageBuild, EventStage, map[string]any{"state": "start"})
	r, err := s.readDest(ctx, f, src)
	j.mu.Lock()
	defer j.mu.Unlock()
	if err == nil {
		err = s.install(j, r)
	}
	if err != nil {
		return nil, s.endCheck(j, StageBuild, err)
	}
	j.state = StateChecked
	// Saved even when the compare was cancelled just now: the build is in
	// place already.
	err = s.save(context.Background(), j)
	return j.result, s.endCheck(j, StageBuild, err)
}

// readProgress sends each page of users read as a progress event.
func (s *Service) readProgress(j *Job) func(source.Progress) {
	return func(p source.Progress) {
		s.emit(j, StageRead, EventProgress, map[string]any{"done": p.Read, "total": p.Total, "pass": p.Pass})
	}
}

// endCheck ends a compare and sends its last event: stage done, or the
// error's code as a log event and then stage failed. The caller holds
// j.mu, so nothing else can begin on the job before that event is out,
// and whatever waits for the lock finds the job free and its new build.
func (s *Service) endCheck(j *Job, stage string, err error) error {
	j.lastUse = s.now()
	s.dropCancel(j.ID)
	j.checking.Store(false)
	if err == nil {
		s.emit(j, stage, EventStage, map[string]any{"state": "done"})
		return nil
	}
	p := problemOf(err)
	if p["code"] == "internal" {
		slog.Warn("compare failed", "job", j.ID, "err", err)
	}
	s.emit(j, stage, EventLog, p)
	s.emit(j, stage, EventStage, map[string]any{"state": "failed"})
	return err
}

// problemOf names a failed compare for its log event: the code the API
// gives the same error, with its extras (the certificate of an old panel
// Karvan cannot verify, where a plan does not build).
func problemOf(err error) map[string]any {
	var srcErr *source.Error
	var planErr *engine.PlanError
	out := map[string]any{"code": "internal"}
	switch {
	case errors.Is(err, context.Canceled):
		out["code"] = "cancelled"
	case errors.As(err, &srcErr):
		maps.Copy(out, srcErr.Extra())
		out["code"] = "source_" + srcErr.Code
	case errors.Is(err, dest.ErrNotReady):
		out["code"] = "panel_not_ready"
	case errors.As(err, &planErr):
		out["code"], out["where"], out["problem"] = "plan_invalid", planErr.Where, planErr.Problem
	}
	return out
}

// readSource reads the old panel: its admin and every user.
func readSource(ctx context.Context, f readFrom) (*engine.Source, error) {
	admin, err := f.source.CurrentAdmin(ctx)
	if err != nil {
		return nil, err
	}
	users, err := f.source.Users(ctx)
	if err != nil {
		return nil, err
	}
	return &engine.Source{Kind: f.kind, Origin: f.origin, Admin: admin, Users: users, Skew: skewOf(f.source)}, nil
}

// readDest reads the destination (read only) and the ledger for a fresh
// read of the old panel.
func (s *Service) readDest(ctx context.Context, f readFrom, src *engine.Source) (reading, error) {
	d, err := dest.Open(ctx, f.envPath)
	if err != nil {
		return reading{}, err
	}
	moved, movedAdmin, err := s.readLedger(ctx, src, d.Target.String())
	if err != nil {
		return reading{}, err
	}
	names := []string{src.Admin.Username}
	for _, n := range []string{f.planAdmin, movedAdmin} {
		if n != "" {
			names = append(names, n)
		}
	}
	facts, err := d.Facts(ctx, names)
	if err != nil {
		return reading{}, err
	}
	return reading{snapshot: src, facts: facts, moved: moved, movedAdmin: movedAdmin}, nil
}

// skewOf reads the clock skew the HTTP client saw, when it has one.
func skewOf(src panel.Source) time.Duration {
	if s, ok := src.(interface{ Skew() time.Duration }); ok {
		return s.Skew()
	}
	return 0
}

// readLedger maps live ledger rows to the users of a source read, and
// finds the admin an earlier run moved. A row only counts if the source
// user still has the same created_at; otherwise it is someone new with a
// reused id or name (D31).
func (s *Service) readLedger(ctx context.Context, src *engine.Source, destID string) (map[string]engine.Moved, string, error) {
	live, err := s.ledger.Live(ctx, src.Origin, destID)
	if err != nil {
		return nil, "", err
	}
	created := map[string]time.Time{}
	for _, u := range src.Users {
		created[engine.KeyOf(src.Kind, u)] = u.CreatedAt
	}
	moved, movedAdmin := map[string]engine.Moved{}, ""
	for _, e := range live {
		if e.Kind == ledger.KindAdmin && e.SourceKey == src.Admin.Username && e.State == ledger.Done {
			movedAdmin = e.Username
		}
		if e.Kind != ledger.KindUser || !e.SourceCreated.Equal(created[e.SourceKey].UTC()) {
			continue
		}
		state := e.State
		if state == ledger.Pending {
			state = ledger.Unknown // settled by the next run before it writes
		}
		moved[e.SourceKey] = engine.Moved{State: state, DestID: e.Row.ID, Username: e.Username, Created: e.CreatedAt}
	}
	return moved, movedAdmin, nil
}

// install builds the plan over a fresh reading and makes both the job's
// own. A plan that does not build leaves the previous reading and build.
func (s *Service) install(j *Job, r reading) error {
	res, err := engine.Build(j.plan, j.inputs(r))
	if err != nil {
		return err
	}
	j.snapshot, j.facts, j.moved, j.movedAdmin = r.snapshot, r.facts, r.moved, r.movedAdmin
	j.result, j.print = res, engine.Fingerprint(res)
	return nil
}

// rebuild builds the plan again over the last reading (a plan edit).
func (s *Service) rebuild(j *Job) error { return s.install(j, j.current()) }

// SetPlan replaces the plan if version is current, and rebuilds on the
// last snapshot (steps 3 and 4).
func (s *Service) SetPlan(ctx context.Context, jobID string, version int, p plan.Plan) (*engine.Result, int, error) {
	j, err := s.job(jobID)
	if err != nil {
		return nil, 0, err
	}
	if !j.lockIdle() {
		return nil, 0, ErrBusy
	}
	defer j.mu.Unlock()
	switch {
	case j.checking.Load():
		return nil, j.version, ErrBusy
	case version != j.version:
		return nil, j.version, ErrStaleVer
	case j.snapshot == nil:
		return nil, j.version, ErrNotChecked
	}
	p.Now = j.plan.Now
	old := j.plan
	j.plan = p
	if err := s.rebuild(j); err != nil {
		j.plan = old
		return nil, j.version, err
	}
	j.version++
	j.lastUse = s.now()
	return j.result, j.version, s.save(ctx, j)
}

// View returns the job's last build and plan version.
func (s *Service) View(jobID string) (*engine.Result, plan.Plan, int, error) {
	j, err := s.job(jobID)
	if err != nil {
		return nil, plan.Plan{}, 0, err
	}
	if !j.lockIdle() {
		f := j.frozen.Load()
		return f.result, f.plan, f.version, nil
	}
	defer j.mu.Unlock()
	j.lastUse = s.now()
	return j.result, j.plan, j.version, nil
}
