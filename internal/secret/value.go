package secret

import (
	"fmt"
	"log/slog"
)

const redacted = "[redacted]"

// Value wraps a secret string so printing, logging or JSON encoding it shows
// "[redacted]". Only Reveal returns the real value.
type Value struct {
	s string
}

// NewValue wraps s.
func NewValue(s string) Value { return Value{s: s} }

// Reveal returns the secret. Call it only where the secret is used.
func (v Value) Reveal() string { return v.s }

// IsZero reports whether the value is empty.
func (v Value) IsZero() bool { return v.s == "" }

// String implements fmt.Stringer.
func (Value) String() string { return redacted }

// GoString implements fmt.GoStringer, which covers %#v.
func (Value) GoString() string { return redacted }

// Format covers every fmt verb, including %q and %x.
func (Value) Format(f fmt.State, _ rune) { _, _ = f.Write([]byte(redacted)) }

// LogValue implements slog.LogValuer.
func (Value) LogValue() slog.Value { return slog.StringValue(redacted) }

// MarshalJSON keeps the secret out of any JSON response or log.
func (Value) MarshalJSON() ([]byte, error) { return []byte(`"` + redacted + `"`), nil }
