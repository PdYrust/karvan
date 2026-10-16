package pasarguard

import (
	"strings"
	"testing"
)

func TestProxySettingsRoundTrip(t *testing.T) {
	raw := `{"vmess":{"id":"3F2504E0-4F89-11D3-9A0C-0305E82C3301"},"vless":{"id":"3f2504e0-4f89-11d3-9a0c-0305e82c3302","flow":"xtls-rprx-vision"},` +
		`"trojan":{"password":"tp"},"shadowsocks":{"password":"` + strings.Repeat("s", 22) + `","method":"aes-128-gcm"},` +
		`"wireguard":{"private_key":"priv","public_key":"pub","peer_ips":["10.0.0.2/32"]},"hysteria":{"auth":"ha"}}`
	c, err := DecodeProxySettings([]byte(raw))
	if err != nil {
		t.Fatal(err)
	}
	if c.VMess.ID != "3f2504e0-4f89-11d3-9a0c-0305e82c3301" || c.VLESS.Flow != "xtls-rprx-vision" ||
		c.WireGuard.PeerIPs[0] != "10.0.0.2/32" || c.Hysteria.Auth.Reveal() != "ha" {
		t.Fatalf("decode: %+v", c)
	}
	out, err := EncodeProxySettings(c)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(out), "flow") {
		t.Error("PasarGuard has no per-user flow; it must not be written")
	}
	back, err := DecodeProxySettings(out)
	if err != nil {
		t.Fatal(err)
	}
	if back.Trojan.Password.Reveal() != "tp" || back.Shadowsocks.Method != "aes-128-gcm" || back.WireGuard.PublicKey != "pub" {
		t.Fatalf("round trip: %+v", back)
	}
}

func TestDecodeEmptyProxySettings(t *testing.T) {
	for _, raw := range []string{"", "{}"} {
		c, err := DecodeProxySettings([]byte(raw))
		if err != nil || len(c.Keys()) != 0 {
			t.Errorf("%q: %+v %v", raw, c, err)
		}
	}
}
