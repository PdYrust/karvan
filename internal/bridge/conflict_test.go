package bridge

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"strings"
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
	s.ID, s.Name = 0, name
	edit(&s)
	return e.b.st.Save(context.Background(), s)
}

// Only an identical host key and path key is an overlap (D52), checked
// against panel and redirect rules alike.
func TestSaveRefusesASecondRuleForTheSameRequests(t *testing.T) {
	e := newEnv(t)
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

// Rules on one port that one name can reach, and rules with no domains on
// one port, must use one certificate (D53).
func TestSaveRefusesTwoCertificatesForOneName(t *testing.T) {
	e := newCertEnv(t)
	cert, key := longCert(t, "a.example.com")
	c1 := e.addCert(CertificateInput{Name: "one", CertPEM: cert, KeyPEM: key})
	c2 := e.addCert(CertificateInput{Name: "two", CertPEM: cert, KeyPEM: key})
	const port = 7443
	e.addRedirectOn("q", `["q.example.com"]`, port, "/q", "exact", c1.ID)
	for _, c := range []struct {
		name    string
		domains []string
		cert    int64
		want    error
	}{
		{"a", []string{"a.example.com"}, c1.ID, nil},
		{"wild other", []string{"*.example.com"}, c2.ID, ErrCertConflict},
		{"wild same", []string{"*.example.com"}, c1.ID, nil},
		{"far", []string{"z.example.org"}, c2.ID, nil},
		{"deeper", []string{"*.z.example.org"}, c1.ID, nil}, // a wildcard and a name it does not cover
		{"all", nil, c1.ID, nil},
		{"all other", nil, c2.ID, ErrCertConflict},
		{"all same", nil, c1.ID, nil},
		{"meets the redirect", []string{"q.example.com"}, c2.ID, ErrCertConflict},
	} {
		_, err := e.saveAs(c.name, func(s *Site) {
			s.Domains, s.OldPath, s.Port, s.CertificateID = c.domains, strings.ReplaceAll(c.name, " ", "-"), port, c.cert
		})
		if !errors.Is(err, c.want) || c.want == nil && err != nil {
			t.Errorf("%s: %v, want %v", c.name, err, c.want)
		}
	}
}

// Saves that race for the same requests store one rule.
func TestRacingSavesStoreOneRule(t *testing.T) {
	e := newEnv(t)
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

// Rows stored before D53 that break it keep today's handshake pick, are
// logged at Reload and carry cert_conflict on their cards.
func TestStoredConflictIsServedAsBeforeAndFlagged(t *testing.T) {
	e := newCertEnv(t)
	port := freePort(t)
	first, firstKey := longCert(t, "first.example.com")
	second, secondKey := longCert(t, "second.example.com")
	e.seedOldSite(oldSite{id: 30, name: "first", path: "a", domains: `[]`, port: port, certPEM: first, keyPEM: firstKey})
	e.seedOldSite(oldSite{id: 31, name: "second", path: "b", domains: `[]`, port: port, certPEM: second, keyPEM: secondKey})
	before := e.restartWithoutAdoption()
	want := handshake(t, port, "")
	before.stopAll()
	var buf bytes.Buffer
	b := e.restart(slog.New(slog.NewTextHandler(&buf, nil)))
	t.Cleanup(b.stopAll)
	if got := handshake(t, port, "any.example.net"); got != want || want == "error" {
		t.Fatalf("handshake %s, before %s", got, want)
	}
	if !strings.Contains(buf.String(), `level=WARN msg="bridge: rules on one port use different certificates" port=`) ||
		!strings.Contains(buf.String(), "rule=first other=second") {
		t.Errorf("no Warn line in %s", buf.String())
	}
	views, err := b.Views(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	for _, v := range views {
		if v.ID >= 30 && !slices.Contains(v.Warnings, WarnCertConflict) {
			t.Errorf("%s: warnings %v", v.Name, v.Warnings)
		}
	}
	if v := views[0]; v.ID != e.site.ID || slices.Contains(v.Warnings, WarnCertConflict) {
		t.Errorf("a rule on another port: %+v", v.Warnings)
	}
}
