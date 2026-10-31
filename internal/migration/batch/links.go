package batch

import (
	"net/url"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"
)

// Facts are what a run learns about the old links: the old path,
// hosts and port of the source users' subscription links, and how Karvan
// reached the old panel (the address it dialed and the certificate pin).
// None of them is secret.
type Facts struct {
	OldPath   string
	OldHosts  []string
	OldPort   int
	SourceIP  string
	SourcePin string
}

// maxHosts caps the old hosts a batch keeps.
const maxHosts = 16

// pathRe is the old path a panel rule accepts (bridge.Site).
var pathRe = regexp.MustCompile(`^[A-Za-z0-9_\-]{1,64}$`)

// OldLinks reads the old path, hosts and port from the source users'
// subscription links. A relative link is read against base, the old
// panel's address, which serves it. The path is kept only when every link
// is /<path>/<token> with one path segment a rule accepts, and the port
// only when every link has the same one; hosts are the distinct ones,
// sorted.
func OldLinks(base string, links []string) (path string, hosts []string, port int) {
	b, err := url.Parse(base)
	if err != nil {
		return "", nil, 0
	}
	paths, ports := map[string]bool{}, map[int]bool{}
	for _, raw := range links {
		u, ok := absLink(b, raw)
		if !ok {
			continue
		}
		if h := strings.ToLower(u.Hostname()); !slices.Contains(hosts, h) && len(hosts) < maxHosts {
			hosts = append(hosts, h)
		}
		paths[onePath(u)] = true
		ports[portOf(u)] = true
	}
	slices.Sort(hosts)
	if len(paths) == 1 {
		for p := range paths {
			path = p
		}
	}
	if len(ports) == 1 {
		for p := range ports {
			port = p
		}
	}
	return path, hosts, port
}

// absLink is a subscription link as a client sees it, or false when it
// is not a plain http or https link.
func absLink(base *url.URL, raw string) (*url.URL, bool) {
	if raw == "" {
		return nil, false
	}
	u, err := url.Parse(raw)
	if err != nil {
		return nil, false
	}
	u = base.ResolveReference(u)
	if u.Scheme != "http" && u.Scheme != "https" || u.Hostname() == "" || u.User != nil {
		return nil, false
	}
	return u, true
}

// onePath is the path segment before the token, or "".
func onePath(u *url.URL) string {
	parts := strings.Split(strings.Trim(u.EscapedPath(), "/"), "/")
	if len(parts) != 2 || parts[1] == "" || !pathRe.MatchString(parts[0]) {
		return ""
	}
	return parts[0]
}

func portOf(u *url.URL) int {
	if p, err := strconv.Atoi(u.Port()); err == nil {
		return p
	}
	if u.Scheme == "https" {
		return 443
	}
	return 80
}

// nameBase is the longest base name, so " 99999" still fits in 64.
const nameBase = 58

// Name is a batch's default name: the source admin's username,
// trimmed and cut to 58 characters, or "batch" when that is empty. It
// trims what the 0008 backfill trims.
func Name(admin string) string {
	s := strings.Trim(admin, " \t\n\v\f\r")
	if utf8.RuneCountInString(s) > nameBase {
		s = string([]rune(s)[:nameBase])
	}
	if s == "" {
		return "batch"
	}
	return s
}

// numbered is base, or base with " 2", " 3" when batches of the same
// old panel already use it: one more than the highest number taken.
func numbered(base string, taken []string) string {
	high := 0
	for _, n := range taken {
		if n == base {
			high = max(high, 1)
			continue
		}
		rest, ok := strings.CutPrefix(n, base+" ")
		if v, err := strconv.Atoi(rest); ok && err == nil && v > 1 && strconv.Itoa(v) == rest {
			high = max(high, v)
		}
	}
	if high == 0 {
		return base
	}
	return base + " " + strconv.Itoa(high+1)
}
