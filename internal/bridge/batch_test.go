package bridge

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/PdYrust/karvan/internal/migration/ledger"
	"github.com/PdYrust/karvan/internal/panel"
)

// batch puts users in a new batch of the env's old panel and destination.
func (e *env) batch(state, reason string, users ...string) int64 {
	e.t.Helper()
	ctx := context.Background()
	var n int
	if err := e.b.DB.QueryRowContext(ctx, "SELECT count(*) FROM jobs").Scan(&n); err != nil {
		e.t.Fatal(err)
	}
	job := fmt.Sprintf("job-%d", n+1)
	if _, err := e.b.DB.ExecContext(ctx, "INSERT INTO jobs (id, state, source_kind, source_origin, panel_id, safe_mode, "+
		"created_at, updated_at) VALUES (?, 'done', 'marzban', ?, 1, 1, 1, 1)", job, e.site.origin); err != nil {
		e.t.Fatal(err)
	}
	res, err := e.b.DB.ExecContext(ctx, "INSERT INTO batches (job_id, panel_id, dest, source_kind, source_origin, name, state, "+
		"inactive_reason, created_at, updated_at) VALUES (?, 1, 'fake-dest', 'marzban', ?, ?, ?, ?, 1, 1)",
		job, e.site.origin, job, state, reason)
	if err != nil {
		e.t.Fatal(err)
	}
	id, _ := res.LastInsertId()
	for _, u := range users {
		if _, err := e.b.DB.ExecContext(ctx, "UPDATE ledger SET batch_id = ? WHERE id = ?", id, e.users[u].ID); err != nil {
			e.t.Fatal(err)
		}
	}
	return id
}

func (e *env) setBatch(id int64, state, reason string) {
	e.t.Helper()
	if _, err := e.b.DB.Exec("UPDATE batches SET state = ?, inactive_reason = ? WHERE id = ?", state, reason, id); err != nil {
		e.t.Fatal(err)
	}
}

// batchRule adds a rule made from batches on port, for the env's old panel.
func (e *env) batchRule(port int, ids ...int64) Site {
	e.t.Helper()
	ctx := context.Background()
	s := e.site
	s.ID, s.Name, s.Port = 0, fmt.Sprintf("batches %d", port), port
	saved, err := e.b.st.Save(ctx, s)
	if err != nil {
		e.t.Fatal(err)
	}
	if _, err := e.b.DB.ExecContext(ctx, "UPDATE legacy_sites SET mode = 'batches' WHERE id = ?", saved.ID); err != nil {
		e.t.Fatal(err)
	}
	for _, id := range ids {
		if _, err := e.b.DB.ExecContext(ctx, "INSERT INTO legacy_site_batches (site_id, batch_id) VALUES (?, ?)", saved.ID, id); err != nil {
			e.t.Fatal(err)
		}
	}
	sites, err := e.b.st.Sites(ctx)
	if err != nil {
		e.t.Fatal(err)
	}
	e.b.mu.Lock()
	e.b.load(sites, nil)
	e.b.mu.Unlock()
	saved, _ = e.b.st.Site(ctx, saved.ID)
	return saved
}

func (e *env) hits() int {
	e.dest.mu.Lock()
	defer e.dest.mu.Unlock()
	return e.dest.hits
}

func (e *env) dropRow(id int64) {
	e.dest.mu.Lock()
	delete(e.dest.rows, id)
	e.dest.mu.Unlock()
}

func (e *env) isClosed(name string) bool {
	e.t.Helper()
	closed, err := e.b.st.closed(context.Background(), e.users[name].ID)
	if err != nil {
		e.t.Fatal(err)
	}
	return closed
}

var held = []struct{ state, reason string }{
	{"inactive", "restoring"}, {"inactive", "rolled_back"}, {"inactive", "restore_failed"}, {"deleted", ""},
}

// A verified token whose row is in an inactive or deleted batch gets
// the 404 at once: the panel is not asked, nothing is dropped or closed,
// even with the user gone from the destination (a restore under way).
func TestHeldBatchStopsAVerifiedToken(t *testing.T) {
	for _, h := range held {
		t.Run(h.state+" "+h.reason, func(t *testing.T) {
			e := newEnv(t, "ali")
			id := e.batch("active", "", "ali")
			tok := e.oldToken("ali", e.clock.now())
			if code, _ := e.get("/sub/"+tok, "198.51.100.7"); code != http.StatusOK {
				t.Fatal(code)
			}
			e.setBatch(id, h.state, h.reason)
			e.dropRow(100)
			hits, calls := e.hits(), e.oldCalls()
			if code, _ := e.get("/sub/"+tok, "198.51.100.7"); code != http.StatusNotFound {
				t.Fatalf("held: %d", code)
			}
			if e.hits() != hits || e.oldCalls() != calls || e.tokens() != 1 || e.isClosed("ali") {
				t.Fatalf("panel hits %d -> %d, old calls %d -> %d, tokens %d, closed %v",
					hits, e.hits(), calls, e.oldCalls(), e.tokens(), e.isClosed("ali"))
			}
		})
	}
}

// A first old link of a user in a held batch gets no pass-through
// and no lazy check, on a manual rule and on a rule of batches.
func TestHeldBatchStopsALazyCheck(t *testing.T) {
	for _, h := range held {
		t.Run(h.state+" "+h.reason, func(t *testing.T) {
			e := newEnv(t, "ali", "bob")
			gone := e.batch(h.state, h.reason, "ali")
			live := e.batch("active", "", "bob")
			e.batchRule(2, gone, live)
			e.dropRow(100)
			for _, port := range []int{1, 2} {
				e.port = port
				hits, calls := e.hits(), e.oldCalls()
				if code, _ := e.get("/sub/"+e.oldToken("ali", e.clock.now()), "198.51.100.7"); code != http.StatusNotFound {
					t.Fatalf("port %d: %d", port, code)
				}
				if e.hits() != hits || e.oldCalls() != calls || e.isClosed("ali") {
					t.Fatalf("port %d: panel hits %d -> %d, old calls %d -> %d, closed %v", port, hits, e.hits(), calls, e.oldCalls(), e.isClosed("ali"))
				}
				if code, _ := e.get("/sub/"+e.oldToken("bob", e.clock.now()), "198.51.100.8"); code != http.StatusOK {
					t.Fatalf("port %d, an active batch: %d", port, code)
				}
			}
		})
	}
}

// A rule whose batches are all held answers 404 to every request,
// tokens of the new panel itself included, and never asks the panel.
func TestRuleOfHeldBatchesHasNoPassThrough(t *testing.T) {
	e := newEnv(t, "ali")
	id := e.batch("active", "", "ali")
	e.batchRule(2, id)
	e.port = 2
	if code, _ := e.get("/sub/native-token-0123456789", "198.51.100.7"); code != http.StatusOK {
		t.Fatalf("active: %d", code)
	}
	for _, h := range held {
		e.setBatch(id, h.state, h.reason)
		hits := e.hits()
		if code, _ := e.get("/sub/native-token-0123456789", "198.51.100.7"); code != http.StatusNotFound || e.hits() != hits {
			t.Fatalf("%s %s: %d, panel hits %d -> %d", h.state, h.reason, code, hits, e.hits())
		}
	}
}

// The upload checks nothing for users in a held batch: no old panel call,
// and a token stored before is not counted as known.
func TestUploadSkipsHeldBatches(t *testing.T) {
	e := newEnv(t, "ali", "bob", "cyrus")
	gone := e.batch("active", "", "ali", "cyrus")
	ali := e.oldToken("ali", e.clock.now())
	if code, _ := e.get("/sub/"+ali, "198.51.100.7"); code != http.StatusOK {
		t.Fatal(code)
	}
	e.setBatch(gone, "inactive", "restoring")
	calls := e.oldCalls()
	list := ali + "\n" + e.oldToken("cyrus", e.clock.now()) + "\n" + e.oldToken("bob", e.clock.now())
	res, err := e.b.Upload(context.Background(), e.site.ID, strings.NewReader(list))
	if err != nil {
		t.Fatal(err)
	}
	if res.Found != 3 || res.Stored != 1 || res.NoUser != 2 || res.Known != 0 || e.oldCalls() != calls+1 {
		t.Fatalf("result %+v, old calls %d -> %d", res, calls, e.oldCalls())
	}
}

// The sweep skips users in a held batch: a restore removes them from the
// destination, and their links must not close for good.
func TestSweepSkipsHeldBatches(t *testing.T) {
	e := newEnv(t, "ali", "bob")
	gone := e.batch("inactive", "restoring", "ali")
	e.batch("active", "", "bob")
	e.dropRow(100)
	e.dropRow(101)
	n, err := e.b.Sweep(context.Background())
	if err != nil || n != 1 || e.isClosed("ali") || !e.isClosed("bob") {
		t.Fatalf("closed %d, %v; ali %v, bob %v", n, err, e.isClosed("ali"), e.isClosed("bob"))
	}
	e.setBatch(gone, "active", "")
	if n, _ := e.b.Sweep(context.Background()); n != 1 || !e.isClosed("ali") {
		t.Fatalf("made active again: closed %d", n)
	}
}

// The daily re-check leaves a token alone once its batch is held, even
// when the old panel no longer knows it.
func TestRecheckSkipsHeldBatches(t *testing.T) {
	e := newEnv(t, "ali")
	id := e.batch("active", "", "ali")
	tok := e.oldToken("ali", e.clock.now())
	if code, _ := e.get("/sub/"+tok, "198.51.100.7"); code != http.StatusOK {
		t.Fatal(code)
	}
	e.old.Subscribe(tok, "") // revoked on the old panel
	q := request{site: e.site, dest: e.dest, token: tok, mac: mac(e.b.macKey, tok)}
	e.setBatch(id, "inactive", "restoring")
	e.b.recheck(q, e.users["ali"])
	e.b.Wait()
	if e.tokens() != 1 {
		t.Fatal("a held token was dropped")
	}
	e.setBatch(id, "active", "")
	e.b.recheck(q, e.users["ali"])
	e.b.Wait()
	if e.tokens() != 0 {
		t.Fatal("an active token revoked on the old panel was kept")
	}
}

// A token proven on one rule counts on another only for a row in
// that rule's scope; out of scope, no lazy check finds the user either.
func TestProofOutsideTheRuleScopeDoesNotCount(t *testing.T) {
	e := newEnv(t, "ali", "bob")
	e.batch("active", "", "ali")
	other := e.batch("active", "", "bob")
	e.batchRule(2, other)
	tok := e.oldToken("ali", e.clock.now())
	if code, _ := e.get("/sub/"+tok, "198.51.100.7"); code != http.StatusOK {
		t.Fatal(code)
	}
	calls := e.oldCalls()
	e.port = 2
	if code, _ := e.get("/sub/"+tok, "198.51.100.7"); code != http.StatusNotFound || e.oldCalls() != calls || e.tokens() != 1 {
		t.Fatalf("outside the scope: %d, old calls %d -> %d, tokens %d", code, calls, e.oldCalls(), e.tokens())
	}
	if code, _ := e.get("/sub/"+e.oldToken("bob", e.clock.now()), "198.51.100.8"); code != http.StatusOK {
		t.Fatalf("in the scope: %d", code)
	}
}

// Of two rows with the same source key, the one the old panel's
// answer matches is served, even when it is not the newest.
func TestLazyCheckPicksTheMatchingRow(t *testing.T) {
	e := newEnv(t, "ali")
	moved := e.users["ali"]
	stale := moved
	stale.SourceCreated, stale.Row = moved.SourceCreated.Add(-time.Hour), panel.Row{ID: 200, Username: "ali_2", CreatedAt: moved.CreatedAt}
	if _, err := ledger.New(e.b.DB).Add(context.Background(), []ledger.Entry{stale}); err != nil {
		t.Fatal(err)
	}
	e.dest.set(200, func(r *Row) { r.Username, r.CreatedAt = stale.Username, stale.CreatedAt })
	code, body := e.get("/sub/"+e.oldToken("ali", e.clock.now()), "198.51.100.7")
	if code != http.StatusOK || body != "config /sub/dest-100-new_ali" {
		t.Fatalf("%d %q", code, body)
	}
}

// Two rows that both match the old panel's answer are a data error:
// 404, nothing stored, and a warning names the rows.
func TestLazyCheckRefusesTwoMatchingRows(t *testing.T) {
	e := newEnv(t, "ali")
	var logs bytes.Buffer
	e.b.Log = slog.New(slog.NewTextHandler(&logs, nil))
	twin := e.users["ali"]
	twin.SourceKey, twin.Row = "ali-copy", panel.Row{ID: 200, Username: "ali_2", CreatedAt: twin.CreatedAt}
	if _, err := ledger.New(e.b.DB).Add(context.Background(), []ledger.Entry{twin}); err != nil {
		t.Fatal(err)
	}
	e.dest.set(200, func(r *Row) { r.Username, r.CreatedAt = twin.Username, twin.CreatedAt })
	if code, _ := e.get("/sub/"+e.oldToken("ali", e.clock.now()), "198.51.100.7"); code != http.StatusNotFound || e.tokens() != 0 {
		t.Fatalf("%d, tokens %d", code, e.tokens())
	}
	if !strings.Contains(logs.String(), "site=old") || !strings.Contains(logs.String(), "ledger=") {
		t.Fatalf("warning: %q", logs.String())
	}
}

// The 404 for a held batch has the same bytes and headers as the one for
// a path no rule serves and for a token no one knows.
func TestHeld404LooksLikeAnyOther404(t *testing.T) {
	e := newEnv(t, "ali")
	id := e.batch("active", "", "ali")
	tok := e.oldToken("ali", e.clock.now())
	if code, _ := e.get("/sub/"+tok, "198.51.100.7"); code != http.StatusOK {
		t.Fatal(code)
	}
	e.setBatch(id, "inactive", "rolled_back")
	var got []*httptest.ResponseRecorder
	for i, path := range []string{"/nothing/here", "/sub/unknown-token-0123456789", "/sub/" + tok} {
		r := httptest.NewRequest(http.MethodGet, "http://old.example.com"+path, nil)
		r.RemoteAddr = fmt.Sprintf("198.51.100.%d:40000", 20+i)
		w := httptest.NewRecorder()
		e.b.Handler(e.port).ServeHTTP(w, r)
		got = append(got, w)
	}
	for i, w := range got {
		if w.Code != http.StatusNotFound || !reflect.DeepEqual(w.Header(), got[0].Header()) || !bytes.Equal(w.Body.Bytes(), got[0].Body.Bytes()) {
			t.Fatalf("answer %d: %d %v %q, want %d %v %q", i, w.Code, w.Header(), w.Body, got[0].Code, got[0].Header(), got[0].Body)
		}
	}
}

// A rule's numbers count its scope: a rule of batches its batches, a
// manual rule every user of its old panel not in a held batch.
func TestStatsCountTheScope(t *testing.T) {
	e := newEnv(t, "ali", "bob", "cyrus")
	gone := e.batch("inactive", "restoring", "ali")
	live := e.batch("active", "", "bob")
	rule := e.batchRule(2, gone, live)
	ctx := context.Background()
	if v := e.b.View(ctx, e.site); v.Stats == nil || v.Stats.Moved != 2 {
		t.Fatalf("manual rule: %+v", v.Stats)
	}
	if v := e.b.View(ctx, rule); v.Stats == nil || v.Stats.Moved != 1 {
		t.Fatalf("rule of batches: %+v", v.Stats)
	}
}

// A close decided just before a restore held the batch does not land:
// the batch state and the close are read and written in one statement.
func TestCloseSkipsAHeldBatch(t *testing.T) {
	e := newEnv(t, "ali")
	id := e.batch("active", "", "ali")
	if code, _ := e.get("/sub/"+e.oldToken("ali", e.clock.now()), "198.51.100.7"); code != http.StatusOK {
		t.Fatal(code)
	}
	e.setBatch(id, "inactive", "restoring")
	e.b.apply(context.Background(), e.users["ali"], verdict{close: ClosedDeleted})
	if e.isClosed("ali") || e.tokens() != 1 {
		t.Fatalf("closed %v, tokens %d", e.isClosed("ali"), e.tokens())
	}
}
