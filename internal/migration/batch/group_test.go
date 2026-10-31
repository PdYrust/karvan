package batch

import (
	"context"
	"errors"
	"slices"
	"testing"
)

const elsewhere = "https://elsewhere.example"

// runs makes three runs of one old panel, a run of another old panel
// and a run onto a second panel, each with one user; run 2 is rolled back.
func runs(t *testing.T) (*Store, map[string]int64) {
	t.Helper()
	s, db := open(t)
	exec(t, db, "INSERT INTO panels (id, env_path, name, scanned_at) VALUES (2, '/q/.env', 'q', 0)")
	ids := map[string]int64{}
	for i, r := range []struct {
		job, from string
		panel     int64
		facts     Facts
	}{
		{"r1", origin, 1, Facts{OldPath: "sub", OldHosts: []string{"a.example"}, OldPort: 443}},
		{"r2", origin, 1, Facts{OldHosts: []string{"b.example"}}},
		{"r3", origin, 1, Facts{OldPath: "s2", OldHosts: []string{"a.example"}}},
		{"other", elsewhere, 1, Facts{}},
		{"panel2", origin, 2, Facts{}},
	} {
		at := int64(1000 * (i + 1))
		job(t, db, r.job, "done", r.from, "alice", at)
		exec(t, db, "UPDATE jobs SET panel_id = ? WHERE id = ?", r.panel, r.job)
		row(t, db, "user", r.job, "done", r.from, dest, at-100)
		ids[r.job] = ensure(t, s, r.job, dest, r.facts)
	}
	exec(t, db, "UPDATE batches SET state = 'inactive', inactive_reason = 'rolled_back' WHERE id = ?", ids["r2"])
	exec(t, db, `INSERT INTO legacy_sites (id, name, panel_id, source_kind, source_url, old_path, listen_port, mode, created_at, updated_at)
		VALUES (1, 'rule', 1, 'marzban', ?, 'sub', 443, 'batches', 1, 1)`, origin)
	return s, ids
}

func batchIDs(bs []Batch) []int64 {
	out := []int64{}
	for _, b := range bs {
		out = append(out, b.ID)
	}
	return out
}

// groupsOf lists the groups for f with up to MaxRuns runs each.
func groupsOf(t *testing.T, s *Store, f Filter) []Group {
	t.Helper()
	got, next, err := s.Groups(context.Background(), f, "", 20, MaxRuns)
	if err != nil || next != "" {
		t.Fatalf("groups %q %v", next, err)
	}
	return got
}

// One old panel on one destination panel is one group, whatever the
// number of its runs; another old panel or panel is another group.
func TestGroupsAreOneOldPanelEach(t *testing.T) {
	s, ids := runs(t)
	got := groupsOf(t, s, Filter{})
	if len(got) != 3 || got[0].ID != ids["panel2"] || got[1].ID != ids["other"] {
		t.Fatalf("groups %+v", got)
	}
	g := got[2]
	if g.ID != ids["r1"] || g.LatestID != ids["r3"] || g.Source.Origin != origin || g.Panel.ID != 1 {
		t.Fatalf("group %+v", g)
	}
	if g.Counts != (GroupCounts{Runs: 3, Active: 2, Inactive: 1, Users: 3}) {
		t.Fatalf("counts %+v", g.Counts)
	}
	if want := []int64{ids["r3"], ids["r2"], ids["r1"]}; !slices.Equal(batchIDs(g.Runs), want) {
		t.Fatalf("runs %v, want %v", batchIDs(g.Runs), want)
	}
}

// A group's old link facts come from its newest runs that know them, its
// hosts from every run, and its rules are those of its old panel only.
func TestGroupFactsAndRules(t *testing.T) {
	s, _ := runs(t)
	got := groupsOf(t, s, Filter{})
	g := got[2]
	if g.OldLinks.Path != "s2" || g.OldLinks.Port != 443 || !slices.Equal(g.OldLinks.Hosts, []string{"a.example", "b.example"}) {
		t.Fatalf("old links %+v", g.OldLinks)
	}
	if !slices.Equal(g.Rules, []Rule{{ID: 1, Name: "rule"}}) || len(got[0].Rules) != 0 || len(got[1].Rules) != 0 {
		t.Fatalf("rules %+v, %+v, %+v", g.Rules, got[0].Rules, got[1].Rules)
	}
}

// The state filter keeps groups with a run in that state and shows those
// runs; the panel filter and the run cap narrow the list.
func TestGroupsFilter(t *testing.T) {
	s, ids := runs(t)
	ctx := context.Background()
	inactive := groupsOf(t, s, Filter{State: "inactive"})
	if len(inactive) != 1 || !slices.Equal(batchIDs(inactive[0].Runs), []int64{ids["r2"]}) {
		t.Fatalf("inactive %+v", inactive)
	}
	one, _, err := s.Groups(ctx, Filter{PanelID: 2}, "", 20, 0)
	if err != nil || len(one) != 1 || one[0].ID != ids["panel2"] || len(one[0].Runs) != 0 {
		t.Fatalf("panel 2 %+v %v", one, err)
	}
	if _, _, err := s.Groups(ctx, Filter{State: "deleted"}, "", 20, 0); !errors.Is(err, ErrBadState) {
		t.Fatalf("bad state: %v", err)
	}
}

// The cursor of a page starts the next one, the latest run first.
func TestGroupsPage(t *testing.T) {
	s, ids := runs(t)
	ctx := context.Background()
	first, next, err := s.Groups(ctx, Filter{}, "", 2, 1)
	if err != nil || len(first) != 2 || next == "" || len(first[0].Runs) != 1 {
		t.Fatalf("first page %+v %q %v", first, next, err)
	}
	rest, next, err := s.Groups(ctx, Filter{}, next, 2, 1)
	if err != nil || len(rest) != 1 || next != "" || rest[0].ID != ids["r1"] {
		t.Fatalf("second page %+v %q %v", rest, next, err)
	}
}

// Any batch of a group, a deleted one too, names it; a group whose
// batches are all deleted is gone.
func TestGetGroupByAnyBatch(t *testing.T) {
	s, ids := runs(t)
	ctx := context.Background()
	exec(t, s.db, "UPDATE batches SET state = 'deleted' WHERE id = ?", ids["r1"])
	for _, id := range []int64{ids["r1"], ids["r2"], ids["r3"]} {
		g, err := s.GetGroup(ctx, id, 0)
		if err != nil || g.ID != ids["r1"] || g.Counts.Runs != 2 || g.OldLinks.Path != "s2" || g.OldLinks.Port != 0 {
			t.Fatalf("group of %d: %+v %v", id, g, err)
		}
	}
	exec(t, s.db, "UPDATE batches SET state = 'deleted' WHERE id = ?", ids["other"])
	if _, err := s.GetGroup(ctx, ids["other"], 0); !errors.Is(err, ErrNotFound) {
		t.Fatalf("all deleted: %v", err)
	}
	if _, err := s.GetGroup(ctx, 999, 0); !errors.Is(err, ErrNotFound) {
		t.Fatalf("unknown: %v", err)
	}
}

// The batch list of one group holds its runs only.
func TestListOfOneGroup(t *testing.T) {
	s, ids := runs(t)
	got, _, err := s.List(context.Background(), Filter{Group: ids["r2"]}, "", 20)
	if want := []int64{ids["r3"], ids["r2"], ids["r1"]}; err != nil || !slices.Equal(batchIDs(got), want) {
		t.Fatalf("runs %v %v, want %v", batchIDs(got), err, want)
	}
}
