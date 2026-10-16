// Package discovery finds Marzban and PasarGuard panels on this server by
// reading their ENV files (never writing them) and probing their
// databases read only, and reports what was found and what is missing.
package discovery

import (
	"regexp"
	"strings"
)

var (
	keyLine   = regexp.MustCompile(`^\s*(?:export\s+)?([A-Za-z_][A-Za-z0-9_.]*)\s*=\s*`)
	expansion = regexp.MustCompile(`\$\{([A-Za-z_][A-Za-z0-9_]*)(?::-([^}]*))?\}`)
)

// ParseDotenv reads an ENV file the way python-dotenv does for both panels:
// optional export, spaces around "=", single quotes (literal), double
// quotes (escapes, multi-line), unquoted values with " #" comments, and
// ${VAR} / ${VAR:-default} expansion from earlier keys. Later keys win.
// Unlike python-dotenv, Karvan does not expand from its own environment.
func ParseDotenv(src []byte) map[string]string {
	s := strings.ReplaceAll(string(src), "\r\n", "\n")
	out := map[string]string{}
	for len(s) > 0 {
		line, rest, _ := strings.Cut(s, "\n")
		m := keyLine.FindStringSubmatch(line)
		if m == nil {
			s = rest
			continue
		}
		key := m[1]
		value, remaining := readValue(s[len(m[0]):])
		switch {
		case strings.HasPrefix(value, "\x00'"):
			value = value[2:]
		default:
			value = expand(strings.TrimPrefix(value, "\x00\""), out)
		}
		out[key] = value
		s = remaining
	}
	return out
}

// readValue reads one value from the start of s and returns it with the
// rest of the input after its line. Quoted values are tagged with a NUL
// and their quote, so the caller knows whether to expand.
func readValue(s string) (string, string) {
	if len(s) > 0 && (s[0] == '"' || s[0] == '\'') {
		q := s[0]
		var b strings.Builder
		for i := 1; i < len(s); i++ {
			c := s[i]
			if c == '\\' && i+1 < len(s) && (q == '"' || s[i+1] == '\'' || s[i+1] == '\\') {
				b.WriteString(unescape(s[i+1], q))
				i++
				continue
			}
			if c == q {
				_, rest, _ := strings.Cut(s[i+1:], "\n")
				return "\x00" + string(q) + b.String(), rest
			}
			b.WriteByte(c)
		}
		// Unterminated quote: python-dotenv drops the line.
		_, rest, _ := strings.Cut(s, "\n")
		return "", rest
	}
	line, rest, _ := strings.Cut(s, "\n")
	if i := strings.Index(line, " #"); i >= 0 {
		line = line[:i]
	} else if i := strings.Index(line, "\t#"); i >= 0 {
		line = line[:i]
	}
	return strings.TrimSpace(line), rest
}

func unescape(c, quote byte) string {
	if quote == '\'' {
		return string(c)
	}
	switch c {
	case 'n':
		return "\n"
	case 't':
		return "\t"
	case 'r':
		return "\r"
	}
	return string(c)
}

func expand(v string, env map[string]string) string {
	return expansion.ReplaceAllStringFunc(v, func(m string) string {
		p := expansion.FindStringSubmatch(m)
		if val, ok := env[p[1]]; ok && val != "" {
			return val
		}
		return p[2]
	})
}
