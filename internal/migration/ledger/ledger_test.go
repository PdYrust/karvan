package ledger

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/PdYrust/karvan/internal/panel"
	"github.com/PdYrust/karvan/internal/store"
)

func open(t *testing.T) *Ledger {
	t.Helper()
	db, err := store.Open(context.Background(), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return New(db)
}

func entry(job, key string, state string) Entry {
	created := time.Date(2025, 1, 2, 3, 4, 5, 123456000, time.UTC)
	return Entry{Kind: KindUser, JobID: job, State: state, SourceOrigin: "https://old", SourceKey: key, SourceCreated: created,
		SourceUsername: key, Dest: "sqlite:///var/lib/marzban/db.sqlite3",
		Row: panel.Row{ID: 7, Username: key, CreatedAt: created}}
}

func TestASourceUserIsClaimedOnce(t *testing.T) {
	ctx := context.Background()
	l := open(t)
	ids, err := l.Add(ctx, []Entry{entry("j1", "ali", Pending), entry("j1", "reza", Pending)})
	if err != nil || len(ids) != 2 {
		t.Fatal(ids, err)
	}
	if _, err := l.Add(ctx, []Entry{entry("j2", "sara", Pending), entry("j2", "ali", Pending)}); !errors.Is(err, ErrClaimed) {
		t.Fatalf("second claim: %v", err)
	}
	if rows, _ := l.Job(ctx, "j2"); len(rows) != 0 {
		t.Fatal("a failed claim must leave nothing behind")
	}
	if err := l.Set(ctx, Compensated, ids[:1]); err != nil {
		t.Fatal(err)
	}
	if _, err := l.Add(ctx, []Entry{entry("j3", "ali", Pending)}); err != nil {
		t.Fatalf("a compensated row frees the identity: %v", err)
	}
	// The same username created again on the source is someone else.
	again := entry("j4", "reza", Pending)
	again.SourceCreated = again.SourceCreated.Add(time.Hour)
	if _, err := l.Add(ctx, []Entry{again}); err != nil {
		t.Fatalf("a new created_at is a new user: %v", err)
	}
}

func TestLiveAndJob(t *testing.T) {
	ctx := context.Background()
	l := open(t)
	ids, err := l.Add(ctx, []Entry{entry("j1", "a", Pending), entry("j1", "b", Pending), entry("j1", "c", Adopted)})
	if err != nil {
		t.Fatal(err)
	}
	if err := l.Set(ctx, Done, ids[:1]); err != nil {
		t.Fatal(err)
	}
	if err := l.Set(ctx, Compensated, ids[1:2]); err != nil {
		t.Fatal(err)
	}
	live, err := l.Live(ctx, "https://old", "sqlite:///var/lib/marzban/db.sqlite3")
	if err != nil || len(live) != 2 {
		t.Fatalf("live = %+v, %v", live, err)
	}
	if !live[0].SourceCreated.Equal(entry("", "", "").SourceCreated) || live[0].Row.ID != 7 {
		t.Fatalf("times or ids lost: %+v", live[0])
	}
	done, err := l.Job(ctx, "j1", Done, Pending)
	if err != nil || len(done) != 1 || done[0].SourceKey != "a" {
		t.Fatalf("job = %+v, %v", done, err)
	}
}
