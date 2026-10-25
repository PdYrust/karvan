package run

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"testing"
	"time"

	"github.com/PdYrust/karvan/internal/panel"
	"github.com/PdYrust/karvan/internal/panel/paneldb/paneldbtest"
	"github.com/PdYrust/karvan/internal/panel/pasarguarddb"
	"github.com/PdYrust/karvan/internal/store"
)

func listService(t *testing.T) *Service {
	t.Helper()
	db, err := store.Open(context.Background(), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	for _, p := range []string{"(1, '/a/.env', 'alpha', 'pasarguard', 0)", "(2, '/b/.env', 'beta', 'marzban', 0)"} {
		if _, err := db.Exec("INSERT INTO panels (id, env_path, name, kind, scanned_at) VALUES " + p); err != nil {
			t.Fatal(err)
		}
	}
	// Five jobs a second apart; the third and fourth share a time, so the
	// id breaks the tie.
	for i, created := range []int64{1000, 2000, 3000, 3000, 5000} {
		state, counts, ended := StateChecked, "", int64(0)
		if i == 4 {
			state, counts, ended = StateDone, `{"read":9,"moved":7,"skipped":2}`, 6000
		}
		_, err := db.Exec(`INSERT INTO jobs (id, state, source_kind, source_origin, source_admin, panel_id, safe_mode, counts, ended_at,
			created_at, updated_at) VALUES (?, ?, 'marzban', 'https://old.example', 'reseller', ?, 1, ?, ?, ?, ?)`,
			fmt.Sprintf("job%d", i), state, 1+i%2, counts, ended, created, created)
		if err != nil {
			t.Fatal(err)
		}
	}
	return New(Deps{DB: db})
}

func ids(rows []JobRow) []string {
	out := make([]string, len(rows))
	for i, r := range rows {
		out[i] = r.ID
	}
	return out
}

func TestJobsPageNewestFirst(t *testing.T) {
	s := listService(t)
	ctx := context.Background()
	var all []string
	next, pages := "", 0
	for {
		rows, after, err := s.Jobs(ctx, JobFilter{}, next, 2)
		if err != nil {
			t.Fatal(err)
		}
		all, next, pages = append(all, ids(rows)...), after, pages+1
		if next == "" {
			break
		}
	}
	if fmt.Sprint(all) != "[job4 job3 job2 job1 job0]" || pages != 3 {
		t.Fatalf("pages %d: %v", pages, all)
	}
}

// A job created while the operator pages does not shift the pages after it.
func TestJobsCursorSurvivesNewJobs(t *testing.T) {
	s := listService(t)
	ctx := context.Background()
	first, next, err := s.Jobs(ctx, JobFilter{}, "", 2)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.db.Exec(`INSERT INTO jobs (id, state, source_kind, source_origin, panel_id, safe_mode, created_at, updated_at)
		VALUES ('job9', 'created', 'marzban', 'https://old.example', 1, 1, 9000, 9000)`); err != nil {
		t.Fatal(err)
	}
	second, _, err := s.Jobs(ctx, JobFilter{}, next, 2)
	if err != nil || fmt.Sprint(ids(first), ids(second)) != "[job4 job3] [job2 job1]" {
		t.Fatalf("pages %v %v %v", ids(first), ids(second), err)
	}
}

func TestJobsCarryTheirCounts(t *testing.T) {
	rows, _, err := listService(t).Jobs(context.Background(), JobFilter{}, "", 1)
	if err != nil || len(rows) != 1 {
		t.Fatalf("first page: %v %v", rows, err)
	}
	done := rows[0]
	if done.Counts == nil || done.Counts.Moved != 7 || done.EndedAt == nil || !done.EndedAt.Equal(time.UnixMilli(6000)) ||
		done.Panel.Name != "alpha" || done.Panel.Kind != "pasarguard" || done.Source.Admin != "reseller" || done.LoginHeld {
		t.Fatalf("done job %+v", done)
	}
}

// job4 is done: it moved 7 and skipped 2. job1 failed and its undo took 2
// of its 5 users back. job0 was rolled back by putting the backup back,
// which leaves the run's counts as they were. job2 and job3 never ran.
func TestTotals(t *testing.T) {
	s := listService(t)
	for _, j := range []struct {
		id, state, counts string
		ended             int64
	}{
		{"job0", StateRolledBack, `{"moved":4,"undone":0,"skipped":1}`, 7000},
		{"job1", StateFailed, `{"moved":5,"undone":2,"skipped":0}`, 6500},
	} {
		if _, err := s.db.Exec("UPDATE jobs SET state = ?, counts = ?, ended_at = ? WHERE id = ?", j.state, j.counts, j.ended, j.id); err != nil {
			t.Fatal(err)
		}
	}
	got, err := s.Totals(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	jobs := map[string]int64{StateDone: 1, StateFailed: 1, StateRolledBack: 1, StateChecked: 2}
	if got.Moved != 10 || got.Skipped != 2 || !maps.Equal(got.Jobs, jobs) || got.LastEnded == nil || !got.LastEnded.Equal(time.UnixMilli(7000)) {
		t.Fatalf("totals %+v", got)
	}
}

func TestTotalsWithoutJobs(t *testing.T) {
	s := listService(t)
	if _, err := s.db.Exec("DELETE FROM jobs"); err != nil {
		t.Fatal(err)
	}
	got, err := s.Totals(context.Background())
	if err != nil || got.Moved != 0 || got.Skipped != 0 || got.LastEnded != nil || got.Jobs == nil || len(got.Jobs) != 0 {
		t.Fatalf("no jobs: %+v %v", got, err)
	}
}

func TestJobsFilters(t *testing.T) {
	s := listService(t)
	ctx := context.Background()
	for name, c := range map[string]struct {
		f    JobFilter
		want string
	}{
		"panel":  {JobFilter{PanelID: 2}, "[job3 job1]"},
		"state":  {JobFilter{States: []string{StateDone}}, "[job4]"},
		"both":   {JobFilter{PanelID: 2, States: []string{StateDone, StateFailed}}, "[]"},
		"nobody": {JobFilter{PanelID: 9}, "[]"},
		"id":     {JobFilter{ID: "job3"}, "[job3]"},
		"no id":  {JobFilter{ID: "job9"}, "[]"},
	} {
		rows, next, err := s.Jobs(ctx, c.f, "", 20)
		if err != nil || fmt.Sprint(ids(rows)) != c.want || next != "" {
			t.Errorf("%s: %v %q %v", name, ids(rows), next, err)
		}
	}
	if _, _, err := s.Jobs(ctx, JobFilter{States: []string{"lost"}}, "", 20); !errors.Is(err, ErrBadState) {
		t.Errorf("unknown state: %v", err)
	}
	for _, bad := range []string{"nodot", "zz!.job1", "10."} {
		if _, _, err := s.Jobs(ctx, JobFilter{}, bad, 20); !errors.Is(err, ErrBadCursor) {
			t.Errorf("cursor %q: %v", bad, err)
		}
	}
}

// A job Karvan holds in memory says whether its login is still there; the
// run drops it at the end.
func TestJobsShowWhoHoldsALogin(t *testing.T) {
	tg := paneldbtest.Load(t, paneldbtest.Fixture{Flavor: "sqlite", Path: "../../panel/pasarguarddb/testdata/" + pasarguarddb.Revision + ".sqlite.sql"})
	h := newHarness(t, panel.Marzban, panel.PasarGuard, tg, 3)
	checked := h.job(nil)
	ran := h.job(nil)
	h.run(ran, true)
	rows, _, err := h.svc.Jobs(context.Background(), JobFilter{}, "", 10)
	if err != nil || len(rows) != 2 {
		t.Fatalf("rows %+v %v", rows, err)
	}
	byID := map[string]JobRow{rows[0].ID: rows[0], rows[1].ID: rows[1]}
	if r := byID[checked.ID]; !r.LoginHeld || r.Counts != nil || r.State != StateChecked {
		t.Errorf("checked job %+v", r)
	}
	if r := byID[ran.ID]; r.LoginHeld || r.Counts == nil || r.Counts.Moved != 3 || r.EndedAt == nil || r.State != StateDone {
		t.Errorf("finished job %+v", r)
	}
}
