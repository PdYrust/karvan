package certorder

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/PdYrust/karvan/internal/bridge"
)

// D67, D69: a domain added by HTTP-01 gets its certificate through the
// bridge's challenge port, which closes after the order; the certificate
// is a row with origin acme, set as the domain's, with its renewal.
func TestHTTP01GetsACertificate(t *testing.T) {
	e := newOrderEnv(t)
	d := e.addACME("a.example.com", HTTP01)
	if d.ACME == nil || d.ACME.Order == nil || d.ACME.Order.Method != HTTP01 || d.Certificate != nil {
		t.Fatalf("added: %+v", d)
	}
	e.waitState(d.ID, "done")
	got := e.domain(d.ID)
	if got.Certificate == nil || got.Certificate.Origin != bridge.OriginACME || got.ACME == nil || got.ACME.Method != HTTP01 ||
		got.ACME.NextRenewal == nil || got.ACME.LastError != nil {
		t.Fatalf("after the order: %+v %+v", got, got.ACME)
	}
	want := e.clock.now().Add(60 * 24 * time.Hour)
	if !got.ACME.NextRenewal.Equal(want) {
		t.Errorf("renews %v, want %v", got.ACME.NextRenewal, want)
	}
	if e.int("SELECT count(*) FROM acme_orders WHERE key_sealed IS NOT NULL") != 0 {
		t.Error("the order kept its key")
	}
	waitClosed(t, e)
	e.waitCount("SELECT count(*) FROM audit_events WHERE action = 'acme.issued' AND actor = 'system'", 1)
}

// A CA that answers finalize with processing and no Location, as Pebble
// and Let's Encrypt do, still gives the certificate once the order is valid.
func TestFinalizeStillProcessing(t *testing.T) {
	e := newOrderEnv(t)
	e.ca.processing = true
	d := e.addACME("p.example.com", HTTP01)
	o := e.waitState(d.ID, "done", "failed")
	if o.State != "done" {
		t.Fatalf("order: %+v %+v", o, o.Error)
	}
	if c := e.domain(d.ID).Certificate; c == nil || c.Origin != bridge.OriginACME {
		t.Fatalf("certificate: %+v", c)
	}
}

// waitClosed waits until no order holds the challenge port.
func waitClosed(t *testing.T, e *orderEnv) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for dialable(e.br.ChallengePort) {
		if time.Now().After(deadline) {
			t.Fatal("the challenge port stays open after the order")
		}
		time.Sleep(10 * time.Millisecond)
	}
}

// D68: a DNS-01 order shows its record and waits; Check looks the record
// up first and asks the CA only once the value is there; force skips the
// lookup.
func TestDNS01WaitsForTheRecordAndCheck(t *testing.T) {
	e := newOrderEnv(t)
	d := e.addACME("*.w.example.com", DNS01)
	o := e.waitState(d.ID, "waiting_dns")
	if o.DNS == nil || o.DNS.Name != "_acme-challenge.w.example.com" || o.DNS.Value == "" {
		t.Fatalf("record: %+v", o.DNS)
	}
	calls := e.ca.count("chal")
	e.setTXT(o.DNS.Name, "old-value", "other")
	_, err := e.svc.Check(t.Context(), d.ID, false)
	var nf *DNSNotFoundError
	if !errors.As(err, &nf) || strings.Join(nf.Seen, ",") != "old-value,other" || e.ca.count("chal") != calls {
		t.Fatalf("check before the record: %v %+v", err, nf)
	}
	e.setTXT(o.DNS.Name, "other", o.DNS.Value)
	if _, err := e.svc.Check(t.Context(), d.ID, false); err != nil {
		t.Fatal(err)
	}
	e.waitState(d.ID, "done")
	if c := e.domain(d.ID).Certificate; c == nil || c.Origin != bridge.OriginACME {
		t.Fatalf("certificate: %+v", c)
	}
	if _, err := e.svc.Check(t.Context(), d.ID, false); !errors.Is(err, ErrState) {
		t.Fatalf("check after done: %v", err)
	}
}

func TestDNS01ForceAsksTheCAWithoutTheLookup(t *testing.T) {
	e := newOrderEnv(t)
	d := e.addACME("f.example.com", DNS01)
	e.waitState(d.ID, "waiting_dns")
	if _, err := e.svc.Check(t.Context(), d.ID, true); err != nil {
		t.Fatal(err)
	}
	o := e.waitState(d.ID, "failed")
	if o.Error == nil || o.Error.Code != CodeChallengeFailed || o.Error.Detail != "no TXT record found" {
		t.Fatalf("forced check without the record: %+v", o.Error)
	}
	if e.domain(d.ID).Certificate != nil {
		t.Fatal("a failed order set a certificate")
	}
}

// D66: methods, wildcards, IP addresses and the subscriber agreement.
func TestOrderRequestsAreChecked(t *testing.T) {
	e := newOrderEnv(t)
	for _, c := range []struct {
		name, method  string
		agree         bool
		field, reason string
	}{
		{"x.example.com", "tls-alpn-01", true, "method", ""},
		{"*.x.example.com", HTTP01, true, "method", "wildcard_needs_dns"},
		{"192.0.2.10", DNS01, true, "name", "ip"},
		{"x.example.com", HTTP01, false, "agreeTos", ""},
	} {
		_, err := e.svc.AddDomain(t.Context(), c.name, c.method, c.agree)
		var inv *bridge.ACMEInvalidError
		if !errors.As(err, &inv) || inv.Field != c.field || inv.Reason != c.reason {
			t.Errorf("%+v: %v", c, err)
		}
	}
	if n := e.int("SELECT count(*) FROM domains"); n != 0 {
		t.Fatalf("refused requests left %d domains", n)
	}
	e.dnsDone(e.addACME("first.example.com", DNS01))
	if _, err := e.svc.AddDomain(t.Context(), "second.example.com", DNS01, false); err != nil {
		t.Fatalf("the account is registered, no agreement needed: %v", err)
	}
}

// D69: one running order per domain; Cancel ends it and lets the port go.
func TestBusyAndCancel(t *testing.T) {
	e := newOrderEnv(t)
	d := e.addACME("b.example.com", DNS01)
	e.waitState(d.ID, "waiting_dns")
	if _, err := e.svc.Start(t.Context(), d.ID, DNS01, false); !errors.Is(err, ErrBusy) {
		t.Fatalf("second start: %v", err)
	}
	if err := e.svc.Cancel(t.Context(), d.ID); err != nil {
		t.Fatal(err)
	}
	e.waitState(d.ID, "cancelled")
	if err := e.svc.Cancel(t.Context(), d.ID); !errors.Is(err, bridge.ErrNotFound) {
		t.Fatalf("nothing running: %v", err)
	}
	deadline := time.Now().Add(5 * time.Second)
	for e.ca.authzState("1") != "deactivated" && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if s := e.ca.authzState("1"); s != "deactivated" {
		t.Errorf("authorization at the CA: %s", s)
	}
}

// A port failure of HTTP-01 creates nothing (section 4).
func TestHTTP01PortFailureCreatesNothing(t *testing.T) {
	e := newOrderEnv(t)
	ln := listenOn(t, e.br.ChallengePort)
	defer ln.Close()
	if _, err := e.svc.AddDomain(t.Context(), "p.example.com", HTTP01, true); !errors.Is(err, bridge.ErrPortBusy) {
		t.Fatalf("busy port: %v", err)
	}
	if e.int("SELECT count(*) FROM domains")+e.int("SELECT count(*) FROM acme_orders") != 0 {
		t.Fatal("something was stored")
	}
}

// D69: at most three orders talk to the CA at once.
func TestAtMostThreeOrdersTalkToTheCA(t *testing.T) {
	e := newOrderEnv(t)
	e.dnsDone(e.addACME("setup.example.com", DNS01))
	e.ca.mu.Lock()
	e.ca.gate = make(chan struct{})
	e.ca.mu.Unlock()
	ids := []int64{}
	for _, n := range []string{"c1", "c2", "c3", "c4", "c5"} {
		ids = append(ids, e.addACME(n+".example.com", DNS01).ID)
	}
	time.Sleep(200 * time.Millisecond)
	e.ca.mu.Lock()
	in := e.ca.inFlight
	close(e.ca.gate)
	e.ca.mu.Unlock()
	if in != talking {
		t.Fatalf("%d orders talk to the CA at once", in)
	}
	for _, id := range ids {
		e.waitState(id, "waiting_dns")
	}
	if e.ca.maxIn > talking {
		t.Fatalf("at most %d at once", e.ca.maxIn)
	}
}

// D71 and the security baseline: no answer, view or log line holds a
// private key, a key authorization or an account URL.
func TestNoKeyInAnswersOrLogs(t *testing.T) {
	e := newOrderEnv(t)
	e.ca.validate = func(o *fakeOrder, keyAuth string) *problem {
		if o.name == "bad.example.com" { // a CA may quote what it expected and the account
			return &problem{status: 403, typ: "incorrectResponse", detail: "expected " + keyAuth + " for " + e.ca.url("/acct/1")}
		}
		return e.validate(o, keyAuth)
	}
	h := e.addACME("k.example.com", HTTP01)
	e.waitState(h.ID, "done")
	e.dnsDone(e.addACME("d.example.com", DNS01))
	e.waitState(e.addACME("bad.example.com", HTTP01).ID, "failed")
	if !strings.Contains(e.logs.String(), "expected tok") {
		t.Fatal("the failure was not logged")
	}
	var out strings.Builder
	for _, v := range []func() (any, error){
		func() (any, error) { return e.br.Domains(context.Background()) },
		func() (any, error) { return e.br.Certificates(context.Background()) },
		func() (any, error) { return e.br.ACME(context.Background(), h.ID) },
		func() (any, error) { return e.svc.GetSettings(context.Background()) },
	} {
		got, err := v()
		if err != nil {
			t.Fatal(err)
		}
		raw, _ := json.Marshal(got)
		out.Write(raw)
	}
	for _, secret := range []string{"PRIVATE KEY", e.ca.url("/acct/"), "." + e.ca.thumb} {
		if strings.Contains(out.String(), secret) || strings.Contains(e.logs.String(), secret) {
			t.Errorf("found %q", secret)
		}
	}
}
