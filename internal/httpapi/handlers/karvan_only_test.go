package handlers_test

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
	"strconv"
	"testing"
)

// atDomain is this browser on the panel's port by the name name: the
// handshake and the host ask for it, the connection goes to 127.0.0.1.
// It starts signed out, as a browser does on a new name.
func (k *karvanEnv) atDomain(name string) *env {
	b := *k.ip
	addr := "127.0.0.1:" + strconv.Itoa(k.port)
	c := *k.ip.client
	c.Transport = &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true, MinVersion: tls.VersionTLS12},
		DialContext: func(ctx context.Context, network, _ string) (net.Conn, error) {
			return (&net.Dialer{}).DialContext(ctx, network, addr)
		}}
	b.client, b.url, b.csrf = &c, "https://"+name+":"+strconv.Itoa(k.port), ""
	return &b
}

type accessBody struct {
	Only   bool   `json:"only"`
	Access string `json:"access"`
}

func access(t *testing.T, e *env) accessBody {
	t.Helper()
	var got accessBody
	res := e.do(http.MethodGet, api+"/settings/karvan-domain", nil)
	res.json(t, &got)
	if res.status != http.StatusOK {
		t.Fatalf("get the Karvan domain: %d %s", res.status, res.body)
	}
	return got
}

// opens reports whether a handshake for name (none for the server's IP)
// gets through.
func (k *karvanEnv) opens(name string) bool {
	conn, err := tls.Dial("tcp", "127.0.0.1:"+strconv.Itoa(k.port), &tls.Config{InsecureSkipVerify: true,
		MinVersion: tls.VersionTLS12, ServerName: name})
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}

// "Only open Karvan on its domain" is on by default and keeps the IP open
// until the operator signs in through the domain; from then on only the
// domain opens the panel. Turning the switch off opens everything again,
// and on again waits for a new sign-in through the domain.
func TestOnlyTheKarvanDomainOpensThePanel(t *testing.T) {
	k := newKarvanEnv(t)
	id, _ := k.domain("panel.example.com", true)
	k.put(map[string]any{"domainId": id}, karvanBody{DomainID: id})
	if got := access(t, k.ip); got != (accessBody{Only: true, Access: "waiting"}) {
		t.Fatalf("before a sign-in through the domain: %+v", got)
	}
	if !k.opens("") || !k.opens("other.example.com") {
		t.Fatal("the IP closed before a sign-in through the domain")
	}
	dom := k.atDomain("panel.example.com")
	if res := dom.login(""); res.status != http.StatusOK {
		t.Fatalf("sign in through the domain: %d %s", res.status, res.body)
	}
	dom.fetchCSRF() // the page loads: armed
	if got := access(t, dom); got.Access != "domain" {
		t.Fatalf("after a sign-in through the domain: %+v", got)
	}
	if k.opens("") || k.opens("other.example.com") || !k.opens("panel.example.com") {
		t.Fatal("the IP or another name still opens the panel")
	}
	if rows := auditRows(t, k.env, "settings.karvan_only"); len(rows) != 1 || rows[0] != `{"domain":"`+strconv.FormatInt(id, 10)+`"}` {
		t.Fatalf("audited %v", rows)
	}
	k.turnTheSwitch(dom)
}

// turnTheSwitch turns "Only open Karvan on its domain" off and on again
// from dom, a browser on the domain: off opens the IP, on waits again for
// a page load through the domain.
func (k *karvanEnv) turnTheSwitch(dom *env) {
	t := k.t
	t.Helper()
	if res := dom.do(http.MethodPut, api+"/settings/karvan-domain", map[string]any{"only": false}); res.status != http.StatusOK {
		t.Fatalf("switch off: %d %s", res.status, res.body)
	}
	if got := access(t, dom); got != (accessBody{Access: "all"}) || !k.opens("") {
		t.Fatalf("switched off: %+v", got)
	}
	if res := dom.do(http.MethodPut, api+"/settings/karvan-domain", map[string]any{"only": true}); res.status != http.StatusOK {
		t.Fatalf("switch on: %d %s", res.status, res.body)
	}
	if !k.opens("") {
		t.Fatal("switched on again: the IP closed before a new sign-in through the domain")
	}
	dom.fetchCSRF()
	if got := access(t, dom); got.Access != "domain" {
		t.Fatalf("switched on again, loaded through the domain: %+v", got)
	}
	if rows := auditRows(t, k.env, "settings.https"); rows[len(rows)-1] != `{"only":"true"}` {
		t.Fatalf("audited %v", rows)
	}
}
