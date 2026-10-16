package marzban

import (
	"strings"
	"testing"
)

func TestSanitizeAlwaysGivesAnAcceptedName(t *testing.T) {
	cases := map[string]string{
		"ali reza":              "ali_reza",
		"a.b":                   "a_b",
		"ab":                    "ab_",
		"علی":                   "___",
		"user:1":                "user_1",
		strings.Repeat("a", 40): strings.Repeat("a", 32),
		"ok.name":               "ok_name",
		"okay.name":             "okay.name",
	}
	for in, want := range cases {
		if got := Sanitize(in); got != want {
			t.Errorf("Sanitize(%q) = %q, want %q", in, got, want)
		}
	}
	for _, v := range loadVectors(t) {
		if s := Sanitize(v.Name); s != "" && !Accepts(s) {
			t.Errorf("Sanitize(%q) = %q is still refused", v.Name, s)
		}
	}
}
