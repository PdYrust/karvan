// Package pasarguard holds what Karvan knows about PasarGuard 5.4.1 that both
// the API source and the database destination need: validation rules and
// value formats.
package pasarguard

import (
	"strings"
	"unicode/utf8"

	"github.com/PdYrust/karvan/internal/panel"
)

// Limits from app/models/validators.py and app/models/proxy.py.
const (
	UsernameMinLen       = 3
	UsernameMaxLen       = 128
	MinShadowsocksLength = 22
)

var methods = map[string]bool{
	"aes-128-gcm": true, "aes-256-gcm": true, "chacha20-ietf-poly1305": true, "xchacha20-poly1305": true,
}

// Accepts reports exactly what validate_username (app/models/validators.py:207)
// allows: 3 to 128 characters, ^[a-zA-Z0-9-_@.]+$, and no two of -_@. in a
// row. The length counts code points, and Python's "$" also matches before
// one final newline.
func Accepts(name string) bool {
	n := utf8.RuneCountInString(name)
	if n < UsernameMinLen || n > UsernameMaxLen {
		return false
	}
	body := strings.TrimSuffix(name, "\n")
	if body == "" || !charsAllowed(body) {
		return false
	}
	return !consecutiveSpecials(body)
}

func special(c byte) bool { return c == '-' || c == '_' || c == '@' || c == '.' }

func allowed(c byte) bool {
	return special(c) || c >= '0' && c <= '9' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z'
}

func charsAllowed(s string) bool {
	for i := 0; i < len(s); i++ {
		if !allowed(s[i]) {
			return false
		}
	}
	return s != ""
}

func consecutiveSpecials(s string) bool {
	for i := 1; i < len(s); i++ {
		if special(s[i]) && special(s[i-1]) {
			return true
		}
	}
	return false
}

// Sanitize turns name into one PasarGuard accepts, changing as little as
// possible: other characters become "_", runs of -_@. keep their first
// character, short names are padded and long ones cut. It returns "" when
// nothing sensible is left.
func Sanitize(name string) string {
	out := make([]byte, 0, len(name))
	for i := 0; i < len(name); i++ {
		c := name[i]
		if c >= 0x80 || !allowed(c) {
			c = '_'
		}
		if special(c) && len(out) > 0 && special(out[len(out)-1]) {
			continue
		}
		out = append(out, c)
	}
	for len(out) < UsernameMinLen {
		out = append(out, '0')
	}
	if len(out) > UsernameMaxLen {
		out = out[:UsernameMaxLen]
	}
	s := string(out)
	if !Accepts(s) {
		return ""
	}
	return s
}

// Rules are PasarGuard's rules. Usernames are case-sensitive on every
// database (CaseSensitiveString in app/db/compiles_types.py).
type Rules struct{}

// ValidateUsername implements panel.Rules.
func (Rules) ValidateUsername(name string) error {
	n := utf8.RuneCountInString(name)
	switch {
	case panel.HasControl(name):
		return &panel.RuleError{Field: "username", Code: panel.CodeUsernameControl}
	case n < UsernameMinLen || n > UsernameMaxLen:
		return &panel.RuleError{Field: "username", Code: panel.CodeUsernameLength}
	case !charsAllowed(name):
		return &panel.RuleError{Field: "username", Code: panel.CodeUsernameChars}
	case consecutiveSpecials(name):
		return &panel.RuleError{Field: "username", Code: panel.CodeUsernameShape}
	}
	return nil
}

// UserKey implements panel.Rules.
func (Rules) UserKey(name string) string { return name }

// ValidateCredentials implements panel.Rules. VLESS flow is not checked:
// PasarGuard has no per-user flow and drops it (conflict C11, not an error).
func (Rules) ValidateCredentials(c panel.Credentials) error {
	if err := panel.ValidateUUIDs(c); err != nil {
		return err
	}
	if c.Trojan != nil && c.Trojan.Password.IsZero() {
		return &panel.RuleError{Field: "trojan.password", Code: panel.CodePasswordEmpty}
	}
	if ss := c.Shadowsocks; ss != nil {
		if utf8.RuneCountInString(ss.Password.Reveal()) < MinShadowsocksLength {
			return &panel.RuleError{Field: "shadowsocks.password", Code: panel.CodePasswordShort}
		}
		if !methods[ss.Method] {
			return &panel.RuleError{Field: "shadowsocks.method", Code: panel.CodeMethod}
		}
	}
	if c.Hysteria != nil && c.Hysteria.Auth.IsZero() {
		return &panel.RuleError{Field: "hysteria.auth", Code: panel.CodePasswordEmpty}
	}
	return nil
}
