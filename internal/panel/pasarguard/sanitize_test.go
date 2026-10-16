package pasarguard

import (
	"strings"
	"testing"
)

func TestSanitizeAlwaysGivesAnAcceptedName(t *testing.T) {
	cases := map[string]string{
		"ali reza":                     "ali_reza",
		"a..b":                         "a.b",
		"ab":                           "ab0",
		"user:1":                       "user_1",
		"x" + strings.Repeat("a", 130): "x" + strings.Repeat("a", 127),
		"ok.name":                      "ok.name",
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
