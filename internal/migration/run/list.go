package run

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"slices"
	"strconv"
	"strings"
	"time"
)

// JobRow is one job as the jobs list shows it.
type JobRow struct {
	ID    string `json:"id"`
	State string `json:"state"`
	Panel struct {
		ID   int64  `json:"id"`
		Name string `json:"name"`
		Kind string `json:"kind"`
	} `json:"panel"`
	Source struct {
		Kind   string `json:"kind"`
		Origin string `json:"origin"`
		Admin  string `json:"admin"`
	} `json:"source"`
	SafeMode bool `json:"safeMode"`
	// Counts and EndedAt are from the job's last run; nil before it ran.
	Counts    *Counts    `json:"counts"`
	CreatedAt time.Time  `json:"createdAt"`
	UpdatedAt time.Time  `json:"updatedAt"`
	EndedAt   *time.Time `json:"endedAt"`
	// LoginHeld says Karvan still has the old panel's login, so the job
	// can be checked and run without signing in again.
	LoginHeld bool `json:"loginHeld"`
}

// JobFilter narrows the jobs list; zero values mean all.
type JobFilter struct {
	PanelID int64
	States  []string
	// ID finds one job, so a page about it reads its row in every state,
	// also while it runs and after a restart.
	ID string
}

// ErrBadCursor means a page cursor was not one this list handed out.
var ErrBadCursor = errors.New("not a jobs list cursor")

// ErrBadState means a filter named a state jobs never have.
var ErrBadState = errors.New("unknown job state")

// MaxPage caps one page of the jobs list.
const MaxPage = 100

var jobStates = []string{StateCreated, StateChecked, StateRunning, StateDone, StateFailed, StateRolledBack, StateInterrupted}

// cursor is where a page ended: the last job's creation time and id, the
// order the list is sorted by.
type cursor struct {
	created int64
	id      string
}

func (c cursor) String() string { return strconv.FormatInt(c.created, 36) + "." + c.id }

func parseCursor(s string) (cursor, error) {
	if s == "" {
		return cursor{created: math.MaxInt64}, nil
	}
	ts, id, ok := strings.Cut(s, ".")
	created, err := strconv.ParseInt(ts, 36, 64)
	if !ok || err != nil || id == "" {
		return cursor{}, ErrBadCursor
	}
	return cursor{created: created, id: id}, nil
}

// Jobs lists jobs newest first, starting after the cursor a previous page
// returned. next is empty on the last page.
func (s *Service) Jobs(ctx context.Context, f JobFilter, after string, limit int) ([]JobRow, string, error) {
	c, err := parseCursor(after)
	if err != nil {
		return nil, "", err
	}
	limit = min(max(limit, 1), MaxPage)
	query, args, err := jobsQuery(f, c, limit+1)
	if err != nil {
		return nil, "", err
	}
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, "", fmt.Errorf("list jobs: %w", err)
	}
	defer rows.Close()
	out := []JobRow{}
	for rows.Next() {
		row, err := scanJob(rows)
		if err != nil {
			return nil, "", err
		}
		out = append(out, row)
	}
	if err := rows.Err(); err != nil {
		return nil, "", fmt.Errorf("list jobs: %w", err)
	}
	next := ""
	if len(out) > limit {
		out = out[:limit]
		last := out[limit-1]
		next = cursor{created: last.CreatedAt.UnixMilli(), id: last.ID}.String()
	}
	return s.withLogins(out), next, nil
}

func jobsQuery(f JobFilter, c cursor, limit int) (string, []any, error) {
	q := `SELECT j.id, j.state, j.panel_id, p.name, p.kind, j.source_kind, j.source_origin, j.source_admin, j.safe_mode,
		j.counts, j.ended_at, j.created_at, j.updated_at
		FROM jobs j JOIN panels p ON p.id = j.panel_id
		WHERE (j.created_at < ? OR (j.created_at = ? AND j.id < ?))`
	args := []any{c.created, c.created, c.id}
	if f.PanelID != 0 {
		q += " AND j.panel_id = ?"
		args = append(args, f.PanelID)
	}
	if f.ID != "" {
		q += " AND j.id = ?"
		args = append(args, f.ID)
	}
	if len(f.States) > 0 {
		for _, st := range f.States {
			if !slices.Contains(jobStates, st) {
				return "", nil, fmt.Errorf("%w: %q", ErrBadState, st)
			}
			args = append(args, st)
		}
		q += " AND j.state IN (?" + strings.Repeat(", ?", len(f.States)-1) + ")"
	}
	return q + " ORDER BY j.created_at DESC, j.id DESC LIMIT ?", append(args, limit), nil
}

func scanJob(rows *sql.Rows) (JobRow, error) {
	var row JobRow
	var counts string
	var ended, created, updated int64
	err := rows.Scan(&row.ID, &row.State, &row.Panel.ID, &row.Panel.Name, &row.Panel.Kind, &row.Source.Kind, &row.Source.Origin,
		&row.Source.Admin, &row.SafeMode, &counts, &ended, &created, &updated)
	if err != nil {
		return JobRow{}, fmt.Errorf("list jobs: %w", err)
	}
	if counts != "" {
		row.Counts = &Counts{}
		if err := json.Unmarshal([]byte(counts), row.Counts); err != nil {
			return JobRow{}, fmt.Errorf("counts of job %s: %w", row.ID, err)
		}
	}
	row.CreatedAt, row.UpdatedAt = time.UnixMilli(created).UTC(), time.UnixMilli(updated).UTC()
	if ended > 0 {
		at := time.UnixMilli(ended).UTC()
		row.EndedAt = &at
	}
	return row, nil
}

// Totals sum up every job, for the overview.
type Totals struct {
	// Jobs counts jobs by state.
	Jobs map[string]int64 `json:"jobs"`
	// Moved is users moved and still there, Skipped users the runs left
	// out. A rolled back job adds to neither: its undo or backup restore
	// took every user back, while its stored counts still show the run.
	Moved   int64 `json:"moved"`
	Skipped int64 `json:"skipped"`
	// LastEnded is when the last job ended; nil when none has.
	LastEnded *time.Time `json:"lastEnded"`
}

// Totals sums up the jobs from their stored counts. counts is empty until
// a job has run; json_extract fails on an empty string but passes NULL
// through, and SUM leaves NULL out.
func (s *Service) Totals(ctx context.Context) (Totals, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT state, COUNT(*), MAX(ended_at),
		COALESCE(SUM(MAX(0, json_extract(c, '$.moved') - COALESCE(json_extract(c, '$.undone'), 0))), 0),
		COALESCE(SUM(json_extract(c, '$.skipped')), 0)
		FROM (SELECT state, ended_at, NULLIF(counts, '') AS c FROM jobs) GROUP BY state`)
	if err != nil {
		return Totals{}, fmt.Errorf("sum up jobs: %w", err)
	}
	defer rows.Close()
	out := Totals{Jobs: map[string]int64{}}
	var last int64
	for rows.Next() {
		var state string
		var n, ended, moved, skipped int64
		if err := rows.Scan(&state, &n, &ended, &moved, &skipped); err != nil {
			return Totals{}, fmt.Errorf("sum up jobs: %w", err)
		}
		out.Jobs[state] = n
		last = max(last, ended)
		if state != StateRolledBack {
			out.Moved += moved
			out.Skipped += skipped
		}
	}
	if err := rows.Err(); err != nil {
		return Totals{}, fmt.Errorf("sum up jobs: %w", err)
	}
	if last > 0 {
		at := time.UnixMilli(last).UTC()
		out.LastEnded = &at
	}
	return out, nil
}

// withLogins fills in which jobs still hold their login. Jobs from before
// a restart are not in memory and never do.
func (s *Service) withLogins(rows []JobRow) []JobRow {
	for i := range rows {
		if j, err := s.job(rows[i].ID); err == nil {
			rows[i].LoginHeld = j.holdsLogin()
			if j.running.Load() {
				rows[i].State = StateRunning
			}
		}
	}
	return rows
}

// holdsLogin says whether the job still has the old panel's login. A job
// that is busy (checking or running) is using it, and waiting for its lock
// could take as long as reading the whole old panel.
func (j *Job) holdsLogin() bool {
	if j.running.Load() || !j.mu.TryLock() {
		return true
	}
	defer j.mu.Unlock()
	return j.source != nil
}
