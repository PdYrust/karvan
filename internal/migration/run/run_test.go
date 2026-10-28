package run

import (
	"context"
	"errors"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/PdYrust/karvan/internal/migration/ledger"
	"github.com/PdYrust/karvan/internal/migration/migrationtest"
	"github.com/PdYrust/karvan/internal/migration/plan"
	"github.com/PdYrust/karvan/internal/panel"
	"github.com/PdYrust/karvan/internal/panel/marzbandb"
	"github.com/PdYrust/karvan/internal/panel/paneldb"
	"github.com/PdYrust/karvan/internal/panel/paneldb/paneldbtest"
	"github.com/PdYrust/karvan/internal/panel/pasarguarddb"
	"github.com/PdYrust/karvan/internal/panel/source"
	"github.com/PdYrust/karvan/internal/panel/subtoken"
	"github.com/PdYrust/karvan/internal/secret"
	"github.com/PdYrust/karvan/internal/store"
)

// harness is one destination, one fake source and a service.
type harness struct {
	t     *testing.T
	svc   *Service
	src   migrationtest.Source
	tg    paneldb.Target
	to    panel.Kind
	panel int64
	fault func(step string, n int) error
}

func newHarness(t *testing.T, from, to panel.Kind, tg paneldb.Target, users int) *harness {
	t.Helper()
	ctx := context.Background()
	dir := t.TempDir()
	db, err := store.Open(ctx, dir)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	keyPath := filepath.Join(dir, "key")
	if err := secret.CreateKeyFile(keyPath); err != nil {
		t.Fatal(err)
	}
	key, err := secret.LoadKeyFile(keyPath)
	if err != nil {
		t.Fatal(err)
	}
	h := &harness{t: t, tg: tg, to: to}
	env := migrationtest.PanelEnv(t, tg, to)
	h.svc = New(Deps{DB: db, Key: key, DataDir: dir, Tools: migrationtest.Tools, ChunkSize: func(paneldb.Dialect) int { return 5 },
		Fault: func(step string, n int) error {
			if h.fault != nil {
				return h.fault(step, n)
			}
			return nil
		}})
	if h.panel, err = h.svc.RegisterPanel(ctx, env, "dest", string(to)); err != nil {
		t.Fatal(err)
	}
	h.src = migrationtest.NewSource(t, from, users)
	t.Cleanup(h.src.Close)
	return h
}

// created starts a job; nothing is read yet.
func (h *harness) created() *Job {
	h.t.Helper()
	j, err := h.svc.Create(context.Background(), Login{URL: h.src.URL, Username: h.src.Username, Password: secret.NewValue(h.src.Password)}, h.panel,
		source.Options{Sleep: func(context.Context, time.Duration) error { return nil }})
	if err != nil {
		h.t.Fatal(err)
	}
	return j
}

// job creates and checks a job, and sets the plan edit gives.
func (h *harness) job(edit func(p *plan.Plan)) *Job {
	h.t.Helper()
	ctx := context.Background()
	j := h.created()
	if _, err := h.svc.Check(ctx, j.ID); err != nil {
		h.t.Fatal(err)
	}
	_, p, v, _ := h.svc.View(j.ID)
	if edit != nil {
		edit(&p)
	}
	r, _, err := h.svc.SetPlan(ctx, j.ID, v, p)
	if err != nil {
		h.t.Fatal(err)
	}
	if r.Blockers > 0 {
		h.t.Fatalf("blockers: %+v %+v", r.Conflicts, r.Groups)
	}
	return j
}

// run runs a checked job to its end and returns the report.
func (h *harness) run(j *Job, safe bool) *Report {
	h.t.Helper()
	_, _, v, _ := h.svc.View(j.ID)
	if err := h.svc.Run(context.Background(), j.ID, Options{Version: v, Reload: ReloadManual, SafeMode: safe}); err != nil {
		h.t.Fatal(err)
	}
	h.svc.Wait()
	j.mu.Lock()
	defer j.mu.Unlock()
	return j.report
}

func (h *harness) count(table string) int {
	h.t.Helper()
	ctx := context.Background()
	r, err := paneldb.OpenReader(ctx, h.tg)
	if err != nil {
		h.t.Fatal(err)
	}
	defer r.Close()
	var n int
	if err := r.View(ctx, func(q paneldb.Querier) error {
		return q.QueryRowContext(ctx, "SELECT count(*) FROM "+table).Scan(&n)
	}); err != nil {
		h.t.Fatal(err)
	}
	return n
}

// each runs fn for the four directions on every loadable database.
func each(t *testing.T, fn func(t *testing.T, from, to panel.Kind, tg paneldb.Target)) {
	dirs := map[panel.Kind][2]string{
		panel.Marzban:    {"../../panel/marzbandb/testdata", marzbandb.Revision},
		panel.PasarGuard: {"../../panel/pasarguarddb/testdata", pasarguarddb.Revision},
	}
	for _, from := range []panel.Kind{panel.Marzban, panel.PasarGuard} {
		for _, to := range []panel.Kind{panel.Marzban, panel.PasarGuard} {
			t.Run(string(from)+"-to-"+string(to), func(t *testing.T) {
				paneldbtest.Each(t, dirs[to][0], dirs[to][1], func(t *testing.T, tg paneldb.Target) { fn(t, from, to, tg) })
			})
		}
	}
}

func TestMoveAndMoveAgain(t *testing.T) {
	each(t, func(t *testing.T, from, to panel.Kind, tg paneldb.Target) {
		h := newHarness(t, from, to, tg, 12)
		rep := h.run(h.job(nil), true)
		if rep.State != StateDone || rep.Counts.Moved != 12 || rep.Counts.Verified != 12 || rep.BackupID == "" {
			t.Fatalf("report: state %s counts %+v backup %q error %q", rep.State, rep.Counts, rep.BackupID, rep.Error)
		}
		if n := h.count("users"); n != 12 {
			t.Fatalf("%d users on the destination", n)
		}
		checkLinks(t, rep, to)
		// The same migration again creates nothing and merges into the admin.
		again := h.run(h.job(nil), true)
		if again.State != StateDone || again.Counts.Moved != 0 || again.Counts.AlreadyMoved != 12 || !again.Admin.Merged {
			t.Fatalf("re-run: state %s counts %+v merged %v error %q", again.State, again.Counts, again.Admin.Merged, again.Error)
		}
		if n := h.count("users"); n != 12 {
			t.Fatalf("re-run created users: %d", n)
		}
	})
}

func TestFailureWithSafeModeUndoesEverything(t *testing.T) {
	for _, step := range []string{"chunk", "ledger"} {
		t.Run(step, func(t *testing.T) { undoesEverything(t, step) })
	}
}

// undoesEverything fails chunk 2 at step: "chunk" before the destination
// commits, "ledger" after it committed but before the ledger says so.
func undoesEverything(t *testing.T, step string) {
	each(t, func(t *testing.T, from, to panel.Kind, tg paneldb.Target) {
		{
			h := newHarness(t, from, to, tg, 12)
			before := paneldbtest.ChecksumExcept(t, tg, "inbounds", "hosts")
			h.fault = func(s string, n int) error {
				if s == step && n == 2 {
					return errors.New("injected")
				}
				return nil
			}
			rep := h.run(h.job(nil), true)
			if rep.State != StateRolledBack {
				t.Fatalf("%s: state %s error %q left %v", step, rep.State, rep.Error, rep.Left)
			}
			if after := paneldbtest.ChecksumExcept(t, tg, "inbounds", "hosts"); after != before {
				t.Fatalf("%s: the destination is not as it was", step)
			}
			rows, _ := h.svc.ledger.Job(context.Background(), rep.JobID, ledger.Done, ledger.Pending)
			if len(rows) != 0 {
				t.Fatalf("%s: ledger still has %d live rows", step, len(rows))
			}
		}
	})
}

func TestFailureWithoutSafeModeKeepsWhatCommitted(t *testing.T) {
	tg := paneldbtest.Load(t, paneldbtest.Fixture{Flavor: "sqlite", Path: "../../panel/pasarguarddb/testdata/" + pasarguarddb.Revision + ".sqlite.sql"})
	h := newHarness(t, panel.Marzban, panel.PasarGuard, tg, 12)
	h.fault = func(s string, n int) error {
		if s == "chunk" && n == 2 {
			return errors.New("injected")
		}
		return nil
	}
	rep := h.run(h.job(nil), false)
	if rep.State != StateFailed || rep.BackupID != "" || h.count("users") != 5 {
		t.Fatalf("state %s backup %q users %d", rep.State, rep.BackupID, h.count("users"))
	}
	h.fault = nil
	rep = h.run(h.job(nil), false)
	if rep.State != StateDone || rep.Counts.Moved != 7 || h.count("users") != 12 {
		t.Fatalf("continue: state %s moved %d users %d (%s)", rep.State, rep.Counts.Moved, h.count("users"), rep.Error)
	}
}

func TestCredentialsAreDroppedAtTheEnd(t *testing.T) {
	tg := paneldbtest.Load(t, paneldbtest.Fixture{Flavor: "sqlite", Path: "../../panel/marzbandb/testdata/" + marzbandb.Revision + ".sqlite.sql"})
	h := newHarness(t, panel.PasarGuard, panel.Marzban, tg, 3)
	j := h.job(nil)
	_ = h.run(j, true)
	j.mu.Lock()
	defer j.mu.Unlock()
	if j.source != nil || !j.login.Password.IsZero() || j.snapshot != nil || len(j.generated) != 0 {
		t.Fatal("the job kept the old panel's login or its users")
	}
}

// The moved reseller keeps the password it signed in to the old panel
// with (owner, 2026-10-16); nothing is generated or handed out.
func TestAdminKeepsTheResellersPassword(t *testing.T) {
	tg := paneldbtest.Load(t, paneldbtest.Fixture{Flavor: "sqlite", Path: "../../panel/marzbandb/testdata/" + marzbandb.Revision + ".sqlite.sql"})
	h := newHarness(t, panel.PasarGuard, panel.Marzban, tg, 2)
	j := h.job(nil)
	rep := h.run(j, false)
	if rep.Admin.Password != PasswordKept || len(j.passwords) != 0 {
		t.Fatalf("password %q, generated %d", rep.Admin.Password, len(j.passwords))
	}
	ctx := context.Background()
	r, err := paneldb.OpenReader(ctx, tg)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	var hash string
	if err := r.View(ctx, func(q paneldb.Querier) error {
		return q.QueryRowContext(ctx, "SELECT hashed_password FROM admins WHERE username = ?", rep.Admin.Username).Scan(&hash)
	}); err != nil {
		t.Fatal(err)
	}
	if bcrypt.CompareHashAndPassword([]byte(hash), []byte(h.src.Password)) != nil {
		t.Fatal("the moved admin cannot sign in with its old password")
	}
}

// A rollback on request runs in the background: its undo reports rows
// handled of the total, the stream ends with done, and the stored report
// counts the undo next to what the run moved.
func TestRollbackOfAFailedJobOnRequest(t *testing.T) {
	h, rep := failedJob(t)
	before := len(trail(t, h, rep.JobID))
	h.rollBackOnRequest(rep.JobID)
	back, err := h.svc.Report(rep.JobID)
	if err != nil || back.State != StateRolledBack || back.Counts.Undone != 10 || back.Counts.Moved != 10 || h.count("users") != 0 || h.count("admins") != 0 {
		t.Fatalf("rollback: %v %+v users %d", err, back, h.count("users"))
	}
	want := []string{"undo stage start", "undo progress 0/10 pass <nil>", "undo progress 5/10 pass <nil>", "undo progress 10/10 pass <nil>",
		"undo stage done", " done rolled_back"}
	if got := trail(t, h, rep.JobID)[before:]; !slices.Equal(got, want) {
		t.Fatalf("events %q", got)
	}
	checkStoredRollback(t, h.svc, rep.JobID, 10)
	if err := h.svc.Rollback(context.Background(), rep.JobID); !errors.Is(err, ErrNoRollback) {
		t.Fatalf("second rollback: %v", err)
	}
}

func TestRecoverSettlesAnInterruptedJob(t *testing.T) {
	tg := paneldbtest.Load(t, paneldbtest.Fixture{Flavor: "sqlite", Path: "../../panel/marzbandb/testdata/" + marzbandb.Revision + ".sqlite.sql"})
	h := newHarness(t, panel.PasarGuard, panel.Marzban, tg, 6)
	// The ledger write after the commit fails, as if Karvan stopped there.
	h.fault = func(s string, n int) error {
		if s == "ledger" && n == 1 {
			return errors.New("stopped")
		}
		return nil
	}
	j := h.job(nil)
	rep := h.run(j, false)
	ctx := context.Background()
	if _, err := h.svc.db.ExecContext(ctx, "UPDATE jobs SET state = ? WHERE id = ?", StateRunning, rep.JobID); err != nil {
		t.Fatal(err)
	}
	if _, err := h.svc.db.ExecContext(ctx, "UPDATE ledger SET state = ? WHERE job_id = ? AND kind = 'user'", ledger.Pending, rep.JobID); err != nil {
		t.Fatal(err)
	}
	if err := h.svc.Recover(ctx); err != nil {
		t.Fatal(err)
	}
	var state string
	_ = h.svc.db.QueryRowContext(ctx, "SELECT state FROM jobs WHERE id = ?", rep.JobID).Scan(&state)
	done, _ := h.svc.ledger.Job(ctx, rep.JobID, ledger.Done)
	if state != StateInterrupted || len(done) != 6 {
		t.Fatalf("state %s, %d rows settled as done", state, len(done))
	}
}

// checkLinks makes sure each new link carries a token the destination
// itself would accept for that user.
func checkLinks(t *testing.T, rep *Report, to panel.Kind) {
	t.Helper()
	for _, u := range rep.Users {
		i := strings.LastIndex(u.NewLink, "/sub/")
		if !u.Moved || i < 0 {
			t.Fatalf("no link for %s: %q", u.Key, u.NewLink)
		}
		token := u.NewLink[i+len("/sub/"):]
		p, ok := subtoken.Parse(token)
		if to == panel.PasarGuard && (!ok || p.UserID != u.DestID || subtoken.PasarGuard(u.DestID, migrationtest.JWTSecret, p.Issued) != token) {
			t.Fatalf("bad PasarGuard link %q", u.NewLink)
		}
		if to == panel.Marzban && (!ok || p.Username != u.Username || !subtoken.LegacyValid(token, migrationtest.JWTSecret)) {
			t.Fatalf("bad Marzban link %q", u.NewLink)
		}
	}
}

// Every method is listed; the API ones need a full admin to act as.
func TestReloadOptions(t *testing.T) {
	tg := paneldbtest.Load(t, paneldbtest.Fixture{Flavor: "sqlite", Path: "../../panel/pasarguarddb/testdata/" + pasarguarddb.Revision + ".sqlite.sql"})
	h := newHarness(t, panel.Marzban, panel.PasarGuard, tg, 2)
	h.svc.Reloaders[ReloadTouch] = manualReload{}
	j := h.job(nil)
	opts, err := h.svc.ReloadOptions(context.Background(), j.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(opts) != 3 || !opts[0].Available || opts[0].Command != "pasarguard restart" ||
		opts[1].Reason != "no_full_admin" || opts[2].Reason != "not_built_in" {
		t.Fatalf("options: %+v", opts)
	}
	addFullAdmin(t, tg)
	if opts, _ = h.svc.ReloadOptions(context.Background(), j.ID); !opts[1].Available {
		t.Fatalf("touch with a full admin: %+v", opts[1])
	}
}

// Two quick submits start one run; the other is busy.
func TestDoubleRunStartsOnce(t *testing.T) {
	tg := paneldbtest.Load(t, paneldbtest.Fixture{Flavor: "sqlite", Path: "../../panel/pasarguarddb/testdata/" + pasarguarddb.Revision + ".sqlite.sql"})
	h := newHarness(t, panel.Marzban, panel.PasarGuard, tg, 3)
	j := h.job(nil)
	_, _, v, _ := h.svc.View(j.ID)
	errs := make(chan error, 2)
	for range 2 {
		go func() { errs <- h.svc.Run(context.Background(), j.ID, Options{Version: v, Reload: ReloadManual}) }()
	}
	a, b := <-errs, <-errs
	h.svc.Wait()
	if (a == nil) == (b == nil) || !errors.Is(a, ErrBusy) && !errors.Is(b, ErrBusy) {
		t.Fatalf("errors %v and %v", a, b)
	}
	if n := h.count("users"); n != 3 {
		t.Fatalf("%d users written", n)
	}
}

// Sweep never waits for a busy job, so it cannot deadlock against Run.
func TestSweepSkipsABusyJob(t *testing.T) {
	tg := paneldbtest.Load(t, paneldbtest.Fixture{Flavor: "sqlite", Path: "../../panel/pasarguarddb/testdata/" + pasarguarddb.Revision + ".sqlite.sql"})
	h := newHarness(t, panel.Marzban, panel.PasarGuard, tg, 1)
	j := h.job(nil)
	j.mu.Lock()
	done := make(chan struct{})
	go func() { h.svc.Sweep(); close(done) }()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("Sweep waited for the job")
	}
	j.mu.Unlock()
}

// A reader that falls this far behind is dropped, so its stream ends and
// the client reconnects instead of waiting for an event it lost.
func TestLaggingEventReaderIsDropped(t *testing.T) {
	db, err := store.Open(context.Background(), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec("INSERT INTO panels (id, env_path, name, scanned_at) VALUES (1, '/x/.env', 'x', 0)"); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec("INSERT INTO jobs (id, state, source_kind, source_origin, panel_id, safe_mode, created_at, updated_at) VALUES ('j', 'running', 'marzban', 'o', 1, 1, 0, 0)"); err != nil {
		t.Fatal(err)
	}
	b := newBus(db, time.Now)
	_, ch, cancel, err := b.subscribe(context.Background(), "j", 0)
	if err != nil {
		t.Fatal(err)
	}
	defer cancel()
	for i := range 300 {
		if err := b.emit(context.Background(), "j", StageExecute, EventProgress, map[string]any{"done": i}); err != nil {
			t.Fatal(err)
		}
	}
	n := 0
	for range ch {
		n++
	}
	if n == 0 || n >= 300 {
		t.Fatalf("read %d events before the stream ended", n)
	}
}

// With an API key there is no password to keep: one is generated and
// waits for its one download.
func TestGeneratedPasswordWaitsForItsDownload(t *testing.T) {
	tg := paneldbtest.Load(t, paneldbtest.Fixture{Flavor: "sqlite", Path: "../../panel/marzbandb/testdata/" + marzbandb.Revision + ".sqlite.sql"})
	h := newHarness(t, panel.PasarGuard, panel.Marzban, tg, 2)
	ctx := context.Background()
	j, err := h.svc.Create(ctx, Login{URL: h.src.URL, Kind: panel.PasarGuard, APIKey: secret.NewValue(h.src.APIKey)}, h.panel,
		source.Options{Sleep: func(context.Context, time.Duration) error { return nil }})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := h.svc.Check(ctx, j.ID); err != nil {
		t.Fatal(err)
	}
	rep := h.run(j, false)
	if rep.State != StateDone || rep.Admin.Password != PasswordGenerated {
		t.Fatalf("state %s password %q", rep.State, rep.Admin.Password)
	}
	first, _ := h.svc.TakePasswords(j.ID)
	again, _ := h.svc.TakePasswords(j.ID)
	if len(first) != 1 || len(again) != 0 {
		t.Fatalf("passwords %d then %d", len(first), len(again))
	}
}
