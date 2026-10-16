package run

import (
	"context"
	"crypto/rand"
	"encoding/hex"
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

// Check reads the old panel and the destination (read only) and builds
// the current plan over them (steps 1 and 2).
func (s *Service) Check(ctx context.Context, jobID string) (*engine.Result, error) {
	j, err := s.job(jobID)
	if err != nil {
		return nil, err
	}
	if !j.lockIdle() {
		return nil, ErrBusy
	}
	defer j.mu.Unlock()
	if j.source == nil {
		return nil, ErrNoLogin
	}
	if err := s.read(ctx, j); err != nil {
		return nil, err
	}
	if err := s.rebuild(j); err != nil {
		return nil, err
	}
	j.state, j.lastUse = StateChecked, s.now()
	return j.result, s.save(ctx, j)
}

// read takes a fresh snapshot of the source, the destination and the
// ledger.
func (s *Service) read(ctx context.Context, j *Job) error {
	admin, err := j.source.CurrentAdmin(ctx)
	if err != nil {
		return err
	}
	users, err := j.source.Users(ctx)
	if err != nil {
		return err
	}
	j.snapshot = &engine.Source{Kind: j.kind, Origin: j.origin, Admin: admin, Users: users, Skew: skewOf(j.source)}
	d, err := dest.Open(ctx, j.envPath)
	if err != nil {
		return err
	}
	if err := s.readLedger(ctx, j, d.Target.String()); err != nil {
		return err
	}
	names := []string{admin.Username}
	for _, n := range []string{j.plan.Admin.Username, j.movedAdmin} {
		if n != "" {
			names = append(names, n)
		}
	}
	j.facts, err = d.Facts(ctx, names)
	return err
}

// skewOf reads the clock skew the HTTP client saw, when it has one.
func skewOf(src panel.Source) time.Duration {
	if s, ok := src.(interface{ Skew() time.Duration }); ok {
		return s.Skew()
	}
	return 0
}

// readLedger maps live ledger rows to source users. A row only counts if
// the source user still has the same created_at; otherwise it is someone
// new with a reused id or name (D31).
func (s *Service) readLedger(ctx context.Context, j *Job, destID string) error {
	live, err := s.ledger.Live(ctx, j.origin, destID)
	if err != nil {
		return err
	}
	created := map[string]time.Time{}
	for _, u := range j.snapshot.Users {
		created[engine.KeyOf(j.kind, u)] = u.CreatedAt
	}
	j.moved, j.movedAdmin = map[string]engine.Moved{}, ""
	for _, e := range live {
		if e.Kind == ledger.KindAdmin && e.SourceKey == j.snapshot.Admin.Username && e.State == ledger.Done {
			j.movedAdmin = e.Username
		}
		if e.Kind != ledger.KindUser || !e.SourceCreated.Equal(created[e.SourceKey].UTC()) {
			continue
		}
		state := e.State
		if state == ledger.Pending {
			state = ledger.Unknown // settled by the next run before it writes
		}
		j.moved[e.SourceKey] = engine.Moved{State: state, DestID: e.Row.ID, Username: e.Username, Created: e.CreatedAt}
	}
	return nil
}

func (s *Service) rebuild(j *Job) error {
	in := engine.Inputs{Source: *j.snapshot, Dest: j.facts, Moved: j.moved, MovedAdmin: j.movedAdmin, Generate: generate, Generated: j.generated}
	r, err := engine.Build(j.plan, in)
	if err != nil {
		return err
	}
	j.result, j.print = r, engine.Fingerprint(r)
	return nil
}

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
