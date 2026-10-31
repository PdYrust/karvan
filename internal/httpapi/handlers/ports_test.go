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

// The overview says whether this page came over plain HTTP, for its
// warning.
func TestOverviewSaysHowThePanelIsServed(t *testing.T) {
	a := newHTTPSEnv(t, 0)
	secure := a.atHTTPSFor(a.port, "")
	secure.login("")
	for want, e := range map[string]*env{"off": a.env, "on": secure} {
		var o struct {
			Security struct {
				TLS string `json:"tls"`
			} `json:"security"`
		}
		res := e.do(http.MethodGet, api+"/overview", nil)
		res.json(t, &o)
		if res.status != http.StatusOK || o.Security.TLS != want {
			t.Fatalf("overview, want %s: %d %s", want, res.status, res.body)
		}
	}
}
