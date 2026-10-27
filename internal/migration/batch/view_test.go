package batch

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"strings"
	"testing"
)

// twoBatches makes an older batch "alice" (job a, two users, one carried
// from a failed run) and a newer inactive one "bob" (job b), on panel 1.
func twoBatches(t *testing.T) (*Store, map[string]int64) {
	t.Helper()
	s, db := open(t)
	job(t, db, "failed", "failed", origin, "alice", 500)
	job(t, db, "a", "done", origin, "alice", 1000)
	job(t, db, "b", "done", origin, "bob", 3000)
	ids := map[string]int64{
		"carried": row(t, db, "user", "failed", "done", origin, dest, 400),
		"adminA":  row(t, db, "admin", "a", "done", origin, dest, 900),
		"moved":   row(t, db, "user", "a", "done", origin, dest, 900),
		"adopted": row(t, db, "user", "a", "adopted", origin, dest, 900),
		"adminB":  row(t, db, "admin", "b", "merged", origin, dest, 2900),
		"bobUser": row(t, db, "user", "b", "done", origin, dest, 2900),
	}
	exec(t, db, "UPDATE ledger SET source_username = 'old_' || source_key, dest_username = 'new_' || source_key, dest_id = id WHERE kind = 'user'")
	exec(t, db, "UPDATE ledger SET source_username = 'alice', dest_username = 'alice_2', dest_id = 70 WHERE id = ?", ids["adminA"])
	exec(t, db, "UPDATE ledger SET source_username = 'bob', dest_username = 'bob', dest_id = 71 WHERE id = ?", ids["adminB"])
	ids["a"] = ensure(t, s, "a", dest, Facts{OldPath: "sub", OldHosts: []string{"s.example"}, OldPort: 443})
	ids["b"] = ensure(t, s, "b", dest, Facts{})
	exec(t, db, "UPDATE batches SET state = 'inactive', inactive_reason = 'rolled_back', inactive_at = 3500 WHERE id = ?", ids["b"])
	exec(t, db, `INSERT INTO legacy_sites (id, name, panel_id, source_kind, source_url, old_path, listen_port, mode, created_at, updated_at)
		VALUES (1, 'rule', 1, 'marzban', ?, 'sub', 443, 'batches', 1, 1)`, origin)
	exec(t, db, "INSERT INTO legacy_site_batches (site_id, batch_id) VALUES (1, ?)", ids["a"])
	exec(t, db, `INSERT INTO legacy_tokens (mac, site_id, ledger_id, issued_at, verified_at, checked_at) VALUES
		(x'01', 1, ?, 1, 10, 20), (x'02', 1, ?, 1, 10, 40)`, ids["moved"], ids["moved"])
	exec(t, db, "INSERT INTO legacy_closed (ledger_id, reason, closed_at) VALUES (?, 'revoked', 50)", ids["adopted"])
	return s, ids
}

func TestGetReadsTheWholeBatch(t *testing.T) {
	s, ids := twoBatches(t)
	b, err := s.Get(context.Background(), ids["a"])
	if err != nil {
		t.Fatal(err)
	}
	type head struct {
		name, state, job, panel, kind, from string
		panelID, created                    int64
		inactive                            bool
	}
	got := head{b.Name, b.State, b.JobID, b.Panel.Name, b.Source.Kind, b.Source.Origin, b.Panel.ID, b.CreatedAt.UnixMilli(), b.InactiveAt != nil}
	if want := (head{"alice", "active", "a", "p", "marzban", origin, 1, 1000, false}); got != want {
		t.Fatalf("batch %+v, want %+v", got, want)
	}
	if b.OldLinks.Path != "sub" || !slices.Equal(b.OldLinks.Hosts, []string{"s.example"}) || b.OldLinks.Port != 443 {
		t.Fatalf("old links %+v", b.OldLinks)
	}
}

func TestGetCountsAdminsAndRules(t *testing.T) {
	s, ids := twoBatches(t)
	b, err := s.Get(context.Background(), ids["a"])
	if err != nil {
		t.Fatal(err)
	}
	if b.Counts != (Counts{Users: 3, Verified: 1, Closed: 1}) {
		t.Fatalf("counts %+v", b.Counts)
	}
	if !slices.Equal(b.Rules, []Rule{{ID: 1, Name: "rule"}}) {
		t.Fatalf("rules %+v", b.Rules)
	}
	// The carried user's own job has no admin row: its job's old admin.
	want := []Admin{{Source: "alice", Dest: "", Users: 1}, {Source: "alice", Dest: "alice_2", Users: 2}}
	if !slices.Equal(b.Admins, want) {
		t.Fatalf("admins %+v", b.Admins)
	}
	inactive, err := s.Get(context.Background(), ids["b"])
	if err != nil || inactive.InactiveReason != "rolled_back" || inactive.InactiveAt == nil || inactive.InactiveAt.UnixMilli() != 3500 {
		t.Fatalf("inactive %+v %v", inactive, err)
	}
}

func TestGetHidesDeletedAndUnknown(t *testing.T) {
	s, ids := twoBatches(t)
	exec(t, s.db, "UPDATE batches SET state = 'deleted' WHERE id = ?", ids["b"])
	for _, id := range []int64{ids["b"], 999} {
		if _, err := s.Get(context.Background(), id); !errors.Is(err, ErrNotFound) {
			t.Errorf("batch %d: %v", id, err)
		}
	}
}

// listed is the ids of one page of the list, and whether more follow.
func listed(t *testing.T, s *Store, f Filter, after string, limit int) ([]int64, string) {
	t.Helper()
	page, next, err := s.List(context.Background(), f, after, limit)
	if err != nil {
		t.Fatal(err)
	}
	out := []int64{}
	for _, b := range page {
		out = append(out, b.ID)
	}
	return out, next
}

func TestListPagesNewestFirst(t *testing.T) {
	s, ids := twoBatches(t)
	if all, next := listed(t, s, Filter{}, "", 50); !slices.Equal(all, []int64{ids["b"], ids["a"]}) || next != "" {
		t.Fatalf("list %v next %q", all, next)
	}
	first, next := listed(t, s, Filter{}, "", 1)
	if !slices.Equal(first, []int64{ids["b"]}) || next == "" {
		t.Fatalf("page 1 %v %q", first, next)
	}
	if second, last := listed(t, s, Filter{}, next, 1); !slices.Equal(second, []int64{ids["a"]}) || last != "" {
		t.Fatalf("page 2 %v %q", second, last)
	}
}

func TestListFilters(t *testing.T) {
	s, ids := twoBatches(t)
	for state, want := range map[string]int64{"active": ids["a"], "inactive": ids["b"]} {
		if got, _ := listed(t, s, Filter{State: state}, "", 50); !slices.Equal(got, []int64{want}) {
			t.Errorf("state %s: %v", state, got)
		}
	}
	if got, _ := listed(t, s, Filter{PanelID: 2}, "", 50); len(got) != 0 {
		t.Errorf("panel 2: %v", got)
	}
	exec(t, s.db, "UPDATE batches SET state = 'deleted' WHERE id = ?", ids["b"])
	if got, _ := listed(t, s, Filter{}, "", 50); !slices.Equal(got, []int64{ids["a"]}) {
		t.Errorf("a deleted batch listed: %v", got)
	}
}

func TestListRefusesBadValues(t *testing.T) {
	s, _ := twoBatches(t)
	ctx := context.Background()
	if _, _, err := s.List(ctx, Filter{State: "deleted"}, "", 50); !errors.Is(err, ErrBadState) {
		t.Errorf("state deleted: %v", err)
	}
	for _, c := range []string{"x", "zz.", ".5", "1.!"} {
		if _, _, err := s.List(ctx, Filter{}, c, 50); !errors.Is(err, ErrBadCursor) {
			t.Errorf("cursor %q: %v", c, err)
		}
	}
}

// userLine is what a test compares of a batch user.
type userLine struct {
	joined, admin    string
	verified         int
	lastChecked      int64
	closed, oldNames bool
}

func lineOf(u User) userLine {
	l := userLine{joined: u.Joined, admin: u.Admin, verified: u.OldLinks.Verified, closed: u.Closed,
		oldNames: strings.HasPrefix(u.SourceUsername, "old_") && strings.HasPrefix(u.Username, "new_")}
	if u.OldLinks.LastChecked != nil {
		l.lastChecked = u.OldLinks.LastChecked.Unix()
	}
	return l
}

func TestUsersSaysHowEachJoined(t *testing.T) {
	s, ids := twoBatches(t)
	users, next, err := s.Users(context.Background(), ids["a"], "", "", 50)
	if err != nil || next != "" || len(users) != 3 {
		t.Fatalf("users %+v %q %v", users, next, err)
	}
	want := map[int64]userLine{
		ids["moved"]:   {joined: "moved", admin: "alice_2", verified: 2, lastChecked: 40, oldNames: true},
		ids["carried"]: {joined: "carried", admin: "alice", oldNames: true},
		ids["adopted"]: {joined: "adopted", admin: "alice_2", closed: true, oldNames: true},
	}
	for _, u := range users {
		if got := lineOf(u); got != want[u.ID] || u.DestID != u.ID {
			t.Errorf("user %d: %+v dest %d, want %+v", u.ID, got, u.DestID, want[u.ID])
		}
	}
}

func TestUsersSearchesAndPages(t *testing.T) {
	s, ids := twoBatches(t)
	ctx := context.Background()
	var name string
	if err := s.db.QueryRow("SELECT source_username FROM ledger WHERE id = ?", ids["adopted"]).Scan(&name); err != nil {
		t.Fatal(err)
	}
	found, _, err := s.Users(ctx, ids["a"], strings.ToUpper(name), "", 50)
	if err != nil || len(found) != 1 || found[0].ID != ids["adopted"] {
		t.Fatalf("search %q: %+v %v", name, found, err)
	}
	page, next, err := s.Users(ctx, ids["a"], "", "", 2)
	if err != nil || len(page) != 2 || next == "" {
		t.Fatalf("page 1 %+v %q %v", page, next, err)
	}
	rest, last, err := s.Users(ctx, ids["a"], "", next, 2)
	if err != nil || len(rest) != 1 || last != "" || rest[0].ID <= page[1].ID {
		t.Fatalf("page 2 %+v %q %v", rest, last, err)
	}
	if _, _, err := s.Users(ctx, ids["a"], "", "x", 2); !errors.Is(err, ErrBadCursor) {
		t.Errorf("bad cursor: %v", err)
	}
	if _, _, err := s.Users(ctx, 999, "", "", 2); !errors.Is(err, ErrNotFound) {
		t.Errorf("unknown batch: %v", err)
	}
}

// No token, MAC or other secret column reaches the JSON of a batch or its
// users (D57).
func TestNoTokenInTheJSON(t *testing.T) {
	s, ids := twoBatches(t)
	b, _ := s.Get(context.Background(), ids["a"])
	users, _, _ := s.Users(context.Background(), ids["a"], "", "", 50)
	raw, err := json.Marshal(map[string]any{"b": b, "u": users})
	if err != nil {
		t.Fatal(err)
	}
	for _, bad := range []string{"mac", "token", "issued", "pin", "sourceIp", "secret"} {
		if strings.Contains(strings.ToLower(string(raw)), strings.ToLower(bad)) {
			t.Errorf("%q in %s", bad, raw)
		}
	}
}

func TestMemberIsOnlyInItsBatch(t *testing.T) {
	s, ids := twoBatches(t)
	ctx := context.Background()
	m, err := s.Member(ctx, ids["a"], ids["moved"])
	if err != nil || m.PanelID != 1 || m.Dest != dest || m.DestID != ids["moved"] || m.AdminDestID != 70 || !strings.HasPrefix(m.Username, "new_") {
		t.Fatalf("member %+v %v", m, err)
	}
	for _, c := range [][2]int64{{ids["a"], ids["bobUser"]}, {ids["a"], ids["adminA"]}, {ids["b"], ids["moved"]}, {999, ids["moved"]}} {
		if _, err := s.Member(ctx, c[0], c[1]); !errors.Is(err, ErrNotFound) {
			t.Errorf("batch %d row %d: %v", c[0], c[1], err)
		}
	}
	exec(t, s.db, "UPDATE batches SET state = 'deleted' WHERE id = ?", ids["b"])
	if _, err := s.Member(ctx, ids["b"], ids["bobUser"]); !errors.Is(err, ErrNotFound) {
		t.Errorf("deleted batch: %v", err)
	}
}

func TestByJob(t *testing.T) {
	s, ids := twoBatches(t)
	ref, err := s.ByJob(context.Background(), "a")
	if err != nil || ref == nil || *ref != (Ref{ID: ids["a"], Name: "alice", State: "active"}) {
		t.Fatalf("ref %+v %v", ref, err)
	}
	if ref, err := s.ByJob(context.Background(), "failed"); err != nil || ref != nil {
		t.Fatalf("job without a batch: %+v %v", ref, err)
	}
}
