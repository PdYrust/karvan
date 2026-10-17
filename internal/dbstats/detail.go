package dbstats

import (
	"context"
	"errors"
	"time"

	"github.com/PdYrust/karvan/internal/panel/paneldb"
	"github.com/PdYrust/karvan/internal/panel/panelstats"
)

// Detail is the slower part of a panel's analytics, read when a page asks
// for it: largest tables, hourly traffic, sign-ups and subscription apps.
type Detail struct {
	Tables  []paneldb.TableSize `json:"tables"`
	Traffic []panelstats.Hour   `json:"traffic"`
	SignUps []panelstats.Count  `json:"signUps"`
	Clients []panelstats.Client `json:"clients"`
}

type cached struct {
	at     time.Time
	detail Detail
}

// detailTTL keeps a detail for a while: counting a large SQLite table or
// scanning dbstat takes time, and the numbers move slowly.
const (
	detailTTL     = 5 * time.Minute
	detailTimeout = 20 * time.Second
	maxTables     = 12
)

// ErrNoPanel means the panel has not been read yet or cannot be read.
var ErrNoPanel = errors.New("no reading of this panel yet")

// Detail reads the panel's detail since a time; key names the range so
// each range is cached apart.
func (s *Sampler) Detail(ctx context.Context, panelID int64, key string, since time.Time) (Detail, error) {
	s.mu.Lock()
	p, ok := s.state[panelID]
	s.mu.Unlock()
	if !ok {
		return Detail{}, ErrNoPanel
	}
	p.db.Lock()
	defer p.db.Unlock()
	now := s.now()
	if c, ok := p.detail[key]; ok && now.Sub(c.at) < detailTTL {
		return c.detail, nil
	}
	if p.reader == nil {
		return Detail{}, ErrNoPanel
	}
	rctx, cancel := context.WithTimeout(ctx, detailTimeout)
	defer cancel()
	d := p.target.Dialect
	var out Detail
	err := p.reader.View(rctx, func(q paneldb.Querier) error {
		var err error
		if out.Tables, err = paneldb.Tables(rctx, q, d, maxTables); err != nil {
			return err
		}
		if out.Traffic, err = panelstats.Traffic(rctx, q, d, since); err != nil {
			return err
		}
		if out.SignUps, err = panelstats.NewUsers(rctx, q, d, since); err != nil {
			return err
		}
		out.Clients, err = panelstats.Clients(rctx, q, d, schemaOf(p.kind), since)
		return err
	})
	if err != nil {
		return Detail{}, err
	}
	p.detail[key] = cached{at: now, detail: out}
	return out, nil
}

// Snapshot returns one panel's last reading.
func (s *Sampler) Snapshot(panelID int64) (Snapshot, bool) {
	s.mu.Lock()
	p, ok := s.state[panelID]
	s.mu.Unlock()
	if !ok {
		return Snapshot{}, false
	}
	p.db.Lock()
	defer p.db.Unlock()
	return p.snap, p.snap.PanelID != 0
}
