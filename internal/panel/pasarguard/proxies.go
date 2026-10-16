package pasarguard

import (
	"encoding/json"
	"fmt"

	"github.com/PdYrust/karvan/internal/panel"
	"github.com/PdYrust/karvan/internal/secret"
)

// proxySettings is users.proxy_settings and the API's proxy_settings
// (app/models/proxy.py:12-112). Pointers tell an absent protocol from an
// empty one.
type proxySettings struct {
	VMess       *idSettings       `json:"vmess,omitempty"`
	VLESS       *idSettings       `json:"vless,omitempty"`
	Trojan      *passwordSettings `json:"trojan,omitempty"`
	Shadowsocks *ssSettings       `json:"shadowsocks,omitempty"`
	WireGuard   *wgSettings       `json:"wireguard,omitempty"`
	Hysteria    *hySettings       `json:"hysteria,omitempty"`
}

type idSettings struct {
	ID string `json:"id"`
	// Flow is only present in data upgraded from Marzban; PasarGuard
	// ignores it and takes flow from the inbound.
	Flow string `json:"flow,omitempty"`
}

type passwordSettings struct {
	Password string `json:"password"`
}

type ssSettings struct {
	Password string `json:"password"`
	Method   string `json:"method"`
}

type wgSettings struct {
	PrivateKey string   `json:"private_key"`
	PublicKey  string   `json:"public_key"`
	PeerIPs    []string `json:"peer_ips"`
}

type hySettings struct {
	Auth string `json:"auth"`
}

// DecodeProxySettings reads a proxy_settings object.
func DecodeProxySettings(raw []byte) (panel.Credentials, error) {
	var s proxySettings
	if len(raw) == 0 {
		return panel.Credentials{}, nil
	}
	if err := json.Unmarshal(raw, &s); err != nil {
		return panel.Credentials{}, fmt.Errorf("proxy_settings: %w", err)
	}
	var c panel.Credentials
	if s.VMess != nil {
		c.VMess = &panel.VMess{ID: panel.CanonicalUUID(s.VMess.ID)}
	}
	if s.VLESS != nil {
		c.VLESS = &panel.VLESS{ID: panel.CanonicalUUID(s.VLESS.ID), Flow: s.VLESS.Flow}
	}
	if s.Trojan != nil {
		c.Trojan = &panel.Trojan{Password: secret.NewValue(s.Trojan.Password)}
	}
	if s.Shadowsocks != nil {
		c.Shadowsocks = &panel.Shadowsocks{Password: secret.NewValue(s.Shadowsocks.Password), Method: s.Shadowsocks.Method}
	}
	if s.WireGuard != nil {
		c.WireGuard = &panel.WireGuard{PrivateKey: secret.NewValue(s.WireGuard.PrivateKey),
			PublicKey: s.WireGuard.PublicKey, PeerIPs: s.WireGuard.PeerIPs}
	}
	if s.Hysteria != nil {
		c.Hysteria = &panel.Hysteria{Auth: secret.NewValue(s.Hysteria.Auth)}
	}
	return c, nil
}

// EncodeProxySettings writes c as a proxy_settings object. VLESS flow is
// left out: PasarGuard has no per-user flow (conflict C11).
func EncodeProxySettings(c panel.Credentials) ([]byte, error) {
	var s proxySettings
	if c.VMess != nil {
		s.VMess = &idSettings{ID: string(c.VMess.ID)}
	}
	if c.VLESS != nil {
		s.VLESS = &idSettings{ID: string(c.VLESS.ID)}
	}
	if c.Trojan != nil {
		s.Trojan = &passwordSettings{Password: c.Trojan.Password.Reveal()}
	}
	if c.Shadowsocks != nil {
		s.Shadowsocks = &ssSettings{Password: c.Shadowsocks.Password.Reveal(), Method: c.Shadowsocks.Method}
	}
	if c.WireGuard != nil {
		ips := c.WireGuard.PeerIPs
		if ips == nil {
			ips = []string{}
		}
		s.WireGuard = &wgSettings{PrivateKey: c.WireGuard.PrivateKey.Reveal(), PublicKey: c.WireGuard.PublicKey, PeerIPs: ips}
	}
	if c.Hysteria != nil {
		s.Hysteria = &hySettings{Auth: c.Hysteria.Auth.Reveal()}
	}
	return json.Marshal(s)
}
