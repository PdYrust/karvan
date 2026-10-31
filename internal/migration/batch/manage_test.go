package batch

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"
)

func TestRenameChecksTheName(t *testing.T) {
	s, ids := twoBatches(t)
	ctx := context.Background()
	b, err := s.Rename(ctx, ids["b"], "  spring move  ")
	if err != nil || b.Name != "spring move" || b.UpdatedAt.UnixMilli() != 99_000 {
		t.Fatalf("rename %+v %v", b, err)
	}
	for _, bad := range []string{"", "   ", strings.Repeat("x", 65), "a\nb", "a\x00b", "tab\there"} {
		if _, err := s.Rename(ctx, ids["a"], bad); !errors.Is(err, ErrInvalidName) {
			t.Errorf("%q: %v", bad, err)
		}
	}
	if b, err := s.Rename(ctx, ids["a"], strings.Repeat("ب", 64)); err != nil || len([]rune(b.Name)) != 64 {
		t.Errorf("64 Persian letters: %v", err)
	}
	exec(t, s.db, "UPDATE batches SET state = 'deleted' WHERE id = ?", ids["b"])
	if _, err := s.Rename(ctx, ids["b"], "again"); !errors.Is(err, ErrNotFound) {
		t.Errorf("deleted: %v", err)
	}
}

func TestDeleteRefusesWhatItMustKeep(t *testing.T) {
	s, ids := twoBatches(t)
	ctx := context.Background()
	if err := s.Delete(ctx, ids["a"], "alice"); !errors.Is(err, ErrActive) {
		t.Errorf("active: %v", err)
	}
	exec(t, s.db, "UPDATE batches SET state = 'inactive', inactive_reason = 'restoring' WHERE id = ?", ids["b"])
	if err := s.Delete(ctx, ids["b"], "bob"); !errors.Is(err, ErrBusy) {
		t.Errorf("restoring: %v", err)
	}
	exec(t, s.db, "UPDATE batches SET inactive_reason = 'rolled_back' WHERE id = ?", ids["b"])
	var c *ConfirmError
	if err := s.Delete(ctx, ids["b"], "Bob"); !errors.As(err, &c) || c.Name != "bob" {
		t.Errorf("wrong name: %v", err)
	}
	if err := s.Delete(ctx, 999, "x"); !errors.Is(err, ErrNotFound) {
		t.Errorf("unknown: %v", err)
	}
}

// The only batch of a rule stays; one of two can go, and the rule keeps
// the other.
func TestDeleteKeepsARulesOnlyBatch(t *testing.T) {
	s, ids := twoBatches(t)
	ctx := context.Background()
	exec(t, s.db, `INSERT INTO legacy_sites (id, name, panel_id, source_kind, source_url, old_path, listen_port, mode, created_at, updated_at)
		VALUES (2, 'only bob', 1, 'marzban', ?, 'sub', 8443, 'batches', 1, 1)`, origin)
	exec(t, s.db, "INSERT INTO legacy_site_batches (site_id, batch_id) VALUES (1, ?), (2, ?)", ids["b"], ids["b"])
	var inUse *InUseError
	if err := s.Delete(ctx, ids["b"], "bob"); !errors.As(err, &inUse) || !slices.Equal(inUse.Rules, []Rule{{ID: 2, Name: "only bob"}}) {
		t.Fatalf("in use: %v", err)
	}
	exec(t, s.db, "DELETE FROM legacy_site_batches WHERE site_id = 2")
	if err := s.Delete(ctx, ids["b"], "bob"); err != nil {
		t.Fatal(err)
	}
	if got := ids2(t, s, "SELECT batch_id FROM legacy_site_batches WHERE site_id = 1"); !slices.Equal(got, []int64{ids["a"]}) {
		t.Fatalf("rule 1 batches %v", got)
	}
}

// A deleted batch keeps its row and its members, loses its tokens
// and rule links, leaves the list, and a later batch does not claim its
// users.
func TestDeleteKeepsMembersOutOfService(t *testing.T) {
	s, ids := twoBatches(t)
	ctx := context.Background()
	exec(t, s.db, `INSERT INTO legacy_tokens (mac, site_id, ledger_id, issued_at, verified_at, checked_at) VALUES (x'03', 1, ?, 1, 1, 1)`, ids["bobUser"])
	exec(t, s.db, "INSERT INTO legacy_site_batches (site_id, batch_id) VALUES (1, ?)", ids["b"])
	if err := s.Delete(ctx, ids["b"], "bob"); err != nil {
		t.Fatal(err)
	}
	var state, reason string
	var deletedAt int64
	if err := s.db.QueryRow("SELECT state, inactive_reason, deleted_at FROM batches WHERE id = ?", ids["b"]).Scan(&state, &reason, &deletedAt); err != nil ||
		state != "deleted" || reason != "rolled_back" || deletedAt != 99_000 {
		t.Fatalf("row %s %s %d %v", state, reason, deletedAt, err)
	}
	if got := members(t, s.db, ids["b"]); !slices.Equal(got, []int64{ids["bobUser"]}) {
		t.Fatalf("members %v", got)
	}
	if got := ids2(t, s, "SELECT ledger_id FROM legacy_tokens WHERE ledger_id = ?", ids["bobUser"]); len(got) != 0 {
		t.Fatalf("tokens left %v", got)
	}
	if got := ids2(t, s, "SELECT site_id FROM legacy_site_batches WHERE batch_id = ?", ids["b"]); len(got) != 0 {
		t.Fatalf("rule links left %v", got)
	}
	if got := ids2(t, s, "SELECT ledger_id FROM legacy_tokens WHERE ledger_id = ?", ids["moved"]); len(got) != 2 {
		t.Fatalf("other batch's tokens %v", got)
	}
	job(t, s.db, "c", "done", origin, "bob", 9000)
	row(t, s.db, "user", "c", "done", origin, dest, 8000)
	later := ensure(t, s, "c", dest, Facts{})
	if got := members(t, s.db, later); slices.Contains(got, ids["bobUser"]) || len(got) != 1 {
		t.Fatalf("later batch claimed %v", got)
	}
	if _, err := s.Get(ctx, ids["b"]); !errors.Is(err, ErrNotFound) {
		t.Fatalf("get after delete: %v", err)
	}
	if err := s.Delete(ctx, ids["b"], "bob"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("second delete: %v", err)
	}
}

func TestReactivateOnlyAfterAFailedRestore(t *testing.T) {
	s, ids := twoBatches(t)
	ctx := context.Background()
	for _, id := range []int64{ids["a"], ids["b"]} {
		if _, err := s.Reactivate(ctx, id); !errors.Is(err, ErrNotReactivatable) {
			t.Errorf("batch %d: %v", id, err)
		}
	}
	exec(t, s.db, "UPDATE batches SET inactive_reason = 'restore_failed' WHERE id = ?", ids["b"])
	b, err := s.Reactivate(ctx, ids["b"])
	if err != nil || b.State != "active" || b.InactiveReason != "" || b.InactiveAt != nil {
		t.Fatalf("reactivate %+v %v", b, err)
	}
	if _, err := s.Reactivate(ctx, 999); !errors.Is(err, ErrNotFound) {
		t.Errorf("unknown: %v", err)
	}
}

func ids2(t *testing.T, s *Store, q string, args ...any) []int64 {
	t.Helper()
	return ids(t, s.db, q, args...)
}
