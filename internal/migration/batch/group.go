package batch

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/PdYrust/karvan/internal/migration/ledger"
	"github.com/PdYrust/karvan/internal/panel/source"
)

// A group is one old panel moved onto one destination panel: every batch
// with the same destination panel, source kind and source origin. Each run
// stays its own batch, for rollback and restore; the group is what a rule
// of batches serves, now and for every later run.
type groupKey struct {
	panel        int64
	kind, origin string
}

// groupOf matches the batches of the group of b.
const groupOf = "b.panel_id = ? AND b.source_kind = ? AND b.source_origin = ?"

func (k groupKey) args() []any { return []any{k.panel, k.kind, k.origin} }

// MaxRuns caps the runs one group of the group list carries.
const MaxRuns = 5

// Group is an old panel and destination with its runs as the API shows
// it. ID is the id of its first batch, deleted ones included, so it never
// changes; LatestID is its newest batch that is not deleted.
type Group struct {
	ID    int64 `json:"id"`
	Panel struct {
		ID   int64  `json:"id"`
		Name string `json:"name"`
		Kind string `json:"kind"`
	} `json:"panel"`
	Source struct {
		Kind   string `json:"kind"`
		Origin string `json:"origin"`
	} `json:"source"`
	Counts   GroupCounts `json:"counts"`
	OldLinks LinkFacts   `json:"oldLinks"`
	Rules    []Rule      `json:"rules"`
	LatestID int64       `json:"latestId"`
	FirstAt  time.Time   `json:"firstAt"`
	LastAt   time.Time   `json:"lastAt"`
	// Runs are its newest batches that match the list's state filter.
	Runs []Batch `json:"runs"`
}

// GroupCounts sum up a group's batches that are not deleted and their users.
type GroupCounts struct {
	Runs     int `json:"runs"`
	Active   int `json:"active"`
	Inactive int `json:"inactive"`
	Users    int `json:"users"`
	Verified int `json:"verified"`
	Closed   int `json:"closed"`
}

const groupSQL = `SELECT * FROM (SELECT min(b.id) AS gid, b.panel_id, p.name, p.kind, b.source_kind, b.source_origin,
	count(*) FILTER (WHERE b.state <> 'deleted') AS runs,
	count(*) FILTER (WHERE b.state = 'active') AS active,
	count(*) FILTER (WHERE b.state = 'inactive') AS inactive,
	COALESCE(max(b.id) FILTER (WHERE b.state <> 'deleted'), 0),
	COALESCE(min(b.created_at) FILTER (WHERE b.state <> 'deleted'), 0),
	COALESCE(max(b.created_at) FILTER (WHERE b.state <> 'deleted'), 0) AS last
	FROM batches b JOIN panels p ON p.id = b.panel_id`

// Groups lists the groups that have a batch not deleted, the latest run
// first, after the cursor a previous page returned. f.State keeps groups
// with a batch in that state, and their runs are the ones in it. Each
// group carries up to runs of its newest runs.
func (s *Store) Groups(ctx context.Context, f Filter, after string, limit, runs int) ([]Group, string, error) {
	c, err := parseCursor(after)
	if err != nil {
		return nil, "", err
	}
	q, args := groupSQL, []any{}
	if f.PanelID != 0 {
		q += " WHERE b.panel_id = ?"
		args = append(args, f.PanelID)
	}
	q += " GROUP BY b.panel_id, b.source_kind, b.source_origin) WHERE runs > 0 AND (last < ? OR (last = ? AND gid < ?))"
	args = append(args, c.created, c.created, c.id)
	switch f.State {
	case "":
	case "active", "inactive":
		q += " AND " + f.State + " > 0"
	default:
		return nil, "", fmt.Errorf("%w: %q", ErrBadState, f.State)
	}
	limit = min(max(limit, 1), MaxPage)
	out, err := s.groups(ctx, f.State, min(max(runs, 0), MaxRuns), q+" ORDER BY last DESC, gid DESC LIMIT ?", append(args, limit+1)...)
	if err != nil {
		return nil, "", err
	}
	next := ""
	if len(out) > limit {
		out = out[:limit]
		last := out[limit-1]
		next = cursor{created: last.LastAt.UnixMilli(), id: last.ID}.String()
	}
	return out, next, nil
}

// GetGroup reads the group of batch id, any of its batches; ErrNotFound
// when there is no such batch or every batch of the group is deleted.
func (s *Store) GetGroup(ctx context.Context, id int64, runs int) (Group, error) {
	k, err := s.keyOf(ctx, id)
	if err != nil {
		return Group{}, err
	}
	out, err := s.groups(ctx, "", min(max(runs, 0), MaxRuns), groupSQL+" WHERE "+groupOf+
		" GROUP BY b.panel_id, b.source_kind, b.source_origin) WHERE runs > 0", k.args()...)
	if err != nil {
		return Group{}, err
	}
	if len(out) == 0 {
		return Group{}, ErrNotFound
	}
	return out[0], nil
}

// keyOf is the group of batch id, deleted ones included.
func (s *Store) keyOf(ctx context.Context, id int64) (groupKey, error) {
	var k groupKey
	err := s.db.QueryRowContext(ctx, "SELECT panel_id, source_kind, source_origin FROM batches WHERE id = ?", id).
		Scan(&k.panel, &k.kind, &k.origin)
	if errors.Is(err, sql.ErrNoRows) {
		return k, ErrNotFound
	}
	if err != nil {
		return k, fmt.Errorf("read batch %d: %w", id, err)
	}
	return k, nil
}

// groups reads groups with their counts, old link facts, rules and runs.
func (s *Store) groups(ctx context.Context, state string, runs int, q string, args ...any) ([]Group, error) {
	rows, err := s.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("read batch groups: %w", err)
	}
	defer rows.Close()
	out := []Group{}
	for rows.Next() {
		var g Group
		var first, last int64
		if err := rows.Scan(&g.ID, &g.Panel.ID, &g.Panel.Name, &g.Panel.Kind, &g.Source.Kind, &g.Source.Origin,
			&g.Counts.Runs, &g.Counts.Active, &g.Counts.Inactive, &g.LatestID, &first, &last); err != nil {
			return nil, fmt.Errorf("read batch groups: %w", err)
		}
		g.FirstAt, g.LastAt = time.UnixMilli(first).UTC(), time.UnixMilli(last).UTC()
		out = append(out, g)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read batch groups: %w", err)
	}
	rules, err := readRuleIndex(ctx, s.db)
	if err != nil {
		return nil, err
	}
	for i := range out {
		if err := s.fillGroup(ctx, &out[i], state, runs, rules); err != nil {
			return nil, err
		}
	}
	return out, nil
}

func (g Group) key() groupKey {
	return groupKey{panel: g.Panel.ID, kind: g.Source.Kind, origin: g.Source.Origin}
}

// fillGroup adds a group's user counts, old link facts, rules and runs.
func (s *Store) fillGroup(ctx context.Context, g *Group, state string, runs int, rules ruleIndex) error {
	k := g.key()
	err := s.db.QueryRowContext(ctx, `SELECT count(*),
		count(*) FILTER (WHERE EXISTS (SELECT 1 FROM legacy_tokens t WHERE t.ledger_id = l.id)),
		count(*) FILTER (WHERE EXISTS (SELECT 1 FROM legacy_closed c WHERE c.ledger_id = l.id))
		FROM ledger l JOIN batches b ON b.id = l.batch_id WHERE b.state <> 'deleted' AND `+groupOf, k.args()...).
		Scan(&g.Counts.Users, &g.Counts.Verified, &g.Counts.Closed)
	if err != nil {
		return fmt.Errorf("count the users of batch group %d: %w", g.ID, err)
	}
	if g.OldLinks, err = s.groupFacts(ctx, k); err != nil {
		return err
	}
	g.Rules = rules.of(k, false)
	g.Runs = []Batch{}
	if runs == 0 {
		return nil
	}
	q, args := "SELECT "+batchColumns+" WHERE b.state <> 'deleted' AND "+groupOf, k.args()
	if state != "" {
		q += " AND b.state = ?"
		args = append(args, state)
	}
	g.Runs, err = s.query(ctx, q+" ORDER BY b.created_at DESC, b.id DESC LIMIT ?", append(args, runs)...)
	return err
}

// groupFacts are the old link facts of a group's batches that are not
// deleted: the newest batch that knows the path or the port gives it, and
// the hosts are every batch's, sorted.
func (s *Store) groupFacts(ctx context.Context, k groupKey) (LinkFacts, error) {
	f := LinkFacts{Hosts: []string{}}
	rows, err := s.db.QueryContext(ctx, "SELECT b.old_path, b.old_hosts, b.old_port FROM batches b WHERE b.state <> 'deleted' AND "+
		groupOf+" ORDER BY b.id", k.args()...)
	if err != nil {
		return f, fmt.Errorf("read the old links of a batch group: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var path, raw string
		var port int
		var hosts []string
		if err := rows.Scan(&path, &raw, &port); err != nil {
			return f, err
		}
		if err := json.Unmarshal([]byte(raw), &hosts); err != nil {
			return f, fmt.Errorf("old hosts of a batch group: %w", err)
		}
		if path != "" {
			f.Path = path
		}
		if port != 0 {
			f.Port = port
		}
		for _, h := range hosts {
			if !slices.Contains(f.Hosts, h) {
				f.Hosts = append(f.Hosts, h)
			}
		}
	}
	slices.Sort(f.Hosts)
	return f, rows.Err()
}

// siteRule is a panel rule as the batch views match it to groups.
type siteRule struct {
	Rule
	batches bool // a rule of batches; else a manual rule
}

// ruleIndex is every panel rule by the group it serves: a rule of
// batches serves its old panel's group, and so does a manual rule of the
// same old panel and destination panel.
type ruleIndex map[groupKey][]siteRule

// readRuleIndex reads every panel rule. A manual rule's address is read
// as the ledger names old panels, so every way of typing it matches.
func readRuleIndex(ctx context.Context, db querier) (ruleIndex, error) {
	rows, err := db.QueryContext(ctx, "SELECT id, name, mode, panel_id, source_kind, source_url FROM legacy_sites ORDER BY name, id")
	if err != nil {
		return nil, fmt.Errorf("read the batch rules: %w", err)
	}
	defer rows.Close()
	out := ruleIndex{}
	for rows.Next() {
		var r siteRule
		var mode, url string
		var k groupKey
		if err := rows.Scan(&r.ID, &r.Name, &mode, &k.panel, &k.kind, &url); err != nil {
			return nil, err
		}
		u, err := source.ParseBase(url)
		if err != nil {
			continue // no rule serves an address that does not parse
		}
		k.origin, r.batches = ledger.Origin(u), mode == "batches"
		out[k] = append(out[k], r)
	}
	return out, rows.Err()
}

// of lists the rules of group k by name, only the rules of batches when
// batchesOnly.
func (ri ruleIndex) of(k groupKey, batchesOnly bool) []Rule {
	out := []Rule{}
	for _, r := range ri[k] {
		if r.batches || !batchesOnly {
			out = append(out, r.Rule)
		}
	}
	return out
}
