package bridge

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

// How a redirect rule matches its path (section 4).
const (
	MatchExact  = "exact"  // the path itself only
	MatchPrefix = "prefix" // the path and anything under it, passed on as a tail
)

// ReasonLoop means a redirect's target is a request a redirect rule of
// Karvan answers.
const ReasonLoop = "loop"

// Limits of a redirect rule (section 4).
const (
	maxRedirectPath = 2048
	maxRedirectURL  = 2048
	maxSegments     = 8
	maxRuleName     = 64
)

// segmentRe is one segment of a from path or a tail. Matching runs on the
// escaped path, so "%2f" and every other escape fail it.
var segmentRe = regexp.MustCompile(`^[A-Za-z0-9._~=-]{1,1000}$`)

// ErrInvalidRedirect means a redirect field is wrong; a
// *RedirectInvalidError says which.
var ErrInvalidRedirect = errors.New("invalid redirect")

// RedirectInvalidError names the wrong field by its JSON key, and the
// reason when there is one (ReasonLoop).
type RedirectInvalidError struct{ Field, Reason string }

func (e *RedirectInvalidError) Error() string {
	return fmt.Sprintf("%v: %s", ErrInvalidRedirect, e.Field)
}

// Is makes errors.Is(err, ErrInvalidRedirect) hold.
func (e *RedirectInvalidError) Is(target error) bool { return target == ErrInvalidRedirect }

func redirectInvalid(field string) error { return &RedirectInvalidError{Field: field} }

// Redirect is a redirect rule: a request on its port, domains and path is
// sent to ToURL with a 302 (D28, D29). Karvan never fetches the target (D30).
type Redirect struct {
	ID            int64    `json:"id"`
	Name          string   `json:"name"`
	Domains       []string `json:"domains"`
	Port          int      `json:"port"`
	FromPath      string   `json:"fromPath"`
	Match         string   `json:"match"`
	ToURL         string   `json:"toUrl"`
	CertificateID int64    `json:"certificateId"`
	cert          certRow  // its certificates row, when CertificateID is set
	target        *url.URL // ToURL parsed; nil when the stored value no longer passes the check
}

// RedirectView is a redirect with its certificate and warnings
// (domain_mismatch, cert_conflict).
type RedirectView struct {
	Redirect
	Certificate *SiteCertificate `json:"certificate"`
	Warnings    []string         `json:"warnings"`
}

// rule is the redirect as the router sees it.
func (d *Redirect) rule() rule {
	return rule{ref: RuleRef{Kind: RuleRedirect, ID: d.ID, Name: d.Name}, domains: d.Domains, port: d.Port,
		certID: d.CertificateID, tls: d.CertificateID != 0, path: strings.Trim(d.FromPath, "/"), exact: d.Match == MatchExact,
		redirect: d}
}

// normalize checks d and puts it in the stored form.
func (d *Redirect) normalize() error {
	name, ok := ruleName(d.Name)
	if !ok {
		return redirectInvalid("name")
	}
	d.Name = name
	s := Site{Domains: d.Domains}
	if s.normalizeDomains() != nil {
		return redirectInvalid("domains")
	}
	d.Domains = append([]string{}, s.Domains...)
	switch {
	case d.Port < 1 || d.Port > 65535:
		return redirectInvalid("port")
	case len(d.FromPath) > maxRedirectPath || !validSegments(strings.TrimPrefix(d.FromPath, "/")):
		return redirectInvalid("fromPath")
	case d.Match != MatchExact && d.Match != MatchPrefix:
		return redirectInvalid("match")
	}
	d.FromPath = "/" + strings.TrimPrefix(d.FromPath, "/")
	u, ok := parseTarget(d.ToURL)
	if !ok {
		return redirectInvalid("toUrl")
	}
	d.ToURL, d.target = u.String(), u
	return nil
}

// ruleName trims a rule name and checks it: 1 to 64 characters, no
// control characters.
func ruleName(s string) (string, bool) {
	s = strings.TrimSpace(s)
	n := utf8.RuneCountInString(s)
	return s, n >= 1 && n <= maxRuleName && utf8.ValidString(s) && !strings.ContainsFunc(s, unicode.IsControl)
}

// validSegments reports whether p is 1 to 8 "/"-separated segments of
// segmentRe, none of them "." or "..".
func validSegments(p string) bool {
	segs := strings.Split(p, "/")
	if len(segs) > maxSegments {
		return false
	}
	for _, s := range segs {
		if !segmentRe.MatchString(s) || s == "." || s == ".." {
			return false
		}
	}
	return true
}

// validTail reports whether the rest of a prefix match may be passed on:
// nothing, or "/" and valid segments.
func validTail(tail string) bool {
	rest, ok := strings.CutPrefix(tail, "/")
	return tail == "" || ok && validSegments(rest)
}

// parseTarget checks a redirect target: an absolute http or https URL with
// a host name or IP, no user info, query or fragment, at most 2048 bytes.
// The host is stored in lower case.
func parseTarget(raw string) (*url.URL, bool) {
	if len(raw) > maxRedirectURL || strings.ContainsAny(raw, `\?#`) {
		return nil, false
	}
	u, err := url.Parse(raw)
	if err != nil {
		return nil, false
	}
	host := strings.ToLower(u.Hostname())
	switch {
	case u.Scheme != "https" && u.Scheme != "http", u.Opaque != "", u.User != nil, strings.HasSuffix(u.Host, ":"):
		return nil, false
	case net.ParseIP(host) == nil && (strings.HasPrefix(host, "*.") || !validPattern(host)):
		return nil, false
	}
	if p := u.Port(); p != "" {
		if n, err := strconv.Atoi(p); err != nil || n < 1 || n > 65535 {
			return nil, false
		}
	}
	u.Host = strings.ToLower(u.Host)
	return u, len(u.String()) <= maxRedirectURL
}

// location is the target with the checked tail added to its path. Scheme
// and host are the stored ones only (D29).
func (d *Redirect) location(tail string) string {
	u := *d.target
	if tail != "" {
		p := strings.TrimSuffix(u.EscapedPath(), "/") + tail // a tail holds unreserved characters only
		u.Path, _ = url.PathUnescape(p)
		u.RawPath = p
	}
	return u.String()
}

// sendRedirect answers 302 to the redirect's target with an empty body.
func sendRedirect(w http.ResponseWriter, d *Redirect, tail string) {
	w.Header().Set("Location", d.location(tail))
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusFound)
}

// loops reports whether a request for target would reach a redirect rule
// in rules, by its host, port and escaped path. Nothing is looked up.
func loops(target *url.URL, rules []rule) bool {
	port := 443
	if target.Scheme == "http" {
		port = 80
	}
	if p := target.Port(); p != "" {
		port, _ = strconv.Atoi(p)
	}
	path := target.EscapedPath()
	if path == "" {
		path = "/"
	}
	t := table{}
	for _, r := range rules {
		t[r.port] = append(t[r.port], route{rule: r})
	}
	rt, _, ok := t.match(port, strings.ToLower(target.Hostname()), path)
	return ok && rt.ref.Kind == RuleRedirect
}
