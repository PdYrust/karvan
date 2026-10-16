package secret

import (
	"encoding/base64"
	"strings"
	"testing"
)

func TestRandomStringUsesOnlyTheAlphabet(t *testing.T) {
	s, err := RandomString("ab", 64)
	if err != nil || len(s) != 64 || strings.Trim(s, "ab") != "" {
		t.Fatalf("RandomString = %q, %v", s, err)
	}
	if !strings.Contains(s, "a") || !strings.Contains(s, "b") {
		t.Fatalf("64 draws from two letters gave only one: %q", s)
	}
}

func TestNewTokenIs256Bits(t *testing.T) {
	a, err := NewToken()
	if err != nil {
		t.Fatal(err)
	}
	raw, err := base64.RawURLEncoding.DecodeString(a)
	if err != nil || len(raw) != 32 {
		t.Fatalf("token %q decodes to %d bytes, %v", a, len(raw), err)
	}
	if b, _ := NewToken(); a == b {
		t.Fatal("two tokens are equal")
	}
}
