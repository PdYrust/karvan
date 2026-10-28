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
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/PdYrust/karvan/internal/panel"
	"github.com/PdYrust/karvan/internal/panel/source"
)

// Site is one old subscription address the bridge answers: a panel rule.
type Site struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
	// Mode is ModeManual (the default) or ModeBatches, fixed when the rule
	// is made. A rule of batches takes PanelID, SourceKind, SourceURL,
	// SourcePin, and OldPath and SourceIP when they know them, from
	// BatchIDs (D56); the body may not send them.
	Mode        string     `json:"mode"`
	BatchIDs    []int64    `json:"batchIds,omitempty"`
	PanelID     int64      `json:"panelId"` // the destination panel
	SourceKind  panel.Kind `json:"sourceKind"`
	SourceURL   string     `json:"sourceUrl"`
	SourceIP    string     `json:"sourceIp,omitempty"`
	SourcePin   string     `json:"sourcePin,omitempty"`
	OldPath     string     `json:"oldPath"`
	Domains     []string   `json:"domains"`
	Port        int        `json:"port"`
	Lazy        bool       `json:"lazy"`
	PlainHTTPOK bool       `json:"plainHttpOk"`
	// HTTPS is whether the rule answers over HTTPS: in a body, nil keeps
	// the stored value on PATCH and takes the D63 default on POST; read
	// back, whether it has a certificate.
	HTTPS *bool `json:"https"`
	// CertificateID is the certificates row the site is served with, or 0:
	// plain HTTP, or its own certificate columns when not adopted (D54).
	// It is derived from the domains on every save (D63), never sent.
	CertificateID int64 `json:"-"`
	// The certificate the site is served with, read from its certificates
	// row, else from its own columns. Never sent or taken as JSON.
	CertFile string `json:"-"`
	KeyFile  string `json:"-"`
	CertPEM  string `json:"-"`
	KeyPEM   string `json:"-"`
	HasKey   bool   `json:"-"`
	certName string // the name of its certificates row
	origin   string
}

// How a panel rule picks the users it serves (D23).
const (
	ModeManual  = "manual"  // the users of its old panel and destination in an active batch or in none
	ModeBatches = "batches" // the users of its batches, while the batch is active
)

// TLS reports whether the site serves HTTPS.
func (s Site) TLS() bool { return s.CertFile != "" || s.CertPEM != "" }

var (
	pathRe  = regexp.MustCompile(`^[A-Za-z0-9_\-]{1,64}$`)
	labelRe = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?$`)
	pinRe   = regexp.MustCompile(`^[0-9a-f]{64}$`)
	// ErrInvalidSite means a site field is wrong; an *InvalidError says which.
	ErrInvalidSite = errors.New("invalid site")
	siteKind       = []panel.Kind{panel.Marzban, panel.PasarGuard}
)

// InvalidError names the first site field that is wrong, by its JSON key,
// the reason when there is one (Reason* constants) and, for domains, the
// names it is about.
type InvalidError struct {
	Field, Reason string
	Names         []string
}

func (e *InvalidError) Error() string { return fmt.Sprintf("%v: %s", ErrInvalidSite, e.Field) }

// Is makes errors.Is(err, ErrInvalidSite) hold.
func (e *InvalidError) Is(target error) bool { return target == ErrInvalidSite }

func invalid(field string) error { return &InvalidError{Field: field} }

// FieldOf is the field an invalid site error names, or "".
func FieldOf(err error) string {
	var e *InvalidError
	if errors.As(err, &e) {
		return e.Field
	}
	return ""
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
		return invalid("sourceUrl")
	}
	s.SourceURL = u.String()
	// Once the old domain points at this server, a lookup would reach the
	// bridge itself, so lazy checks need the address the migration used.
	if _, err := netip.ParseAddr(u.Hostname()); err != nil && s.Lazy && s.SourceIP == "" {
		return invalid("sourceIp") // lazy checks need the old panel's address
	}
	return s.normalizeDomains()
}

func (s *Site) checkFields() error {
	switch {
	case s.Name == "" || len(s.Name) > 64:
		return invalid("name")
	case !slices.Contains(siteKind, s.SourceKind):
		return invalid("sourceKind")
	case !pathRe.MatchString(s.OldPath):
		return invalid("oldPath")
	case s.Port < 1 || s.Port > 65535:
		return invalid("port")
	case s.SourcePin != "" && !pinRe.MatchString(s.SourcePin):
		return invalid("sourcePin")
	}
	if s.SourceIP != "" {
		if a, err := netip.ParseAddr(s.SourceIP); err != nil || !source.Allowed(a) {
			return invalid("sourceIp")
		}
	}
	return nil
}

// checkCert checks a certificate to inspect: files on this server
// (absolute paths, both), a pasted certificate with its key, or none.
func (s *Site) checkCert() error {
	switch {
	case s.CertFile != "" && s.CertPEM != "":
		return invalid("certPem") // a certificate file or a pasted certificate, not both
	case s.CertFile != "" && !filepath.IsAbs(s.CertFile):
		return invalid("certFile")
	case (s.CertFile == "") != (s.KeyFile == ""):
		return invalid("keyFile")
	case s.KeyFile != "" && !filepath.IsAbs(s.KeyFile):
		return invalid("keyFile")
	case s.KeyPEM != "" && s.CertPEM == "":
		return invalid("keyPem")
	}
	return nil
}

func (s *Site) normalizeDomains() error {
	s.Domains = slices.Clone(s.Domains) // never rewrite the caller's slice
	for i, d := range s.Domains {
		d = strings.ToLower(strings.TrimSuffix(strings.TrimSpace(d), "."))
		if !validPattern(d) {
			return invalid("domains")
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
