package handlers_test

import (
	"net"
	"net/http"
	"testing"
)

type portsOut struct {
	Ports []struct {
		Port  int    `json:"port"`
		Owner string `json:"owner"`
	} `json:"ports"`
	Proposal int `json:"proposal"`
}

func (p portsOut) owner(port int) string {
	for _, x := range p.Ports {
		if x.Port == port {
			return x.Owner
		}
	}
	return ""
}

// GET /ports names who holds each listening port and proposes one.
func TestPortsNameTheirOwners(t *testing.T) {
	a := newHTTPSEnv(t, 0)
	busy, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer busy.Close()
	other := busy.Addr().(*net.TCPAddr).Port
	var out portsOut
	res := a.do(http.MethodGet, api+"/ports?prefer=2096", nil)
	res.json(t, &out)
	if res.status != http.StatusOK || out.owner(a.port) != "panel" || out.owner(other) != "other" {
		t.Fatalf("ports: %d %s", res.status, res.body)
	}
	if out.Proposal == a.port || out.Proposal == other || out.owner(out.Proposal) == "other" {
		t.Fatalf("proposal %d is taken: %s", out.Proposal, res.body)
	}
	for _, q := range []string{"?prefer=0", "?prefer=70000", "?prefer=x"} {
		wantError(t, a.do(http.MethodGet, api+"/ports"+q, nil), http.StatusBadRequest, "bad_request")
	}
}

// GET /settings/https lists the ports where an HTTPS rule holds a domain,
// for a shared panel address, and says when a new path waits too.
func TestHTTPSListsSharedPorts(t *testing.T) {
	a := newHTTPSEnv(t, 0)
	cert, key := testCert(t, "panel.example.com")
	a.addDomain(t, "panel.example.com", cert, key)
	port := freePort(t)
	body := map[string]any{"name": "go", "domains": []string{"panel.example.com"}, "port": port, "fromPath": "/go",
		"match": "prefix", "toUrl": "https://new.example.com/", "https": true}
	if res := a.do(http.MethodPost, api+"/link-redirects", body); res.status != http.StatusCreated {
		t.Fatalf("add redirect: %d %s", res.status, res.body)
	}
	var h struct {
		SharedPorts []struct {
			Domain string `json:"domain"`
			Port   int    `json:"port"`
		} `json:"sharedPorts"`
	}
	res := a.do(http.MethodGet, api+"/settings/https", nil)
	res.json(t, &h)
	if len(h.SharedPorts) != 1 || h.SharedPorts[0].Domain != "panel.example.com" || h.SharedPorts[0].Port != port {
		t.Fatalf("shared ports: %s", res.body)
	}
	var p portsOut
	a.do(http.MethodGet, api+"/ports", nil).json(t, &p)
	if o := p.owner(port); o != "" && o != "rule" {
		t.Fatalf("rule port %d owner %q", port, o)
	}
	start := map[string]any{"newPath": true, "port": 0, "password": password}
	if res := a.do(http.MethodPost, api+"/settings/address", start); res.status != http.StatusOK {
		t.Fatalf("new path: %d %s", res.status, res.body)
	}
	var w struct {
		Pending *struct {
			PathChanged bool `json:"pathChanged"`
		} `json:"pending"`
	}
	a.do(http.MethodGet, api+"/settings/https", nil).json(t, &w)
	if w.Pending == nil || !w.Pending.PathChanged {
		t.Fatalf("a new path waiting: %+v", w.Pending)
	}
}

// The overview says how the panel is served, for its plain HTTP warning.
func TestOverviewSaysHowThePanelIsServed(t *testing.T) {
	a := newHTTPSEnv(t, 0)
	var o struct {
		Security struct {
			TLS string `json:"tls"`
		} `json:"security"`
	}
	res := a.do(http.MethodGet, api+"/overview", nil)
	res.json(t, &o)
	if res.status != http.StatusOK || o.Security.TLS != "off" {
		t.Fatalf("overview: %d %s", res.status, res.body)
	}
}
