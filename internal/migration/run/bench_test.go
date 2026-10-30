//go:build benchtest

package run

import (
	"context"
	"fmt"
	"runtime"
	"syscall"
	"testing"
	"time"

	"github.com/PdYrust/karvan/internal/panel"
	"github.com/PdYrust/karvan/internal/panel/marzbandb"
	"github.com/PdYrust/karvan/internal/panel/paneldb"
	"github.com/PdYrust/karvan/internal/panel/paneldb/paneldbtest"
)

// The engine benchmark of D37: a Marzban reseller panel with benchAdmins
// admins owning benchPerAdmin users each, moved to a Marzban on SQLite.
const (
	benchAdmins   = 2000
	benchPerAdmin = 20
)

// TestBenchEngine2000Admins times check, resolve and execute, with the
// allocations and the peak RSS of the whole test process. Run it with
// make bench; the admin passwords use the real bcrypt cost.
func TestBenchEngine2000Admins(t *testing.T) {
	paneldbtest.Each(t, "../../panel/marzbandb/testdata", marzbandb.Revision, func(t *testing.T, tg paneldb.Target) {
		if tg.Dialect != paneldb.SQLite {
			return // the recorded numbers are on SQLite
		}
		useHashCost(t, startHashCost)
		h := newHarness(t, panel.Marzban, panel.Marzban, tg, benchAdmins*benchPerAdmin)
		h.svc.ChunkSize = func(paneldb.Dialect) int { return 50 } // the default on SQLite
		h.src.Mutate(func(users *[]map[string]any) {
			for i := range *users {
				a := i % benchAdmins
				(*users)[i]["admin"] = map[string]any{"id": a + 10, "username": fmt.Sprintf("admin%04d", a), "is_sudo": false}
			}
		})
		ctx := context.Background()
		m := measure(t)
		j := h.created()
		if _, err := h.svc.Check(ctx, j.ID); err != nil {
			t.Fatal(err)
		}
		m.step("check")
		_, p, v, _ := h.svc.View(j.ID)
		r, _, err := h.svc.SetPlan(ctx, j.ID, v, p)
		if err != nil || r.Blockers > 0 {
			t.Fatalf("resolve: %v, %d blockers", err, r.Blockers)
		}
		m.step("resolve")
		rep := h.run(j, false)
		m.step("execute")
		if rep.State != StateDone || rep.Counts.Moved != benchAdmins*benchPerAdmin {
			t.Fatalf("report: state %s counts %+v error %q", rep.State, rep.Counts, rep.Error)
		}
		t.Logf("admins %d, users %d, peak RSS %d MiB", benchAdmins, benchAdmins*benchPerAdmin, peakRSS()>>20)
	})
}

// meter logs the time and allocations of each step since the last one.
type meter struct {
	t      *testing.T
	at     time.Time
	allocs uint64
	bytes  uint64
}

func measure(t *testing.T) *meter {
	m := &meter{t: t}
	m.reset()
	return m
}

func (m *meter) reset() {
	var s runtime.MemStats
	runtime.ReadMemStats(&s)
	m.at, m.allocs, m.bytes = time.Now(), s.Mallocs, s.TotalAlloc
}

func (m *meter) step(name string) {
	var s runtime.MemStats
	runtime.ReadMemStats(&s)
	m.t.Logf("%-8s %8.2f s %10d allocs %8d MiB allocated", name, time.Since(m.at).Seconds(), s.Mallocs-m.allocs, (s.TotalAlloc-m.bytes)>>20)
	m.reset()
}

// peakRSS is the most memory the process held, in bytes.
func peakRSS() int64 {
	var u syscall.Rusage
	if err := syscall.Getrusage(syscall.RUSAGE_SELF, &u); err != nil {
		return 0
	}
	return u.Maxrss << 10 // Linux reports KiB
}
