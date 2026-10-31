package bridge

import (
	"context"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"os"
	"testing"
	"time"
)

func leafOf(t *testing.T, certPEM string) []byte {
	t.Helper()
	b, _ := pem.Decode([]byte(certPEM))
	if b == nil {
		t.Fatal("no PEM")
	}
	if _, err := x509.ParseCertificate(b.Bytes); err != nil {
		t.Fatal(err)
	}
	return b.Bytes
}

// The panel reads its domain's certificate through the cache, and a
// renewed file is served without a restart.
func TestPanelCertificateFollowsARenewedFile(t *testing.T) {
	e := newEnv(t)
	ctx := context.Background()
	dir := t.TempDir()
	cf, kf := longCertFiles(t, dir, "panel.example.com")
	d := e.addDomainWith("panel.example.com", CertChoice{Kind: ChoiceFile, CertFile: cf, KeyFile: kf})
	first, err := e.b.PanelCertificate(ctx, d.ID)
	if err != nil {
		t.Fatal(err)
	}
	if again, _ := e.b.PanelCertificate(ctx, d.ID); again != first {
		t.Fatal("an unchanged file was read again")
	}
	c, k := longCert(t, "panel.example.com")
	for path, data := range map[string]string{cf: c, kf: k} {
		if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	later := time.Now().Add(time.Minute)
	if err := os.Chtimes(cf, later, later); err != nil {
		t.Fatal(err)
	}
	renewed, err := e.b.PanelCertificate(ctx, d.ID)
	if err != nil || string(renewed.Certificate[0]) != string(leafOf(t, c)) {
		t.Fatalf("the renewed file is not served: %v", err)
	}
	plain := e.addDomain("plain.example.com")
	for _, id := range []int64{plain.ID, 9999} {
		if _, err := e.b.PanelCertificate(ctx, id); !errors.Is(err, ErrNoPanelCertificate) {
			t.Errorf("domain %d: %v", id, err)
		}
	}
}

// PanelAnswers matches a domain's name like a rule does and says whether
// it has a certificate.
func TestPanelAnswers(t *testing.T) {
	e := newEnv(t)
	ctx := context.Background()
	dir := t.TempDir()
	cf, kf := longCertFiles(t, dir, "*.example.com")
	wild := e.addDomainWith("*.example.com", CertChoice{Kind: ChoiceFile, CertFile: cf, KeyFile: kf})
	plain := e.addDomain("panel.example.org")
	for _, c := range []struct {
		id            int64
		host          string
		answers, cert bool
	}{
		{wild.ID, "a.example.com", true, true},
		{wild.ID, "a.b.example.com", false, true},
		{wild.ID, "example.com", false, true},
		{plain.ID, "panel.example.org", true, false},
		{plain.ID, "other.example.org", false, false},
	} {
		answers, cert, err := e.b.PanelAnswers(ctx, c.id, c.host)
		if err != nil || answers != c.answers || cert != c.cert {
			t.Errorf("PanelAnswers(%d, %s) = %v, %v, %v", c.id, c.host, answers, cert, err)
		}
	}
	if _, _, err := e.b.PanelAnswers(ctx, 9999, "a.example.com"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("a domain that is gone: %v", err)
	}
}
