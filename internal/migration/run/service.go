// Package run is the migration job runner: it creates jobs, checks them
// (read only), and runs them through backup, write, verify and reload,
// with the ledger protocol, settlement and Safe Mode rollback (D30-D50).
package run

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"path/filepath"
	"sync"
	"time"

	"github.com/PdYrust/karvan/internal/migration/ledger"
	"github.com/PdYrust/karvan/internal/panel/paneldb"
	"github.com/PdYrust/karvan/internal/safemode"
	"github.com/PdYrust/karvan/internal/secret"
)

// IdleTimeout drops a job's credentials when nobody used it (D11).
const IdleTimeout = 2 * time.Hour

// Errors the HTTP layer maps to codes.
var (
	ErrNoJob       = errors.New("no such job")
	ErrStaleVer    = errors.New("the plan changed since it was read")
	ErrBlocked     = errors.New("conflicts still need a choice")
	ErrBusy        = errors.New("the job is running")
	ErrNotChecked  = errors.New("check the job first")
	ErrNoLogin     = errors.New("the old panel's login is gone; sign in again")
	ErrBadPassword = errors.New("the admin password does not meet the policy")
	ErrNoRollback  = errors.New("only a failed or interrupted job can be rolled back")
	ErrBadReload   = errors.New("unknown reload method")
	// ErrLocked means another job or another Karvan writes to the
	// destination now.
	ErrLocked = paneldb.ErrLocked
	// ErrNoPanelControl means Karvan cannot stop and start the panel, so
	// it cannot put a whole backup back.
	ErrNoPanelControl = errors.New("cannot stop and start this panel")
)

// WeakPasswordError is ErrBadPassword with the policy rule the typed
// password broke: one of the pasarguard.Password* codes.
type WeakPasswordError struct{ Rule string }

func (e *WeakPasswordError) Error() string { return ErrBadPassword.Error() + ": " + e.Rule }
func (e *WeakPasswordError) Unwrap() error { return ErrBadPassword }

// PanelControl stops and starts a destination panel, for a full backup
// restore (D36).
type PanelControl interface {
	Stop(ctx context.Context) error
	Start(ctx context.Context) error
}

// Deps are what the service needs.
type Deps struct {
	DB      *sql.DB
	Key     *secret.Key
	DataDir string
	Now     func() time.Time
	Tools   safemode.Runner // runs dump tools
	// Control returns the control for the panel whose ENV is at envPath,
	// or why Karvan cannot stop and start it. Nil: never.
	Control func(ctx context.Context, envPath string) (PanelControl, error)
	// Reloaders are the ways to make a panel pick up written users (D46).
	Reloaders map[string]Reloader
	// BatchSize is users per destination transaction (D15).
	BatchSize func(d paneldb.Dialect) int
	// Fault, in tests, fails a step on purpose.
	Fault func(step string, batch int) error
}

// Service owns all jobs.
type Service struct {
	Deps
	db     *sql.DB
	now    func() time.Time
	ledger *ledger.Ledger
	events *bus

	mu      sync.Mutex
	jobs    map[string]*Job
	cancels map[string]context.CancelFunc
	probes  map[string]controlProbe // panel control by ENV path
	wg      sync.WaitGroup
}

// New returns a service. Call Recover once at start.
func New(d Deps) *Service {
	if d.Now == nil {
		d.Now = time.Now
	}
	if d.Tools == nil {
		d.Tools = safemode.DefaultRunner
	}
	if d.BatchSize == nil {
		d.BatchSize = func(d paneldb.Dialect) int {
			if d == paneldb.SQLite {
				return 50
			}
			return 500
		}
	}
	if d.Reloaders == nil {
		d.Reloaders = map[string]Reloader{}
	}
	if _, ok := d.Reloaders[ReloadManual]; !ok {
		d.Reloaders[ReloadManual] = manualReload{}
	}
	return &Service{Deps: d, db: d.DB, now: d.Now, ledger: ledger.New(d.DB), events: newBus(d.DB, d.Now), jobs: map[string]*Job{},
		cancels: map[string]context.CancelFunc{}, probes: map[string]controlProbe{}}
}

// control opens the panel control for envPath.
func (s *Service) control(ctx context.Context, envPath string) (PanelControl, error) {
	if s.Control == nil {
		return nil, ErrNoPanelControl
	}
	c, err := s.Control(ctx, envPath)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrNoPanelControl, err)
	}
	return c, nil
}

// SafeModeDir is where Safe Mode backups live, outside any rotation.
func (s *Service) SafeModeDir() string { return filepath.Join(s.DataDir, "safe-mode") }

// Wait blocks until running jobs end (shutdown, tests).
func (s *Service) Wait() { s.wg.Wait() }

// Shutdown stops every running job between batches (Safe Mode undoes
// them) and waits for them to end.
func (s *Service) Shutdown() {
	s.mu.Lock()
	for _, c := range s.cancels {
		c()
	}
	s.mu.Unlock()
	s.wg.Wait()
}

func (s *Service) job(id string) (*Job, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	j, ok := s.jobs[id]
	if !ok {
		return nil, ErrNoJob
	}
	return j, nil
}

// RegisterPanel records a discovered panel and returns its id.
func (s *Service) RegisterPanel(ctx context.Context, envPath, name, kind string) (int64, error) {
	_, err := s.db.ExecContext(ctx, `INSERT INTO panels (env_path, name, kind, scanned_at) VALUES (?, ?, ?, ?)
		ON CONFLICT (env_path) DO UPDATE SET name = excluded.name, kind = excluded.kind, scanned_at = excluded.scanned_at`,
		envPath, name, kind, s.now().UnixMilli())
	if err != nil {
		return 0, fmt.Errorf("register panel: %w", err)
	}
	var id int64
	err = s.db.QueryRowContext(ctx, "SELECT id FROM panels WHERE env_path = ?", envPath).Scan(&id)
	return id, err
}

func (s *Service) envPath(ctx context.Context, panelID int64) (string, error) {
	var p string
	err := s.db.QueryRowContext(ctx, "SELECT env_path FROM panels WHERE id = ?", panelID).Scan(&p)
	if errors.Is(err, sql.ErrNoRows) {
		return "", fmt.Errorf("%w: panel %d", ErrNotFound, panelID)
	}
	return p, err
}

// Sweep drops the credentials of jobs idle longer than IdleTimeout. It
// never holds s.mu while it takes a job's lock (Run takes them the other
// way round), and skips a job that is busy now.
func (s *Service) Sweep() {
	s.mu.Lock()
	jobs := make([]*Job, 0, len(s.jobs))
	for _, j := range s.jobs {
		jobs = append(jobs, j)
	}
	s.mu.Unlock()
	for _, j := range jobs {
		if j.running.Load() || !j.mu.TryLock() {
			continue
		}
		// A compare reads without the lock and sets checking under it, so
		// the flag only counts once the lock is held.
		if !j.running.Load() && !j.checking.Load() && s.now().Sub(j.lastUse) > IdleTimeout {
			j.wipe()
		}
		j.mu.Unlock()
	}
}

// dropCancel forgets a job's cancel once its background work ends. The
// caller holds the job's lock, so the job's next run or compare cannot
// have put its own there yet.
func (s *Service) dropCancel(id string) {
	s.mu.Lock()
	delete(s.cancels, id)
	s.mu.Unlock()
}

// Events returns stored events after seq and a channel of new ones.
func (s *Service) Events(ctx context.Context, jobID string, after int64) ([]Event, <-chan Event, func(), error) {
	return s.events.subscribe(ctx, jobID, after)
}

func (s *Service) emit(j *Job, stage, typ string, data map[string]any) {
	// Events are best effort: a failed event store must not fail a write.
	_ = s.events.emit(context.Background(), j.ID, stage, typ, data)
}
