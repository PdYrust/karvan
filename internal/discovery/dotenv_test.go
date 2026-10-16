package discovery

import (
	"testing"
)

func TestParseDotenv(t *testing.T) {
	src := `# Marzban installer style
SQLALCHEMY_DATABASE_URL = "mysql+pymysql://marzban:p#ss@127.0.0.1:3306/marzban"
UVICORN_PORT=8000
export XRAY_JSON = '/var/lib/marzban/xray_config.json'
XRAY_SUBSCRIPTION_URL_PREFIX = https://sub.example.com   # trailing comment
EMPTY=
SPACED =   value with spaces   
HASH_IN_VALUE=abc#def
QUOTED_ESCAPES="line1\nline2 \"q\""
SINGLE='no $EXPANSION \n here'
REF=${UVICORN_PORT}
REF_DEFAULT=${MISSING:-fallback}
MULTI="first
second"
# DUP=commented
DUP=one
DUP=two
not a valid line
`
	got := ParseDotenv([]byte(src))
	want := map[string]string{
		"SQLALCHEMY_DATABASE_URL":      "mysql+pymysql://marzban:p#ss@127.0.0.1:3306/marzban",
		"UVICORN_PORT":                 "8000",
		"XRAY_JSON":                    "/var/lib/marzban/xray_config.json",
		"XRAY_SUBSCRIPTION_URL_PREFIX": "https://sub.example.com",
		"EMPTY":                        "",
		"SPACED":                       "value with spaces",
		"HASH_IN_VALUE":                "abc#def",
		"QUOTED_ESCAPES":               "line1\nline2 \"q\"",
		"SINGLE":                       `no $EXPANSION \n here`,
		"REF":                          "8000",
		"REF_DEFAULT":                  "fallback",
		"MULTI":                        "first\nsecond",
		"DUP":                          "two",
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("%s = %q, want %q", k, got[k], v)
		}
	}
	if len(got) != len(want) {
		t.Errorf("extra keys: %v", got)
	}
}

func TestParseDotenvCRLF(t *testing.T) {
	got := ParseDotenv([]byte("A=1\r\nB = \"2\"\r\n"))
	if got["A"] != "1" || got["B"] != "2" {
		t.Fatalf("%v", got)
	}
}
