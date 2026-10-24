package run

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/PdYrust/karvan/internal/migration/engine"
	"github.com/PdYrust/karvan/internal/migration/plan"
	"github.com/PdYrust/karvan/internal/panel"
	"github.com/PdYrust/karvan/internal/secret"
)

// Job states.
const (
	StateCreated     = "created"
	StateChecked     = "checked"
	StateRunning     = "running"
	StateDone        = "done"
	StateFailed      = "failed"
	StateRolledBack  = "rolled_back"
	StateInterrupted = "interrupted"
)

// Login is how Karvan reads the old panel. It lives only in the job's
// memory and is dropped when the job ends or sits idle (D11).
type Login struct {
	URL       string
	Kind      panel.Kind // empty: detect
	Username  string
	Password  secret.Value
	APIKey    secret.Value
	PinSHA256 string
}

// Job is one migration from one old panel into one discovered panel.
type Job struct {
	ID      string
	PanelID int64

	// running is set for the whole run, during which the run goroutine
	// holds mu; readers then use frozen instead of waiting.
	running atomic.Bool
	frozen  atomic.Pointer[frozenView]
	// checking is set while a compare reads the panels in the background.
	// It holds mu only to put its build in place, so readers keep the
	// previous build meanwhile; whatever would change the job is busy.
	checking atomic.Bool

	mu         sync.Mutex
	state      string
	envPath    string
	login      Login
	source     panel.Source
	kind       panel.Kind
	origin     string
	snapshot   *engine.Source
	facts      engine.Dest
	moved      map[string]engine.Moved
	movedAdmin string
	plan       plan.Plan
	version    int
	generated  map[string]panel.Credentials
	result     *engine.Result
	// changes is what the last read changed in the outcome, and checkedAt
	// when the panels were read.
	changes   engine.Changes
	checkedAt time.Time
	safeMode  bool
	rollback  string // RollbackUndo or RollbackBackup
	backupID  string
	lastUse   time.Time
	report    *Report
	passwords map[string]string // generated admin passwords, shown once
	// passwordFrom says where the moved admin's password came from.
	passwordFrom string
}

// frozenView is what readers see while the job runs.
type frozenView struct {
	result  *engine.Result
	plan    plan.Plan
	version int
}

// readFrom is what reading the panels needs from a job, copied under its
// lock so a compare can read without holding it.
type readFrom struct {
	source    panel.Source
	kind      panel.Kind
	origin    string
	envPath   string
	planAdmin string // the admin's name in the plan, when it renames it
}

func (j *Job) readFrom() readFrom {
	return readFrom{source: j.source, kind: j.kind, origin: j.origin, envPath: j.envPath, planAdmin: j.plan.Admin.Username}
}

// reading is one read of both panels: the old panel's admin and users,
// the destination's facts, and who the ledger says moved before.
type reading struct {
	snapshot   *engine.Source
	facts      engine.Dest
	moved      map[string]engine.Moved
	movedAdmin string
}

// current is the job's last reading.
func (j *Job) current() reading {
	return reading{snapshot: j.snapshot, facts: j.facts, moved: j.moved, movedAdmin: j.movedAdmin}
}

// inputs are what a build of the job's plan over r reads.
func (j *Job) inputs(r reading) engine.Inputs {
	return engine.Inputs{Source: *r.snapshot, Dest: r.facts, Moved: r.moved, MovedAdmin: r.movedAdmin, Generate: generate, Generated: j.generated}
}

func (j *Job) rollbackOrUndo() string {
	if j.rollback == RollbackBackup {
		return RollbackBackup
	}
	return RollbackUndo
}

// ownPassword reports whether the reseller signed in to the old panel with
// their own username and password, so the moved admin can keep it (owner,
// 2026-10-16). bcrypt reads 72 bytes. The caller holds mu.
func (j *Job) ownPassword() bool {
	l := j.login
	return j.snapshot != nil && l.APIKey.IsZero() && !l.Password.IsZero() && len(l.Password.Reveal()) <= 72 &&
		l.Username == j.snapshot.Admin.Username
}

// lockIdle locks j unless it runs, and never waits for a run, which holds
// mu from start to end. A caller that loses the race to a starting run
// would otherwise wait for the whole migration.
func (j *Job) lockIdle() bool {
	for {
		if j.running.Load() {
			return false
		}
		if j.mu.TryLock() {
			if j.running.Load() {
				j.mu.Unlock()
				return false
			}
			return true
		}
		time.Sleep(10 * time.Millisecond)
	}
}

// State reports the job state.
func (j *Job) State() string {
	if !j.lockIdle() {
		return StateRunning
	}
	defer j.mu.Unlock()
	return j.state
}

// wipe drops everything secret the job holds (D11).
func (j *Job) wipe() {
	if j.source != nil {
		j.source.Close()
		j.source = nil
	}
	j.login = Login{}
	j.generated = map[string]panel.Credentials{}
	j.snapshot = nil
}

func (s *Service) save(ctx context.Context, j *Job) error {
	raw, err := json.Marshal(j.plan)
	if err != nil {
		return err
	}
	var report, counts []byte
	var ended int64
	if j.report != nil {
		if report, err = json.Marshal(j.report.WithoutLinks()); err != nil {
			return err
		}
		if counts, err = json.Marshal(j.report.Counts); err != nil {
			return err
		}
		if !j.report.EndedAt.IsZero() {
			ended = j.report.EndedAt.UnixMilli()
		}
	}
	admin := ""
	if j.snapshot != nil {
		admin = j.snapshot.Admin.Username
	}
	_, err = s.db.ExecContext(ctx, `INSERT INTO jobs (id, state, source_kind, source_origin, source_admin, panel_id, plan,
		plan_version, safe_mode, rollback, backup_id, report, counts, ended_at, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT (id) DO UPDATE SET state = excluded.state, source_kind = excluded.source_kind,
		source_admin = CASE WHEN excluded.source_admin = '' THEN jobs.source_admin ELSE excluded.source_admin END,
		plan = excluded.plan, plan_version = excluded.plan_version, safe_mode = excluded.safe_mode, rollback = excluded.rollback,
		backup_id = excluded.backup_id, report = excluded.report, counts = excluded.counts, ended_at = excluded.ended_at,
		updated_at = excluded.updated_at`,
		j.ID, j.state, string(j.kind), j.origin, admin, j.PanelID, string(raw), j.version, j.safeMode, j.rollbackOrUndo(), j.backupID,
		string(report), string(counts), ended, s.now().UnixMilli(), s.now().UnixMilli())
	if err != nil {
		return fmt.Errorf("save job %s: %w", j.ID, err)
	}
	return nil
}
