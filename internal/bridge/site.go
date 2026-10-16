// Package bridge answers old subscription links for users Karvan moved:
// a request with an old token is checked against what Karvan recorded,
// then served from the new panel with a fresh token for the same user
// (D55-D64). The new panel stays the judge of its own users: a user it
// revoked or deleted loses the old link too (owner rule).
package bridge

import (
	"errors"
	"fmt"
	"net"
	"net/netip"
	"regexp"
	"slices"
	"strings"

	"github.com/PdYrust/karvan/internal/panel"
	"github.com/PdYrust/karvan/internal/panel/source"
)

// Site is one old subscription address the bridge answers.
type Site struct {
	ID          int64      `json:"id"`
	Name        string     `json:"name"`
	PanelID     int64      `json:"panelId"` // the destination panel
	SourceKind  panel.Kind `json:"sourceKind"`
	SourceURL   string     `json:"sourceUrl"`
	SourceIP    string     `json:"sourceIp,omitempty"`
	SourcePin   string     `json:"sourcePin,omitempty"`
	OldPath     string     `json:"oldPath"`
	Domains     []string   `json:"domains"`
	Port        int        `json:"port"`
	CertFile    string     `json:"certFile,omitempty"`
	KeyFile     string     `json:"keyFile,omitempty"`
	CertPEM     string     `json:"certPem,omitempty"`
	KeyPEM      string     `json:"-"` // write only; stored sealed
	HasKey      bool       `json:"hasKey"`
	Lazy        bool       `json:"lazy"`
	PlainHTTPOK bool       `json:"plainHttpOk"`
	origin      string
}

// TLS reports whether the site serves HTTPS.
func (s Site) TLS() bool { return s.CertFile != "" || s.CertPEM != "" }

var (
	pathRe  = regexp.MustCompile(`^[A-Za-z0-9_\-]{1,64}$`)
	labelRe = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?$`)
	pinRe   = regexp.MustCompile(`^[0-9a-f]{64}$`)
	// ErrInvalidSite means a site field is wrong; the message says which.
	ErrInvalidSite = errors.New("invalid site")
	siteKind       = []panel.Kind{panel.Marzban, panel.PasarGuard}
)

func invalid(f string, a ...any) error {
	return fmt.Errorf("%w: %s", ErrInvalidSite, fmt.Sprintf(f, a...))
}

// normalize checks s and puts it in the stored form.
func (s *Site) normalize() error {
	s.Name = strings.TrimSpace(s.Name)
	s.OldPath = strings.Trim(strings.TrimSpace(s.OldPath), "/")
	s.SourcePin = strings.ToLower(strings.ReplaceAll(s.SourcePin, ":", ""))
	if err := s.checkFields(); err != nil {
		return err
	}
	u, err := source.ParseBase(s.SourceURL)
	if err != nil {
		return invalid("%v", err)
	}
	s.SourceURL = u.String()
	return s.normalizeDomains()
}

func (s *Site) checkFields() error {
	switch {
	case s.Name == "" || len(s.Name) > 64:
		return invalid("name")
	case !slices.Contains(siteKind, s.SourceKind):
		return invalid("source kind %q", s.SourceKind)
	case !pathRe.MatchString(s.OldPath):
		return invalid("old path %q", s.OldPath)
	case s.Port < 1 || s.Port > 65535:
		return invalid("port %d", s.Port)
	case s.SourcePin != "" && !pinRe.MatchString(s.SourcePin):
		return invalid("certificate pin")
	case (s.CertFile == "") != (s.KeyFile == ""):
		return invalid("a certificate file needs its key file")
	case s.CertFile != "" && s.CertPEM != "":
		return invalid("a certificate file or a pasted certificate, not both")
	}
	if s.SourceIP != "" {
		if a, err := netip.ParseAddr(s.SourceIP); err != nil || !source.Allowed(a) {
			return invalid("source address %q", s.SourceIP)
		}
	}
	return nil
}

func (s *Site) normalizeDomains() error {
	for i, d := range s.Domains {
		d = strings.ToLower(strings.TrimSuffix(strings.TrimSpace(d), "."))
		if !validPattern(d) {
			return invalid("domain %q", d)
		}
		s.Domains[i] = d
	}
	slices.Sort(s.Domains)
	s.Domains = slices.Compact(s.Domains)
	return nil
}

// validPattern accepts host names and one leading "*." wildcard.
func validPattern(d string) bool {
	if net.ParseIP(d) != nil {
		return true
	}
	d = strings.TrimPrefix(d, "*.")
	if d == "" || len(d) > 253 {
		return false
	}
	for _, l := range strings.Split(d, ".") {
		if !labelRe.MatchString(l) {
			return false
		}
	}
	return true
}

// matches reports whether host (no port, lower case) fits pattern. A
// wildcard covers exactly one label, like a certificate.
func matches(pattern, host string) bool {
	if rest, ok := strings.CutPrefix(pattern, "*."); ok {
		i := strings.IndexByte(host, '.')
		return i > 0 && host[i+1:] == rest
	}
	return pattern == host
}

// covers reports whether the site answers host. A site with no domains
// answers every host on its port.
func (s Site) covers(host string) bool {
	if len(s.Domains) == 0 {
		return true
	}
	return slices.ContainsFunc(s.Domains, func(p string) bool { return matches(p, host) })
}

// overlaps reports whether two sites could both claim one request.
func (s Site) overlaps(o Site) bool {
	if s.Port != o.Port || s.OldPath != o.OldPath {
		return false
	}
	if len(s.Domains) == 0 || len(o.Domains) == 0 {
		return true
	}
	for _, a := range s.Domains {
		for _, b := range o.Domains {
			if a == b || matches(a, strings.TrimPrefix(b, "*.")) || matches(b, strings.TrimPrefix(a, "*.")) {
				return true
			}
		}
	}
	return false
}
