package run

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/PdYrust/karvan/internal/migration/dest"
	"github.com/PdYrust/karvan/internal/migration/engine"
	"github.com/PdYrust/karvan/internal/panel"
	"github.com/PdYrust/karvan/internal/panel/paneldb"
	"github.com/PdYrust/karvan/internal/panel/paneldb/paneldbtest"
	"github.com/PdYrust/karvan/internal/panel/pasarguarddb"
	"github.com/PdYrust/karvan/internal/panel/source"
)

func pasarGuardDB(t *testing.T) paneldb.Target {
	t.Helper()
	return paneldbtest.Load(t, paneldbtest.Fixture{Flavor: "sqlite", Path: "../../panel/pasarguarddb/testdata/" + pasarguarddb.Revision + ".sqlite.sql"})
}

// trail lists a job's events as "stage type what": what is the state of
// a stage event, the code of a log event, or a progress event's numbers.
func trail(t *testing.T, h *harness, jobID string) []string {
	t.Helper()
	past, _, stop, err := h.svc.Events(context.Background(), jobID, 0)
	if err != nil {
		t.Fatal(err)
	}
	stop()
	var out []string
	for _, e := range past {
		what := fmt.Sprint(e.Data["state"])
		switch e.Type {
		case EventLog:
			what = fmt.Sprint(e.Data["code"])
		case EventProgress:
			what = fmt.Sprintf("%v/%v pass %v", e.Data["done"], e.Data["total"], e.Data["pass"])
		}
		out = append(out, e.Stage+" "+e.Type+" "+what)
	}
	return out
}

// hold makes the old panel wait before it answers, until release, which
// may be called twice (early and deferred).
func (h *harness) hold() (release func()) {
	held, done := make(chan struct{}), make(chan struct{})
	go h.src.Mutate(func(*[]map[string]any) { close(held); <-done })
	<-held
	return sync.OnceFunc(func() { close(done) })
}

func TestCompareReadsInTheBackground(t *testing.T) {
	h := newHarness(t, panel.Marzban, panel.PasarGuard, pasarGuardDB(t), 12)
	j := h.created()
	if err := h.svc.Compare(j.ID); err != nil {
		t.Fatal(err)
	}
	h.svc.Wait()
	// The list is read until two passes agree, so it is read twice.
	want := []string{"read stage start", "read progress 12/12 pass 1", "read progress 12/12 pass 2", "read stage done",
		"build stage start", "build stage done"}
	if got := trail(t, h, j.ID); !slices.Equal(got, want) {
		t.Fatalf("events %q", got)
	}
	if sum, err := h.svc.Summary(j.ID); err != nil || sum.State != StateChecked || sum.Users != 12 {
		t.Fatalf("summary %+v %v", sum, err)
	}
}

// A compare that fails says why on the stream and leaves the build the
// operator reviewed.
func TestFailedCompareKeepsThePreviousBuild(t *testing.T) {
	h := newHarness(t, panel.Marzban, panel.PasarGuard, pasarGuardDB(t), 3)
	j := h.job(nil)
	before, _, v, _ := h.svc.View(j.ID)
	h.src.Close()
	if err := h.svc.Compare(j.ID); err != nil {
		t.Fatal(err)
	}
	h.svc.Wait()
	got := trail(t, h, j.ID)
	if end := got[len(got)-3:]; !slices.Equal(end, []string{"read stage start", "read log source_unreachable", "read stage failed"}) {
		t.Fatalf("events %q", got)
	}
	after, _, again, _ := h.svc.View(j.ID)
	if after != before || again != v || j.State() != StateChecked {
		t.Fatalf("the build or its version changed: %d to %d", v, again)
	}
}

// While a compare reads, readers keep the previous build, and whatever
// would change the job is busy until the read ends.
func TestACompareMakesTheJobBusy(t *testing.T) {
	h := newHarness(t, panel.Marzban, panel.PasarGuard, pasarGuardDB(t), 3)
	j := h.job(nil)
	_, p, v, _ := h.svc.View(j.ID)
	release := h.hold()
	defer release()
	if err := h.svc.Compare(j.ID); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if err := h.svc.Compare(j.ID); !errors.Is(err, ErrBusy) {
		t.Errorf("a second compare: %v", err)
	}
	if _, _, err := h.svc.SetPlan(ctx, j.ID, v, p); !errors.Is(err, ErrBusy) {
		t.Errorf("a plan edit: %v", err)
	}
	if err := h.svc.Run(ctx, j.ID, Options{Version: v, Reload: ReloadManual}); !errors.Is(err, ErrBusy) {
		t.Errorf("a run: %v", err)
	}
	if sum, err := h.svc.Summary(j.ID); err != nil || sum.Users != 3 {
		t.Errorf("summary during the read: %+v %v", sum, err)
	}
	// Idle for hours on paper, but the read uses the login: it stays.
	j.mu.Lock()
	j.lastUse = time.Now().Add(-3 * IdleTimeout)
	j.mu.Unlock()
	h.svc.Sweep()
	release()
	h.svc.Wait()
	if got := trail(t, h, j.ID); got[len(got)-1] != "build stage done" || !j.holdsLogin() {
		t.Fatalf("events %q", got)
	}
}

// Shutdown stops a compare that waits for the old panel.
func TestShutdownStopsACompare(t *testing.T) {
	h := newHarness(t, panel.Marzban, panel.PasarGuard, pasarGuardDB(t), 3)
	j := h.created()
	release := h.hold()
	defer release()
	if err := h.svc.Compare(j.ID); err != nil {
		t.Fatal(err)
	}
	h.svc.Shutdown()
	got := trail(t, h, j.ID)
	if !slices.Equal(got, []string{"read stage start", "read log cancelled", "read stage failed"}) || j.State() != StateCreated {
		t.Fatalf("events %q", got)
	}
}

// A failed compare's log event carries the code the API would answer
// with, and what the operator needs to act on it.
func TestProblemOfAFailedCompare(t *testing.T) {
	cert := &source.Certificate{SHA256: "ab12", Subject: "CN=old.example", NotAfter: time.Date(2027, 1, 2, 0, 0, 0, 0, time.UTC)}
	for _, c := range []struct {
		err  error
		want map[string]any
	}{
		{&source.Error{Code: source.CodeTLSUntrusted, Cert: cert}, map[string]any{"code": "source_tls_untrusted", "sha256": "ab12",
			"subject": "CN=old.example", "notAfter": cert.NotAfter}},
		{&source.Error{Code: source.CodeUnauthorized, Status: 401}, map[string]any{"code": "source_unauthorized"}},
		{fmt.Errorf("open: %w", dest.ErrNotReady), map[string]any{"code": "panel_not_ready"}},
		{&engine.PlanError{Where: "category", Problem: "a category has no id"}, map[string]any{"code": "plan_invalid",
			"where": "category", "problem": "a category has no id"}},
		{&source.Error{Code: source.CodeUnreachable, Err: context.Canceled}, map[string]any{"code": "cancelled"}},
		{errors.New("disk full"), map[string]any{"code": "internal"}},
	} {
		if got := problemOf(c.err); fmt.Sprint(got) != fmt.Sprint(c.want) {
			t.Errorf("%v: %v", c.err, got)
		}
	}
}
