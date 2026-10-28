package bridge

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"log/slog"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/PdYrust/karvan/internal/secret"
)

// moveCase is section 3's P41-32 database: one domain on two TLS ports
// with the same certificate, one with different certificates, a wildcard
// rule before an exact rule with another certificate, a plain rule and a
// TLS rule sharing a domain, a rule with no domains, an IP domain, a site
// not adopted and a redirect rule with a certificate.
type moveCase struct {
	e                *env
	a, b, c, d, bad  int // ports: TLS a, b, c and the not adopted site's, plain d
	certs            map[string][2]string
	redirectCertName string
}

// moveNames are the SNI names tried on every TLS port: every domain, a
// name only the wildcard covers, an unknown name and no SNI.
var moveNames = []string{"conf.example.com", "same.example.com", "a.example.net", "y.example.net", "127.0.0.1",
	"redir.example.com", "bad.example.org", "unknown.example.org", ""}

func newMoveCase(t *testing.T) *moveCase {
	e := newEnv(t, "ali")
	dir := t.TempDir()
	m := &moveCase{e: e, a: freePort(t), b: freePort(t), c: freePort(t), d: freePort(t), bad: freePort(t), certs: map[string][2]string{}}
	for _, n := range []string{"one.example.com", "two.example.com", "wild.example.net", "exact.example.net", "127.0.0.1", "redir.example.com"} {
		cf, kf := certFiles(t, dir, n)
		m.certs[n] = [2]string{cf, kf}
	}
	if _, err := e.b.DB.Exec("UPDATE legacy_sites SET listen_port = ? WHERE id = ?", freePort(t), e.site.ID); err != nil {
		t.Fatal(err)
	}
	pc, pk := selfSigned(t, "any.example.com")
	bc, bk := selfSigned(t, "bad.example.org")
	f := func(n string) (string, string) { return m.certs[n][0], m.certs[n][1] }
	one, oneKey := f("one.example.com")
	two, twoKey := f("two.example.com")
	wild, wildKey := f("wild.example.net")
	exact, exactKey := f("exact.example.net")
	ip, ipKey := f("127.0.0.1")
	for _, s := range []oldSite{
		{id: 10, name: "a-main", path: "sub", domains: `["conf.example.com","same.example.com"]`, port: m.a, certFile: one, keyFile: oneKey},
		{id: 11, name: "a-any", path: "sub", domains: `[]`, port: m.a, certPEM: pc, keyPEM: pk},
		{id: 12, name: "b-same", path: "sub", domains: `["same.example.com"]`, port: m.b, certFile: one, keyFile: oneKey},
		{id: 13, name: "b-conf", path: "sub", domains: `["conf.example.com"]`, port: m.b, certFile: two, keyFile: twoKey},
		{id: 14, name: "c-wild", path: "sub", domains: `["*.example.net"]`, port: m.c, certFile: wild, keyFile: wildKey},
		{id: 15, name: "c-exact", path: "sub", domains: `["a.example.net"]`, port: m.c, certFile: exact, keyFile: exactKey},
		{id: 16, name: "c-ip", path: "sub", domains: `["127.0.0.1"]`, port: m.c, certFile: ip, keyFile: ipKey},
		{id: 17, name: "d-plain", path: "sub", domains: `["same.example.com"]`, port: m.d},
		{id: 18, name: "e-bad", path: "sub", domains: `["bad.example.org"]`, port: m.bad, certPEM: bc, keyPEM: bk, aadID: 99},
	} {
		e.seedOldSite(s)
	}
	m.seedRedirect(t)
	// 0012 leaves every domain from before waiting for the fill.
	for _, q := range []string{"INSERT INTO domains (name, created_at, updated_at) SELECT DISTINCT j.value, 1, 1 FROM " +
		"(SELECT domains FROM legacy_sites UNION ALL SELECT domains FROM link_redirects) r, json_each(r.domains) j WHERE true " +
		"ON CONFLICT (name) DO NOTHING", "UPDATE domains SET cert_filled = 0"} {
		if _, err := e.b.DB.Exec(q); err != nil {
			t.Fatal(err)
		}
	}
	return m
}

// seedRedirect writes a redirect rule on port b with its own certificate.
func (m *moveCase) seedRedirect(t *testing.T) {
	cf, kf := m.certs["redir.example.com"][0], m.certs["redir.example.com"][1]
	m.redirectCertName = "redirect"
	if _, err := m.e.b.DB.Exec("INSERT INTO certificates (id, name, cert_file, key_file, created_at, updated_at) VALUES (50, ?, ?, ?, 1, 1)",
		m.redirectCertName, cf, kf); err != nil {
		t.Fatal(err)
	}
	if _, err := m.e.b.DB.Exec("INSERT INTO link_redirects (id, name, domains, listen_port, from_path, match, to_url, certificate_id, "+
		"created_at, updated_at) VALUES (1, 'r-b', '[\"redir.example.com\"]', ?, '/rb', 'prefix', 'https://new.example.com/x', 50, 1, 1)",
		m.b); err != nil {
		t.Fatal(err)
	}
}

// handshakes is the served certificate for every moveNames name on every
// TLS port, as "port name hash" lines. The port of the site not adopted
// fails every handshake, before and after, since its key does not open.
func (m *moveCase) handshakes(t *testing.T, ports ...int) []string {
	t.Helper()
	if len(ports) == 0 {
		ports = []int{m.a, m.b, m.c, m.bad}
	}
	var out []string
	for _, p := range ports {
		for _, n := range moveNames {
			out = append(out, fmt.Sprintf("%d %q %s", p, n, handshake(t, p, n)))
		}
	}
	return out
}

// answers asks every rule for ali's old link.
func (m *moveCase) answers(t *testing.T, b *Bridge) []string {
	t.Helper()
	tok := m.e.oldToken("ali", m.e.clock.now().Add(-time.Minute))
	var out []string
	for _, q := range []struct {
		port       int
		host, path string
	}{{m.a, "conf.example.com", "sub"}, {m.a, "q.example.org", "sub"}, {m.b, "same.example.com", "sub"}, {m.b, "conf.example.com", "sub"},
		{m.c, "z.example.net", "sub"}, {m.c, "a.example.net", "sub"}, {m.c, "127.0.0.1", "sub"}, {m.d, "same.example.com", "sub"},
		{m.bad, "bad.example.org", "sub"}, {m.b, "redir.example.com", "rb"}} {
		code, body := requestOn(b, q.port, q.host, "/"+q.path+"/"+tok)
		out = append(out, fmt.Sprintf("%s/%s %d %s", q.host, q.path, code, body))
	}
	return out
}

// moved is the domains with their certificate row names and fill flag,
// and the pins, as stored.
func (m *moveCase) moved() []string {
	return append(m.e.strs("SELECT d.name || ' ' || COALESCE(c.name, '-') || ' ' || d.cert_filled FROM domains d "+
		"LEFT JOIN certificates c ON c.id = d.certificate_id ORDER BY d.name"),
		m.e.strs(fmt.Sprintf("SELECT 'pin ' || CASE p.port WHEN %d THEN 'b' ELSE p.port END || ' ' || d.name || ' ' || c.name "+
			"FROM domain_pins p JOIN domains d ON d.id = p.domain_id JOIN certificates c ON c.id = p.certificate_id ORDER BY d.name", m.b))...)
}

// D64: the move fills every domain from its rules and pins the port that
// served another certificate, with the same handshake bytes on every
// port and name, the same answers and tokens, and a plain rule plain.
func TestDomainFillKeepsHandshakes(t *testing.T) {
	m := newMoveCase(t)
	e := m.e
	before := e.restartWithoutAdoption()
	shakes, answers := m.handshakes(t), m.answers(t, before)
	tokens := e.strs("SELECT hex(mac) || ' ' || site_id || ' ' || ledger_id FROM legacy_tokens ORDER BY mac")
	served := !slices.ContainsFunc(answers[:9], func(a string) bool { return !strings.Contains(a, " 200 config ") })
	ok := !slices.ContainsFunc(shakes, func(s string) bool {
		return strings.HasSuffix(s, " error") != strings.HasPrefix(s, strconv.Itoa(m.bad)+" ")
	})
	if len(tokens) == 0 || !served || !ok {
		t.Fatalf("setup: tokens %v, handshakes %v, answers %v", tokens, shakes, answers)
	}
	before.stopAll()
	var log bytes.Buffer
	after := e.restart(slog.New(slog.NewTextHandler(&log, nil)))
	if got := m.handshakes(t); !slices.Equal(got, shakes) {
		t.Errorf("handshakes changed:\n%v\nwant\n%v", got, shakes)
	}
	if got := m.answers(t, after); !slices.Equal(got, answers) {
		t.Errorf("answers %v, want %v", got, answers)
	}
	if got := e.strs("SELECT hex(mac) || ' ' || site_id || ' ' || ledger_id FROM legacy_tokens ORDER BY mac"); !slices.Equal(got, tokens) {
		t.Errorf("tokens %v, want %v", got, tokens)
	}
	want := []string{"*.example.net c-wild 1", "127.0.0.1 c-ip 1", "a.example.net c-wild 1", "bad.example.org - 0",
		"conf.example.com a-main 1", "old.example.com - 1", "redir.example.com redirect 1", "same.example.com a-main 1",
		"pin b conf.example.com b-conf"}
	if got := m.moved(); !slices.Equal(got, want) {
		t.Errorf("domains and pins %v, want %v", got, want)
	}
	if got := e.siteCert(17); got != "- 1" {
		t.Errorf("plain rule after the move: %q", got)
	}
	for _, line := range []string{`level=INFO msg="bridge: domain certificates filled" domains=7 pins=1`,
		`level=WARN msg="bridge: domain certificate not filled" domain=bad.example.org rule=e-bad`} {
		if !strings.Contains(log.String(), line) {
			t.Errorf("no log line %s in %s", line, log.String())
		}
	}
}

// A second start finds nothing to fill and changes no row.
func TestDomainFillRunsOnce(t *testing.T) {
	m := newMoveCase(t)
	e := m.e
	e.restart(slog.New(slog.DiscardHandler)).stopAll()
	const all = "SELECT 'd ' || id || name || COALESCE(certificate_id, '-') || cert_filled || updated_at FROM domains " +
		"UNION ALL SELECT 'p ' || port || domain_id || certificate_id FROM domain_pins " +
		"UNION ALL SELECT 'c ' || id || name || updated_at FROM certificates " +
		"UNION ALL SELECT 's ' || id || COALESCE(certificate_id, '-') || updated_at FROM legacy_sites ORDER BY 1"
	first := e.strs(all)
	e.clock.add(time.Hour)
	e.restart(slog.New(slog.DiscardHandler)).stopAll()
	if got := e.strs(all); !slices.Equal(got, first) {
		t.Fatalf("second start:\n%v\nwant\n%v", got, first)
	}
}

// Once its site is adopted, a domain left by the fill is filled at the
// next start.
func TestDomainFillWaitsForTheAdoption(t *testing.T) {
	m := newMoveCase(t)
	e := m.e
	e.restart(slog.New(slog.DiscardHandler)).stopAll()
	var sealed []byte
	if err := e.b.DB.QueryRow("SELECT key_sealed FROM legacy_sites WHERE id = 18").Scan(&sealed); err != nil {
		t.Fatal(err)
	}
	key, err := e.b.Key.Open(sealed, secret.AAD("legacy_sites", "key_sealed", 99))
	if err != nil {
		t.Fatal(err)
	}
	if sealed, err = e.b.Key.Seal(key, secret.AAD("legacy_sites", "key_sealed", 18)); err != nil {
		t.Fatal(err)
	}
	if _, err := e.b.DB.Exec("UPDATE legacy_sites SET key_sealed = ? WHERE id = 18", sealed); err != nil {
		t.Fatal(err)
	}
	e.restart(slog.New(slog.DiscardHandler)).stopAll()
	if got := e.strs("SELECT d.name || ' ' || c.name || ' ' || d.cert_filled FROM domains d JOIN certificates c ON c.id = d.certificate_id " +
		"WHERE d.name = 'bad.example.org'"); !slices.Equal(got, []string{"bad.example.org e-bad 1"}) {
		t.Fatalf("domain after the adoption: %v", got)
	}
}

// D63: a save that changes nothing, or only the name, of every migrated
// rule keeps every handshake, the pinned port's included.
func TestNoOpSavesKeepHandshakes(t *testing.T) {
	m := newMoveCase(t)
	e := m.e
	ctx := context.Background()
	b := e.restart(slog.New(slog.DiscardHandler))
	shakes := m.handshakes(t)
	for _, id := range []int64{10, 11, 12, 13, 14, 15, 16} {
		s := mustSite(t, e, id)
		if _, err := b.SaveSite(ctx, s); err != nil {
			t.Fatalf("save %s unchanged: %v", s.Name, err)
		}
		s.Name, s.HTTPS = s.Name+" renamed", nil
		if _, err := b.SaveSite(ctx, s); err != nil {
			t.Fatalf("rename %s: %v", s.Name, err)
		}
	}
	r, err := b.st.redirect(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{r.Name, "r-b renamed"} {
		r.Name = name
		if _, err := b.SaveRedirect(ctx, r); err != nil {
			t.Fatalf("save the redirect as %s: %v", name, err)
		}
	}
	if got := m.handshakes(t); !slices.Equal(got, shakes) {
		t.Errorf("handshakes changed:\n%v\nwant\n%v", got, shakes)
	}
	if got := m.moved(); !slices.Contains(got, "pin b conf.example.com b-conf") {
		t.Errorf("the pin went with a save that changed nothing: %v", got)
	}
}

// certHash is the hash handshake gives for the certificate in certPEM.
func certHash(certPEM string) string {
	block, _ := pem.Decode([]byte(certPEM))
	sum := sha256.Sum256(block.Bytes)
	return hex.EncodeToString(sum[:])
}

// D65 on the moved database: a new certificate for the conflicting domain
// drops its pin and every port serves it; the plain rule stays plain; a
// rule with no domains on a new port is plain unless it asks for HTTPS,
// which is refused; a file pair a row has is that row.
func TestMovedDomainTakesANewCertificate(t *testing.T) {
	m := newMoveCase(t)
	e := m.e
	ctx := context.Background()
	b := e.restart(slog.New(slog.DiscardHandler))
	var conf int64
	if err := e.b.DB.QueryRow("SELECT id FROM domains WHERE name = 'conf.example.com'").Scan(&conf); err != nil {
		t.Fatal(err)
	}
	cert, key := selfSigned(t, "conf.example.com")
	d, err := b.SetDomainCertificate(ctx, conf, CertChoice{Kind: ChoicePasted, CertPEM: cert, KeyPEM: key})
	if err != nil || len(d.Pins) != 0 || slices.Contains(d.Warnings, WarnCertConflict) {
		t.Fatalf("set: %+v %v", d, err)
	}
	for _, port := range []int{m.a, m.b} {
		if got := handshake(t, port, "conf.example.com"); got != certHash(cert) {
			t.Errorf("port %d serves %s", port, got)
		}
	}
	plain := mustSite(t, e, 17)
	plain.HTTPS = nil
	if s, err := b.SaveSite(ctx, plain); err != nil || *s.HTTPS {
		t.Fatalf("the plain rule after a save: %+v %v", s, err)
	}
	on := true
	_, err = b.SaveSite(ctx, e.rule("bare", freePort(t), &on))
	wantSiteInvalid(t, err, "https", ReasonNoCert)
	if s, err := b.SaveSite(ctx, e.rule("bare", freePort(t), nil)); err != nil || *s.HTTPS {
		t.Fatalf("no domains on a new port: %+v %v", s, err)
	}
	one := m.certs["one.example.com"]
	got, err := b.AddDomain(ctx, "one.example.com", CertChoice{Kind: ChoiceFile, CertFile: one[0], KeyFile: one[1]})
	if err != nil || got.Certificate == nil || got.Certificate.Name != "a-main" {
		t.Fatalf("a stored pair: %+v %v", got, err)
	}
}
