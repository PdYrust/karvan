// Package dbstats watches every registered panel's database for the
// analytics pages: the engine's own numbers and the panel's users, once a
// minute, through read-only handles. Nothing here writes to a panel.
package dbstats

import (
	"context"
	"log/slog"
	"slices"
	"sync"
	"time"

	"github.com/PdYrust/karvan/internal/discovery"
	"github.com/PdYrust/karvan/internal/panel"
	"github.com/PdYrust/karvan/internal/panel/paneldb"
	"github.com/PdYrust/karvan/internal/panel/panelstats"
)

// Intervals: a reading per minute, the panel's ENV read again every ten
// minutes (it opens the database too), each reading bounded in time.
const (
	Interval     = time.Minute
	reinspect    = 10 * time.Minute
	readTimeout  = 10 * time.Second
	Retention    = 30 * 24 * time.Hour
	trimInterval = 24 * time.Hour
)

// PanelRef is a registered panel.
type PanelRef struct {
	ID      int64
	EnvPath string
	Name    string
}

// Problem says why a panel's database could not be read, as a code the
// web panel words: a discovery finding, or "db_unreachable".
type Problem struct {
	Code string `json:"code"`
}

// Snapshot is a panel database at its last reading.
type Snapshot struct {
	PanelID  int64                `json:"panelId"`
	Name     string               `json:"name"`
	Kind     panel.Kind           `json:"kind,omitempty"`
	At       time.Time            `json:"at"`
	Problem  *Problem             `json:"problem,omitempty"`
	Engine   *paneldb.EngineStats `json:"engine,omitempty"`
	QPS      float64              `json:"qps"`
	CacheHit *float64             `json:"cacheHit"`
	Data     *panelstats.Data     `json:"data,omitempty"`
}

// Sampler reads every panel once per Interval.
type Sampler struct {
	Panels  func(ctx context.Context) ([]PanelRef, error)
	Store   *Store
	Now     func() time.Time
	Log     *slog.Logger
	Inspect func(ctx context.Context, envPath string) discovery.Panel

	mu       sync.Mutex
	state    map[int64]*panelState
	lastTrim time.Time
}

// panelState is what the sampler keeps per panel. db serializes the use of
// reader between the minute reading and detail requests.
type panelState struct {
	db        sync.Mutex
	reader    *paneldb.Reader
	target    paneldb.Target
	kind      panel.Kind
	inspected time.Time
	prev      *paneldb.EngineStats
	prevAt    time.Time
	snap      Snapshot
	detail    map[string]cached
}

// Run reads the panels every Interval until ctx ends, then closes the
// handles.
func (s *Sampler) Run(ctx context.Context) {
	t := time.NewTicker(Interval)
	defer t.Stop()
	defer s.close()
	for {
		s.Tick(ctx)
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
	}
}

func (s *Sampler) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now()
}

func (s *Sampler) log() *slog.Logger {
	if s.Log != nil {
		return s.Log
	}
	return slog.Default()
}

// Tick reads every registered panel once.
func (s *Sampler) Tick(ctx context.Context) {
	refs, err := s.Panels(ctx)
	if err != nil {
		if ctx.Err() == nil {
			s.log().Warn("list panels for analytics", "err", err)
		}
		return
	}
	for _, ref := range refs {
		s.readPanel(ctx, ref)
	}
	if now := s.now(); now.Sub(s.lastTrim) >= trimInterval && s.Store != nil {
		s.lastTrim = now
		if err := s.Store.Trim(ctx, now.Add(-Retention).Unix()); err != nil && ctx.Err() == nil {
			s.log().Warn("trim database minutes", "err", err)
		}
	}
}

func (s *Sampler) panel(id int64) *panelState {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.state == nil {
		s.state = map[int64]*panelState{}
	}
	p, ok := s.state[id]
	if !ok {
		p = &panelState{detail: map[string]cached{}}
		s.state[id] = p
	}
	return p
}

// Snapshots returns the last reading of every panel read so far, by id.
func (s *Sampler) Snapshots() []Snapshot {
	s.mu.Lock()
	ps := make([]*panelState, 0, len(s.state))
	for _, p := range s.state {
		ps = append(ps, p)
	}
	s.mu.Unlock()
	out := make([]Snapshot, 0, len(ps))
	for _, p := range ps {
		p.db.Lock()
		if p.snap.PanelID != 0 {
			out = append(out, p.snap)
		}
		p.db.Unlock()
	}
	slices.SortFunc(out, func(a, b Snapshot) int { return int(a.PanelID - b.PanelID) })
	return out
}

func (s *Sampler) close() {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, p := range s.state {
		p.db.Lock()
		if p.reader != nil {
			_ = p.reader.Close()
			p.reader = nil
		}
		p.db.Unlock()
	}
}
