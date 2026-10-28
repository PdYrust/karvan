package certorder

import (
	"context"
	"crypto/x509"
	"errors"
	"slices"
	"strconv"
	"testing"
	"time"

	"github.com/PdYrust/karvan/internal/bridge"
)

func TestRenewAtAndBackoff(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	for life, lead := range map[time.Duration]time.Duration{
		90 * 24 * time.Hour:  30 * 24 * time.Hour,
		200 * 24 * time.Hour: 30 * 24 * time.Hour,
		6 * 24 * time.Hour:   2 * 24 * time.Hour,
		3 * time.Hour:        time.Hour,
	} {
		leaf := &x509.Certificate{NotBefore: start, NotAfter: start.Add(life)}
		if got := renewAt(leaf); !got.Equal(start.Add(life - lead)) {
			t.Errorf("life %v: renews %v before expiry", life, start.Add(life).Sub(got))
		}
	}
	var got []time.Duration
	for tries := 1; tries <= 7; tries++ {
		got = append(got, backoff(tries))
	}
	want := []time.Duration{time.Hour, 2 * time.Hour, 4 * time.Hour, 8 * time.Hour, 16 * time.Hour, 24 * time.Hour, 24 * time.Hour}
	if !slices.Equal(got, want) {
		t.Fatalf("backoff %v", got)
	}
}

// httpsRule serves name over HTTPS on a free port, so its certificate is
// read from the row at each handshake.
func (e *orderEnv) httpsRule(name string) int {
	e.t.Helper()
	port := freePort(e.t)
	_, err := e.br.SaveRedirect(context.Background(), bridge.Redirect{Name: name + " " + strconv.Itoa(port), Domains: []string{name},
		Port: port, FromPath: "/go", Match: bridge.MatchPrefix, ToURL: "https://new.example.com/"})
	if err != nil {
		e.t.Fatal(err)
	}
	return port
}

// D65, D70: a renewal rewrites the domain's row and reloads, so a warm
// listener and a fresh one both serve the new certificate.
func TestRenewalServesTheNewCertificate(t *testing.T) {
	e := newOrderEnv(t)
	d := e.addACME("r.example.com", HTTP01)
	e.waitState(d.ID, "done")
	first := e.domain(d.ID).Certificate
	warm := e.httpsRule("r.example.com")
	before := handshake(t, warm, "r.example.com")
	e.svc.renewDue(t.Context())
	if n := e.int("SELECT count(*) FROM acme_orders"); n != 1 {
		t.Fatalf("not due yet, but %d orders", n)
	}
	e.clock.add(61 * 24 * time.Hour)
	e.svc.renewDue(t.Context())
	if o := e.waitState(d.ID, "done"); !o.Renewal {
		t.Fatalf("latest order: %+v", o)
	}
	got := e.domain(d.ID)
	if got.Certificate.ID != first.ID || e.int("SELECT count(*) FROM certificates") != 1 {
		t.Fatalf("the renewal made a new row: %+v", got.Certificate)
	}
	if !got.ACME.NextRenewal.Equal(e.clock.now().Add(60 * 24 * time.Hour)) {
		t.Errorf("next renewal %v", got.ACME.NextRenewal)
	}
	after := handshake(t, warm, "r.example.com")
	for deadline := time.Now().Add(5 * time.Second); after == before && time.Now().Before(deadline); { // the reload follows the commit
		time.Sleep(10 * time.Millisecond)
		after = handshake(t, warm, "r.example.com")
	}
	fresh := handshake(t, e.httpsRule("r.example.com"), "r.example.com")
	if after == before || fresh != after {
		t.Fatalf("before %s, warm %s, fresh %s", before, after, fresh)
	}
}

// D70: a certificate living 6 days renews 2 days before expiry, never in
// a loop.
func TestShortLivedCertificateDoesNotLoop(t *testing.T) {
	e := newOrderEnv(t)
	e.ca.life = 6 * 24 * time.Hour
	e.dnsDone(e.addACME("s.example.com", DNS01))
	for range 3 {
		e.svc.renewDue(t.Context())
	}
	if n := e.int("SELECT count(*) FROM acme_orders"); n != 1 {
		t.Fatalf("%d orders right after the first", n)
	}
	e.clock.add(4*24*time.Hour + time.Second)
	e.svc.renewDue(t.Context())
	if n := e.int("SELECT count(*) FROM acme_orders"); n != 2 {
		t.Fatalf("%d orders when due", n)
	}
}

// waitTries waits for domain id's renewal to count tries failures.
func (e *orderEnv) waitTries(id int64, tries int) bridge.Domain {
	e.t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for {
		d := e.domain(id)
		if d.ACME != nil && d.ACME.Tries == tries {
			return d
		}
		if time.Now().After(deadline) {
			e.t.Fatalf("domain %d: %+v, want %d tries", id, d.ACME, tries)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

// D70: a failed renewal keeps the old certificate, shows renewal_failed
// and tries again after 1, then 2 hours; a success clears it.
func TestRenewalBacksOff(t *testing.T) {
	e := newOrderEnv(t)
	d := e.addACME("o.example.com", HTTP01)
	e.waitState(d.ID, "done")
	cert := e.domain(d.ID).Certificate.ID
	e.ca.set("order", problem{status: 400, typ: "rejectedIdentifier", detail: "not now"})
	e.clock.add(61 * 24 * time.Hour)
	for tries, wait := range []time.Duration{time.Hour, 2 * time.Hour} {
		e.svc.renewDue(t.Context())
		got := e.waitTries(d.ID, tries+1)
		if got.ACME.LastError == nil || got.ACME.LastError.Code != CodeRejected || !slices.Contains(got.Warnings, bridge.WarnRenewalFailed) ||
			!got.ACME.NextRenewal.Equal(e.clock.now().Add(wait)) || got.Certificate.ID != cert {
			t.Fatalf("try %d: %+v %+v", tries+1, got.ACME, got.Warnings)
		}
		e.svc.renewDue(t.Context()) // not due again yet
		e.clock.add(wait)
	}
	e.waitCount("SELECT count(*) FROM audit_events WHERE action = 'acme.renew_failed'", 2)
	e.ca.clear("order")
	e.svc.renewDue(t.Context())
	e.waitState(d.ID, "done")
	if got := e.domain(d.ID); got.ACME.Tries != 0 || got.ACME.LastError != nil || slices.Contains(got.Warnings, bridge.WarnRenewalFailed) {
		t.Fatalf("after a success: %+v %v", got.ACME, got.Warnings)
	}
}

// D70: an HTTP-01 renewal that cannot open the challenge port stores
// why, with the same backoff.
func TestRenewalPortFailures(t *testing.T) {
	e := newOrderEnv(t)
	d := e.addACME("q.example.com", HTTP01)
	e.waitState(d.ID, "done")
	waitClosed(t, e)
	e.clock.add(61 * 24 * time.Hour)
	ln := listenOn(t, e.br.ChallengePort)
	e.svc.renewDue(t.Context())
	if got := e.waitTries(d.ID, 1); got.ACME.LastError.Code != CodePortBusy {
		t.Fatalf("busy: %+v", got.ACME.LastError)
	}
	_ = ln.Close()
	if _, err := e.br.SaveRedirect(t.Context(), bridge.Redirect{Name: "tls", Domains: []string{"q.example.com"},
		Port: e.br.ChallengePort, FromPath: "/go", Match: bridge.MatchPrefix, ToURL: "https://new.example.com/"}); err != nil {
		t.Fatal(err)
	}
	e.clock.add(time.Hour)
	e.svc.renewDue(t.Context())
	if got := e.waitTries(d.ID, 2); got.ACME.LastError.Code != CodePortTLS {
		t.Fatalf("HTTPS rule on the port: %+v", got.ACME.LastError)
	}
}

// D70: a DNS-01 renewal waits for Check; when the CA expires it, the
// domain counts acme_expired and the next due run starts a new order with
// a new record.
func TestExpiredDNSRenewalStartsAgain(t *testing.T) {
	e := newOrderEnv(t)
	d := e.addACME("x.example.com", DNS01)
	e.dnsDone(d)
	id := d.ID
	e.clock.add(61 * 24 * time.Hour)
	e.svc.renewDue(t.Context())
	first := e.waitState(id, "waiting_dns")
	if got := e.domain(id); !first.Renewal || !slices.Contains(got.Warnings, bridge.WarnRenewalWaitingDNS) {
		t.Fatalf("waiting renewal: %+v %v", first, got.Warnings)
	}
	e.clock.add(8 * 24 * time.Hour)
	e.svc.renewDue(t.Context())
	got := e.waitTries(id, 1)
	if got.ACME.LastError.Code != CodeExpired || got.ACME.Order.State != "failed" {
		t.Fatalf("expired: %+v %+v", got.ACME.LastError, got.ACME.Order)
	}
	e.clock.add(time.Hour)
	e.svc.renewDue(t.Context())
	next := e.waitState(id, "waiting_dns")
	got = e.domain(id)
	if next.ID == first.ID || next.DNS.Value == first.DNS.Value || !slices.Contains(got.Warnings, bridge.WarnRenewalWaitingDNS) ||
		!slices.Contains(got.Warnings, bridge.WarnRenewalFailed) {
		t.Fatalf("new order %+v, warnings %v", next, got.Warnings)
	}
	if _, err := e.svc.Check(t.Context(), id, false); !errors.As(err, new(*DNSNotFoundError)) {
		t.Fatalf("the old record does not hold the new value: %v", err)
	}
}

// D73: a change of directory (the staging switch) changes new orders
// only; a renewal keeps the account of the domain's certificate.
func TestRenewalKeepsItsDirectory(t *testing.T) {
	e := newOrderEnv(t)
	d := e.addACME("k.example.com", HTTP01)
	e.waitState(d.ID, "done")
	other := newFakeCA(t, e.clock.now)
	e.svc.TestDirectory = other.url("/dir")
	e.clock.add(61 * 24 * time.Hour)
	e.svc.renewDue(t.Context())
	e.waitState(d.ID, "done")
	if e.ca.count("order") != 2 || other.count("order") != 0 {
		t.Fatalf("orders: first CA %d, other %d", e.ca.count("order"), other.count("order"))
	}
	if _, err := e.svc.AddDomain(t.Context(), "new.example.com", HTTP01, false); !errors.As(err, new(*bridge.ACMEInvalidError)) {
		t.Fatalf("a new order on the other directory needs the agreement: %v", err)
	}
}
