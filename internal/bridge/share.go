package bridge

import (
	"context"
	"crypto/tls"
	"net/http"

	"github.com/PdYrust/karvan/internal/httpapi/middleware"
)

// PanelSite is Karvan's web panel on a shared HTTPS port (ADR 0011):
// requests on Port whose host is the domain and whose path is under
// /BasePath go to Handler, before the rules.
type PanelSite struct {
	Port     int
	DomainID int64
	BasePath string
	Handler  http.Handler
}

// sharedPanel is a PanelSite with its domain name.
type sharedPanel struct {
	PanelSite
	name string
}

// SharePanel sets the panel addresses served on bridge ports; nil stops
// them. A site whose domain is gone is left out and logged. The ports
// stay open while the panel is on them, even with no rule left there, so
// deleting a rule does not lock the operator out.
func (b *Bridge) SharePanel(ctx context.Context, sites []PanelSite) {
	out := make([]sharedPanel, 0, len(sites))
	for _, s := range sites {
		d, err := b.Domain(ctx, s.DomainID)
		if err != nil {
			b.Log.Error("bridge: the panel domain cannot be read", "domain", s.DomainID, "err", err)
			continue
		}
		out = append(out, sharedPanel{PanelSite: s, name: d.Name})
	}
	b.reloadMu.Lock() // a Reload running now must see the ports
	defer b.reloadMu.Unlock()
	b.panel.Store(&out)
	for _, s := range out {
		b.listen(s.Port, true) // a port that listens already is kept as it is
	}
}

// sharedPorts are the ports the panel is shared on.
func (b *Bridge) sharedPorts() []int {
	var out []int
	if p := b.panel.Load(); p != nil {
		for _, s := range *p {
			out = append(out, s.Port)
		}
	}
	return out
}

// panelFor is the panel handler for an HTTPS request on port, or nil. The
// hidden path is compared in constant time, as on the panel's own port;
// any other request goes to the rules and gets their answers.
func (b *Bridge) panelFor(port int, r *http.Request) http.Handler {
	p := b.panel.Load()
	if p == nil || r.TLS == nil {
		return nil
	}
	host := requestHost(r)
	for _, s := range *p {
		if s.Port == port && s.name == host && middleware.UnderBase(r.URL.Path, s.BasePath) {
			return s.Handler
		}
	}
	return nil
}

// panelCertificate is the panel domain's certificate for a name on port
// that no rule there serves; ok is false when there is none.
func (b *Bridge) panelCertificate(ctx context.Context, port int, name string) (*tls.Certificate, bool) {
	p := b.panel.Load()
	if p == nil {
		return nil, false
	}
	for _, s := range *p {
		if s.Port == port && s.name == name {
			c, err := b.PanelCertificate(ctx, s.DomainID)
			return c, err == nil
		}
	}
	return nil, false
}
