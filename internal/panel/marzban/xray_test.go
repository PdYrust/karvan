package marzban

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/PdYrust/karvan/internal/panel"
	"github.com/PdYrust/karvan/internal/secret"
)

const xrayConfig = `{
  // Marzban's installer writes comments like this
  "log": {"loglevel": "warning"}, # and like this
  "inbounds": [
    {"tag": "VLESS TCP REALITY", "protocol": "vless", "port": 443},
    {"tag": "VMess # WS", "protocol": "vmess", "port": 8080},
    {"tag": "Shadowsocks TCP", "protocol": "shadowsocks", "port": 1080},
    {"tag": "api", "protocol": "dokodemo-door", "port": 62789},
    {"tag": "hidden", "protocol": "trojan", "port": 2083},
    {"tag": "url // not a comment", "protocol": "trojan", "port": 2096}
  ]
}`

func TestInboundsFollowsMarzban(t *testing.T) {
	p := filepath.Join(t.TempDir(), "xray_config.json")
	if err := os.WriteFile(p, []byte(xrayConfig), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := Inbounds(p, []string{"hidden"})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"shadowsocks/Shadowsocks TCP", "trojan/url // not a comment", "vless/VLESS TCP REALITY", "vmess/VMess # WS"}
	if len(got) != len(want) {
		t.Fatalf("got %+v", got)
	}
	for i, w := range want {
		if got[i].ID != w {
			t.Errorf("target %d = %q, want %q", i, got[i].ID, w)
		}
	}
}

func TestProxySettingsRoundTrip(t *testing.T) {
	in := panel.Credentials{
		VMess:       &panel.VMess{ID: "3f2504e0-4f89-11d3-9a0c-0305e82c3301"},
		VLESS:       &panel.VLESS{ID: "3f2504e0-4f89-11d3-9a0c-0305e82c3302", Flow: "xtls-rprx-vision"},
		Trojan:      &panel.Trojan{Password: secret.NewValue("t-pass")},
		Shadowsocks: &panel.Shadowsocks{Password: secret.NewValue("s-pass"), Method: "aes-256-gcm"},
	}
	var out panel.Credentials
	for _, p := range Protocols {
		raw, err := ProxySettings(in, p)
		if err != nil {
			t.Fatal(err)
		}
		if err := SetProxy(&out, p, raw); err != nil {
			t.Fatal(err)
		}
	}
	if out.VMess.ID != in.VMess.ID || out.VLESS.Flow != in.VLESS.Flow ||
		out.Trojan.Password.Reveal() != "t-pass" || out.Shadowsocks.Method != "aes-256-gcm" {
		t.Fatalf("round trip lost data: %+v", out)
	}
	if n, ok := ProtocolOf(TypeName("shadowsocks")); !ok || n != "shadowsocks" {
		t.Fatal("type name mapping is not symmetric")
	}
}
