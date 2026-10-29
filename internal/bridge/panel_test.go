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

// D6: the panel reads its domain's certificate through the cache, and a
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

func TestServesHTTPS(t *testing.T) {
	e, _, _ := newRuleCertEnv(t)
	ctx := context.Background()
	port := freePort(t)
	if _, err := e.b.SaveSite(ctx, e.rule("tls", port, nil, "a.example.com")); err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct {
		port int
		name string
		want bool
	}{{port, "a.example.com", true}, {port, "b.example.com", false}, {e.port, "old.example.com", false}, {port + 1, "a.example.com", false}} {
		if got, err := e.b.ServesHTTPS(ctx, c.port, c.name); err != nil || got != c.want {
			t.Errorf("ServesHTTPS(%d, %s) = %v, %v", c.port, c.name, got, err)
		}
	}
}
