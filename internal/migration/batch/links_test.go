package batch

import (
	"slices"
	"strings"
	"testing"
)

func TestName(t *testing.T) {
	cases := map[string]string{
		"alice":                 "alice",
		" \talice\n":            "alice",
		"":                      "batch",
		" \t ":                  "batch",
		strings.Repeat("a", 70): strings.Repeat("a", 58),
		strings.Repeat("ب", 70): strings.Repeat("ب", 58),
	}
	for in, want := range cases {
		if got := Name(in); got != want {
			t.Errorf("Name(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestNumbered(t *testing.T) {
	cases := []struct {
		taken []string
		want  string
	}{
		{nil, "alice"},
		{[]string{"bob"}, "alice"},
		{[]string{"alice"}, "alice 2"},
		{[]string{"alice", "alice 2"}, "alice 3"},
		// A renamed first batch leaves its number free, but later ones
		// still count up from the highest.
		{[]string{"alice 2"}, "alice 3"},
		{[]string{"alice", "alice x", "alice 02x"}, "alice 2"},
	}
	for _, c := range cases {
		if got := numbered("alice", c.taken); got != c.want {
			t.Errorf("numbered(%v) = %q, want %q", c.taken, got, c.want)
		}
	}
}

func TestOldLinks(t *testing.T) {
	cases := []struct {
		name  string
		base  string
		links []string
		path  string
		hosts []string
		port  int
	}{
		{"one host", "https://panel.example", []string{"https://Sub.Example/sub/a", "https://sub.example/sub/b/"}, "sub", []string{"sub.example"}, 443},
		{"relative links use the panel", "http://10.0.0.5:8000", []string{"/sub/a", "/sub/b"}, "sub", []string{"10.0.0.5"}, 8000},
		{"plain http", "https://p.example", []string{"http://s.example/sub/a"}, "sub", []string{"s.example"}, 80},
		{"several hosts", "https://p.example", []string{"https://b.example/s/1", "https://a.example/s/2", "https://b.example/s/3"}, "s", []string{"a.example", "b.example"}, 443},
		{"two path segments", "https://p.example", []string{"https://s.example/api/sub/a"}, "", []string{"s.example"}, 443},
		{"mixed paths", "https://p.example", []string{"https://s.example/sub/a", "https://s.example/feed/b"}, "", []string{"s.example"}, 443},
		{"a path the rule refuses", "https://p.example", []string{"https://s.example/s.b/a"}, "", []string{"s.example"}, 443},
		{"mixed ports", "https://p.example", []string{"https://s.example/sub/a", "https://s.example:8443/sub/b"}, "sub", []string{"s.example"}, 0},
		{"no links", "https://p.example", []string{"", ""}, "", nil, 0},
		{"no token", "https://p.example", []string{"https://s.example/sub"}, "", []string{"s.example"}, 443},
		{"junk", "https://p.example", []string{"ftp://s.example/sub/a", "https://u:p@s.example/sub/a", "::bad"}, "", nil, 0},
	}
	for _, c := range cases {
		path, hosts, port := OldLinks(c.base, c.links)
		if path != c.path || !slices.Equal(hosts, c.hosts) || port != c.port {
			t.Errorf("%s: got %q %v %d, want %q %v %d", c.name, path, hosts, port, c.path, c.hosts, c.port)
		}
	}
}

// A panel that sends a link per user on its own host gives at most
// maxHosts hosts, so a batch row stays small.
func TestOldLinksCapsTheHosts(t *testing.T) {
	var links []string
	for i := range 3 * maxHosts {
		links = append(links, "https://h"+strings.Repeat("x", i)+".example/sub/t")
	}
	if _, hosts, _ := OldLinks("https://p.example", links); len(hosts) != maxHosts {
		t.Fatalf("%d hosts", len(hosts))
	}
}
