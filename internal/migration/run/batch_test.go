package run

import (
	"context"
	"errors"
	"net/url"
	"strconv"
	"strings"
	"testing"

	"github.com/PdYrust/karvan/internal/panel"
	"github.com/PdYrust/karvan/internal/panel/marzbandb"
	"github.com/PdYrust/karvan/internal/panel/paneldb/paneldbtest"
	"github.com/PdYrust/karvan/internal/panel/pasarguarddb"
)

// batchRow is what a test reads of a job's batch.
type batchRow struct {
	id                    int64
	name, path, hosts, ip string
	port                  int
	members               int
}

func (h *harness) batchOf(jobID string) (batchRow, bool) {
	h.t.Helper()
	var b batchRow
	err := h.svc.db.QueryRow(`SELECT b.id, b.name, b.old_path, b.old_hosts, b.source_ip, b.old_port,
		(SELECT count(*) FROM ledger l WHERE l.batch_id = b.id) FROM batches b WHERE b.job_id = ?`, jobID).
		Scan(&b.id, &b.name, &b.path, &b.hosts, &b.ip, &b.port, &b.members)
	if err != nil {
		return b, false
	}
	return b, true
}

// A run that ends done becomes a batch of its users, with the old link
// facts read from the old panel's subscription links and the address
// Karvan reached it at (D5, D7).
func TestDoneRunMakesItsBatch(t *testing.T) {
	tg := paneldbtest.Load(t, paneldbtest.Fixture{Flavor: "sqlite", Path: "../../panel/pasarguarddb/testdata/" + pasarguarddb.Revision + ".sqlite.sql"})
	h := newHarness(t, panel.Marzban, panel.PasarGuard, tg, 12)
	rep := h.run(h.job(nil), false)
	if rep.State != StateDone {
		t.Fatalf("state %s error %q", rep.State, rep.Error)
	}
	b, ok := h.batchOf(rep.JobID)
	u, _ := url.Parse(h.src.URL)
	port, _ := strconv.Atoi(u.Port())
	want := batchRow{id: b.id, name: "reseller", path: "sub", hosts: `["127.0.0.1"]`, ip: "127.0.0.1", port: port, members: 12}
	if !ok || b != want {
		t.Fatalf("batch %+v (found %v), want %+v", b, ok, want)
	}
}

// A done job whose batch was not stored (Karvan stopped between the job's
// save and the batch) gets it at the next start (D5).
func TestRecoverMakesAMissingBatch(t *testing.T) {
	tg := paneldbtest.Load(t, paneldbtest.Fixture{Flavor: "sqlite", Path: "../../panel/marzbandb/testdata/" + marzbandb.Revision + ".sqlite.sql"})
	h := newHarness(t, panel.PasarGuard, panel.Marzban, tg, 6)
	rep := h.run(h.job(nil), false)
	ctx := context.Background()
	for _, q := range []string{"UPDATE ledger SET batch_id = NULL", "DELETE FROM batches"} {
		if _, err := h.svc.db.ExecContext(ctx, q); err != nil {
			t.Fatal(err)
		}
	}
	if err := New(h.svc.Deps).Recover(ctx); err != nil {
		t.Fatal(err)
	}
	if b, ok := h.batchOf(rep.JobID); !ok || b.members != 6 || b.name != "shop" || b.path != "" || b.hosts != "[]" {
		t.Fatalf("batch %+v found %v", b, ok)
	}
}

// A batch user's new link is built on request from the destination panel,
// the same link the report hands out (D57), and only while the panel
// still points at the database the batch moved it to.
func TestUserLinkOfABatchUser(t *testing.T) {
	tg := paneldbtest.Load(t, paneldbtest.Fixture{Flavor: "sqlite", Path: "../../panel/pasarguarddb/testdata/" + pasarguarddb.Revision + ".sqlite.sql"})
	h := newHarness(t, panel.Marzban, panel.PasarGuard, tg, 2)
	rep := h.run(h.job(nil), false)
	b, ok := h.batchOf(rep.JobID)
	if !ok {
		t.Fatal("no batch")
	}
	ctx := context.Background()
	var ledgerID int64
	if err := h.svc.db.QueryRow("SELECT min(id) FROM ledger WHERE batch_id = ?", b.id).Scan(&ledgerID); err != nil {
		t.Fatal(err)
	}
	m, err := h.svc.Batches().Member(ctx, b.id, ledgerID)
	if err != nil {
		t.Fatal(err)
	}
	link, err := h.svc.UserLink(ctx, m)
	if err != nil {
		t.Fatal(err)
	}
	full, err := h.svc.WithLinks(ctx, rep.JobID)
	if err != nil {
		t.Fatal(err)
	}
	want := ""
	for _, u := range full.Users {
		if u.DestID == m.DestID {
			want = u.NewLink
		}
	}
	if link == "" || want == "" || link[:strings.LastIndex(link, "/")] != want[:strings.LastIndex(want, "/")] {
		t.Fatalf("link %q, report link %q", link, want)
	}
	m.Dest = "sqlite:///elsewhere.db"
	if _, err := h.svc.UserLink(ctx, m); !errors.Is(err, ErrPanelChanged) {
		t.Fatalf("another database: %v", err)
	}
}
