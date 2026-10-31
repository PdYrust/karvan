package bridge

import (
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"fmt"
	"log/slog"
	"maps"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/PdYrust/karvan/internal/secret"
)

// oldSite is a site as stored before the certificates table: its own certificate columns, no
// certificate_id, the key of a pasted certificate sealed under the site.
type oldSite struct {
	id                int64
	name, path        string
	domains           string
	port              int
	certFile, keyFile string
	certPEM, keyPEM   string
	aadID             int64 // the id the key is sealed under, when not id
}

// seedOldSite writes s straight into legacy_sites, as version 9 left it.
func (e *env) seedOldSite(s oldSite) {
	e.t.Helper()
	var sealed []byte
	if s.keyPEM != "" {
		aad := s.id
		if s.aadID != 0 {
			aad = s.aadID
		}
		var err error
		if sealed, err = e.b.Key.Seal([]byte(s.keyPEM), secret.AAD("legacy_sites", "key_sealed", aad)); err != nil {
			e.t.Fatal(err)
		}
	}
	_, err := e.b.DB.Exec("INSERT INTO legacy_sites (id, name, panel_id, source_kind, source_url, source_ip, old_path, domains, "+
		"listen_port, cert_file, key_file, cert_pem, key_sealed, lazy, plain_http_ok, created_at, updated_at) "+
		"VALUES (?, ?, 1, 'marzban', ?, '127.0.0.1', ?, ?, ?, ?, ?, ?, ?, 1, 1, 1, 1)",
		s.id, s.name, e.site.SourceURL, s.path, s.domains, s.port, s.certFile, s.keyFile, s.certPEM, sealed)
	if err != nil {
		e.t.Fatal(err)
	}
}

// certFiles writes a self-signed certificate for name and its key to dir.
func certFiles(t *testing.T, dir, name string) (string, string) {
	t.Helper()
	c, k := selfSigned(t, name)
	cf, kf := filepath.Join(dir, name+".crt"), filepath.Join(dir, name+".key")
	if err := os.WriteFile(cf, []byte(c), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(kf, []byte(k), 0o600); err != nil {
		t.Fatal(err)
	}
	return cf, kf
}

// handshake is the leaf certificate the bridge serves on port for sni.
func handshake(t *testing.T, port int, sni string) string {
	t.Helper()
	d := &net.Dialer{Timeout: 5 * time.Second}
	conn, err := tls.DialWithDialer(d, "tcp", "127.0.0.1:"+strconv.Itoa(port),
		&tls.Config{ServerName: sni, InsecureSkipVerify: true, MinVersion: tls.VersionTLS12})
	if err != nil {
		return "error"
	}
	defer conn.Close()
	sum := sha256.Sum256(conn.ConnectionState().PeerCertificates[0].Raw)
	return hex.EncodeToString(sum[:])
}

// restart is a new bridge on the same database, as at the next start.
func (e *env) restart(log *slog.Logger) *Bridge { return e.start(log, true) }

// restartWithoutAdoption serves the sites as before the certificates
// table.
func (e *env) restartWithoutAdoption() *Bridge { return e.start(slog.New(slog.DiscardHandler), false) }

func (e *env) start(log *slog.Logger, adopt bool) *Bridge {
	e.t.Helper()
	b := New(Deps{DB: e.b.DB, Key: e.b.Key, Now: e.clock.now, Log: log, OpenDest: e.b.OpenDest})
	if adopt {
		b.adoptCertificates(context.Background())
		b.fillDomains(context.Background())
	}
	if err := b.Reload(context.Background()); err != nil {
		e.t.Fatal(err)
	}
	e.t.Cleanup(b.stopAll)
	return b
}

type adoptCase struct {
	e          *env
	tlsA, tlsB int
	names      map[int][]string // SNI names tried per port
}

// newAdoptCase seeds the sites of the adoption test: a file pair, a
// pasted certificate, two more sites on the first pair, plain HTTP (the
// env's own site), a wildcard domain and no domains.
func newAdoptCase(t *testing.T) *adoptCase {
	e := newEnv(t, "ali")
	dir := t.TempDir()
	a, ak := certFiles(t, dir, "a.example.com")
	w, wk := certFiles(t, dir, "x.example.net")
	pc, pk := selfSigned(t, "p.example.com")
	c := &adoptCase{e: e, tlsA: freePort(t), tlsB: freePort(t)}
	if _, err := e.b.DB.Exec("UPDATE legacy_sites SET listen_port = ? WHERE id = ?", freePort(t), e.site.ID); err != nil {
		t.Fatal(err)
	}
	// A certificate row that takes the first site's name, so its pair gets "file 2".
	if _, err := e.b.DB.Exec("INSERT INTO certificates (name, cert_file, key_file, created_at, updated_at) " +
		"VALUES ('file', '/elsewhere.crt', '/elsewhere.key', 1, 1)"); err != nil {
		t.Fatal(err)
	}
	for _, s := range []oldSite{
		{id: 10, name: "file", path: "sub", domains: `["a.example.com"]`, port: c.tlsA, certFile: a, keyFile: ak},
		{id: 11, name: "pasted", path: "sub", domains: `["p.example.com"]`, port: c.tlsA, certPEM: pc, keyPEM: pk},
		{id: 12, name: "shared", path: "sub", domains: `["b.example.com"]`, port: c.tlsA, certFile: a, keyFile: ak},
		{id: 13, name: "wild", path: "sub", domains: `["*.example.net"]`, port: c.tlsB, certFile: w, keyFile: wk},
		{id: 14, name: "any", path: "an", domains: `[]`, port: c.tlsB, certFile: a, keyFile: ak},
	} {
		e.seedOldSite(s)
	}
	// 0012 leaves every domain from before waiting for the fill.
	for _, q := range []string{"INSERT INTO domains (name, created_at, updated_at) SELECT DISTINCT j.value, 1, 1 FROM legacy_sites s, " +
		"json_each(s.domains) j WHERE true ON CONFLICT (name) DO NOTHING", "UPDATE domains SET cert_filled = 0"} {
		if _, err := e.b.DB.Exec(q); err != nil {
			t.Fatal(err)
		}
	}
	c.names = map[int][]string{
		c.tlsA: {"a.example.com", "p.example.com", "b.example.com", "", "unknown.example.org"},
		c.tlsB: {"x.example.net", "y.example.net", "", "other.example.org"},
	}
	return c
}

func (c *adoptCase) handshakes(t *testing.T) map[string]string {
	t.Helper()
	out := map[string]string{}
	for port, names := range c.names {
		for _, n := range names {
			out[fmt.Sprintf("%d/%s", port, n)] = handshake(t, port, n)
		}
	}
	return out
}

// answers asks every site for ali's old link and returns the answers.
func (c *adoptCase) answers(t *testing.T, b *Bridge) []string {
	t.Helper()
	tok := c.e.oldToken("ali", c.e.clock.now().Add(-time.Minute))
	var out []string
	for _, s := range []struct {
		port       int
		host, path string
	}{{c.tlsA, "a.example.com", "sub"}, {c.tlsA, "p.example.com", "sub"}, {c.tlsA, "b.example.com", "sub"},
		{c.tlsB, "z.example.net", "sub"}, {c.tlsB, "q.example.org", "an"}} {
		code, body := requestOn(b, s.port, s.host, "/"+s.path+"/"+tok)
		out = append(out, fmt.Sprintf("%s %d %s", s.host, code, body))
	}
	return out
}

func requestOn(b *Bridge, port int, host, path string) (int, string) {
	r := httptest.NewRequest(http.MethodGet, "http://"+host+path, nil)
	r.RemoteAddr = "198.51.100.7:40000"
	r.Header.Set("User-Agent", "v2rayNG/1.9")
	w := httptest.NewRecorder()
	b.Handler(port).ServeHTTP(w, r)
	return w.Code, w.Body.String()
}

// strs runs a query whose rows are one text column each.
func (e *env) strs(q string) []string {
	e.t.Helper()
	rows, err := e.b.DB.Query(q)
	if err != nil {
		e.t.Fatal(err)
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			e.t.Fatal(err)
		}
		out = append(out, s)
	}
	if err := rows.Err(); err != nil {
		e.t.Fatal(err)
	}
	return out
}

// A restart moves every site certificate into a certificates row,
// one per file pair and per paste, with the same sites, tokens,
// handshakes and answers.
func TestAdoptionKeepsHandshakesAndTokens(t *testing.T) {
	c := newAdoptCase(t)
	e := c.e
	before := e.restartWithoutAdoption()
	shakes := c.handshakes(t)
	answers := c.answers(t, before)
	tokens := e.strs("SELECT hex(mac) || ' ' || site_id || ' ' || ledger_id FROM legacy_tokens ORDER BY mac")
	served := !slices.ContainsFunc(answers, func(a string) bool { return !strings.Contains(a, " 200 config ") })
	if len(tokens) != 1 || !served || slices.Contains(slices.Collect(maps.Values(shakes)), "error") {
		t.Fatalf("setup: tokens %v, handshakes %v, answers %v", tokens, shakes, answers)
	}
	before.stopAll()
	after := e.restart(slog.New(slog.DiscardHandler))
	if got := c.handshakes(t); fmt.Sprint(got) != fmt.Sprint(shakes) {
		t.Error("a handshake changed after adoption")
	}
	if got := c.answers(t, after); !slices.Equal(got, answers) {
		t.Errorf("answers %v, want %v", got, answers)
	}
	if got := e.strs("SELECT hex(mac) || ' ' || site_id || ' ' || ledger_id FROM legacy_tokens ORDER BY mac"); !slices.Equal(got, tokens) {
		t.Errorf("tokens %v, want %v", got, tokens)
	}
	got := e.strs("SELECT s.id || ' ' || s.name || ' ' || COALESCE(c.name, '-') || ' ' || (s.cert_file || s.key_file || s.cert_pem = '') " +
		"|| ' ' || (s.key_sealed IS NULL) FROM legacy_sites s LEFT JOIN certificates c ON c.id = s.certificate_id ORDER BY s.id")
	want := []string{"1 old - 1 1", "10 file file 2 1 1", "11 pasted pasted 1 1", "12 shared file 2 1 1", "13 wild wild 1 1", "14 any file 2 1 1"}
	if !slices.Equal(got, want) {
		t.Errorf("sites %v, want %v", got, want)
	}
	if n := e.strs("SELECT count(*) FROM certificates"); n[0] != "4" {
		t.Errorf("certificates = %v, want 4 (3 adopted, 1 seeded)", n)
	}
}

// The pasted key is sealed under the certificate row, and opens there.
func TestAdoptionSealsThePastedKeyUnderTheCertificate(t *testing.T) {
	c := newAdoptCase(t)
	e := c.e
	e.restart(slog.New(slog.DiscardHandler))
	var id int64
	var sealed []byte
	if err := e.b.DB.QueryRow("SELECT id, key_sealed FROM certificates WHERE name = 'pasted'").Scan(&id, &sealed); err != nil {
		t.Fatal(err)
	}
	if _, err := e.b.Key.Open(sealed, secret.AAD("legacy_sites", "key_sealed", 11)); err == nil {
		t.Fatal("key still opens under the site")
	}
	key, err := e.b.Key.Open(sealed, secret.AAD("certificates", "key_sealed", id))
	if err != nil || !strings.Contains(string(key), "PRIVATE KEY") {
		t.Fatalf("key under the certificate: %v", err)
	}
}

// A site whose key cannot be opened, or whose pair does not match, stays
// as it is, logged at Warn and flagged on its view; a second start adds
// nothing.
func TestAdoptionFailureLeavesTheSite(t *testing.T) {
	e := newEnv(t, "ali")
	pc, pk := selfSigned(t, "p.example.com")
	_, other := selfSigned(t, "p.example.com")
	port := freePort(t)
	e.seedOldSite(oldSite{id: 20, name: "sealed elsewhere", path: "pa", domains: `["p.example.com"]`, port: port, certPEM: pc, keyPEM: pk, aadID: 99})
	e.seedOldSite(oldSite{id: 21, name: "wrong key", path: "pb", domains: `["p.example.com"]`, port: port, certPEM: pc, keyPEM: other})
	const cols = "SELECT id || ' ' || cert_pem || ' ' || hex(key_sealed) || ' ' || COALESCE(certificate_id, '-') FROM legacy_sites WHERE id >= 20 ORDER BY id"
	before := e.strs(cols)
	var buf bytes.Buffer
	for range 2 {
		b := e.restart(slog.New(slog.NewTextHandler(&buf, nil)))
		b.stopAll()
	}
	if got := e.strs(cols); !slices.Equal(got, before) {
		t.Error("a site that failed adoption changed")
	}
	if n := e.strs("SELECT count(*) FROM certificates"); n[0] != "0" {
		t.Errorf("certificates = %v, want none", n)
	}
	for _, name := range []string{"sealed elsewhere", "wrong key"} {
		if !strings.Contains(buf.String(), "level=WARN msg=\"bridge: certificate not adopted\" site=\""+name+"\"") {
			t.Errorf("no Warn line for %q in %s", name, buf.String())
		}
	}
	s, err := e.b.st.Site(context.Background(), 21)
	if err != nil {
		t.Fatal(err)
	}
	v := e.b.View(context.Background(), s)
	if !slices.Equal(v.Warnings, []string{WarnNotAdopted}) || v.Cert == nil || !slices.Contains(v.Cert.Warnings, WarnKeyMismatch) {
		t.Fatalf("view warnings %v, cert %+v", v.Warnings, v.Cert)
	}
}

// A second start finds nothing left to adopt and adds no row.
func TestAdoptionRunsOnce(t *testing.T) {
	c := newAdoptCase(t)
	e := c.e
	e.restart(slog.New(slog.DiscardHandler)).stopAll()
	first := e.strs("SELECT id || ' ' || name || ' ' || updated_at FROM certificates ORDER BY id")
	e.clock.add(time.Hour)
	e.restart(slog.New(slog.DiscardHandler)).stopAll()
	if got := e.strs("SELECT id || ' ' || name || ' ' || updated_at FROM certificates ORDER BY id"); !slices.Equal(got, first) {
		t.Fatalf("second start: %v, want %v", got, first)
	}
	// The env's clock is before the test certificates, so only their own warning shows.
	if v := e.b.View(context.Background(), mustSite(t, e, 10)); !slices.Equal(v.Warnings, []string{WarnNotYetValid}) || v.Cert == nil {
		t.Fatalf("adopted view: warnings %v, cert %+v", v.Warnings, v.Cert)
	}
}

// siteCert is a site's certificate row name and whether its own columns
// are empty.
func (e *env) siteCert(id int64) string {
	e.t.Helper()
	return e.strs(fmt.Sprintf("SELECT COALESCE(c.name, '-') || ' ' || (s.cert_file || s.key_file || s.cert_pem = '' AND s.key_sealed IS NULL) "+
		"FROM legacy_sites s LEFT JOIN certificates c ON c.id = s.certificate_id WHERE s.id = %d", id))[0]
}

// A save that keeps a site's certificateId keeps its row and its key.
func TestSaveKeepsTheCertificate(t *testing.T) {
	c := newAdoptCase(t)
	e := c.e
	ctx := context.Background()
	e.restart(slog.New(slog.DiscardHandler)).stopAll()
	t.Cleanup(e.b.stopAll)
	pasted := mustSite(t, e, 11)
	pasted.Name = "pasted again"
	if _, err := e.b.SaveSite(ctx, pasted); err != nil {
		t.Fatal(err)
	}
	if got := e.siteCert(11); got != "pasted 1" {
		t.Errorf("pasted site after save: %q", got)
	}
	if key, err := e.b.st.siteKey(ctx, 11); err != nil || !strings.Contains(string(key), "PRIVATE KEY") {
		t.Errorf("pasted key after save: %v", err)
	}
}
