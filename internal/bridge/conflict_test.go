package bridge

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"testing"
)

// addRedirectOn writes a redirect rule with its path straight into its table.
func (e *env) addRedirectOn(name, domains string, port int, fromPath, match string, certID any) {
	e.t.Helper()
	_, err := e.b.DB.Exec("INSERT INTO link_redirects (name, domains, listen_port, from_path, match, to_url, certificate_id, "+
		"created_at, updated_at) VALUES (?, ?, ?, ?, ?, 'https://new.example.com/', ?, 1, 1)", name, domains, port, fromPath, match, certID)
	if err != nil {
		e.t.Fatal(err)
	}
}

// saveAs saves a copy of the env's site with edit applied.
func (e *env) saveAs(name string, edit func(s *Site)) (Site, error) {
	s := e.site
	s.ID, s.Name, s.HTTPS = 0, name, nil // a new rule, as the web panel sends it
	edit(&s)
	return e.b.st.Save(context.Background(), s)
}

// Only an identical host key and path key is an overlap, checked
// against panel and redirect rules alike.
func TestSaveRefusesASecondRuleForTheSameRequests(t *testing.T) {
	e := newEnv(t)
	for _, d := range []string{"z.example.com", "*.example.com", "r.example.com", "x.example.com"} {
		e.addDomain(d)
	}
	e.addRedirectOn("prefix", `["r.example.com"]`, e.port, "/sub", "prefix", nil)
	e.addRedirectOn("exact", `["x.example.com"]`, e.port, "/sub", "exact", nil)
	for _, c := range []struct {
		name    string
		domains []string
		path    string
		want    error
	}{
		{"same", []string{"old.example.com"}, "sub", ErrOverlap},
		{"shares one", []string{"old.example.com", "z.example.com"}, "sub", ErrOverlap},
		{"wildcard", []string{"*.example.com"}, "sub", nil},
		{"same wildcard", []string{"*.example.com"}, "sub", ErrOverlap},
		{"all", nil, "sub", nil},
		{"all again", nil, "sub", ErrOverlap},
		{"all elsewhere", nil, "other", nil},
		{"redirect prefix", []string{"r.example.com"}, "sub", ErrOverlap},
		{"redirect exact", []string{"x.example.com"}, "sub", nil},
	} {
		_, err := e.saveAs(c.name, func(s *Site) { s.Domains, s.OldPath = c.domains, c.path })
		if !errors.Is(err, c.want) || c.want == nil && err != nil {
			t.Errorf("%s: %v, want %v", c.name, err, c.want)
		}
	}
}

// Rules on one port that one name can reach may take other certificates
// now: each domain has its own, and an exact name beats a wildcard.
func TestSaveTakesOtherCertificatesForOneName(t *testing.T) {
	e := newCertEnv(t)
	t.Cleanup(e.b.stopAll)
	a, aKey := longCert(t, "a.example.com")
	w, wKey := longCert(t, "*.example.com")
	e.addDomainWith("a.example.com", CertChoice{Kind: ChoicePasted, CertPEM: a, KeyPEM: aKey})
	e.addDomainWith("*.example.com", CertChoice{Kind: ChoicePasted, CertPEM: w, KeyPEM: wKey})
	port := freePort(t)
	for _, c := range []struct {
		name    string
		domains []string
	}{{"a", []string{"a.example.com"}}, {"wild", []string{"*.example.com"}}} {
		if _, err := e.saveAs(c.name, func(s *Site) { s.Domains, s.OldPath, s.Port = c.domains, c.name, port }); err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
	}
	if err := e.b.Reload(context.Background()); err != nil {
		t.Fatal(err)
	}
	exact, wild := handshake(t, port, "a.example.com"), handshake(t, port, "b.example.com")
	if exact == wild || exact == "error" || wild == "error" {
		t.Fatalf("handshakes %s %s", exact, wild)
	}
}

// Saves that race for the same requests store one rule.
func TestRacingSavesStoreOneRule(t *testing.T) {
	e := newEnv(t)
	e.addDomain("race.example.com")
	errs := make(chan error, 8)
	for i := range 8 {
		go func() {
			_, err := e.saveAs(fmt.Sprint("racer ", i), func(s *Site) { s.Domains, s.OldPath = []string{"race.example.com"}, "race" })
			errs <- err
		}()
	}
	saved := 0
	for range 8 {
		if err := <-errs; err == nil {
			saved++
		} else if !errors.Is(err, ErrOverlap) {
			t.Fatal(err)
		}
	}
	if saved != 1 {
		t.Fatalf("%d rules saved for the same requests", saved)
	}
}

// No rule may take the port of Karvan's own web panel.
func TestSaveRefusesKarvansPanelPort(t *testing.T) {
	e := newEnv(t)
	e.b.st.panelPorts = func() []int { return []int{0, 8080} }
	if _, err := e.saveAs("panel port", func(s *Site) { s.Port = 8080 }); !errors.Is(err, ErrPortTaken) {
		t.Fatalf("on the panel's port: %v", err)
	}
	if _, err := e.saveAs("next port", func(s *Site) { s.Port = 8081 }); err != nil {
		t.Fatal(err)
	}
	if _, err := e.b.st.Save(context.Background(), e.site); err != nil {
		t.Fatalf("a rule on its own port: %v", err)
	}
}

// Rows stored before the move whose rules on one port use other
// certificates keep today's handshake pick, and their cards no longer
// carry cert_conflict: only a pin does.
func TestStoredRulesWithOtherCertificatesServeAsBefore(t *testing.T) {
	e := newCertEnv(t)
	port := freePort(t)
	first, firstKey := longCert(t, "first.example.com")
	second, secondKey := longCert(t, "second.example.com")
	e.seedOldSite(oldSite{id: 30, name: "first", path: "a", domains: `[]`, port: port, certPEM: first, keyPEM: firstKey})
	e.seedOldSite(oldSite{id: 31, name: "second", path: "b", domains: `[]`, port: port, certPEM: second, keyPEM: secondKey})
	before := e.restartWithoutAdoption()
	want := handshake(t, port, "")
	before.stopAll()
	b := e.restart(slog.New(slog.DiscardHandler))
	t.Cleanup(b.stopAll)
	if got := handshake(t, port, "any.example.net"); got != want || want == "error" {
		t.Fatalf("handshake %s, before %s", got, want)
	}
	views, err := b.Views(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	for _, v := range views {
		if slices.Contains(v.Warnings, WarnCertConflict) {
			t.Errorf("%s: warnings %v", v.Name, v.Warnings)
		}
	}
}
