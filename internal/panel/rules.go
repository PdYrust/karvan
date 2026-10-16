package panel

import (
	"fmt"
	"regexp"
	"unicode"
)

// Rules are the destination's validation rules. Karvan enforces them itself,
// because database writes skip the panel's own API checks.
type Rules interface {
	// ValidateUsername reports why the destination would refuse name for a
	// user, or nil.
	ValidateUsername(name string) error
	// ValidateCredentials reports the first credential the destination would
	// refuse, or nil.
	ValidateCredentials(c Credentials) error
	// UserKey is the form under which the destination's unique index on user
	// names compares name (folded where the column is case-insensitive).
	UserKey(name string) string
}

// VersionError means Karvan has no tested writer for this database.
type VersionError struct {
	Kind     Kind
	Revision string
}

func (e *VersionError) Error() string {
	return fmt.Sprintf("%s revision %q is not supported", e.Kind, e.Revision)
}

// SchemaError means the database is at a known revision but its shape is
// not what that revision creates (a fork, a hand edit, an extra table that
// references users).
type SchemaError struct {
	Kind   Kind
	Detail string
}

func (e *SchemaError) Error() string { return fmt.Sprintf("%s schema: %s", e.Kind, e.Detail) }

// RuleError is a value the destination would refuse. Code is stable and
// machine-readable; the UI turns it into words.
type RuleError struct {
	Field string // "username", "vless.flow", "shadowsocks.password"...
	Code  string
}

func (e *RuleError) Error() string { return fmt.Sprintf("%s: %s", e.Field, e.Code) }

// Rule codes shared by both panels.
const (
	CodeUsernameChars   = "username_chars"
	CodeUsernameLength  = "username_length"
	CodeUsernameShape   = "username_shape"
	CodeUsernameControl = "username_control"
	CodeUUIDFormat      = "uuid_format"
	CodeFlow            = "flow_unsupported"
	CodeMethod          = "method_unsupported"
	CodePasswordShort   = "password_short"
	CodePasswordEmpty   = "password_empty"
)

var canonicalUUID = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

// IsCanonicalUUID reports whether u is lowercase, hyphenated hex.
func IsCanonicalUUID(u UUID) bool { return canonicalUUID.MatchString(string(u)) }

// HasControl reports whether s holds a control character. Both panels
// accept a trailing newline through Python's "$"; Karvan never does.
func HasControl(s string) bool {
	for _, r := range s {
		if unicode.IsControl(r) {
			return true
		}
	}
	return false
}

// ValidateUUIDs checks the VMess and VLESS ids that are present.
func ValidateUUIDs(c Credentials) error {
	if c.VMess != nil && !IsCanonicalUUID(c.VMess.ID) {
		return &RuleError{Field: "vmess.id", Code: CodeUUIDFormat}
	}
	if c.VLESS != nil && !IsCanonicalUUID(c.VLESS.ID) {
		return &RuleError{Field: "vless.id", Code: CodeUUIDFormat}
	}
	return nil
}
