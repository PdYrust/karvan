package run

import (
	"context"
	"errors"
	"slices"
	"testing"

	"golang.org/x/crypto/bcrypt"

	"github.com/PdYrust/karvan/internal/migration/ledger"
	"github.com/PdYrust/karvan/internal/migration/migrationtest"
	"github.com/PdYrust/karvan/internal/migration/plan"
	"github.com/PdYrust/karvan/internal/panel"
	"github.com/PdYrust/karvan/internal/panel/marzbandb"
	"github.com/PdYrust/karvan/internal/panel/paneldb"
	"github.com/PdYrust/karvan/internal/panel/paneldb/paneldbtest"
)

// giveToNorth makes the users from the fourth on belong to the old
// panel's admin "north" (D34).
func giveToNorth(src migrationtest.Source) {
	src.Mutate(func(users *[]map[string]any) {
		for i := 3; i < len(*users); i++ {
			(*users)[i]["admin"] = map[string]any{"id": 8, "username": "north", "is_sudo": false}
		}
	})
}

// movedOnly is rep with only its moved users.
func movedOnly(rep *Report) *Report {
	out := *rep
	out.Users = slices.DeleteFunc(slices.Clone(rep.Users), func(u UserLine) bool { return !u.Moved })
	return &out
}

// owners reads each destination user's admin name.
func (h *harness) owners() map[string]string {
	h.t.Helper()
	return h.query("SELECT u.username, a.username FROM users u JOIN admins a ON a.id = u.admin_id")
}

// query reads two text columns into a map.
func (h *harness) query(q string) map[string]string {
	h.t.Helper()
	ctx := context.Background()
	r, err := paneldb.OpenReader(ctx, h.tg)
	if err != nil {
		h.t.Fatal(err)
	}
	defer r.Close()
	out := map[string]string{}
	err = r.View(ctx, func(q2 paneldb.Querier) error {
		rows, err := q2.QueryContext(ctx, q)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var k, v string
			if err := rows.Scan(&k, &v); err != nil {
				return err
			}
			out[k] = v
		}
		return rows.Err()
	})
	if err != nil {
		h.t.Fatal(err)
	}
	return out
}

// D34: each admin moves with its own users. The login's admin keeps its
// password, the other one gets a generated one for passwords.csv. A
// re-run moves only the user left out before, into the admin its own
// admin moved to; the login's admin, with nothing new, merges as ever.
func TestTwoAdminsMoveAndMoveAgain(t *testing.T) {
	each(t, func(t *testing.T, from, to panel.Kind, tg paneldb.Target) {
		h := newHarness(t, from, to, tg, 6)
		giveToNorth(h.src)
		admins := h.count("admins")
		later := map[panel.Kind]string{panel.Marzban: "user004", panel.PasarGuard: "5"}[from]
		j := h.job(func(p *plan.Plan) { p.Users = map[string]plan.UserDecision{later: {Exclude: true}} })
		rep := h.run(j, true)
		if rep.State != StateDone || rep.Counts.Moved != 5 || rep.Counts.Verified != 5 || h.count("admins") != admins+2 {
			t.Fatalf("report: state %s counts %+v error %q admins %d", rep.State, rep.Counts, rep.Error, h.count("admins"))
		}
		checkTwoAdmins(t, h, j, rep)
		checkLinks(t, movedOnly(rep), to)
		again := h.run(h.job(nil), true)
		checkMovedAgain(t, h, again, admins)
		checkLinks(t, movedOnly(again), to)
	})
}

// checkTwoAdmins checks the first run's admins, owners and passwords.
func checkTwoAdmins(t *testing.T, h *harness, j *Job, rep *Report) {
	t.Helper()
	login := rep.Admin.Username
	if len(rep.Admins) != 2 || rep.Admins[0].Username != login || rep.Admins[0].Password != PasswordKept ||
		rep.Admins[1].Username != "north" || rep.Admins[1].Password != PasswordGenerated || rep.Admins[1].Users != 2 || rep.Admins[1].ID == 0 {
		t.Fatalf("admins %+v", rep.Admins)
	}
	owners := h.owners()
	for u, want := range map[string]string{"user000": login, "user001": login, "user002": login, "user003": "north", "user005": "north"} {
		if owners[u] != want {
			t.Fatalf("%s belongs to %q, want %q", u, owners[u], want)
		}
	}
	pw := j.passwords["north"]
	hashes := h.query("SELECT username, hashed_password FROM admins")
	if pw == "" || len(j.passwords) != 1 || bcrypt.CompareHashAndPassword([]byte(hashes["north"]), []byte(pw)) != nil {
		t.Fatalf("north's password: %d generated", len(j.passwords))
	}
	if cost, err := bcrypt.Cost([]byte(hashes["north"])); err != nil || cost != hashCost {
		t.Fatalf("north's bcrypt cost = %d, %v, want hashCost %d", cost, err, hashCost)
	}
}

// checkMovedAgain checks a re-run that moves user004 into north's admin.
func checkMovedAgain(t *testing.T, h *harness, again *Report, admins int) {
	t.Helper()
	if again.State != StateDone || again.Counts.Moved != 1 || again.Counts.Verified != 1 || len(again.Admins) != 2 ||
		!again.Admins[0].Merged || !again.Admins[1].Merged || again.Admins[1].Users != 1 || again.Admins[1].Password != "" {
		t.Fatalf("re-run: state %s counts %+v admins %+v error %q", again.State, again.Counts, again.Admins, again.Error)
	}
	if h.count("users") != 6 || h.count("admins") != admins+2 || h.owners()["user004"] != "north" {
		t.Fatalf("re-run: users %d admins %d user004 under %q", h.count("users"), h.count("admins"), h.owners()["user004"])
	}
}

// An undo takes back both admins the job created with their users.
func TestTwoAdminsUndo(t *testing.T) {
	each(t, func(t *testing.T, from, to panel.Kind, tg paneldb.Target) {
		h := newHarness(t, from, to, tg, 6)
		giveToNorth(h.src)
		before := paneldbtest.ChecksumExcept(t, tg, "inbounds", "hosts")
		h.fault = func(s string, n int) error {
			if s == "chunk" && n == 2 {
				return errors.New("injected")
			}
			return nil
		}
		rep := h.run(h.job(nil), true)
		if rep.State != StateRolledBack {
			t.Fatalf("state %s error %q left %v", rep.State, rep.Error, rep.Left)
		}
		if after := paneldbtest.ChecksumExcept(t, tg, "inbounds", "hosts"); after != before {
			t.Fatal("the destination is not as it was")
		}
		if rows, _ := h.svc.ledger.Job(context.Background(), rep.JobID, ledger.Done, ledger.Pending); len(rows) != 0 {
			t.Fatalf("ledger still has %d live rows", len(rows))
		}
	})
}

// Marzban destination admins are never sudo, a sudo login and its
// other admins included.
func TestMarzbanAdminsAreNeverSudo(t *testing.T) {
	tg := paneldbtest.Load(t, paneldbtest.Fixture{Flavor: "sqlite", Path: "../../panel/marzbandb/testdata/" + marzbandb.Revision + ".sqlite.sql"})
	h := newHarness(t, panel.Marzban, panel.Marzban, tg, 6)
	h.src.Close()
	h.src = migrationtest.NewSudoMarzban(t, 6)
	t.Cleanup(h.src.Close)
	giveToNorth(h.src)
	rep := h.run(h.job(nil), false)
	if rep.State != StateDone || len(rep.Admins) != 2 {
		t.Fatalf("state %s admins %+v error %q", rep.State, rep.Admins, rep.Error)
	}
	sudo := h.query("SELECT username, CAST(is_sudo AS TEXT) FROM admins")
	for _, a := range rep.Admins {
		if v := sudo[a.Username]; v != "0" && v != "false" {
			t.Fatalf("%s is_sudo %q", a.Username, v)
		}
	}
}
