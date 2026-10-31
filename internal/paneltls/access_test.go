package paneltls

import (
	"context"
	"crypto/tls"
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/PdYrust/karvan/internal/audit"
	"github.com/PdYrust/karvan/internal/settings"
)

// opensFor is which of names the panel opens for.
func (f *fixture) opensFor(names ...string) []string {
	var out []string
	for _, n := range names {
		if f.m.Allows(context.Background(), n) {
			out = append(out, n)
		}
	}
	return out
}

// armed is a manager with domain 4, panel.example.com, enabled and armed.
func armedFixture(t *testing.T) (*fixture, *tls.Certificate) {
	t.Helper()
	f := newFixture(t)
	domain := otherCert(t, f)
	f.registry(map[int64]registered{4: {name: "panel.example.com", cert: domain}})
	f.m.Audit = audit.New(f.sql)
	f.m.Set(settings.TLS{DomainID: 4, Armed: true})
	return f, domain
}

// Who the panel opens for: everyone with no domain or the switch off, and
// until the first sign-in through the domain; then only the domain.
func TestAccessByTheSwitch(t *testing.T) {
	f, _ := armedFixture(t)
	ctx := context.Background()
	for _, tc := range []struct {
		tls  settings.TLS
		want Access
	}{
		{settings.TLS{}, AccessAll},
		{settings.TLS{Armed: true}, AccessAll},
		{settings.TLS{DomainID: 4, OpenIP: true, Armed: true}, AccessAll},
		{settings.TLS{DomainID: 4}, AccessWaiting},
		{settings.TLS{DomainID: 4, Armed: true}, AccessDomain},
	} {
		f.m.Set(tc.tls)
		if got, _ := f.m.Access(ctx); got != tc.want {
			t.Errorf("%+v: %s, want %s", tc.tls, got, tc.want)
		}
		want := names
		if tc.want == AccessDomain {
			want = []string{"panel.example.com", "PANEL.example.com."}
		}
		if got := f.opensFor(names...); !slices.Equal(got, want) {
			t.Errorf("%+v: opens for %q, want %q", tc.tls, got, want)
		}
	}
}

// The first signed-in request through the domain arms it, and is stored;
// a request by the IP, or a handshake for another name, does not.
func TestArmOnlyThroughTheDomain(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	f.registry(map[int64]registered{4: {name: "panel.example.com"}})
	if err := f.db.SetHTTP(ctx, settings.HTTP{Port: 34567, BasePath: "abcDEF2345ghijkm", TLS: settings.TLS{DomainID: 4}}); err != nil {
		t.Fatal(err)
	}
	f.m.Set(settings.TLS{DomainID: 4})
	for _, by := range [][]string{{"203.0.113.7"}, {"panel.example.com", ""}, {"panel.example.com", "other.example.com"}, {}} {
		if armed, err := f.m.Arm(ctx, by...); armed || err != nil {
			t.Fatalf("armed by %q: %v", by, err)
		}
	}
	if a, _ := f.m.Access(ctx); a != AccessWaiting {
		t.Fatalf("before the domain: %s", a)
	}
	if armed, err := f.m.Arm(ctx, "panel.example.com", "panel.example.com"); !armed || err != nil {
		t.Fatalf("through the domain: %v %v", armed, err)
	}
	if armed, _ := f.m.Arm(ctx, "panel.example.com"); armed {
		t.Fatal("armed twice")
	}
	if a, _ := f.m.Access(ctx); a != AccessDomain {
		t.Fatalf("after the domain: %s", a)
	}
	if h, err := f.db.HTTP(ctx); err != nil || !h.TLS.Armed {
		t.Fatalf("arming was not stored: %+v, %v", h.TLS, err)
	}
}

// A domain whose certificate ends opens the IP again, logged and audited
// once, and only the domain opens the panel again once it is renewed.
func TestFallbackWhenTheCertificateEnds(t *testing.T) {
	f, domain := armedFixture(t)
	ctx := context.Background()
	if a, _ := f.m.Access(ctx); a != AccessDomain {
		t.Fatalf("a good certificate: %s", a)
	}
	f.now = domain.Leaf.NotAfter.Add(accessFor)
	for range 3 {
		if a, why := f.m.Access(ctx); a != AccessFallback || why != FallbackExpired {
			t.Fatalf("an ended certificate: %s %s", a, why)
		}
	}
	if got := f.opensFor(names...); !slices.Equal(got, names) {
		t.Fatalf("fallback opens for %q", got)
	}
	if n := strings.Count(f.log.String(), "server's IP opens the panel again"); n != 1 {
		t.Fatalf("logged %d times: %s", n, f.log)
	}
	renewed := otherCert(t, f)
	f.registry(map[int64]registered{4: {name: "panel.example.com", cert: renewed}})
	f.now = f.now.Add(accessFor)
	if a, _ := f.m.Access(ctx); a != AccessDomain {
		t.Fatalf("renewed: %s", a)
	}
	events, err := f.m.Audit.List(ctx, audit.Filter{})
	if err != nil || len(events) != 2 || events[1].Action != audit.ActionKarvanFallback || events[1].Detail["reason"] != FallbackExpired ||
		events[1].Actor != audit.ActorSystem || events[0].Action != audit.ActionKarvanOnly || events[0].Outcome != audit.Success {
		t.Fatalf("audit %+v, %v", events, err)
	}
}

// A domain certificate whose files are gone opens the IP again too.
func TestFallbackWhenTheFilesAreGone(t *testing.T) {
	f, _ := armedFixture(t)
	f.m.Domain = func(context.Context, int64) (*tls.Certificate, error) {
		return nil, errors.New("open /etc/letsencrypt/live/x/fullchain.pem: no such file")
	}
	if a, why := f.m.Access(context.Background()); a != AccessFallback || why != FallbackMissing {
		t.Fatalf("files gone: %s %s", a, why)
	}
}

// A domain without a certificate is served with Karvan's own and is no
// reason to open the IP.
func TestDomainWithoutACertificateStaysTheOnlyWay(t *testing.T) {
	f, _ := armedFixture(t)
	f.registry(map[int64]registered{4: {name: "panel.example.com"}})
	f.m.Set(settings.TLS{DomainID: 4, Armed: true})
	if a, _ := f.m.Access(context.Background()); a != AccessDomain {
		t.Fatalf("no certificate: %s", a)
	}
}

// karvan access open-ip writes the settings from the command line; the
// running manager takes them at its next reload.
func TestReloadTakesAChangeFromTheCommandLine(t *testing.T) {
	f, _ := armedFixture(t)
	ctx := context.Background()
	h := settings.HTTP{Port: 34567, BasePath: "abcDEF2345ghijkm", TLS: settings.TLS{DomainID: 4, Armed: true}}
	if err := f.db.SetHTTP(ctx, h); err != nil {
		t.Fatal(err)
	}
	if err := f.db.SetKarvan(ctx, settings.TLS{DomainID: 4, OpenIP: true}); err != nil {
		t.Fatal(err)
	}
	if a, _ := f.m.Access(ctx); a != AccessDomain {
		t.Fatalf("before the reload: %s", a)
	}
	if err := f.m.Reload(ctx); err != nil {
		t.Fatal(err)
	}
	if a, _ := f.m.Access(ctx); a != AccessAll || !f.m.Panel().OpenIP {
		t.Fatalf("after the reload: %s %+v", a, f.m.Panel())
	}
}
