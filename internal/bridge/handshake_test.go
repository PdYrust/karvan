package bridge

import (
	"fmt"
	"testing"
)

// pickedOn loads sites and facts into a bridge and returns the
// certificate id pickCert gives for each name on port, "s<id>" for a
// rule's own certificate and "none" for no answer.
func pickedOn(sites []Site, f shakeFacts, port int, names ...string) map[string]string {
	b := &Bridge{}
	b.load(sites, nil)
	f.certs = map[int64]certSnap{1: {}, 2: {}, 3: {}, 4: {}, 5: {}}
	b.shakes, b.certRows = buildShakes(b.routes, f), f.certs
	out := map[string]string{}
	for _, n := range names {
		src, ok := b.pickCert(port, n)
		switch {
		case !ok:
			out[n] = "none"
		case src.site != nil:
			out[n] = fmt.Sprintf("s%d", src.site.ID)
		default:
			out[n] = fmt.Sprint(src.id)
		}
	}
	return out
}

func tlsSite(id int64, port int, cert int64, domains ...string) Site {
	return Site{ID: id, Port: port, Domains: domains, CertificateID: cert, CertPEM: "pem"}
}

// D62: the exact domain, then the wildcard that covers the name, the
// port's pin first; no SNI and unknown names take the D53 order.
func TestPickByDomainRows(t *testing.T) {
	sites := []Site{tlsSite(1, 443, 1, "*.example.net"), tlsSite(2, 443, 2, "a.example.net", "b.example.com"), tlsSite(3, 443, 3)}
	f := shakeFacts{domains: map[string]shakeDomain{"*.example.net": {4, true}, "a.example.net": {5, true}, "b.example.com": {5, true}},
		pins: map[int]map[string]int64{443: {"b.example.com": 2}, 8443: {"a.example.net": 1}}}
	got := pickedOn(sites, f, 443, "a.example.net", "y.example.net", "b.example.com", "unknown.example.org", "", "x.y.example.net")
	want := map[string]string{"a.example.net": "5", "y.example.net": "4", "b.example.com": "2", "unknown.example.org": "3", "": "3",
		"x.y.example.net": "3"}
	if fmt.Sprint(got) != fmt.Sprint(want) {
		t.Fatalf("picks %v, want %v", got, want)
	}
}

// A port with a domain not filled, or with a rule served from its own
// certificate, keeps the D53 pick for every name (D64).
func TestPickOnAPortNotMoved(t *testing.T) {
	sites := []Site{tlsSite(1, 443, 1, "*.example.net"), tlsSite(2, 443, 2, "a.example.net")}
	f := shakeFacts{domains: map[string]shakeDomain{"*.example.net": {4, true}, "a.example.net": {5, false}}}
	want := map[string]string{"a.example.net": "1", "y.example.net": "1", "": "1"}
	if got := pickedOn(sites, f, 443, "a.example.net", "y.example.net", ""); fmt.Sprint(got) != fmt.Sprint(want) {
		t.Fatalf("not filled: picks %v, want %v", got, want)
	}
	f.domains["a.example.net"] = shakeDomain{5, true}
	sites = append(sites, Site{ID: 3, Port: 443, Domains: []string{"own.example.org"}, CertPEM: "pem"})
	f.domains["own.example.org"] = shakeDomain{0, true}
	want = map[string]string{"a.example.net": "1", "own.example.org": "s3", "": "1"}
	if got := pickedOn(sites, f, 443, "a.example.net", "own.example.org", ""); fmt.Sprint(got) != fmt.Sprint(want) {
		t.Fatalf("own certificate: picks %v, want %v", got, want)
	}
	if got := pickedOn(sites, f, 80, "a.example.net"); got["a.example.net"] != "none" {
		t.Fatalf("a port with no rule: %v", got)
	}
}
