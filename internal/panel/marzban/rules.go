// Package marzban holds what Karvan knows about Marzban 0.8.4 that both the
// API source and the database destination need: validation rules and
// value formats.
package marzban

import (
	"strings"

	"github.com/PdYrust/karvan/internal/panel"
)

// UsernameColumnLen is users.username String(34). SQLite does not enforce
// it and MySQL strict mode errors, so Karvan enforces it on every backend.
const UsernameColumnLen = 34

// Flows and shadowsocks methods the API accepts (xray_api/types/account.py).
var (
	flows   = map[string]bool{"": true, "xtls-rprx-vision": true}
	methods = map[string]bool{"aes-128-gcm": true, "aes-256-gcm": true, "chacha20-ietf-poly1305": true}
)

// Accepts reports exactly what USERNAME_REGEXP (app/models/user.py:16)
// matches: ^(?=\w{3,32}\b)[a-zA-Z0-9-_@.]+(?:_[a-zA-Z0-9-_@.]+)*$
// The trailing group adds nothing, because the class already holds "_".
// Python's "$" also matches before one final newline.
func Accepts(name string) bool {
	body := strings.TrimSuffix(name, "\n")
	if body == "" {
		return false
	}
	run := -1
	for i := 0; i < len(body); i++ {
		c := body[i]
		if !allowed(c) {
			return false
		}
		if run < 0 && !word(c) {
			run = i
		}
	}
	if run < 0 {
		run = len(body)
	}
	return run >= 3 && run <= 32
}

func word(c byte) bool {
	return c == '_' || c >= '0' && c <= '9' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z'
}

func allowed(c byte) bool { return word(c) || c == '-' || c == '@' || c == '.' }

// Rules are Marzban's rules for one destination database.
type Rules struct {
	foldUsers bool
}

// NewRules returns the rules for a database whose users.username column is
// case-insensitive (SQLite NOCASE) or not (MySQL utf8mb4_bin).
func NewRules(caseInsensitiveUsers bool) Rules { return Rules{foldUsers: caseInsensitiveUsers} }

// ValidateUsername implements panel.Rules.
func (Rules) ValidateUsername(name string) error {
	switch {
	case panel.HasControl(name):
		return &panel.RuleError{Field: "username", Code: panel.CodeUsernameControl}
	case !charsAllowed(name):
		return &panel.RuleError{Field: "username", Code: panel.CodeUsernameChars}
	case !Accepts(name):
		return &panel.RuleError{Field: "username", Code: panel.CodeUsernameShape}
	case len(name) > UsernameColumnLen:
		return &panel.RuleError{Field: "username", Code: panel.CodeUsernameLength}
	}
	return nil
}

func charsAllowed(name string) bool {
	for i := 0; i < len(name); i++ {
		if !allowed(name[i]) {
			return false
		}
	}
	return name != ""
}

// UserKey implements panel.Rules. SQLite NOCASE folds ASCII letters only.
func (r Rules) UserKey(name string) string {
	if !r.foldUsers {
		return name
	}
	return asciiLower(name)
}

func asciiLower(s string) string {
	b := []byte(s)
	for i, c := range b {
		if c >= 'A' && c <= 'Z' {
			b[i] = c + 'a' - 'A'
		}
	}
	return string(b)
}

// ValidateCredentials implements panel.Rules.
func (Rules) ValidateCredentials(c panel.Credentials) error {
	if err := panel.ValidateUUIDs(c); err != nil {
		return err
	}
	if c.VLESS != nil && !flows[c.VLESS.Flow] {
		return &panel.RuleError{Field: "vless.flow", Code: panel.CodeFlow}
	}
	if c.Trojan != nil {
		if c.Trojan.Password.IsZero() {
			return &panel.RuleError{Field: "trojan.password", Code: panel.CodePasswordEmpty}
		}
		if !flows[c.Trojan.Flow] {
			return &panel.RuleError{Field: "trojan.flow", Code: panel.CodeFlow}
		}
	}
	if c.Shadowsocks != nil {
		if c.Shadowsocks.Password.IsZero() {
			return &panel.RuleError{Field: "shadowsocks.password", Code: panel.CodePasswordEmpty}
		}
		if !methods[c.Shadowsocks.Method] {
			return &panel.RuleError{Field: "shadowsocks.method", Code: panel.CodeMethod}
		}
	}
	return nil
}
