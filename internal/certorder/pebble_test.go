//go:build acmetest

package certorder

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/PdYrust/karvan/internal/bridge"
)

// The live test runs against Pebble from scripts/acmetest (make test-acme):
// the bridge answers HTTP-01 on 5002, where Pebble checks it, and
// DNS-01 records are set on the challenge server, whose DNS the Check and
// Pebble both ask.
const (
	pebbleDir    = "https://127.0.0.1:14000/dir"
	pebbleRoots  = "https://127.0.0.1:15000/roots/0"
	pebbleHTTP   = 5002
	pebbleDNS    = "127.0.0.1:8053"
	pebbleManage = "http://127.0.0.1:8055"
)

func pebbleClient(t *testing.T) *http.Client {
	t.Helper()
	dir := os.Getenv("ACMETEST_DIR")
	if dir == "" {
		dir = "/tmp/karvan-acmetest"
	}
	data, err := os.ReadFile(filepath.Join(dir, "pebble.minica.pem"))
	if err != nil {
		t.Fatalf("Pebble's API root (run scripts/acmetest/up.sh): %v", err)
	}
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(data) {
		t.Fatal("no certificate in pebble.minica.pem")
	}
	tr := http.DefaultTransport.(*http.Transport).Clone()
	tr.TLSClientConfig = &tls.Config{RootCAs: pool, MinVersion: tls.VersionTLS12}
	return &http.Client{Transport: tr, Timeout: 30 * time.Second}
}

func newPebbleEnv(t *testing.T) *orderEnv {
	t.Helper()
	client := pebbleClient(t)
	r := &net.Resolver{PreferGo: true, Dial: func(ctx context.Context, network, _ string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, network, pebbleDNS)
	}}
	e := newEnvOn(t, pebbleHTTP, func(e *orderEnv) { e.dir, e.client, e.lookup = pebbleDir, client, r.LookupTXT })
	e.svc.wait = 2 * time.Minute
	return e
}

// setTXT sets a TXT record on the challenge server.
func setTXT(t *testing.T, name, value string) {
	t.Helper()
	body, _ := json.Marshal(map[string]string{"host": name + ".", "value": value})
	resp, err := http.Post(pebbleManage+"/set-txt", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("set-txt: %d", resp.StatusCode)
	}
}

// waitLive waits up to a minute for domain id's latest order to reach state.
func (e *orderEnv) waitLive(id int64, state string) bridge.ACMEOrder {
	e.t.Helper()
	deadline := time.Now().Add(time.Minute)
	for {
		st, err := e.br.ACME(context.Background(), id)
		if err != nil {
			e.t.Fatal(err)
		}
		if o := st.Order; o != nil && (o.State == state || (o.State == "failed" && state != "failed")) {
			if o.State != state {
				e.t.Fatalf("order failed: %+v\n%s", o.Error, e.logs.String())
			}
			return *o
		}
		if time.Now().After(deadline) {
			e.t.Fatalf("order: %+v, want %s", st.Order, state)
		}
		time.Sleep(100 * time.Millisecond)
	}
}

// verifyChain checks the domain's stored chain against Pebble's root.
func (e *orderEnv) verifyChain(id int64, name string, client *http.Client) {
	e.t.Helper()
	var chain string
	if err := e.db.QueryRow("SELECT c.cert_pem FROM certificates c JOIN domains d ON d.certificate_id = c.id WHERE d.id = ? "+
		"AND c.origin = 'acme'", id).Scan(&chain); err != nil {
		e.t.Fatal(err)
	}
	resp, err := client.Get(pebbleRoots)
	if err != nil {
		e.t.Fatal(err)
	}
	rootPEM, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	roots, inter := x509.NewCertPool(), x509.NewCertPool()
	if !roots.AppendCertsFromPEM(rootPEM) {
		e.t.Fatalf("Pebble's root: %q", rootPEM)
	}
	var leaf *x509.Certificate
	for rest := []byte(chain); ; {
		var b *pem.Block
		if b, rest = pem.Decode(rest); b == nil {
			break
		}
		c, err := x509.ParseCertificate(b.Bytes)
		if err != nil {
			e.t.Fatal(err)
		}
		if leaf == nil {
			leaf = c
		} else {
			inter.AddCert(c)
		}
	}
	if leaf == nil {
		e.t.Fatal("no certificate stored")
	}
	if _, err := leaf.Verify(x509.VerifyOptions{DNSName: name, Roots: roots, Intermediates: inter}); err != nil {
		e.t.Fatalf("chain of %s: %v", name, err)
	}
}

// One certificate by HTTP-01 through the bridge's challenge port and one
// by DNS-01 through the challenge server's TXT record, both checked
// against Pebble's root.
func TestPebbleIssuesByHTTP01AndDNS01(t *testing.T) {
	e := newPebbleEnv(t)
	h := e.addACME("http.karvan.test", HTTP01)
	e.waitLive(h.ID, "done")
	e.verifyChain(h.ID, "http.karvan.test", e.client)
	waitClosed(t, e)

	d := e.addACME("dns.karvan.test", DNS01)
	o := e.waitLive(d.ID, "waiting_dns")
	setTXT(t, o.DNS.Name, o.DNS.Value)
	if _, err := e.svc.Check(t.Context(), d.ID, false); err != nil {
		t.Fatalf("check: %v", err)
	}
	e.waitLive(d.ID, "done")
	e.verifyChain(d.ID, "dns.karvan.test", e.client)
}
