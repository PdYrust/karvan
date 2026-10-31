package bridge

import (
	"context"
	"errors"
	"slices"
	"testing"
)

// newRuleCertEnv has a.example.com and b.example.com with certificates
// and plain.example.com without one.
func newRuleCertEnv(t *testing.T) (*env, Domain, Domain) {
	e := newCertEnv(t)
	t.Cleanup(e.b.stopAll)
	ac, ak := longCert(t, "a.example.com")
	bc, bk := longCert(t, "b.example.com")
	a := e.addDomainWith("a.example.com", CertChoice{Kind: ChoicePasted, CertPEM: ac, KeyPEM: ak})
	b := e.addDomainWith("b.example.com", CertChoice{Kind: ChoicePasted, CertPEM: bc, KeyPEM: bk})
	e.addDomain("plain.example.com")
	return e, a, b
}

// rule is a new panel rule on port with domains, as the web panel sends it.
func (e *env) rule(name string, port int, https *bool, domains ...string) Site {
	s := e.site
	s.ID, s.Name, s.OldPath, s.Port, s.Domains, s.HTTPS = 0, name, name, port, domains, https
	return s
}

func wantSiteInvalid(t *testing.T, err error, field, reason string, names ...string) {
	t.Helper()
	var e *InvalidError
	if !errors.As(err, &e) || e.Field != field || e.Reason != reason || !slices.Equal(e.Names, names) {
		t.Errorf("want site_invalid %s %s %v, got %v", field, reason, names, err)
	}
}

// On POST https is on only when every domain has a certificate;
// asking for it with a domain without one is refused with the names; on
// PATCH an absent https keeps the stored value.
func TestRuleHTTPSComesFromItsDomains(t *testing.T) {
	e, a, _ := newRuleCertEnv(t)
	ctx := context.Background()
	on, off := true, false
	tls, err := e.b.SaveSite(ctx, e.rule("tls", freePort(t), nil, "b.example.com", "a.example.com"))
	if err != nil || !*tls.HTTPS || tls.CertificateID != a.Certificate.ID {
		t.Fatalf("all with certificates: %+v %v", tls, err)
	}
	plain, err := e.b.SaveSite(ctx, e.rule("plain", freePort(t), nil, "a.example.com", "plain.example.com"))
	if err != nil || *plain.HTTPS || plain.CertificateID != 0 {
		t.Fatalf("one without a certificate: %+v %v", plain, err)
	}
	_, err = e.b.SaveSite(ctx, e.rule("asked", freePort(t), &on, "a.example.com", "plain.example.com", "z.example.com"))
	wantSiteInvalid(t, err, "domains", ReasonNotRegistered, "z.example.com")
	_, err = e.b.SaveSite(ctx, e.rule("asked", freePort(t), &on, "a.example.com", "plain.example.com"))
	wantSiteInvalid(t, err, "domains", ReasonNoCert, "plain.example.com")
	tls.Name, tls.HTTPS = "tls renamed", nil
	if kept, err := e.b.SaveSite(ctx, tls); err != nil || !*kept.HTTPS {
		t.Fatalf("PATCH keeps HTTPS: %+v %v", kept, err)
	}
	plain.Domains, plain.HTTPS = []string{"a.example.com"}, nil
	if kept, err := e.b.SaveSite(ctx, plain); err != nil || *kept.HTTPS {
		t.Fatalf("PATCH keeps plain HTTP: %+v %v", kept, err)
	}
	tls.HTTPS = &off
	if turned, err := e.b.SaveSite(ctx, tls); err != nil || *turned.HTTPS || turned.CertificateID != 0 {
		t.Fatalf("turned off: %+v %v", turned, err)
	}
}

// A domain whose Let's Encrypt order runs is pending: refused only when
// the save asks for HTTPS.
func TestRulePendingDomain(t *testing.T) {
	e, _, _ := newRuleCertEnv(t)
	ctx := context.Background()
	if _, err := e.b.DB.Exec("INSERT INTO acme_accounts (id, directory, created_at) VALUES (1, 'https://ca.example/dir', 1)"); err != nil {
		t.Fatal(err)
	}
	if _, err := e.b.DB.Exec("INSERT INTO acme_orders (domain_id, account_id, method, state, created_at, updated_at) " +
		"SELECT id, 1, 'dns-01', 'waiting_dns', 1, 1 FROM domains WHERE name = 'plain.example.com'"); err != nil {
		t.Fatal(err)
	}
	on := true
	_, err := e.b.SaveSite(ctx, e.rule("asked", freePort(t), &on, "plain.example.com"))
	wantSiteInvalid(t, err, "domains", ReasonPending, "plain.example.com")
	if s, err := e.b.SaveSite(ctx, e.rule("plain", freePort(t), nil, "plain.example.com")); err != nil || *s.HTTPS {
		t.Fatalf("a plain save of a pending domain: %+v %v", s, err)
	}
}

// The derived certificate is the port's pin of the first domain with a
// certificate, else its certificate; a pin goes once no rule on its port
// holds the domain.
func TestRuleCertificateTakesThePinFirst(t *testing.T) {
	e, a, b := newRuleCertEnv(t)
	ctx := context.Background()
	port := freePort(t)
	if _, err := e.b.DB.Exec("INSERT INTO domain_pins (port, domain_id, certificate_id) VALUES (?, ?, ?)", port, a.ID, b.Certificate.ID); err != nil {
		t.Fatal(err)
	}
	s, err := e.b.SaveSite(ctx, e.rule("pinned", port, nil, "a.example.com"))
	if err != nil || s.CertificateID != b.Certificate.ID {
		t.Fatalf("pinned: %+v %v", s, err)
	}
	s.Port, s.HTTPS = freePort(t), nil
	if moved, err := e.b.SaveSite(ctx, s); err != nil || moved.CertificateID != a.Certificate.ID {
		t.Fatalf("moved off the pinned port: %+v %v", moved, err)
	}
	if n := e.strs("SELECT count(*) FROM domain_pins"); n[0] != "0" {
		t.Fatalf("pins left: %v", n)
	}
}

// A rule with no domains takes the port's fallback: none on a new port,
// where asking for HTTPS is refused and absent https is plain; the no-SNI
// certificate on a port that has one.
func TestRuleWithNoDomainsTakesThePortFallback(t *testing.T) {
	e, a, _ := newRuleCertEnv(t)
	ctx := context.Background()
	on := true
	_, err := e.b.SaveSite(ctx, e.rule("asked", freePort(t), &on))
	wantSiteInvalid(t, err, "https", ReasonNoCert)
	if s, err := e.b.SaveSite(ctx, e.rule("bare", freePort(t), nil)); err != nil || *s.HTTPS {
		t.Fatalf("no domains on a new port: %+v %v", s, err)
	}
	port := freePort(t)
	if _, err := e.b.SaveSite(ctx, e.rule("first", port, nil, "a.example.com")); err != nil {
		t.Fatal(err)
	}
	s, err := e.b.SaveSite(ctx, e.rule("any", port, nil))
	if err != nil || !*s.HTTPS || s.CertificateID != a.Certificate.ID {
		t.Fatalf("no domains next to an HTTPS rule: %+v %v", s, err)
	}
	r := e.redirectFor("redirect", "/r", MatchExact, "https://new.example.com/")
	r.Port, r.Domains, r.HTTPS = freePort(t), nil, &on
	var invalid *RedirectInvalidError
	if _, err := e.b.SaveRedirect(ctx, r); !errors.As(err, &invalid) || invalid.Field != "https" || invalid.Reason != ReasonNoCert {
		t.Fatalf("a redirect asking for HTTPS with no fallback: %v", err)
	}
	r.Domains, r.HTTPS = []string{"z.example.com"}, nil
	if _, err := e.b.SaveRedirect(ctx, r); !errors.As(err, &invalid) || invalid.Field != "domains" || invalid.Reason != ReasonNotRegistered ||
		!slices.Equal(invalid.Names, []string{"z.example.com"}) {
		t.Fatalf("a redirect domain without a row: %v", err)
	}
}
