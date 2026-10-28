// Package certorder gets certificates for domains from Let's Encrypt
// (phase-4.1 D66 to D73): one ACME account per directory, orders by
// HTTP-01 through the bridge's challenge port or by DNS-01 with a Check,
// resume at start, and renewal before expiry. The certificates are stored
// through the bridge, which serves them.
package certorder

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/PdYrust/karvan/internal/audit"
	"github.com/PdYrust/karvan/internal/bridge"
	"github.com/PdYrust/karvan/internal/secret"
	"github.com/PdYrust/karvan/internal/settings"
)

// Let's Encrypt directories (D66).
const (
	Production = "https://acme-v02.api.letsencrypt.org/directory"
	Staging    = "https://acme-staging-v02.api.letsencrypt.org/directory"
)

// Methods (D66).
const (
	HTTP01 = "http-01"
	DNS01  = "dns-01"
)

// Order states (D69).
const (
	stateStarting   = "starting"
	stateWaitingDNS = "waiting_dns"
	stateValidating = "validating"
	stateIssuing    = "issuing"
	stateDone       = "done"
	stateFailed     = "failed"
	stateCancelled  = "cancelled"
)

// runningStates is the SQL list of the states the running index covers.
const runningStates = "('starting', 'waiting_dns', 'validating', 'issuing')"

// talking is how many orders may talk to the CA at once (D69).
const talking = 3

var (
	// ErrBusy means the domain has a running order (acme_busy).
	ErrBusy = errors.New("the domain has a running Let's Encrypt order")
	// ErrState means the order is not waiting for its DNS record (acme_state).
	ErrState = errors.New("the order is not waiting for its DNS record")
	// errGone means the order was cancelled or deleted under its work.
	errGone = bridge.ErrOrderGone
)

// Recorder writes audit events.
type Recorder interface {
	Record(ctx context.Context, e audit.Event) error
}

// Deps are what the service runs on.
type Deps struct {
	DB       *sql.DB
	Key      *secret.Key
	Bridge   *bridge.Bridge
	Settings *settings.Store
	Audit    Recorder // nil: nothing is audited
	Log      *slog.Logger
	Now      func() time.Time
	// TestDirectory replaces both Let's Encrypt directories
	// (acmeTestDirectory, D74).
	TestDirectory string
	// HTTPClient talks to the CA; nil: http.DefaultClient.
	HTTPClient *http.Client
	// LookupTXT reads the TXT records of a name for a DNS-01 Check; nil:
	// the system resolver.
	LookupTXT func(ctx context.Context, name string) ([]string, error)
}

// Service runs Let's Encrypt orders.
type Service struct {
	Deps
	sem   chan struct{}
	mu    sync.Mutex
	jobs  map[int64]*job // by order id
	seq   int
	bg    context.Context
	stop  context.CancelFunc
	wg    sync.WaitGroup
	call  time.Duration // one CA call
	wait  time.Duration // an authorization or order wait
	retry time.Duration // the first retry of a failed CA call
}

// job is an order working in the background.
type job struct {
	domainID int64
	cancel   context.CancelFunc
	holder   string // its hold on the challenge port, "" for none
}

// New returns a service; Run resumes its orders and renews.
func New(d Deps) *Service {
	if d.Now == nil {
		d.Now = time.Now
	}
	if d.Log == nil {
		d.Log = slog.Default()
	}
	if d.HTTPClient == nil {
		d.HTTPClient = http.DefaultClient
	}
	if d.LookupTXT == nil {
		d.LookupTXT = net.DefaultResolver.LookupTXT
	}
	bg, stop := context.WithCancel(context.Background())
	return &Service{Deps: d, sem: make(chan struct{}, talking), jobs: map[int64]*job{}, bg: bg, stop: stop,
		call: 30 * time.Second, wait: 2 * time.Minute, retry: time.Second}
}

// directory is where new orders go: the test CA from the config, else
// staging or production as the setting says (D66, D74).
func (s *Service) directory(ctx context.Context) (string, error) {
	if s.TestDirectory != "" {
		return s.TestDirectory, nil
	}
	a, err := s.Settings.ACME(ctx)
	if err != nil {
		return "", err
	}
	if a.Staging {
		return Staging, nil
	}
	return Production, nil
}

// launch works order id of domainID in the background; holder is its
// hold on the challenge port, let go when the work ends.
func (s *Service) launch(id, domainID int64, holder string) {
	ctx, cancel := context.WithCancel(s.bg)
	j := &job{domainID: domainID, cancel: cancel, holder: holder}
	s.mu.Lock()
	s.jobs[id] = j
	s.mu.Unlock()
	s.wg.Go(func() {
		defer s.end(id, j)
		s.drive(ctx, id)
	})
}

// end lets go of what job j of order id held.
func (s *Service) end(id int64, j *job) {
	j.cancel()
	s.mu.Lock()
	if s.jobs[id] == j {
		delete(s.jobs, id)
	}
	h := j.holder
	j.holder = ""
	s.mu.Unlock()
	s.release(h)
}

// release lets go of a hold on the challenge port. While the service
// stops the bridge closes every port itself, and a reload now would open
// them again.
func (s *Service) release(holder string) {
	if holder == "" || s.bg.Err() != nil {
		return
	}
	if err := s.Bridge.WantChallengePort(context.WithoutCancel(s.bg), holder, false); err != nil {
		s.Log.Warn("certorder: close challenge port", "err", err)
	}
}

// holder is a new name for one hold on the challenge port.
func (s *Service) holder(domainID int64) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.seq++
	return fmt.Sprintf("order:%d:%d", domainID, s.seq)
}

// Forget stops the background work of domainID's orders, after the
// bridge cancelled them or deleted the domain.
func (s *Service) Forget(domainID int64) {
	s.mu.Lock()
	var stop []*job
	for _, j := range s.jobs {
		if j.domainID == domainID {
			stop = append(stop, j)
		}
	}
	s.mu.Unlock()
	for _, j := range stop {
		j.cancel()
	}
}

// Run resumes the running orders, opens the challenge port when
// acme.keepPort80 is on, and renews due certificates at start and
// hourly until ctx ends (D69, D70).
func (s *Service) Run(ctx context.Context) {
	if a, err := s.Settings.ACME(ctx); err != nil {
		s.Log.Warn("certorder: read settings", "err", err)
	} else if a.KeepPort80 {
		if err := s.Bridge.WantChallengePort(ctx, keepHolder, true); err != nil {
			s.Log.Info("certorder: keep challenge port", "port", s.Bridge.ChallengePort, "err", err)
		}
	}
	s.resume(ctx)
	t := time.NewTicker(time.Hour)
	defer t.Stop()
	for {
		s.renewDue(ctx)
		select {
		case <-ctx.Done():
			s.stop()
			s.wg.Wait()
			return
		case <-t.C:
		}
	}
}

// keepHolder is acme.keepPort80's hold on the challenge port.
const keepHolder = "keep"

// timed runs fn with a limit of d.
func timed[T any](ctx context.Context, d time.Duration, fn func(context.Context) (T, error)) (T, error) {
	ctx, cancel := context.WithTimeout(ctx, d)
	defer cancel()
	return fn(ctx)
}

// domainName reads the name of domain id.
func (s *Service) domainName(ctx context.Context, id int64) (string, error) {
	var name string
	err := s.DB.QueryRowContext(ctx, "SELECT name FROM domains WHERE id = ?", id).Scan(&name)
	if errors.Is(err, sql.ErrNoRows) {
		return "", bridge.ErrNotFound
	}
	return name, err
}

// record writes an audit event of the system, when an audit log is set.
func (s *Service) record(ctx context.Context, action string, outcome audit.Outcome, detail map[string]string) {
	if s.Audit == nil {
		return
	}
	if err := s.Audit.Record(ctx, audit.Event{Actor: audit.ActorSystem, Action: action, Outcome: outcome, Detail: detail}); err != nil {
		s.Log.Warn("certorder: audit", "action", action, "err", err)
	}
}
