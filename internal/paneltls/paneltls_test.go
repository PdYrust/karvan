package paneltls

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"database/sql"
	"errors"
	"log/slog"
	"net"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/PdYrust/karvan/internal/secret"
	"github.com/PdYrust/karvan/internal/settings"
	"github.com/PdYrust/karvan/internal/store"
)

type fixture struct {
	m   *Manager
	db  *settings.Store
	sql *sql.DB
	now time.Time
	log *bytes.Buffer
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	dir := t.TempDir()
	db, err := store.Open(context.Background(), dir)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	keyFile := filepath.Join(dir, "secret.key")
	if err := secret.CreateKeyFile(keyFile); err != nil {
		t.Fatal(err)
	}
	key, err := secret.LoadKeyFile(keyFile)
	if err != nil {
		t.Fatal(err)
	}
	f := &fixture{db: settings.New(db), sql: db, now: time.Date(2026, 10, 29, 12, 0, 0, 0, time.UTC), log: &bytes.Buffer{}}
	f.m = &Manager{Settings: f.db, Key: key, Now: func() time.Time { return f.now },
		IPs: func() []net.IP { return []net.IP{net.ParseIP("203.0.113.7")} },
		Log: slog.New(slog.NewTextHandler(f.log, &slog.HandlerOptions{Level: slog.LevelDebug}))}
	return f
}

// fresh is a manager over the same store, as after a restart.
func (f *fixture) fresh() *Manager {
	return &Manager{Settings: f.m.Settings, Key: f.m.Key, Now: f.m.Now, IPs: f.m.IPs, Log: f.m.Log}
}

func wantSelfLeaf(t *testing.T, f *fixture, leaf *x509.Certificate) {
	t.Helper()
	if leaf.NotAfter.Sub(f.now) != selfLife || len(leaf.IPAddresses) != 1 || !leaf.IPAddresses[0].Equal(net.ParseIP("203.0.113.7")) ||
		len(leaf.DNSNames) != 1 || leaf.DNSNames[0] != "localhost" || leaf.PublicKeyAlgorithm != x509.ECDSA {
		t.Fatalf("certificate: until %v, ips %v, names %v, %v", leaf.NotAfter, leaf.IPAddresses, leaf.DNSNames, leaf.PublicKeyAlgorithm)
	}
}

func TestSelfSignedCertificate(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	c, err := f.m.Certificate(ctx, settings.TLS{Mode: settings.TLSSelf})
	if err != nil {
		t.Fatal(err)
	}
	leaf := c.Leaf
	wantSelfLeaf(t, f, leaf)
	sc, ok, err := f.db.SelfCert(ctx)
	if !ok || err != nil || strings.Contains(sc.SealedKey, "PRIVATE KEY") {
		t.Fatalf("stored: %v, %v, key in the clear: %v", ok, err, strings.Contains(sc.SealedKey, "PRIVATE KEY"))
	}
	again, err := f.fresh().Certificate(ctx, settings.TLS{Mode: settings.TLSSelf})
	if err != nil || !bytes.Equal(again.Leaf.Raw, leaf.Raw) {
		t.Fatalf("after a restart the certificate changed: %v", err)
	}
	fp, err := f.m.Fingerprint(ctx)
	if err != nil || len(fp) != 95 || fp != Fingerprint(leaf.Raw) || strings.ToUpper(fp) != fp {
		t.Fatalf("fingerprint %q, %v", fp, err)
	}
	if _, err := f.m.Certificate(ctx, settings.TLS{}); !errors.Is(err, ErrPlain) {
		t.Fatalf("plain mode: %v", err)
	}
}

func TestSelfSignedRenewal(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	first, err := f.m.Self(ctx)
	if err != nil {
		t.Fatal(err)
	}
	f.now = f.now.Add(selfLife - renewBefore - time.Minute)
	if c, _ := f.m.Self(ctx); c != first {
		t.Fatal("renewed more than 30 days before the end")
	}
	f.now = f.now.Add(2 * time.Minute)
	renewed, err := f.m.Self(ctx)
	if err != nil || renewed == first || renewed.Leaf.NotAfter.Sub(f.now) != selfLife {
		t.Fatalf("not renewed 30 days before the end: %v", err)
	}
	if c, _ := f.fresh().Self(ctx); !bytes.Equal(c.Leaf.Raw, renewed.Leaf.Raw) {
		t.Fatal("the renewed certificate was not stored")
	}
}

func TestFailedRenewalKeepsTheOldCertificate(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	first, err := f.m.Self(ctx)
	if err != nil {
		t.Fatal(err)
	}
	f.now = f.now.Add(selfLife - renewBefore)
	_ = f.sql.Close() // the new one cannot be stored
	c, err := f.m.Self(ctx)
	if err != nil || c != first || strings.Count(f.log.String(), "level=ERROR") != 1 {
		t.Fatalf("Self = %v, same %v; log %s", err, c == first, f.log)
	}
	if c, _ := f.m.Self(ctx); c != first || strings.Count(f.log.String(), "level=ERROR") != 1 {
		t.Fatal("tried again before RenewEvery")
	}
}

func TestDomainFallbackOrder(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	self, err := f.m.Self(ctx)
	if err != nil {
		t.Fatal(err)
	}
	domain := otherCert(t, f)
	var fail error
	f.m.Domain = func(_ context.Context, id int64) (*tls.Certificate, error) {
		if id != 4 {
			t.Fatalf("domain %d", id)
		}
		return domain, fail
	}
	mode := settings.TLS{Mode: settings.TLSDomain, DomainID: 4}
	// Gone at start, nothing loaded yet: the self-signed one, logged once.
	fail = errors.New("open /etc/letsencrypt/live/x/fullchain.pem: no such file")
	for range 3 {
		if c, err := f.m.Certificate(ctx, mode); err != nil || c != self {
			t.Fatalf("no certificate yet: %v, self %v", err, c == self)
		}
	}
	if n := strings.Count(f.log.String(), "level=ERROR"); n != 1 || !strings.Contains(f.log.String(), "level=WARN") {
		t.Fatalf("logged %d errors: %s", n, f.log)
	}
	// Loads: the domain's.
	fail = nil
	if c, err := f.m.Certificate(ctx, mode); err != nil || c != domain {
		t.Fatalf("domain certificate: %v", err)
	}
	// Gone again: the last good one.
	fail = errors.New("tls: private key does not match public key")
	if c, err := f.m.Certificate(ctx, mode); err != nil || c != domain {
		t.Fatalf("last good certificate: %v", err)
	}
}

// otherCert is a certificate that is not the panel's self-signed one.
func otherCert(t *testing.T, f *fixture) *tls.Certificate {
	t.Helper()
	certPEM, keyPEM, err := newSelf(nil, f.now)
	if err != nil {
		t.Fatal(err)
	}
	c, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		t.Fatal(err)
	}
	return &c
}
