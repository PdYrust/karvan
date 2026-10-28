package bridge

import (
	"context"
	"errors"
	"slices"
	"testing"
)

// domainField is the field a domain_invalid error names, or "".
func domainField(err error) string {
	var e *DomainInvalidError
	if errors.As(err, &e) {
		return e.Field
	}
	return ""
}

// certify gives the domain name the pasted certificate, registering it
// when needed.
func (e *env) certify(name, certPEM, keyPEM string) Domain {
	e.t.Helper()
	d := e.addDomain(name)
	got, err := e.b.SetDomainCertificate(context.Background(), d.ID, CertChoice{Kind: ChoicePasted, CertPEM: certPEM, KeyPEM: keyPEM})
	if err != nil {
		e.t.Fatalf("certify %s: %v", name, err)
	}
	return got
}

// Every way of D61 stores what it says: a stored row, a file pair (the row
// of the same pair when there is one), a paste sealed in a new row named
// after the domain, or none.
func TestAddDomainWithItsCertificate(t *testing.T) {
	e := newCertEnv(t)
	dir := t.TempDir()
	cf, kf := longCertFiles(t, dir, "a.example.com")
	stored := e.addCert(CertificateInput{Name: "stored", CertFile: cf, KeyFile: kf})
	cert, key := longCert(t, "*.example.com")
	bf, bk := longCertFiles(t, dir, "b.example.com")
	e.addCert(CertificateInput{Name: "b.example.com", CertPEM: cert, KeyPEM: key}) // takes the default name
	for _, c := range []struct {
		name string
		ch   CertChoice
		want string
	}{
		{"a.example.com", CertChoice{Kind: ChoiceStored, CertificateID: stored.ID}, "stored"},
		{"www.a.example.com", CertChoice{Kind: ChoiceNone, PlainOK: true}, "-"},
		{"*.example.com", CertChoice{Kind: ChoicePasted, CertPEM: cert, KeyPEM: key, Name: "wild"}, "wild"},
		{"x.example.com", CertChoice{Kind: ChoicePasted, CertPEM: cert, KeyPEM: key}, "x.example.com"},
		{"b.example.com", CertChoice{Kind: ChoiceFile, CertFile: bf, KeyFile: bk}, "b.example.com 2"},
	} {
		d := e.addDomainWith(c.name, c.ch)
		got := "-"
		if d.Certificate != nil {
			got = d.Certificate.Name
		}
		if got != c.want || d.Name != c.name {
			t.Errorf("%s: certificate %q, want %q", c.name, got, c.want)
		}
	}
	// A file pair that a stored row has reuses that row, whatever name is sent.
	ccf, ckf := longCertFiles(t, dir, "c.example.com")
	scanned := e.addCert(CertificateInput{Name: "scanned", CertFile: ccf, KeyFile: ckf})
	again := e.addDomainWith("C.example.com.", CertChoice{Kind: ChoiceFile, CertFile: ccf, KeyFile: ckf, Name: "ignored"})
	if again.Certificate == nil || again.Certificate.ID != scanned.ID || again.Name != "c.example.com" {
		t.Fatalf("same pair: %+v", again)
	}
	if n := e.strs("SELECT count(*) FROM certificates"); n[0] != "6" {
		t.Errorf("certificates = %v, want 6", n)
	}
	if got := e.strs("SELECT key_sealed IS NOT NULL FROM certificates WHERE name = 'x.example.com'"); !slices.Equal(got, []string{"1"}) {
		t.Errorf("pasted key sealed: %v", got)
	}
}

// A refused choice names its field and stores nothing: neither the domain
// nor a certificate.
func TestAddDomainRefusals(t *testing.T) {
	e := newCertEnv(t)
	dir := t.TempDir()
	cf, kf := longCertFiles(t, dir, "a.example.com")
	_, otherKF := longCertFiles(t, t.TempDir(), "a.example.com")
	stored := e.addCert(CertificateInput{Name: "stored", CertFile: cf, KeyFile: kf})
	cert, key := longCert(t, "a.example.com")
	_, otherKey := longCert(t, "a.example.com")
	wantDomain := func(field string) func(error) bool { return func(err error) bool { return domainField(err) == field } }
	wantCert := func(field, reason string) func(error) bool {
		return func(err error) bool {
			var c *CertInvalidError
			return errors.As(err, &c) && c.Field == field && c.Reason == reason
		}
	}
	for _, c := range []struct {
		name string
		ch   CertChoice
		ok   func(error) bool
	}{
		{"a.example.com", CertChoice{}, wantDomain("certificate")},
		{"a.example.com", CertChoice{Kind: "paste"}, wantDomain("certificate")},
		{"a.example.com", CertChoice{Kind: ChoiceNone}, wantDomain("plainOk")},
		{"bad name", plainOK, wantDomain("name")},
		{"a.example.com", CertChoice{Kind: ChoiceStored, CertificateID: 999}, wantCert("certificateId", "")},
		{"b.example.com", CertChoice{Kind: ChoiceStored, CertificateID: stored.ID}, wantCert("certificate", ReasonNotCovered)},
		{"b.example.com", CertChoice{Kind: ChoiceFile, CertFile: cf, KeyFile: kf}, wantCert("certificate", ReasonNotCovered)},
		{"b.example.com", CertChoice{Kind: ChoicePasted, CertPEM: cert, KeyPEM: key}, wantCert("certificate", ReasonNotCovered)},
		{"a.example.com", CertChoice{Kind: ChoiceFile, CertFile: "cert.pem", KeyFile: kf}, wantCert("certFile", "format")},
		{"a.example.com", CertChoice{Kind: ChoiceFile, CertFile: cf, KeyFile: otherKF}, wantCert("keyFile", "key_mismatch")},
		{"a.example.com", CertChoice{Kind: ChoicePasted, CertPEM: cert}, wantCert("keyPem", "format")},
		{"a.example.com", CertChoice{Kind: ChoicePasted, CertPEM: cert, KeyPEM: otherKey}, wantCert("keyPem", "key_mismatch")},
		{"a.example.com", CertChoice{Kind: ChoicePasted, CertPEM: cert, KeyPEM: key, Name: "stored"}, func(err error) bool {
			return errors.Is(err, ErrNameTaken)
		}},
		{"a.example.com", CertChoice{Kind: ChoiceACME, Method: "http-01"}, func(err error) bool { return errors.Is(err, ErrInvalidACME) }},
		{"old.example.com", plainOK, func(err error) bool { return errors.Is(err, ErrNameTaken) }},
	} {
		if _, err := e.b.AddDomain(context.Background(), c.name, c.ch); !c.ok(err) {
			t.Errorf("%s %+v: %v", c.name, c.ch, err)
		}
	}
	if got := e.strs("SELECT name FROM domains UNION ALL SELECT name FROM certificates"); !slices.Equal(got, []string{"old.example.com", "stored"}) {
		t.Fatalf("rows after the refusals: %v", got)
	}
}

// D65: a new certificate drops the domain's pins, the HTTPS rules that
// hold it take it, and every port serves it; none is refused while an
// HTTPS rule holds the domain, and plain rules stay plain.
func TestSetDomainCertificate(t *testing.T) {
	e := newCertEnv(t)
	ctx := context.Background()
	t.Cleanup(e.b.stopAll)
	cert, key := longCert(t, "tls.example.com")
	d := e.addDomainWith("tls.example.com", CertChoice{Kind: ChoicePasted, CertPEM: cert, KeyPEM: key})
	tlsPort, plainPort := freePort(t), freePort(t)
	s := e.site
	s.ID, s.Name, s.OldPath, s.Domains, s.Port, s.HTTPS = 0, "tls", "tls", []string{"tls.example.com"}, tlsPort, nil
	saved, err := e.b.SaveSite(ctx, s)
	if err != nil || !*saved.HTTPS {
		t.Fatalf("https rule: %+v %v", saved, err)
	}
	off := false
	s.Name, s.OldPath, s.Port, s.HTTPS = "plain", "plain", plainPort, &off
	plain, err := e.b.SaveSite(ctx, s)
	if err != nil || *plain.HTTPS {
		t.Fatalf("plain rule: %+v %v", plain, err)
	}
	if _, err := e.b.DB.Exec("INSERT INTO domain_pins (port, domain_id, certificate_id) VALUES (?, ?, ?)", tlsPort, d.ID, d.Certificate.ID); err != nil {
		t.Fatal(err)
	}
	before := handshake(t, tlsPort, "tls.example.com")
	other, otherKey := longCert(t, "tls.example.com")
	got, err := e.b.SetDomainCertificate(ctx, d.ID, CertChoice{Kind: ChoicePasted, CertPEM: other, KeyPEM: otherKey, Name: "renewed"})
	if err != nil || got.Certificate == nil || got.Certificate.Name != "renewed" || len(got.Pins) != 0 {
		t.Fatalf("change: %+v %v", got, err)
	}
	if after := handshake(t, tlsPort, "tls.example.com"); after == before || after == "error" {
		t.Fatalf("handshake after the change: %s", after)
	}
	if c := e.strs("SELECT s.name || ' ' || COALESCE(c.name, '-') FROM legacy_sites s LEFT JOIN certificates c ON c.id = s.certificate_id " +
		"WHERE s.name IN ('tls', 'plain') ORDER BY s.name"); !slices.Equal(c, []string{"plain -", "tls renewed"}) {
		t.Fatalf("rules after the change: %v", c)
	}
}

// None is refused while an HTTPS rule holds the domain, and nothing
// changes; an unknown domain is not found.
func TestSetDomainCertificateRefusals(t *testing.T) {
	e := newCertEnv(t)
	ctx := context.Background()
	t.Cleanup(e.b.stopAll)
	cert, key := longCert(t, "tls.example.com")
	d := e.addDomainWith("tls.example.com", CertChoice{Kind: ChoicePasted, CertPEM: cert, KeyPEM: key})
	s := e.site
	s.ID, s.Name, s.OldPath, s.Domains, s.Port, s.HTTPS = 0, "tls", "tls", []string{"tls.example.com"}, freePort(t), nil
	saved, err := e.b.SaveSite(ctx, s)
	if err != nil {
		t.Fatal(err)
	}
	var inUse *InUseError
	if _, err := e.b.SetDomainCertificate(ctx, d.ID, plainOK); !errors.As(err, &inUse) || inUse.What != "domain" ||
		!slices.Equal(inUse.Rules, []RuleRef{{RulePanel, saved.ID, "tls"}}) {
		t.Fatalf("none under an HTTPS rule: %v", err)
	}
	if got := e.domainsByName(2)["tls.example.com"]; got.Certificate == nil || got.Certificate.ID != d.Certificate.ID {
		t.Fatalf("after the refusal: %+v", got)
	}
	if _, err := e.b.SetDomainCertificate(ctx, 999, plainOK); !errors.Is(err, ErrNotFound) {
		t.Fatalf("unknown domain: %v", err)
	}
	if _, err := e.b.SetDomainCertificate(ctx, d.ID, CertChoice{Kind: ChoiceACME}); !errors.Is(err, ErrInvalidACME) {
		t.Fatalf("acme before its build: %v", err)
	}
}

// domainsByName lists the domains, checks there are n in name order and
// returns them by name.
func (e *env) domainsByName(n int) map[string]Domain {
	e.t.Helper()
	list, err := e.b.Domains(context.Background())
	if err != nil || len(list) != n {
		e.t.Fatalf("domains %+v %v", list, err)
	}
	by := map[string]Domain{}
	var names []string
	for _, d := range list {
		by[d.Name], names = d, append(names, d.Name)
	}
	if !slices.IsSorted(names) {
		e.t.Errorf("order %v", names)
	}
	return by
}

// The list names the domains whose certificate each one is, with a domain
// the certificate does not cover.
func TestCertificatesListTheirDomains(t *testing.T) {
	e := newCertEnv(t)
	cf, kf := longCertFiles(t, t.TempDir(), "old.example.com")
	c := e.addCert(CertificateInput{Name: "file", CertFile: cf, KeyFile: kf})
	other, otherKey := longCert(t, "other.example.com")
	unused := e.addCert(CertificateInput{Name: "unused", CertPEM: other, KeyPEM: otherKey})
	e.addDomainWith("other.example.com", CertChoice{Kind: ChoiceStored, CertificateID: unused.ID})
	if _, err := e.b.DB.Exec("UPDATE domains SET certificate_id = ? WHERE name = 'old.example.com'", c.ID); err != nil {
		t.Fatal(err)
	}
	list, err := e.b.Certificates(context.Background())
	if err != nil || len(list[0].Domains) != 1 || list[0].Domains[0].Name != "old.example.com" || len(list[0].Domains[0].Warnings) != 0 ||
		len(list[1].Domains) != 1 || list[1].Domains[0].Name != "other.example.com" {
		t.Fatalf("domains of certificates %+v %v", list, err)
	}
	if _, err := e.b.DB.Exec("UPDATE domains SET certificate_id = ? WHERE name = 'other.example.com'", c.ID); err != nil {
		t.Fatal(err)
	}
	list, _ = e.b.Certificates(context.Background())
	if d := list[0].Domains; len(d) != 2 || d[1].Name != "other.example.com" || !slices.Equal(d[1].Warnings, []string{WarnDomain}) {
		t.Fatalf("a domain the certificate does not cover %+v", d)
	}
}
