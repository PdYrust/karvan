package address

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"net/netip"
	"strings"
	"testing"
	"time"
)

// onlyDomain is Allow while only the Karvan domain opens the panel.
func onlyDomain(_ context.Context, name string) bool { return name == "panel.example.com" }

// over is a GET of the hidden path over HTTPS whose handshake asks for sni
// and whose request names host ("" for the IP address).
func (f *fixture) over(sni, host string) string {
	f.t.Helper()
	c := &http.Client{Timeout: 5 * time.Second, Transport: &http.Transport{DisableKeepAlives: true,
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true, MinVersion: tls.VersionTLS12, ServerName: sni}}}
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("https://127.0.0.1:%d/%s/", f.port, oldPath), nil)
	if err != nil {
		f.t.Fatal(err)
	}
	if host != "" {
		req.Host = host
	}
	res, err := c.Do(req)
	if err != nil {
		return "refused"
	}
	defer res.Body.Close()
	b, _ := io.ReadAll(res.Body)
	if res.StatusCode != http.StatusOK {
		return fmt.Sprint(res.StatusCode, " ", strings.Contains(string(b), oldPath))
	}
	return string(b)
}

// While only the domain opens the panel, the domain does; a handshake for
// another name or for none (the IP) is refused; a request whose host is
// not the domain gets the plain 404, over HTTPS and plain HTTP alike.
func TestOnlyTheDomainOpensThePanel(t *testing.T) {
	f := newFixtureWith(t, 0, func(s *Server) { s.Allow = onlyDomain })
	via := fmt.Sprintf("%s via %d/%s", oldPath, f.port, oldPath)
	if got := f.over("panel.example.com", "PANEL.example.com:443"); got != via {
		t.Fatalf("the domain: %s", got)
	}
	for _, sni := range []string{"other.example.com", ""} {
		if got := f.over(sni, ""); got != "refused" {
			t.Errorf("handshake for %q: %s, want it refused", sni, got)
		}
	}
	if got := f.over("panel.example.com", "127.0.0.1"); got != "404 false" {
		t.Errorf("the domain's handshake with the IP as host: %s", got)
	}
	if got, _ := f.fetch("http", f.port, "other.example.com", oldPath); got != "404 " {
		t.Errorf("plain HTTP for another host: %s", got)
	}
	if got, _ := f.fetch("http", f.port, "", oldPath); got != "404 " {
		t.Errorf("plain HTTP for the IP: %s", got)
	}
	if got, _ := f.fetch("http", f.port, "panel.example.com", oldPath); !strings.HasPrefix(got, "307 https://panel.example.com:") {
		t.Errorf("plain HTTP for the domain: %s", got)
	}
}

// This server's own addresses always get the panel, for any name, as
// through an SSH tunnel.
func TestLocalPeersAlwaysGetThePanel(t *testing.T) {
	f := newFixtureWith(t, 0, func(s *Server) {
		s.Allow = func(context.Context, string) bool { return false }
		s.Local = []netip.Prefix{netip.MustParsePrefix("127.0.0.0/8")}
	})
	via := fmt.Sprintf("%s via %d/%s", oldPath, f.port, oldPath)
	for _, name := range []string{"", "localhost", "other.example.com"} {
		if got := f.over(name, name); got != via {
			t.Errorf("from this server, name %q: %s", name, got)
		}
	}
}

// The rule is the panel port's own: the site handed to the subscription
// bridge on a shared port answers as before, the bridge picking the host.
func TestDomainRuleLeavesTheBridgeAlone(t *testing.T) {
	f, current := sharedFixture(t)
	f.srv.Allow = func(context.Context, string) bool { return false }
	if want := fmt.Sprintf("%s via %d/%s ", oldPath, current.Port, oldPath); f.viaBridge(current) != want {
		t.Fatalf("via the bridge: %q", f.viaBridge(current))
	}
}
