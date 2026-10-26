package dbstats

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/PdYrust/karvan/internal/discovery"
	"github.com/PdYrust/karvan/internal/panel"
	"github.com/PdYrust/karvan/internal/panel/paneldb"
	"github.com/PdYrust/karvan/internal/panel/paneldb/paneldbtest"
	"github.com/PdYrust/karvan/internal/panel/pasarguarddb"
	"github.com/PdYrust/karvan/internal/store"
)

// rig is one PasarGuard database on SQLite behind a fake discovery.
type rig struct {
	t       *testing.T
	s       *Sampler
	tg      paneldb.Target
	now     time.Time
	finding string
	karvan  *sql.DB
}

func newRig(t *testing.T) *rig {
	t.Helper()
	tg := paneldbtest.Load(t, paneldbtest.Fixture{Flavor: "sqlite", Path: "../panel/pasarguarddb/testdata/" + pasarguarddb.Revision + ".sqlite.sql"})
	db, err := store.Open(context.Background(), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if _, err := db.Exec("INSERT INTO panels (id, env_path, name, kind, scanned_at) VALUES (7, '/opt/pg/.env', 'pg', 'pasarguard', 0)"); err != nil {
		t.Fatal(err)
	}
	r := &rig{t: t, tg: tg, now: time.Date(2026, 10, 17, 12, 0, 0, 0, time.UTC), karvan: db}
	r.s = &Sampler{
		Store: NewStore(db),
		Now:   func() time.Time { return r.now },
		Panels: func(context.Context) ([]PanelRef, error) {
			return []PanelRef{{ID: 7, EnvPath: "/opt/pg/.env", Name: "pg"}}, nil
		},
		Inspect: func(_ context.Context, env string) discovery.Panel {
			p := discovery.Panel{EnvPath: env, Kind: panel.PasarGuard}
			p.Settings.DatabaseURL = "sqlite:///" + tg.Path
			if r.finding != "" {
				p.Findings = []discovery.Finding{{Code: r.finding}}
			}
			return p
		},
	}
	t.Cleanup(r.s.close)
	r.exec("INSERT INTO admins (username, hashed_password, created_at, role_id) VALUES ('reseller', 'x', ?, 3)")
	r.exec("INSERT INTO users (username, status, used_traffic, created_at, admin_id, online_at, proxy_settings) VALUES ('u1', 'active', 500, ?, 1, ?, '{}')")
	return r
}

func (r *rig) exec(query string) {
	r.t.Helper()
	w, err := paneldb.OpenWriter(context.Background(), r.tg)
	if err != nil {
		r.t.Fatal(err)
	}
	defer w.Close()
	at := paneldb.TimeArg(paneldb.SQLite, r.now.Add(-time.Minute))
	err = w.Batch(context.Background(), func(x paneldb.Execer) error {
		args := []any{at, at}[:countArgs(query)]
		_, err := x.ExecContext(context.Background(), query, args...)
		return err
	})
	if err != nil {
		r.t.Fatal(err)
	}
}

func countArgs(q string) int {
	n := 0
	for _, c := range q {
		if c == '?' {
			n++
		}
	}
	return n
}

func TestTickReadsEngineAndUsers(t *testing.T) {
	r := newRig(t)
	ctx := context.Background()
	r.s.Tick(ctx)
	snap, ok := r.s.Snapshot(7)
	if !ok || snap.Problem != nil || snap.Engine == nil || snap.Data == nil {
		t.Fatalf("snapshot %+v", snap)
	}
	r.now = r.now.Add(time.Minute)
	r.s.Tick(ctx)
	mins, err := r.s.Store.Range(ctx, 7, 0, r.now.Unix()+60, 60)
	if err != nil || len(mins) != 2 {
		t.Fatalf("minutes %+v %v", mins, err)
	}
	checks := map[string][2]any{
		"kind":       {snap.Kind, panel.PasarGuard},
		"users":      {snap.Data.Users, int64(1)},
		"online":     {snap.Data.OnlineNow, int64(1)},
		"journal":    {snap.Engine.JournalMode, "wal"},
		"no cache":   {snap.CacheHit == nil, true},
		"saved user": {mins[0].Users, int64(1)},
		"active":     {mins[0].Active, int64(1)},
		"size":       {mins[0].Size > 0, true},
		"snapshots":  {len(r.s.Snapshots()), 1},
	}
	for name, c := range checks {
		if c[0] != c[1] {
			t.Errorf("%s: got %v, want %v", name, c[0], c[1])
		}
	}
}

func TestBlockingFindingStopsReading(t *testing.T) {
	r := newRig(t)
	r.finding = discovery.EnvUnreadable
	r.s.Tick(context.Background())
	snap, ok := r.s.Snapshot(7)
	if !ok || snap.Problem == nil || snap.Problem.Code != discovery.EnvUnreadable || snap.Data != nil {
		t.Fatalf("snapshot %+v", snap)
	}
	if _, err := r.s.Detail(context.Background(), 7, "24h", r.now.Add(-24*time.Hour)); err == nil {
		t.Fatal("detail of an unreadable panel")
	}
	// Other findings, such as an unknown revision, leave the numbers readable.
	r.finding = discovery.UnknownRevision
	r.s.Tick(context.Background())
	if snap, _ := r.s.Snapshot(7); snap.Problem != nil || snap.Data == nil {
		t.Fatalf("non-blocking finding: %+v", snap)
	}
}

func TestDetailIsCached(t *testing.T) {
	r := newRig(t)
	ctx := context.Background()
	if _, err := r.s.Detail(ctx, 7, "24h", r.now.Add(-24*time.Hour)); err == nil {
		t.Fatal("detail before any reading")
	}
	r.s.Tick(ctx)
	d, err := r.s.Detail(ctx, 7, "24h", r.now.Add(-24*time.Hour))
	if err != nil || len(d.Tables) == 0 || len(d.SignUps) != 1 || d.Traffic == nil || d.Clients == nil {
		t.Fatalf("detail %+v %v", d, err)
	}
	r.exec("INSERT INTO users (username, status, used_traffic, created_at, proxy_settings) VALUES ('u2', 'active', 0, ?, '{}')")
	if again, _ := r.s.Detail(ctx, 7, "24h", r.now.Add(-24*time.Hour)); len(again.SignUps) != 1 {
		t.Fatalf("cached detail changed: %+v", again.SignUps)
	}
	r.now = r.now.Add(detailTTL)
	if fresh, _ := r.s.Detail(ctx, 7, "24h", r.now.Add(-24*time.Hour)); fresh.SignUps[0].N != 2 {
		t.Fatalf("detail not read again: %+v", fresh.SignUps)
	}
}

func TestRates(t *testing.T) {
	prev := &paneldb.EngineStats{Queries: 1000, CacheReads: 10_000, CacheMisses: 100}
	cur := &paneldb.EngineStats{Queries: 1600, CacheReads: 11_000, CacheMisses: 110}
	qps, hit := rates(prev, cur, time.Minute)
	if qps != 10 || hit == nil || *hit != 0.99 {
		t.Fatalf("qps %v hit %v", qps, hit)
	}
	if qps, hit := rates(nil, cur, time.Minute); qps != 0 || hit != nil {
		t.Fatal("rate without a previous reading")
	}
	restarted := &paneldb.EngineStats{Queries: 5}
	if qps, _ := rates(cur, restarted, time.Minute); qps != 0 {
		t.Fatal("rate across a server restart")
	}
	if _, hit := rates(prev, &paneldb.EngineStats{Queries: 1001, CacheReads: 10_000}, time.Minute); hit != nil {
		t.Fatal("hit rate without cache reads")
	}
}

func TestTickTrimsMinutesPastRetention(t *testing.T) {
	r := newRig(t)
	ctx := context.Background()
	r.s.Tick(ctx)
	r.now = r.now.Add(Retention + time.Hour)
	r.s.Tick(ctx)
	mins, err := r.s.Store.Range(ctx, 7, 0, r.now.Unix()+60, 60)
	if err != nil || len(mins) != 1 || mins[0].TS < r.now.Add(-Retention).Unix() {
		t.Fatalf("minutes %+v %v", mins, err)
	}
}
