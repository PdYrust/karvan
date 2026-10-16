package marzban

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

func TestAcceptsMatchesPanelRegex(t *testing.T) {
	for _, v := range loadVectors(t) {
		if got := Accepts(v.Name); got != v.OK {
			t.Errorf("Accepts(%q) = %v, panel says %v", v.Name, got, v.OK)
		}
	}
}

func TestValidateUsernameIsStricterThanPanel(t *testing.T) {
	r := NewRules(false)
	for _, v := range loadVectors(t) {
		err := r.ValidateUsername(v.Name)
		if err == nil && !v.OK {
			t.Errorf("ValidateUsername(%q) accepted a name the panel refuses", v.Name)
		}
	}
	cases := map[string]string{
		"abc\n": panel.CodeUsernameControl,
		strings.Repeat("a", 3) + "." + strings.Repeat("b", 31): panel.CodeUsernameLength,
		"ab":    panel.CodeUsernameShape,
		"abc d": panel.CodeUsernameChars,
	}
	for name, code := range cases {
		var re *panel.RuleError
		if err := r.ValidateUsername(name); !errors.As(err, &re) || re.Code != code {
			t.Errorf("ValidateUsername(%q) = %v, want %s", name, err, code)
		}
	}
	if err := r.ValidateUsername("ali.reza_2"); err != nil {
		t.Errorf("valid name refused: %v", err)
	}
}

func TestUserKeyFollowsCollation(t *testing.T) {
	if NewRules(true).UserKey("Ali") != NewRules(true).UserKey("aLI") {
		t.Error("NOCASE column must fold case")
	}
	if NewRules(false).UserKey("Ali") == NewRules(false).UserKey("ali") {
		t.Error("binary column must keep case")
	}
}

func TestValidateCredentials(t *testing.T) {
	id := panel.CanonicalUUID("3f2504e0-4f89-11d3-9a0c-0305e82c3301")
	r := NewRules(false)
	good := panel.Credentials{
		VMess:       &panel.VMess{ID: id},
		VLESS:       &panel.VLESS{ID: id, Flow: "xtls-rprx-vision"},
		Trojan:      &panel.Trojan{Password: secret.NewValue("p"), Flow: ""},
		Shadowsocks: &panel.Shadowsocks{Password: secret.NewValue("short"), Method: "chacha20-ietf-poly1305"},
	}
	if err := r.ValidateCredentials(good); err != nil {
		t.Fatalf("valid credentials refused: %v", err)
	}
	bad := map[string]panel.Credentials{
		panel.CodeUUIDFormat:    {VMess: &panel.VMess{ID: "3F2504E0"}},
		panel.CodeFlow:          {VLESS: &panel.VLESS{ID: id, Flow: "xtls-rprx-direct"}},
		panel.CodeMethod:        {Shadowsocks: &panel.Shadowsocks{Password: secret.NewValue("p"), Method: "xchacha20-poly1305"}},
		panel.CodePasswordEmpty: {Trojan: &panel.Trojan{}},
	}
	for code, c := range bad {
		var re *panel.RuleError
		if err := r.ValidateCredentials(c); !errors.As(err, &re) || re.Code != code {
			t.Errorf("want %s, got %v", code, err)
		}
	}
}
