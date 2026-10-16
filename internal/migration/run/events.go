package run

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

// Stages of a run.
const (
	StageCheck   = "check"
	StageBackup  = "backup"
	StageExecute = "execute"
	StageVerify  = "verify"
	StageReload  = "reload"
	StageUndo    = "undo"
	StageRestore = "restore"
)

// Event types.
const (
	EventStage    = "stage"    // a stage started or ended: {"state": "start"|"done"|"failed"}
	EventProgress = "progress" // {"done": n, "total": n, "etaSeconds": n}
	EventLog      = "log"      // {"code": "...", ...}: the UI words it
	EventDone     = "done"     // {"state": final job state}
)

// Event is one thing that happened in a job. Data never holds a secret.
type Event struct {
	Seq   int64          `json:"seq"`
	At    time.Time      `json:"at"`
	Stage string         `json:"stage"`
	Type  string         `json:"type"`
	Data  map[string]any `json:"data,omitempty"`
}

// bus stores events in job_events and fans them out to live subscribers.
type bus struct {
	db  *sql.DB
	now func() time.Time

	mu   sync.Mutex
	seq  map[string]int64
	subs map[string]map[chan Event]struct{}
}

func newBus(db *sql.DB, now func() time.Time) *bus {
	return &bus{db: db, now: now, seq: map[string]int64{}, subs: map[string]map[chan Event]struct{}{}}
}

func (b *bus) emit(ctx context.Context, job, stage, typ string, data map[string]any) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if _, ok := b.seq[job]; !ok {
		var last sql.NullInt64
		if err := b.db.QueryRowContext(ctx, "SELECT MAX(seq) FROM job_events WHERE job_id = ?", job).Scan(&last); err != nil {
			return err
		}
		b.seq[job] = last.Int64
	}
	b.seq[job]++
	e := Event{Seq: b.seq[job], At: b.now().UTC(), Stage: stage, Type: typ, Data: data}
	raw, err := json.Marshal(e.Data)
	if err != nil {
		return err
	}
	if _, err := b.db.ExecContext(ctx, "INSERT INTO job_events (job_id, seq, at, stage, type, data) VALUES (?, ?, ?, ?, ?, ?)",
		job, e.Seq, e.At.UnixMilli(), stage, typ, string(raw)); err != nil {
		return fmt.Errorf("store event: %w", err)
	}
	for ch := range b.subs[job] {
		select {
		case ch <- e:
		default:
			// A slow reader catches up from the table with Last-Event-ID.
		}
	}
	return nil
}

// Events returns the stored events after seq and a channel of new ones.
// cancel must be called to stop the subscription.
func (b *bus) subscribe(ctx context.Context, job string, after int64) ([]Event, <-chan Event, func(), error) {
	b.mu.Lock()
	ch := make(chan Event, 256)
	if b.subs[job] == nil {
		b.subs[job] = map[chan Event]struct{}{}
	}
	b.subs[job][ch] = struct{}{}
	b.mu.Unlock()
	cancel := func() {
		b.mu.Lock()
		defer b.mu.Unlock()
		if _, ok := b.subs[job][ch]; ok {
			delete(b.subs[job], ch)
			close(ch)
		}
	}
	past, err := b.stored(ctx, job, after)
	if err != nil {
		cancel()
		return nil, nil, nil, err
	}
	return past, ch, cancel, nil
}

func (b *bus) stored(ctx context.Context, job string, after int64) ([]Event, error) {
	rows, err := b.db.QueryContext(ctx, "SELECT seq, at, stage, type, data FROM job_events WHERE job_id = ? AND seq > ? ORDER BY seq", job, after)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Event
	for rows.Next() {
		var e Event
		var at int64
		var raw string
		if err := rows.Scan(&e.Seq, &at, &e.Stage, &e.Type, &raw); err != nil {
			return nil, err
		}
		e.At = time.UnixMilli(at).UTC()
		if err := json.Unmarshal([]byte(raw), &e.Data); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}
