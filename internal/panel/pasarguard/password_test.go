package pasarguard

import (
	"strings"
	"testing"
)

func TestCheckPassword(t *testing.T) {
	cases := map[string]string{
		"Aa1!Aa1!":                 PasswordShort,
		"aaaaaaaaaaaa":             PasswordWeak,
		"AbcdefGHij12":             PasswordWeak, // no special
		"Shop12!!Pass":             PasswordHasName,
		`Ab12"Cd34!xy`:             PasswordHasQuote,
		strings.Repeat("Ab1!", 19): PasswordLong,
		"Good.Pass12word":          "",
	}
	for pw, want := range cases {
		if got := CheckPassword(pw, "shop"); got != want {
			t.Errorf("CheckPassword(%q) = %q, want %q", pw, got, want)
		}
	}
}

func TestGeneratedPasswordsMeetThePolicy(t *testing.T) {
	seen := map[string]bool{}
	for range 200 {
		p, err := GeneratePassword("shop")
		if err != nil {
			t.Fatal(err)
		}
		if code := CheckPassword(p, "shop"); code != "" || seen[p] {
			t.Fatalf("%q: %q (repeat %v)", p, code, seen[p])
		}
		seen[p] = true
	}
}
