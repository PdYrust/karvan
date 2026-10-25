package bridge

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"math/big"
	"slices"
	"strings"
	"testing"
	"time"
)

// certFor makes a self-signed certificate for names (or only a common
// name when names is empty) valid from nb to na, and its key.
func certFor(t *testing.T, cn string, names []string, nb, na time.Time) ([]byte, []byte) {
	t.Helper()
	k, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	tpl := &x509.Certificate{SerialNumber: big.NewInt(7), Subject: pkix.Name{CommonName: cn, Organization: []string{"Old CA"}},
		DNSNames: names, NotBefore: nb, NotAfter: na}
	der, err := x509.CreateCertificate(rand.Reader, tpl, tpl, &k.PublicKey, k)
	if err != nil {
		t.Fatal(err)
	}
	kb, err := x509.MarshalPKCS8PrivateKey(k)
	if err != nil {
		t.Fatal(err)
	}
	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: kb})
}

func TestCertWarnings(t *testing.T) {
	now := time.Date(2026, 10, 16, 12, 0, 0, 0, time.UTC)
	day := 24 * time.Hour
	good, goodKey := certFor(t, "old.example.com", []string{"old.example.com", "*.cdn.example.com"}, now.Add(-day), now.Add(90*day))
	_, otherKey := certFor(t, "x", nil, now.Add(-day), now.Add(day))
	for name, c := range map[string]struct {
		cert, key []byte
		domains   []string
		want      []string
	}{
		"fine":          {good, goodKey, []string{"old.example.com", "a.cdn.example.com", "*.cdn.example.com"}, []string{}},
		"catch-all":     {good, goodKey, nil, []string{}},
		"no key":        {good, nil, nil, []string{WarnNoKey}},
		"wrong key":     {good, otherKey, nil, []string{WarnKeyMismatch}},
		"other domain":  {good, goodKey, []string{"new.example.com"}, []string{WarnDomain}},
		"wider pattern": {good, goodKey, []string{"*.example.com"}, []string{WarnDomain}},
		"not a cert":    {[]byte("hello"), goodKey, nil, []string{WarnUnreadable}},
	} {
		got := inspect(c.cert, c.key, c.domains, now)
		if !slices.Equal(got.Warnings, c.want) {
			t.Errorf("%s: %v, want %v", name, got.Warnings, c.want)
		}
	}
	f := inspect(good, goodKey, nil, now)
	if !slices.Equal(f.Domains, []string{"old.example.com", "*.cdn.example.com"}) || f.Issuer != "old.example.com" || !f.NotAfter.Equal(now.Add(90*day)) {
		t.Errorf("facts %+v", f)
	}
}

func TestCertTimeWarnings(t *testing.T) {
	now := time.Date(2026, 10, 16, 12, 0, 0, 0, time.UTC)
	day := 24 * time.Hour
	for name, c := range map[string]struct {
		nb, na time.Time
		want   string
	}{
		"expires in 14 days": {now.Add(-day), now.Add(14 * day), WarnExpiresSoon},
		"expired":            {now.Add(-2 * day), now.Add(-day), WarnExpired},
		"not yet valid":      {now.Add(day), now.Add(90 * day), WarnNotYetValid},
	} {
		cert, key := certFor(t, "", nil, c.nb, c.na)
		if got := inspect(cert, key, nil, now); !slices.Equal(got.Warnings, []string{c.want}) {
			t.Errorf("%s: %v", name, got.Warnings)
		}
	}
	cert, key := certFor(t, "Only.Example.com", nil, now.Add(-day), now.Add(90*day))
	if f := inspect(cert, key, []string{"only.example.com"}, now); !slices.Equal(f.Domains, []string{"only.example.com"}) ||
		f.Issuer != "Only.Example.com" {
		t.Errorf("common name only: %+v", f)
	}
}

// A changed pasted certificate needs its own key; the stored one is kept
// only while it still fits.
func TestPastedCertNeedsItsKey(t *testing.T) {
	e := newEnv(t)
	ctx := context.Background()
	now := e.clock.now()
	a, aKey := certFor(t, "old.example.com", []string{"old.example.com"}, now.Add(-time.Hour), now.Add(90*24*time.Hour))
	b, _ := certFor(t, "old.example.com", []string{"old.example.com"}, now.Add(-time.Hour), now.Add(time.Hour))
	s := e.site
	s.CertPEM, s.KeyPEM = string(a), string(aKey)
	saved, err := e.b.st.Save(ctx, s)
	if err != nil {
		t.Fatal(err)
	}
	saved.CertPEM = string(b)
	if _, err := e.b.st.Save(ctx, saved); FieldOf(err) != "keyPem" {
		t.Fatalf("a new certificate with the old key: %v", err)
	}
	saved.CertPEM = "-----BEGIN CERTIFICATE-----\nAAAA\n-----END CERTIFICATE-----\n"
	if _, err := e.b.st.Save(ctx, saved); FieldOf(err) != "certPem" {
		t.Fatalf("a broken certificate: %v", err)
	}
	v := e.b.View(ctx, mustSite(t, e, saved.ID))
	if v.Cert == nil || len(v.Cert.Warnings) != 0 {
		t.Fatalf("stored facts %+v", v.Cert)
	}
}

func mustSite(t *testing.T, e *env, id int64) Site {
	t.Helper()
	s, err := e.b.st.Site(context.Background(), id)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

// The numbers count moved users, proven ones and closed ones, for the
// site's old panel and destination.
func TestSiteStats(t *testing.T) {
	e := newEnv(t, "ali", "bob", "cyrus")
	ctx := context.Background()
	if _, err := e.b.Upload(ctx, e.site.ID, strings.NewReader(e.oldToken("ali", e.clock.now()))); err != nil {
		t.Fatal(err)
	}
	if err := e.b.st.close(ctx, e.users["bob"].ID, "revoked"); err != nil {
		t.Fatal(err)
	}
	v := e.b.View(ctx, mustSite(t, e, e.site.ID))
	if v.Stats == nil || *v.Stats != (SiteStats{Moved: 3, Verified: 1, Closed: 1}) || v.Cert != nil {
		t.Fatalf("view %+v %+v", v.Stats, v.Cert)
	}
	views, err := e.b.Views(ctx)
	if err != nil || len(views) != 1 || views[0].Stats.Moved != 3 {
		t.Fatalf("views %+v %v", views, err)
	}
}

// A token list is checked in the background; its progress is read while
// it runs, and a second list for the same site waits for the first.
func TestUploadInTheBackground(t *testing.T) {
	e := newEnv(t, "ali", "bob")
	list := e.oldToken("ali", e.clock.now()) + "\n" + e.oldToken("bob", e.clock.now())
	if p, err := e.b.UploadStatus(e.site.ID); err != nil || p.State != UploadNone {
		t.Fatalf("before: %+v %v", p, err)
	}
	first, err := e.b.StartUpload(e.site.ID, []byte(list))
	if err != nil || first.State != UploadRunning || first.Found != 2 {
		t.Fatalf("start: %+v %v", first, err)
	}
	e.b.upWG.Wait()
	p, err := e.b.UploadStatus(e.site.ID)
	if err != nil || p.State != UploadDone || p.Checked != 2 || p.Stored != 2 || p.EndedAt.IsZero() {
		t.Fatalf("done: %+v %v", p, err)
	}
	e.b.upMu.Lock()
	e.b.ups[e.site.ID].State = UploadRunning
	e.b.upMu.Unlock()
	if _, err := e.b.StartUpload(e.site.ID, []byte(list)); !errors.Is(err, ErrUploadBusy) {
		t.Fatalf("second list: %v", err)
	}
	if _, err := e.b.UploadStatus(99); !errors.Is(err, ErrNotFound) {
		t.Fatalf("unknown site: %v", err)
	}
}

// Stopping the bridge stops a list's checks, and its state says so.
func TestUploadStopsWithTheBridge(t *testing.T) {
	e := newEnv(t, "ali")
	if _, err := e.b.dest(context.Background(), e.site.PanelID); err != nil { // opened before the stop
		t.Fatal(err)
	}
	e.b.stopBg()
	if _, err := e.b.StartUpload(e.site.ID, []byte(e.oldToken("ali", e.clock.now()))); err != nil {
		t.Fatal(err)
	}
	e.b.upWG.Wait()
	if p, _ := e.b.UploadStatus(e.site.ID); p.State != UploadStopped || p.Stored != 0 {
		t.Fatalf("stopped: %+v", p)
	}
}
