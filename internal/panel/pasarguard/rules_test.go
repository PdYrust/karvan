package pasarguard

import (
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/PdYrust/karvan/internal/panel"
	"github.com/PdYrust/karvan/internal/secret"
)

type vector struct {
	Name string `json:"name"`
	OK   bool   `json:"ok"`
}

func loadVectors(t *testing.T) []vector {
	t.Helper()
	b, err := os.ReadFile("testdata/usernames.json")
	if err != nil {
		t.Fatal(err)
	}
	var v []vector
	if err := json.Unmarshal(b, &v); err != nil {
		t.Fatal(err)
	}
	return v
}

func TestAcceptsMatchesPanelValidator(t *testing.T) {
	for _, v := range loadVectors(t) {
		if got := Accepts(v.Name); got != v.OK {
			t.Errorf("Accepts(%q) = %v, panel says %v", v.Name, got, v.OK)
		}
	}
}

func TestValidateUsername(t *testing.T) {
	r := Rules{}
	for _, v := range loadVectors(t) {
		if r.ValidateUsername(v.Name) == nil && !v.OK {
			t.Errorf("ValidateUsername(%q) accepted a name the panel refuses", v.Name)
		}
	}
	cases := map[string]string{
		"abc\n":                  panel.CodeUsernameControl,
		"ab":                     panel.CodeUsernameLength,
		strings.Repeat("a", 129): panel.CodeUsernameLength,
		"ab..c":                  panel.CodeUsernameShape,
		"ab c":                   panel.CodeUsernameChars,
	}
	for name, code := range cases {
		var re *panel.RuleError
		if err := r.ValidateUsername(name); !errors.As(err, &re) || re.Code != code {
			t.Errorf("ValidateUsername(%q) = %v, want %s", name, err, code)
		}
	}
	if r.UserKey("Ali") == r.UserKey("ali") {
		t.Error("PasarGuard usernames are case-sensitive on every database")
	}
}

func TestValidateCredentials(t *testing.T) {
	id := panel.CanonicalUUID("3f2504e0-4f89-11d3-9a0c-0305e82c3301")
	long := secret.NewValue(strings.Repeat("x", 22))
	good := panel.Credentials{
		VMess:       &panel.VMess{ID: id},
		VLESS:       &panel.VLESS{ID: id},
		Trojan:      &panel.Trojan{Password: secret.NewValue("p")},
		Shadowsocks: &panel.Shadowsocks{Password: long, Method: "xchacha20-poly1305"},
		Hysteria:    &panel.Hysteria{Auth: secret.NewValue("a")},
	}
	if err := (Rules{}).ValidateCredentials(good); err != nil {
		t.Fatalf("valid credentials refused: %v", err)
	}
	bad := map[string]panel.Credentials{
		panel.CodeUUIDFormat:    {VLESS: &panel.VLESS{ID: "not-a-uuid"}},
		panel.CodePasswordShort: {Shadowsocks: &panel.Shadowsocks{Password: secret.NewValue(strings.Repeat("x", 21)), Method: "aes-128-gcm"}},
		panel.CodeMethod:        {Shadowsocks: &panel.Shadowsocks{Password: long, Method: "rc4-md5"}},
		panel.CodePasswordEmpty: {Hysteria: &panel.Hysteria{}},
	}
	for code, c := range bad {
		var re *panel.RuleError
		if err := (Rules{}).ValidateCredentials(c); !errors.As(err, &re) || re.Code != code {
			t.Errorf("want %s, got %v", code, err)
		}
	}
}
