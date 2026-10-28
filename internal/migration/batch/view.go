package batch

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
)

// ErrNotFound means no batch has this id, the batch was deleted, or the
// user is not in it.
var ErrNotFound = errors.New("batch not found")

// ErrBadCursor means a page cursor was not one these lists handed out.
var ErrBadCursor = errors.New("not a batch list cursor")

// ErrBadState means a list filter named a state the list does not show.
var ErrBadState = errors.New("unknown batch state")

// MaxPage caps one page of the batch and batch user lists.
const MaxPage = 100

// Batch is a batch as the API shows it. It holds no token, MAC, pin or
// address (D57).
type Batch struct {
	ID             int64      `json:"id"`
	Name           string     `json:"name"`
	State          string     `json:"state"`
	InactiveReason string     `json:"inactiveReason"`
	InactiveAt     *time.Time `json:"inactiveAt"`
	JobID          string     `json:"jobId"`
	Panel          struct {
		ID   int64  `json:"id"`
		Name string `json:"name"`
		Kind string `json:"kind"`
	} `json:"panel"`
	Source struct {
		Kind   string `json:"kind"`
		Origin string `json:"origin"`
	} `json:"source"`
	Admins    []Admin   `json:"admins"`
	OldLinks  LinkFacts `json:"oldLinks"`
	Counts    Counts    `json:"counts"`
	Rules     []Rule    `json:"rules"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// Admin is one admin of a batch: its username on the old panel and on
// the destination (empty when Karvan has no record of it), and how many
// of the batch's users it owns.
type Admin struct {
	Source string `json:"source"`
	Dest   string `json:"dest"`
	Users  int    `json:"users"`
}

// LinkFacts are the old link facts read during the run (D7).
type LinkFacts struct {
	Path  string   `json:"path"`
	Hosts []string `json:"hosts"`
	Port  int      `json:"port"`
}

// Counts sum up a batch's users: all of them, those with at least one
// verified old link, and those whose old links are closed.
type Counts struct {
	Users    int `json:"users"`
	Verified int `json:"verified"`
	Closed   int `json:"closed"`
}

// Rule is a panel rule that serves a batch.
type Rule struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

// Ref names a job's batch on the migrations list.
type Ref struct {
	ID    int64  `json:"id"`
	Name  string `json:"name"`
	State string `json:"state"`
}

// Filter narrows the batch list; zero values mean all.
type Filter struct {
	PanelID int64
	State   string // active or inactive
}

const batchColumns = `b.id, b.name, b.state, b.inactive_reason, b.inactive_at, b.job_id, b.panel_id, p.name, p.kind,
	b.source_kind, b.source_origin, b.old_path, b.old_hosts, b.old_port, b.created_at, b.updated_at
	FROM batches b JOIN panels p ON p.id = b.panel_id`

// List lists batches newest first, never deleted ones, starting after the
// cursor a previous page returned. next is empty on the last page.
func (s *Store) List(ctx context.Context, f Filter, after string, limit int) ([]Batch, string, error) {
	c, err := parseCursor(after)
	if err != nil {
		return nil, "", err
	}
	q := "SELECT " + batchColumns + " WHERE b.state <> 'deleted' AND (b.created_at < ? OR (b.created_at = ? AND b.id < ?))"
	args := []any{c.created, c.created, c.id}
	switch f.State {
	case "":
	case "active", "inactive":
		q += " AND b.state = ?"
		args = append(args, f.State)
	default:
		return nil, "", fmt.Errorf("%w: %q", ErrBadState, f.State)
	}
	if f.PanelID != 0 {
		q += " AND b.panel_id = ?"
		args = append(args, f.PanelID)
	}
	limit = min(max(limit, 1), MaxPage)
	out, err := s.query(ctx, q+" ORDER BY b.created_at DESC, b.id DESC LIMIT ?", append(args, limit+1)...)
	if err != nil {
		return nil, "", err
	}
	next := ""
	if len(out) > limit {
		out = out[:limit]
		last := out[limit-1]
		next = cursor{created: last.CreatedAt.UnixMilli(), id: last.ID}.String()
	}
	return out, next, nil
}

// Get reads one batch; a deleted one is ErrNotFound.
func (s *Store) Get(ctx context.Context, id int64) (Batch, error) {
	out, err := s.query(ctx, "SELECT "+batchColumns+" WHERE b.id = ? AND b.state <> 'deleted'", id)
	if err != nil {
		return Batch{}, err
	}
	if len(out) == 0 {
		return Batch{}, ErrNotFound
	}
	return out[0], nil
}

// ByJob names the batch of a job, deleted ones included; nil when the
// job has none.
func (s *Store) ByJob(ctx context.Context, jobID string) (*Ref, error) {
	var r Ref
	err := s.db.QueryRowContext(ctx, "SELECT id, name, state FROM batches WHERE job_id = ?", jobID).Scan(&r.ID, &r.Name, &r.State)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read the batch of job %s: %w", jobID, err)
	}
	return &r, nil
}

// query reads batches with their admins, counts and rules.
func (s *Store) query(ctx context.Context, q string, args ...any) ([]Batch, error) {
	rows, err := s.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("read batches: %w", err)
	}
	defer rows.Close()
	out := []Batch{}
	for rows.Next() {
		b, err := scanBatch(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read batches: %w", err)
	}
	for i := range out {
		if err := s.details(ctx, &out[i]); err != nil {
			return nil, err
		}
	}
	return out, nil
}

func scanBatch(rows *sql.Rows) (Batch, error) {
	var b Batch
	var hosts string
	var inactive, created, updated int64
	err := rows.Scan(&b.ID, &b.Name, &b.State, &b.InactiveReason, &inactive, &b.JobID, &b.Panel.ID, &b.Panel.Name, &b.Panel.Kind,
		&b.Source.Kind, &b.Source.Origin, &b.OldLinks.Path, &hosts, &b.OldLinks.Port, &created, &updated)
	if err != nil {
		return b, fmt.Errorf("read batches: %w", err)
	}
	if err := json.Unmarshal([]byte(hosts), &b.OldLinks.Hosts); err != nil {
		return b, fmt.Errorf("old hosts of batch %d: %w", b.ID, err)
	}
	if b.OldLinks.Hosts == nil {
		b.OldLinks.Hosts = []string{}
	}
	if inactive > 0 && b.State == "inactive" {
		at := time.UnixMilli(inactive).UTC()
		b.InactiveAt = &at
	}
	b.CreatedAt, b.UpdatedAt = time.UnixMilli(created).UTC(), time.UnixMilli(updated).UTC()
	return b, nil
}

// adminOf is a user row's admin row: the row its own job wrote or merged
// into for the user's source admin (D34), or the job's only admin row for
// a row from before. A row's job may have none (a run that failed before).
const adminOf = `LEFT JOIN ledger a ON a.id = (SELECT min(x.id) FROM ledger x WHERE x.kind = 'admin' AND x.job_id = l.job_id
		AND (l.source_admin = '' OR x.source_key = l.source_admin))
	JOIN jobs j ON j.id = l.job_id`

// details fills in a batch's admins, counts and rules.
func (s *Store) details(ctx context.Context, b *Batch) error {
	err := s.db.QueryRowContext(ctx, `SELECT count(*),
		count(*) FILTER (WHERE EXISTS (SELECT 1 FROM legacy_tokens t WHERE t.ledger_id = l.id)),
		count(*) FILTER (WHERE EXISTS (SELECT 1 FROM legacy_closed c WHERE c.ledger_id = l.id))
		FROM ledger l WHERE l.batch_id = ?`, b.ID).Scan(&b.Counts.Users, &b.Counts.Verified, &b.Counts.Closed)
	if err != nil {
		return fmt.Errorf("count the users of batch %d: %w", b.ID, err)
	}
	rows, err := s.db.QueryContext(ctx, `SELECT COALESCE(a.source_username, j.source_admin), COALESCE(a.dest_username, ''), count(*)
		FROM ledger l `+adminOf+` WHERE l.batch_id = ? GROUP BY 1, 2 ORDER BY 1, 2`, b.ID)
	if err != nil {
		return fmt.Errorf("read the admins of batch %d: %w", b.ID, err)
	}
	defer rows.Close()
	b.Admins = []Admin{}
	for rows.Next() {
		var a Admin
		if err := rows.Scan(&a.Source, &a.Dest, &a.Users); err != nil {
			return err
		}
		b.Admins = append(b.Admins, a)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	b.Rules, err = readRules(ctx, s.db, `SELECT s.id, s.name FROM legacy_sites s JOIN legacy_site_batches sb ON sb.site_id = s.id
		WHERE sb.batch_id = ? ORDER BY s.name, s.id`, b.ID)
	return err
}

// readRules runs a query whose rows are a rule's id and name.
func readRules(ctx context.Context, db querier, q string, args ...any) ([]Rule, error) {
	rows, err := db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("read the batch's rules: %w", err)
	}
	defer rows.Close()
	out := []Rule{}
	for rows.Next() {
		var r Rule
		if err := rows.Scan(&r.ID, &r.Name); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// cursor is where a page of batches ended: the last batch's creation
// time and id, the order the list is sorted by.
type cursor struct {
	created int64
	id      int64
}

func (c cursor) String() string {
	return strconv.FormatInt(c.created, 36) + "." + strconv.FormatInt(c.id, 36)
}

func parseCursor(s string) (cursor, error) {
	if s == "" {
		return cursor{created: math.MaxInt64, id: math.MaxInt64}, nil
	}
	ts, id, ok := strings.Cut(s, ".")
	created, err1 := strconv.ParseInt(ts, 36, 64)
	n, err2 := strconv.ParseInt(id, 36, 64)
	if !ok || err1 != nil || err2 != nil {
		return cursor{}, ErrBadCursor
	}
	return cursor{created: created, id: n}, nil
}
