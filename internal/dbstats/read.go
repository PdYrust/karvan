package dbstats

import (
	"context"
	"time"

	"github.com/PdYrust/karvan/internal/discovery"
	"github.com/PdYrust/karvan/internal/panel"
	"github.com/PdYrust/karvan/internal/panel/marzbandb"
	"github.com/PdYrust/karvan/internal/panel/paneldb"
	"github.com/PdYrust/karvan/internal/panel/panelstats"
	"github.com/PdYrust/karvan/internal/panel/pasarguarddb"
)

// schemaOf is where the two panels' tables differ for analytics.
func schemaOf(k panel.Kind) panelstats.Schema {
	if k == panel.Marzban {
		return marzbandb.Stats
	}
	return pasarguarddb.Stats
}

// blocking findings mean there is no database to read; other findings
// (a relative Xray path, an unknown revision) do not stop analytics.
var blocking = map[string]bool{
	discovery.EnvUnreadable: true, discovery.DBURLMissing: true, discovery.DBRelativePath: true,
	discovery.DBUnsupported: true, discovery.DBUnreachable: true, discovery.NotAPanel: true,
}

// open makes sure p has a reader for the panel's current database. The
// ENV is read again every reinspect, so a changed database is followed.
func (s *Sampler) open(ctx context.Context, p *panelState, ref PanelRef, now time.Time) *Problem {
	if p.reader != nil && now.Sub(p.inspected) < reinspect {
		return nil
	}
	inspect := s.Inspect
	if inspect == nil {
		inspect = discovery.Inspect
	}
	found := inspect(ctx, ref.EnvPath)
	p.inspected = now
	for _, f := range found.Findings {
		if blocking[f.Code] {
			return &Problem{Code: f.Code}
		}
	}
	if found.Kind == "" {
		return &Problem{Code: discovery.NotAPanel}
	}
	t, err := paneldb.ParseURL(found.Settings.DatabaseURL)
	if err != nil {
		return &Problem{Code: discovery.DBUnsupported}
	}
	if p.reader != nil && t.String() == p.target.String() {
		p.kind = found.Kind
		return nil
	}
	p.closeReader()
	r, err := paneldb.OpenReader(ctx, t)
	if err != nil {
		return &Problem{Code: discovery.DBUnreachable}
	}
	p.reader, p.target, p.kind, p.prev = r, t, found.Kind, nil
	return nil
}

func (p *panelState) closeReader() {
	if p.reader != nil {
		_ = p.reader.Close()
		p.reader = nil
	}
}

// readPanel takes one reading of one panel and saves its minute.
func (s *Sampler) readPanel(ctx context.Context, ref PanelRef) {
	p := s.panel(ref.ID)
	p.db.Lock()
	defer p.db.Unlock()
	now := s.now()
	snap := Snapshot{PanelID: ref.ID, Name: ref.Name, At: now.UTC()}
	if problem := s.open(ctx, p, ref, now); problem != nil {
		p.closeReader()
		snap.Problem = problem
		p.snap = snap
		return
	}
	snap.Kind = p.kind
	rctx, cancel := context.WithTimeout(ctx, readTimeout)
	defer cancel()
	var eng paneldb.EngineStats
	var data panelstats.Data
	err := p.reader.View(rctx, func(q paneldb.Querier) error {
		var err error
		if eng, err = paneldb.Engine(rctx, q, p.target, p.reader.Server()); err != nil {
			return err
		}
		data, err = panelstats.Read(rctx, q, p.target.Dialect, schemaOf(p.kind), now)
		return err
	})
	if err != nil {
		if ctx.Err() == nil {
			s.log().Warn("read panel database", "panel", ref.Name, "err", err)
		}
		// Read the ENV again next time: the database may have moved.
		p.closeReader()
		snap.Problem = &Problem{Code: discovery.DBUnreachable}
		p.snap = snap
		return
	}
	snap.Engine, snap.Data = &eng, &data
	snap.QPS, snap.CacheHit = rates(p.prev, &eng, now.Sub(p.prevAt))
	p.prev, p.prevAt, p.snap = &eng, now, snap
	if s.Store != nil {
		m := Minute{TS: now.Unix() / 60 * 60, Size: eng.Size, Connections: eng.Connections, QPS: snap.QPS, CacheHit: snap.CacheHit,
			Users: data.Users, Active: data.ByStatus[string(panel.StatusActive)], Online: data.OnlineNow, UsedTraffic: data.UsedTraffic}
		if err := s.Store.Put(ctx, ref.ID, m); err != nil && ctx.Err() == nil {
			s.log().Warn("save database minute", "panel", ref.Name, "err", err)
		}
	}
}

// rates turns two readings of the engine's counters into queries per
// second and a cache hit rate. A server restart resets the counters, so a
// counter that went down gives no rate.
func rates(prev, cur *paneldb.EngineStats, since time.Duration) (float64, *float64) {
	if prev == nil || since <= 0 || cur.Queries < prev.Queries {
		return 0, nil
	}
	qps := float64(cur.Queries-prev.Queries) / since.Seconds()
	reads, misses := cur.CacheReads-prev.CacheReads, cur.CacheMisses-prev.CacheMisses
	if reads <= 0 || misses < 0 || misses > reads {
		return qps, nil
	}
	hit := 1 - float64(misses)/float64(reads)
	return qps, &hit
}
