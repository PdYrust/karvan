package bridge

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/PdYrust/karvan/internal/migration/ledger"
	"github.com/PdYrust/karvan/internal/panel"
	"github.com/PdYrust/karvan/internal/panel/fakepanel"
	"github.com/PdYrust/karvan/internal/panel/pasarguardapi"
	"github.com/PdYrust/karvan/internal/panel/source"
	"github.com/PdYrust/karvan/internal/panel/subtoken"
)

// A PasarGuard old panel: v3 tokens name the user by id, and the oldest
// links name them by username; both reach the same ledger row.
func TestPasarGuardOldPanel(t *testing.T) {
	e := newEnv(t)
	ctx := context.Background()
	old := fakepanel.NewPasarGuard(fakepanel.PasarGuardAdmin{ID: 7, Username: "shop", Password: "pw"},
		[]map[string]any{fakepanel.PasarGuardUser(42, "sara", nil)}, fakepanel.Faults{})
	t.Cleanup(old.Close)
	e.addDomain("pg.example.com")
	site, err := e.b.st.Save(ctx, Site{Name: "pg", PanelID: 1, SourceKind: panel.PasarGuard, SourceURL: old.URL,
		SourceIP: "127.0.0.1", OldPath: "sub", Domains: []string{"pg.example.com"}, Port: e.port, Lazy: true, PlainHTTPOK: true})
	if err != nil {
		t.Fatal(err)
	}
	e.b.mu.Lock()
	e.b.load(append(e.b.sites, site), nil)
	e.b.mu.Unlock()

	v3 := subtoken.PasarGuard(42, oldSecret, e.clock.now())
	old.Subscribe(v3, "sara")
	c, _ := source.New(source.Options{BaseURL: old.URL, Attempts: 1})
	u, err := pasarguardapi.SubscriptionUser(ctx, c, "sub", v3)
	if err != nil {
		t.Fatal(err)
	}
	row := panel.Row{ID: 300, Username: "sara", CreatedAt: u.CreatedAt}
	if _, err := ledger.New(e.b.DB).Add(ctx, []ledger.Entry{{Kind: ledger.KindUser, JobID: "job", State: ledger.Done,
		SourceOrigin: site.origin, SourceKey: strconv.FormatInt(u.SourceID, 10), SourceCreated: u.CreatedAt,
		SourceUsername: "sara", CredPrint: ledger.Fingerprint(u.Credentials), Dest: "fake-dest", Row: row}}); err != nil {
		t.Fatal(err)
	}
	e.dest.set(300, func(r *Row) { r.Username, r.CreatedAt = row.Username, row.CreatedAt })

	legacy := subtoken.Marzban("sara", oldSecret, e.clock.now().Add(time.Minute))
	old.Subscribe(legacy, "sara")
	for _, tok := range []string{v3, legacy} {
		code, body := e.getHost("pg.example.com", "/sub/"+tok)
		if code != http.StatusOK || body != "config /sub/dest-300-sara" {
			t.Fatalf("%s: %d %q", tok, code, body)
		}
	}
	// A token for an id Karvan did not move is refused.
	other := subtoken.PasarGuard(43, oldSecret, e.clock.now())
	old.Subscribe(other, "sara")
	if code, _ := e.getHost("pg.example.com", "/sub/"+other); code != http.StatusNotFound {
		t.Fatalf("a token naming id 43: %d", code)
	}
}

// legacyV3 mints the older PasarGuard link shape: b64("v3,id,ts") and 10
// hex characters of sha256, as the old links in the field look.
func legacyV3(id int64, at time.Time) string {
	data := base64.RawURLEncoding.EncodeToString([]byte(fmt.Sprintf("v3,%d,%d", id, at.Unix())))
	sum := sha256.Sum256([]byte(data + oldSecret))
	return data + hex.EncodeToString(sum[:])[:10]
}

// PasarGuard 5.5 leaves proxy_settings out of /info, and mints a new
// token on every read of the link. Every token the old panel knows for
// the moved user is served, each checked once; the ledger's fingerprint
// came from the admin API, which still has the credentials.
func TestPasarGuard55InfoWithoutCredentials(t *testing.T) {
	e, old := pasarGuard55Env(t)
	calls := old.SubCalls()
	first := legacyV3(42, e.clock.now().Add(-30*24*time.Hour))
	second := subtoken.PasarGuard(42, oldSecret, e.clock.now().Add(-time.Hour))
	for _, tok := range []string{first, second} {
		old.Subscribe(tok, "Owner")
		for range 2 {
			if code, body := e.getHost("pg.example.com", "/sub/"+tok); code != http.StatusOK || body != "config /sub/dest-300-Owner" {
				t.Fatalf("%s: %d %q", tok, code, body)
			}
		}
	}
	if e.tokens() != 2 || old.SubCalls()-calls != 2 {
		t.Fatalf("tokens %d, old panel asked %d times", e.tokens(), old.SubCalls()-calls)
	}
	// A token whose answer names another user, and the same id and name
	// created again, are still refused.
	e.clock.add(ipWindow)
	for _, tc := range []struct {
		name string
		at   string
	}{{"other", ""}, {"Owner", "2026-09-01T00:00:00Z"}} {
		if tc.at != "" {
			old.Mutate(func(users *[]map[string]any) { (*users)[0]["created_at"] = tc.at })
		}
		tok := subtoken.PasarGuard(42, oldSecret, e.clock.now().Add(-time.Minute))
		old.Subscribe(tok, tc.name)
		if code, _ := e.getHost("pg.example.com", "/sub/"+tok); code != http.StatusNotFound || e.tokens() != 2 {
			t.Fatalf("%s %s: %d, tokens %d", tc.name, tc.at, code, e.tokens())
		}
	}
}

// pasarGuard55Env is a rule on a PasarGuard 5.5 old panel with user 42,
// Owner, moved to destination row 300 with the fingerprint of the
// credentials the migration read.
func pasarGuard55Env(t *testing.T) (*env, *fakepanel.PasarGuard) {
	t.Helper()
	e := newEnv(t)
	ctx := context.Background()
	user := fakepanel.PasarGuardUser(42, "Owner", map[string]any{"created_at": "2026-03-04T05:06:07Z",
		"edit_at": "2026-10-01T08:00:00Z", "expire": nil, "data_limit_reset_strategy": "day", "group_ids": []int64{1, 5, 7}, "hwid_limit": 0})
	old := fakepanel.NewPasarGuard(fakepanel.PasarGuardAdmin{ID: 7, Username: "shop", Password: "pw"},
		[]map[string]any{user, fakepanel.PasarGuardUser(43, "other", nil)}, fakepanel.Faults{})
	t.Cleanup(old.Close)
	e.addDomain("pg.example.com")
	site, err := e.b.st.Save(ctx, Site{Name: "pg", PanelID: 1, SourceKind: panel.PasarGuard, SourceURL: old.URL,
		SourceIP: "127.0.0.1", OldPath: "sub", Domains: []string{"pg.example.com"}, Port: e.port, Lazy: true, PlainHTTPOK: true})
	if err != nil {
		t.Fatal(err)
	}
	e.b.mu.Lock()
	e.b.load(append(e.b.sites, site), nil)
	e.b.mu.Unlock()
	// The migration's read, with the credentials.
	setup := subtoken.PasarGuard(42, oldSecret, e.clock.now())
	old.Subscribe(setup, "Owner")
	c, _ := source.New(source.Options{BaseURL: old.URL, Attempts: 1})
	u, err := pasarguardapi.SubscriptionUser(ctx, c, "sub", setup)
	if err != nil || len(u.Credentials.Keys()) == 0 {
		t.Fatal(err, u.Credentials.Keys())
	}
	row := panel.Row{ID: 300, Username: "Owner", CreatedAt: u.CreatedAt}
	if _, err := ledger.New(e.b.DB).Add(ctx, []ledger.Entry{{Kind: ledger.KindUser, JobID: "job", State: ledger.Done,
		SourceOrigin: site.origin, SourceKey: "42", SourceCreated: u.CreatedAt, SourceUsername: "Owner",
		CredPrint: ledger.Fingerprint(u.Credentials), Dest: "fake-dest", Row: row}}); err != nil {
		t.Fatal(err)
	}
	e.dest.set(300, func(r *Row) { r.Username, r.CreatedAt = row.Username, row.CreatedAt })
	old.AnswerSubAs55()
	if u, err := pasarguardapi.SubscriptionUser(ctx, c, "sub", setup); err != nil || len(u.Credentials.Keys()) != 0 {
		t.Fatal("the fake still sends credentials", err)
	}
	return e, old
}

// Credentials the answer carries must match; only PasarGuard's withheld
// ones are skipped.
func TestSameUserCredentials(t *testing.T) {
	at := time.Date(2026, 3, 4, 5, 6, 7, 0, time.UTC)
	creds := panel.Credentials{VLESS: &panel.VLESS{ID: "22222222-0000-4000-8000-000000000042"}}
	e := ledger.Entry{SourceKey: "42", SourceUsername: "Owner", SourceCreated: at, CredPrint: ledger.Fingerprint(creds)}
	other := panel.Credentials{VLESS: &panel.VLESS{ID: "22222222-0000-4000-8000-000000000043"}}
	for _, tc := range []struct {
		kind panel.Kind
		u    panel.User
		want bool
	}{
		{panel.PasarGuard, panel.User{SourceID: 42, Username: "Owner", CreatedAt: at.Add(400 * time.Nanosecond)}, true},
		{panel.PasarGuard, panel.User{SourceID: 42, Username: "Owner", CreatedAt: at, Credentials: creds}, true},
		{panel.PasarGuard, panel.User{SourceID: 42, Username: "Owner", CreatedAt: at, Credentials: other}, false},
		{panel.PasarGuard, panel.User{SourceID: 42, Username: "Owner", CreatedAt: at.Add(time.Microsecond)}, false},
		{panel.PasarGuard, panel.User{SourceID: 43, Username: "Owner", CreatedAt: at}, false},
		{panel.Marzban, panel.User{Username: "Owner", CreatedAt: at}, false},
	} {
		if got := sameUser(tc.kind, tc.u, e); got != tc.want {
			t.Errorf("%s %+v: %t", tc.kind, tc.u, got)
		}
	}
}
