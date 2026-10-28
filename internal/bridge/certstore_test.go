package bridge

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/PdYrust/karvan/internal/panel"
	"github.com/PdYrust/karvan/internal/secret"
)

// newCertEnv is a bridge env whose clock is now, so certificates made for
// the test are valid.
func newCertEnv(t *testing.T) *env {
	t.Helper()
	e := newEnv(t)
	e.clock.t = time.Now()
	return e
}

// longCert is a self-signed certificate valid for 90 days, so it has no
// expiry warning.
func longCert(t *testing.T, name string) (string, string) {
	t.Helper()
	return selfSignedFor(t, name, 90*24*time.Hour)
}

// longCertFiles writes longCert for name and its key to dir.
func longCertFiles(t *testing.T, dir, name string) (string, string) {
	t.Helper()
	c, k := longCert(t, name)
	cf, kf := filepath.Join(dir, name+".crt"), filepath.Join(dir, name+".key")
	for path, data := range map[string]string{cf: c, kf: k} {
		if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return cf, kf
}

func (e *env) addDomain(name string) Domain {
	e.t.Helper()
	d, err := e.b.AddDomain(context.Background(), name)
	if errors.Is(err, ErrNameTaken) { // a rule's domain has its row already (D48)
		list, _ := e.b.Domains(context.Background())
		for _, d := range list {
			if d.Name == name {
				return d
			}
		}
	}
	if err != nil {
		e.t.Fatalf("add domain %q: %v", name, err)
	}
	return d
}

func (e *env) addCert(in CertificateInput) Certificate {
	e.t.Helper()
	c, err := e.b.SaveCertificate(context.Background(), 0, in)
	if err != nil {
		e.t.Fatalf("add certificate %q: %v", in.Name, err)
	}
	return c
}

// pointSite makes site id use certificate certID, as a save does.
func (e *env) pointSite(id, certID int64) {
	e.t.Helper()
	if _, err := e.b.DB.Exec("UPDATE legacy_sites SET certificate_id = ? WHERE id = ?", certID, id); err != nil {
		e.t.Fatal(err)
	}
}

// addRedirect writes a redirect rule straight into its table.
func (e *env) addRedirect(name, domains string, port int, certID any) int64 {
	e.t.Helper()
	res, err := e.b.DB.Exec("INSERT INTO link_redirects (name, domains, listen_port, from_path, match, to_url, certificate_id, "+
		"created_at, updated_at) VALUES (?, ?, ?, '/x', 'exact', 'https://new.example.com/', ?, 1, 1)", name, domains, port, certID)
	if err != nil {
		e.t.Fatal(err)
	}
	id, _ := res.LastInsertId()
	return id
}

func wantCertInvalid(t *testing.T, err error, field, reason string) {
	t.Helper()
	var ce *CertInvalidError
	if !errors.As(err, &ce) || !errors.Is(err, ErrInvalidCert) || ce.Field != field || ce.Reason != reason {
		t.Errorf("want certificate_invalid %s %s, got %v", field, reason, err)
	}
}

// A domain is stored in its normal form; anything that is not a host
// name, one leading wildcard or an IP address is refused, and so is a
// name taken already.
func TestAddDomainChecksTheName(t *testing.T) {
	e := newCertEnv(t)
	for in, want := range map[string]string{" Sub.Example.COM. ": "sub.example.com", "*.example.com": "*.example.com",
		"192.0.2.1": "192.0.2.1", "2001:DB8::1": "2001:db8::1"} {
		if d := e.addDomain(in); d.Name != want || d.ID == 0 {
			t.Errorf("add %q: %+v", in, d)
		}
	}
	long := strings.Repeat("a.", 125) + "com" // 253 bytes after "*."
	for _, bad := range []string{"", " ", "a..b", "*.*.example.com", "-a.example.com", "a_b.example.com", "*." + long,
		"exa mple.com", "*"} {
		if _, err := e.b.AddDomain(context.Background(), bad); !errors.Is(err, ErrInvalidDomain) {
			t.Errorf("add %q: %v", bad, err)
		}
	}
	if _, err := e.b.AddDomain(context.Background(), "SUB.example.com"); !errors.Is(err, ErrNameTaken) {
		t.Errorf("a taken name: %v", err)
	}
}

// A domain a panel rule or a redirect rule holds stays; an unused one goes.
func TestDomainInUseIsNotDeleted(t *testing.T) {
	e := newCertEnv(t)
	ctx := context.Background()
	used := e.addDomain("old.example.com")
	redirected := e.addDomain("r.example.com")
	e.addRedirect("to new", `["r.example.com"]`, 8443, nil)
	free := e.addDomain("free.example.com")
	err := e.b.DeleteDomain(ctx, used.ID)
	var inUse *InUseError
	if !errors.As(err, &inUse) || !errors.Is(err, ErrInUse) ||
		!slices.Equal(inUse.Rules, []RuleRef{{Kind: RulePanel, ID: e.site.ID, Name: "old"}}) {
		t.Fatalf("delete a used domain: %v", err)
	}
	if err := e.b.DeleteDomain(ctx, redirected.ID); !errors.As(err, &inUse) || inUse.Rules[0].Kind != RuleRedirect {
		t.Fatalf("delete a redirected domain: %v", err)
	}
	if err := e.b.DeleteDomain(ctx, free.ID); err != nil {
		t.Fatal(err)
	}
	if err := e.b.DeleteDomain(ctx, free.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("delete again: %v", err)
	}
	if got := e.strs("SELECT name FROM domains ORDER BY name"); !slices.Equal(got, []string{"old.example.com", "r.example.com"}) {
		t.Fatalf("domains left %v", got)
	}
}

// Every way a certificate can be wrong names its field and reason.
func TestCertificateInputIsChecked(t *testing.T) {
	e := newCertEnv(t)
	dir := t.TempDir()
	cf, kf := longCertFiles(t, dir, "a.example.com")
	_, otherKF := longCertFiles(t, t.TempDir(), "a.example.com")
	cert, key := longCert(t, "a.example.com")
	_, otherKey := longCert(t, "a.example.com")
	cases := []struct {
		in            CertificateInput
		field, reason string
	}{
		{CertificateInput{Name: " ", CertPEM: cert, KeyPEM: key}, "name", "format"},
		{CertificateInput{Name: strings.Repeat("n", 65), CertPEM: cert, KeyPEM: key}, "name", "format"},
		{CertificateInput{Name: "a\nb", CertPEM: cert, KeyPEM: key}, "name", "format"},
		{CertificateInput{Name: "x"}, "certPem", "no_cert"},
		{CertificateInput{Name: "x", CertFile: cf, KeyFile: kf, CertPEM: cert}, "certPem", "format"},
		{CertificateInput{Name: "x", CertFile: cf}, "keyFile", "format"},
		{CertificateInput{Name: "x", KeyFile: kf}, "certFile", "format"},
		{CertificateInput{Name: "x", CertFile: "cert.pem", KeyFile: kf}, "certFile", "format"},
		{CertificateInput{Name: "x", CertFile: dir + "/../" + filepath.Base(dir) + "/a.example.com.crt", KeyFile: kf}, "certFile", "format"},
		{CertificateInput{Name: "x", CertFile: cf, KeyFile: "/k\x00ey"}, "keyFile", "format"},
		{CertificateInput{Name: "x", CertFile: "/" + strings.Repeat("a", 4096), KeyFile: kf}, "certFile", "format"},
		{CertificateInput{Name: "x", CertFile: filepath.Join(dir, "missing.crt"), KeyFile: kf}, "certFile", "unreadable"},
		{CertificateInput{Name: "x", CertFile: "/dev/zero", KeyFile: kf}, "certFile", "unreadable"},
		{CertificateInput{Name: "x", CertFile: kf, KeyFile: kf}, "certFile", "no_cert"},
		{CertificateInput{Name: "x", CertFile: cf, KeyFile: filepath.Join(dir, "missing.key")}, "keyFile", "unreadable"},
		{CertificateInput{Name: "x", CertFile: cf, KeyFile: otherKF}, "keyFile", "key_mismatch"},
		{CertificateInput{Name: "x", CertPEM: "not a certificate", KeyPEM: key}, "certPem", "no_cert"},
		{CertificateInput{Name: "x", CertPEM: cert}, "keyPem", "format"},
		{CertificateInput{Name: "x", CertPEM: cert, KeyPEM: otherKey}, "keyPem", "key_mismatch"},
		{CertificateInput{Name: "x", CertPEM: cert + strings.Repeat("#", 64<<10), KeyPEM: key}, "certPem", "format"},
		{CertificateInput{Name: "x", CertPEM: cert, KeyPEM: key + strings.Repeat("#", 64<<10)}, "keyPem", "format"},
	}
	for _, c := range cases {
		_, err := e.b.SaveCertificate(context.Background(), 0, c.in)
		wantCertInvalid(t, err, c.field, c.reason)
	}
	e.addCert(CertificateInput{Name: "a", CertFile: cf, KeyFile: kf})
	if _, err := e.b.SaveCertificate(context.Background(), 0, CertificateInput{Name: " a ", CertPEM: cert, KeyPEM: key}); !errors.Is(err, ErrNameTaken) {
		t.Errorf("a taken name: %v", err)
	}
	if got := e.strs("SELECT count(*) FROM certificates"); got[0] != "1" {
		t.Errorf("certificates stored: %v", got)
	}
}

// A pasted key is sealed under its own certificate row.
func TestPastedKeyIsSealedUnderTheCertificate(t *testing.T) {
	e := newCertEnv(t)
	cert, key := longCert(t, "a.example.com")
	c := e.addCert(CertificateInput{Name: "pasted", CertPEM: cert, KeyPEM: key})
	if c.Kind != CertKindPasted || !c.HasKey || c.CertPEM != cert || c.CertFile != "" || len(c.Facts.Warnings) != 0 {
		t.Fatalf("added %+v", c)
	}
	sealed := func() []byte {
		var b []byte
		if err := e.b.DB.QueryRow("SELECT key_sealed FROM certificates WHERE id = ?", c.ID).Scan(&b); err != nil {
			t.Fatal(err)
		}
		return b
	}
	if got, err := e.b.Key.Open(sealed(), secret.AAD("certificates", "key_sealed", c.ID)); err != nil || string(got) != key {
		t.Fatalf("open under its row: %v", err)
	}
	if _, err := e.b.Key.Open(sealed(), secret.AAD("certificates", "key_sealed", c.ID+1)); err == nil {
		t.Fatal("the key opens under another row")
	}
	if bytes.Contains(sealed(), []byte("PRIVATE KEY")) {
		t.Fatal("the key is stored in the clear")
	}
}

// A change without the key keeps it, a change to files drops it, and a
// change back to a paste needs a key again.
func TestCertificateChangeKeepsOrDropsTheKey(t *testing.T) {
	e := newCertEnv(t)
	ctx := context.Background()
	cert, key := longCert(t, "a.example.com")
	c := e.addCert(CertificateInput{Name: "pasted", CertPEM: cert, KeyPEM: key})
	sealed := e.strs(fmt.Sprintf("SELECT key_sealed IS NULL FROM certificates WHERE id = %d", c.ID))
	if sealed[0] != "0" {
		t.Fatal("no key stored")
	}
	kept, err := e.b.SaveCertificate(ctx, c.ID, CertificateInput{Name: "renamed", CertPEM: cert})
	if err != nil || kept.Name != "renamed" || !kept.HasKey {
		t.Fatalf("change without the key: %+v %v", kept, err)
	}
	cf, kf := longCertFiles(t, t.TempDir(), "a.example.com")
	files, err := e.b.SaveCertificate(ctx, c.ID, CertificateInput{Name: "renamed", CertFile: cf, KeyFile: kf})
	if err != nil || files.Kind != CertKindFile || files.CertFile != cf || files.CertPEM != "" ||
		e.strs(fmt.Sprintf("SELECT key_sealed IS NULL FROM certificates WHERE id = %d", c.ID))[0] != "1" {
		t.Fatalf("change to files: %+v %v", files, err)
	}
	if _, err := e.b.SaveCertificate(ctx, c.ID, CertificateInput{Name: "renamed", CertPEM: cert}); err == nil {
		t.Fatal("files to pasted without a key")
	}
	if _, err := e.b.SaveCertificate(ctx, 999, CertificateInput{Name: "x", CertPEM: cert, KeyPEM: key}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("change an unknown one: %v", err)
	}
}

// The list says what each certificate covers and which rules use it,
// with a rule domain the certificate does not cover.
func TestCertificatesListFactsAndRules(t *testing.T) {
	e := newCertEnv(t)
	cf, kf := longCertFiles(t, t.TempDir(), "old.example.com")
	c := e.addCert(CertificateInput{Name: "file", CertFile: cf, KeyFile: kf})
	other, otherKey := longCert(t, "b.example.com")
	unused := e.addCert(CertificateInput{Name: "unused", CertPEM: other, KeyPEM: otherKey})
	e.pointSite(e.site.ID, c.ID)
	e.addRedirect("to new", `["x.example.com"]`, 8443, c.ID)
	list, err := e.b.Certificates(context.Background())
	if err != nil || len(list) != 2 || list[0].ID != c.ID || list[1].ID != unused.ID {
		t.Fatalf("list %+v %v", list, err)
	}
	got := list[0]
	if got.Kind != CertKindFile || got.CertFile != cf || got.KeyFile != kf || !got.HasKey ||
		!slices.Equal(got.Facts.Domains, []string{"old.example.com"}) || len(got.Facts.Warnings) != 0 {
		t.Fatalf("facts %+v", got)
	}
	want := []CertRule{{RuleRef{Kind: RulePanel, ID: e.site.ID, Name: "old"}, []string{}},
		{RuleRef{Kind: RuleRedirect, ID: 1, Name: "to new"}, []string{WarnDomain}}}
	if !slices.EqualFunc(got.Rules, want, func(a, b CertRule) bool { return a.RuleRef == b.RuleRef && slices.Equal(a.Warnings, b.Warnings) }) {
		t.Fatalf("rules %+v", got.Rules)
	}
	if len(list[1].Rules) != 0 || list[1].Rules == nil {
		t.Fatalf("an unused certificate's rules %+v", list[1].Rules)
	}
}

// A certificate a rule uses stays; an unused one goes.
func TestCertificateInUseIsNotDeleted(t *testing.T) {
	e := newCertEnv(t)
	ctx := context.Background()
	cert, key := longCert(t, "old.example.com")
	used := e.addCert(CertificateInput{Name: "used", CertPEM: cert, KeyPEM: key})
	redirected := e.addCert(CertificateInput{Name: "redirected", CertPEM: cert, KeyPEM: key})
	free := e.addCert(CertificateInput{Name: "free", CertPEM: cert, KeyPEM: key})
	e.pointSite(e.site.ID, used.ID)
	e.addRedirect("to new", `[]`, 8443, redirected.ID)
	var inUse *InUseError
	if err := e.b.DeleteCertificate(ctx, used.ID); !errors.As(err, &inUse) || inUse.What != "certificate" ||
		!slices.Equal(inUse.Rules, []RuleRef{{Kind: RulePanel, ID: e.site.ID, Name: "old"}}) {
		t.Fatalf("delete a used certificate: %v", err)
	}
	if err := e.b.DeleteCertificate(ctx, redirected.ID); !errors.As(err, &inUse) || inUse.Rules[0].Kind != RuleRedirect {
		t.Fatalf("delete a redirect's certificate: %v", err)
	}
	if err := e.b.DeleteCertificate(ctx, free.ID); err != nil {
		t.Fatal(err)
	}
	if err := e.b.DeleteCertificate(ctx, free.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("delete again: %v", err)
	}
}

// A domain lists the certificates that cover it, the rules that hold it
// and the warnings of D50.
func TestDomainsShowCertificatesRulesAndWarnings(t *testing.T) {
	e := newCertEnv(t)
	ctx := context.Background()
	wild, wildKey := longCert(t, "*.example.com")
	exact, exactKey := longCert(t, "old.example.com")
	w := e.addCert(CertificateInput{Name: "wild", CertPEM: wild, KeyPEM: wildKey})
	x := e.addCert(CertificateInput{Name: "exact", CertPEM: exact, KeyPEM: exactKey})
	e.pointSite(e.site.ID, x.ID) // old.example.com on port 1 with "exact"
	e.addRedirect("wild rule", `["*.example.com"]`, e.site.Port, w.ID)
	e.addRedirect("bare", `["bare.example.org"]`, 9000, w.ID)
	for _, n := range []string{"old.example.com", "*.example.com", "bare.example.org", "z.example.net"} {
		e.addDomain(n)
	}
	list, err := e.b.Domains(ctx)
	if err != nil || len(list) != 4 {
		t.Fatalf("domains %+v %v", list, err)
	}
	by := map[string]Domain{}
	for _, d := range list {
		by[d.Name] = d
	}
	if names := []string{list[0].Name, list[1].Name, list[2].Name, list[3].Name}; !slices.IsSorted(names) {
		t.Errorf("order %v", names)
	}
	check := func(name string, certs []Ref, rules []RuleRef, warnings []string) {
		t.Helper()
		d := by[name]
		if !slices.Equal(d.Certificates, certs) || !slices.Equal(d.Rules, rules) || !slices.Equal(d.Warnings, warnings) {
			t.Errorf("%s: %+v", name, d)
		}
	}
	check("old.example.com", []Ref{{x.ID, "exact"}, {w.ID, "wild"}}, []RuleRef{{RulePanel, e.site.ID, "old"}},
		[]string{WarnCertConflict})
	check("*.example.com", []Ref{{w.ID, "wild"}}, []RuleRef{{RuleRedirect, 1, "wild rule"}}, []string{WarnCertConflict})
	check("bare.example.org", []Ref{}, []RuleRef{{RuleRedirect, 2, "bare"}}, []string{WarnDomain, WarnNoCertificate})
	check("z.example.net", []Ref{}, []RuleRef{}, []string{WarnNoCertificate})
}

// A domain shows the warnings of the certificates its rules use.
func TestDomainCarriesCertificateWarnings(t *testing.T) {
	e := newCertEnv(t)
	cert, key := selfSigned(t, "old.example.com")
	c := e.addCert(CertificateInput{Name: "soon", CertPEM: cert, KeyPEM: key})
	e.pointSite(e.site.ID, c.ID)
	e.addDomain("old.example.com")
	e.clock.add(2 * time.Hour) // the certificate is valid for an hour
	list, err := e.b.Domains(context.Background())
	if err != nil || len(list) != 1 || !slices.Equal(list[0].Warnings, []string{WarnExpired}) {
		t.Fatalf("domains %+v %v", list, err)
	}
}

// A stored certificate can be checked against domains before a rule uses it.
func TestInspectAStoredCertificate(t *testing.T) {
	e := newCertEnv(t)
	ctx := context.Background()
	cert, key := longCert(t, "old.example.com")
	c := e.addCert(CertificateInput{Name: "pasted", CertPEM: cert, KeyPEM: key})
	f, err := e.b.InspectCert(ctx, CertInput{CertificateID: c.ID, Domains: []string{"new.example.com"}})
	if err != nil || !slices.Equal(f.Warnings, []string{WarnDomain}) || !slices.Equal(f.Domains, []string{"old.example.com"}) {
		t.Fatalf("inspect %+v %v", f, err)
	}
	if _, err := e.b.InspectCert(ctx, CertInput{CertificateID: 999}); FieldOf(err) != "certificateId" {
		t.Fatalf("unknown id: %v", err)
	}
	if _, err := e.b.InspectCert(ctx, CertInput{CertificateID: c.ID, CertPEM: cert}); FieldOf(err) != "certificateId" {
		t.Fatalf("an id and a certificate: %v", err)
	}
}

// A changed certificate is served by the rules that use it after the save.
func TestChangedCertificateIsServed(t *testing.T) {
	e := newCertEnv(t)
	ctx := context.Background()
	t.Cleanup(e.b.stopAll)
	cf, kf := longCertFiles(t, t.TempDir(), "old.example.com")
	port := freePort(t)
	c := e.addCert(CertificateInput{Name: "tls", CertFile: cf, KeyFile: kf})
	s, err := e.b.SaveSite(ctx, Site{Name: "tls", PanelID: 1, SourceKind: panel.Marzban, SourceURL: e.old.URL, SourceIP: "127.0.0.1",
		OldPath: "tls", Domains: []string{"old.example.com"}, Port: port, CertificateID: c.ID})
	if err != nil || s.CertificateID == 0 {
		t.Fatalf("save %+v %v", s, err)
	}
	before := handshake(t, port, "old.example.com")
	cf2, kf2 := longCertFiles(t, t.TempDir(), "old.example.com")
	if _, err := e.b.SaveCertificate(ctx, s.CertificateID, CertificateInput{Name: "tls", CertFile: cf2, KeyFile: kf2}); err != nil {
		t.Fatal(err)
	}
	if after := handshake(t, port, "old.example.com"); after == before || after == "error" {
		t.Fatalf("handshake after the change: %s", after)
	}
}

// No private key reaches a log line, even when a key cannot be opened.
func TestCertificateKeysStayOutOfLogs(t *testing.T) {
	e := newCertEnv(t)
	ctx := context.Background()
	var logs bytes.Buffer
	e.b.Log = slog.New(slog.NewTextHandler(&logs, &slog.HandlerOptions{Level: slog.LevelDebug}))
	cert, key := longCert(t, "old.example.com")
	_, otherKey := longCert(t, "old.example.com")
	c := e.addCert(CertificateInput{Name: "pasted", CertPEM: cert, KeyPEM: key})
	_, _ = e.b.SaveCertificate(ctx, c.ID, CertificateInput{Name: "pasted", CertPEM: cert, KeyPEM: otherKey})
	if _, err := e.b.DB.Exec("UPDATE certificates SET key_sealed = x'00' WHERE id = ?", c.ID); err != nil {
		t.Fatal(err)
	}
	e.pointSite(e.site.ID, c.ID)
	e.addDomain("old.example.com")
	if _, err := e.b.Certificates(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := e.b.Domains(ctx); err != nil {
		t.Fatal(err)
	}
	_, _ = e.b.InspectCert(ctx, CertInput{CertificateID: c.ID})
	if !strings.Contains(logs.String(), "pasted") {
		t.Fatalf("the unopenable key was not logged: %s", logs.String())
	}
	body := strings.Split(strings.TrimSpace(key), "\n")[1]
	if strings.Contains(logs.String(), "PRIVATE KEY") || strings.Contains(logs.String(), body) ||
		strings.Contains(logs.String(), strings.Split(strings.TrimSpace(otherKey), "\n")[1]) {
		t.Fatalf("a key reached the log: %s", logs.String())
	}
}
