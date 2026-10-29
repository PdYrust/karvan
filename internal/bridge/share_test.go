package bridge

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"slices"
	"strconv"
	"testing"
)

const sharedBase = "hiddenPath1234"

// panelHandler answers every request it gets with "panel".
var panelHandler = http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = io.WriteString(w, "panel") })

// answerOn is what the bridge answers on port: status, content type and body.
func answerOn(b *Bridge, port int, method, url string) string {
	r := httptest.NewRequest(method, url, nil) // an https URL sets r.TLS
	w := httptest.NewRecorder()
	b.Handler(port).ServeHTTP(w, r)
	return strconv.Itoa(w.Code) + " " + w.Header().Get("Content-Type") + " " + w.Body.String()
}

// D12: the panel domain under the hidden path reaches the panel; every
// other host, path, scheme and port gets the bridge's own answer, the
// plain 404 bytes for a wrong hidden path included.
func TestSharedPortHandsThePanelItsPaths(t *testing.T) {
	e, a, _ := newRuleCertEnv(t)
	ctx := context.Background()
	port := freePort(t)
	if _, err := e.b.SaveSite(ctx, e.rule("tls", port, nil, "a.example.com")); err != nil {
		t.Fatal(err)
	}
	notFound := answerOn(e.b, port, http.MethodGet, "https://a.example.com/"+sharedBase+"/")
	notAllowed := answerOn(e.b, port, http.MethodPost, "https://a.example.com/"+sharedBase+"/api/v1/session")
	e.b.SharePanel(ctx, []PanelSite{{Port: port, DomainID: a.ID, BasePath: sharedBase, Handler: panelHandler}})
	panel := "200 text/plain; charset=utf-8 panel"
	for _, c := range []struct{ method, url, want string }{
		{http.MethodGet, "https://a.example.com/" + sharedBase + "/", panel},
		{http.MethodGet, "https://A.Example.com:" + strconv.Itoa(port) + "/" + sharedBase, panel},
		{http.MethodPost, "https://a.example.com/" + sharedBase + "/api/v1/session", panel},
		{http.MethodGet, "https://b.example.com/" + sharedBase + "/", notFound},
		{http.MethodGet, "https://a.example.com/hiddenPath1235/", notFound},
		{http.MethodGet, "https://a.example.com/" + sharedBase + "5/", notFound},
		{http.MethodGet, "https://a.example.com/", notFound},
		{http.MethodGet, "http://a.example.com/" + sharedBase + "/", notFound},
		{http.MethodPost, "https://a.example.com/tls/x", notAllowed},
	} {
		if got := answerOn(e.b, port, c.method, c.url); got != c.want {
			t.Errorf("%s %s = %q, want %q", c.method, c.url, got, c.want)
		}
	}
	if got := answerOn(e.b, port+1, http.MethodGet, "https://a.example.com/"+sharedBase+"/"); got != notFound {
		t.Errorf("another port = %q", got)
	}
	e.b.SharePanel(ctx, nil)
	if got := answerOn(e.b, port, http.MethodGet, "https://a.example.com/"+sharedBase+"/"); got != notFound {
		t.Errorf("after the panel left = %q", got)
	}
}

// The panel on a shared port changes no handshake of any rule's port and
// name (D12, as P41-32).
func TestSharedPortKeepsHandshakes(t *testing.T) {
	m := newMoveCase(t)
	b := m.e.restart(slog.New(slog.DiscardHandler))
	shakes := m.handshakes(t)
	list, err := b.Domains(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	i := slices.IndexFunc(list, func(d Domain) bool { return d.Name == "conf.example.com" })
	if i < 0 {
		t.Fatalf("no conf.example.com in %v", list)
	}
	b.SharePanel(context.Background(), []PanelSite{{Port: m.a, DomainID: list[i].ID, BasePath: sharedBase, Handler: panelHandler},
		{Port: m.b, DomainID: list[i].ID, BasePath: sharedBase, Handler: panelHandler}})
	if got := m.handshakes(t); !slices.Equal(got, shakes) {
		t.Errorf("handshakes changed:\n%v\nwant\n%v", got, shakes)
	}
}

// Deleting the rule the panel shares a port with keeps the port and the
// domain's certificate, so the operator is not locked out.
func TestSharedPortOutlivesItsRule(t *testing.T) {
	e, a, _ := newRuleCertEnv(t)
	ctx := context.Background()
	port := freePort(t)
	s, err := e.b.SaveSite(ctx, e.rule("tls", port, nil, "a.example.com"))
	if err != nil {
		t.Fatal(err)
	}
	before := handshake(t, port, "a.example.com")
	e.b.SharePanel(ctx, []PanelSite{{Port: port, DomainID: a.ID, BasePath: sharedBase, Handler: panelHandler}})
	if err := e.b.DeleteSite(ctx, s.ID); err != nil {
		t.Fatal(err)
	}
	if got := handshake(t, port, "a.example.com"); before == "error" || got != before {
		t.Fatalf("handshake %s, want %s", got, before)
	}
	if got := handshake(t, port, "b.example.com"); got != "error" {
		t.Errorf("another name got a certificate: %s", got)
	}
	if got := answerOn(e.b, port, http.MethodGet, "https://a.example.com/"+sharedBase+"/"); got != "200 text/plain; charset=utf-8 panel" {
		t.Errorf("panel after the rule went = %q", got)
	}
}
