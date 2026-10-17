package hoststats

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

// Defaults for a server: a sample every 5 s, the last 15 minutes kept in
// memory, minutes kept for 30 days.
const (
	Interval  = 5 * time.Second
	keep      = 180
	Retention = 30 * 24 * time.Hour
	trimEvery = 24 * time.Hour
)

// Sampler samples the server and keeps what the analytics pages show.
type Sampler struct {
	Reader Reader
	Store  *Store
	Now    func() time.Time
	Log    *slog.Logger

	mu       sync.Mutex
	ring     []Sample
	seq      int64
	prev     *raw
	acc      minuteAcc
	lastTrim time.Time
	err      error
}

// Run samples every Interval until ctx ends.
func (s *Sampler) Run(ctx context.Context) {
	t := time.NewTicker(Interval)
	defer t.Stop()
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

// Tick takes one sample, and saves the finished minute when a new one began.
func (s *Sampler) Tick(ctx context.Context) {
	now := s.now()
	r, err := s.Reader.read(now)
	s.mu.Lock()
	s.err = err
	if err != nil && r.cpu.total == 0 {
		s.mu.Unlock()
		return
	}
	var done *Minute
	if s.prev != nil {
		sample := between(*s.prev, r)
		s.seq++
		sample.Seq = s.seq
		s.ring = append(s.ring, sample)
		if len(s.ring) > keep {
			s.ring = s.ring[len(s.ring)-keep:]
		}
		done = s.acc.add(sample)
	}
	s.prev = &r
	trim := now.Sub(s.lastTrim) >= trimEvery
	if trim {
		s.lastTrim = now
	}
	s.mu.Unlock()

	if done != nil && s.Store != nil {
		if err := s.Store.Put(ctx, *done); err != nil && ctx.Err() == nil {
			s.log().Warn("save server minute", "err", err)
		}
	}
	if trim && s.Store != nil {
		if _, err := s.Store.Trim(ctx, now.Add(-Retention).Unix()); err != nil && ctx.Err() == nil {
			s.log().Warn("trim server minutes", "err", err)
		}
	}
}

func (s *Sampler) log() *slog.Logger {
	if s.Log != nil {
		return s.Log
	}
	return slog.Default()
}

// Since returns the samples newer than seq, oldest first.
func (s *Sampler) Since(seq int64) []Sample {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := []Sample{}
	for _, x := range s.ring {
		if x.Seq > seq {
			out = append(out, x)
		}
	}
	return out
}

// Latest is the newest sample, if any, and the last read error.
func (s *Sampler) Latest() (Sample, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.ring) == 0 {
		return Sample{}, false, s.err
	}
	return s.ring[len(s.ring)-1], true, s.err
}

// minuteAcc sums the samples of one minute.
type minuteAcc struct {
	ts                   int64
	n                    float64
	sum                  Minute
	cpuMax, rxMax, txMax float64
}

// add takes a sample and returns the previous minute when s starts a new one.
func (a *minuteAcc) add(s Sample) *Minute {
	ts := s.At.Unix() / 60 * 60
	var done *Minute
	if a.n > 0 && ts != a.ts {
		m := a.minute()
		done = &m
		*a = minuteAcc{}
	}
	a.ts = ts
	a.n++
	a.sum.CPU += s.CPU
	a.sum.Load1 += s.Load[0]
	a.sum.MemUsed += int64(s.Mem.Used)      //nolint:gosec // G115: memory sizes fit in int64.
	a.sum.MemCache += int64(s.Mem.Cache)    //nolint:gosec // G115: as above.
	a.sum.SwapUsed += int64(s.Mem.SwapUsed) //nolint:gosec // G115: as above.
	a.sum.MemTotal = int64(s.Mem.Total)     //nolint:gosec // G115: as above.
	a.sum.NetRx += s.NetRx
	a.sum.NetTx += s.NetTx
	a.sum.DiskRead += s.DiskRead
	a.sum.DiskWrite += s.DiskWrite
	a.sum.TCP += int64(s.TCP)
	a.sum.Procs += int64(s.Procs)
	a.cpuMax, a.rxMax, a.txMax = max(a.cpuMax, s.CPU), max(a.rxMax, s.NetRx), max(a.txMax, s.NetTx)
	return done
}

func (a *minuteAcc) minute() Minute {
	n := a.n
	ni := int64(n)
	return Minute{
		TS: a.ts, CPU: a.sum.CPU / n, CPUMax: a.cpuMax, Load1: a.sum.Load1 / n,
		MemUsed: a.sum.MemUsed / ni, MemCache: a.sum.MemCache / ni, MemTotal: a.sum.MemTotal, SwapUsed: a.sum.SwapUsed / ni,
		NetRx: a.sum.NetRx / n, NetTx: a.sum.NetTx / n, NetRxMax: a.rxMax, NetTxMax: a.txMax,
		DiskRead: a.sum.DiskRead / n, DiskWrite: a.sum.DiskWrite / n, TCP: a.sum.TCP / ni, Procs: a.sum.Procs / ni,
	}
}
