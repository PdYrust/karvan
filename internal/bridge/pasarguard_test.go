package bridge

import (
	"context"
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
