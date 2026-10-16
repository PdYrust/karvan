package marzban

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/PdYrust/karvan/internal/panel"
	"github.com/PdYrust/karvan/internal/secret"
)

// Protocols in the order Marzban lists them. The API uses the value
// (vmess), the database stores the enum name (VMess).
var Protocols = []string{"vmess", "vless", "trojan", "shadowsocks"}

var typeNames = map[string]string{"vmess": "VMess", "vless": "VLESS", "trojan": "Trojan", "shadowsocks": "Shadowsocks"}

// TypeName is the proxies.type value for protocol.
func TypeName(protocol string) string { return typeNames[protocol] }

// ProtocolOf maps a proxies.type value back to its protocol.
func ProtocolOf(typeName string) (string, bool) {
	for p, n := range typeNames {
		if n == typeName {
			return p, true
		}
	}
	return "", false
}

// settings is one proxy's settings object, the same in the API and in the
// proxies.settings JSON column (app/models/proxy.py:72-104).
type settings struct {
	ID       string `json:"id,omitempty"`
	Flow     string `json:"flow,omitempty"`
	Password string `json:"password,omitempty"`
	Method   string `json:"method,omitempty"`
}

// SetProxy decodes raw (a settings object) for protocol into c.
func SetProxy(c *panel.Credentials, protocol string, raw []byte) error {
	var s settings
	if err := json.Unmarshal(raw, &s); err != nil {
		return fmt.Errorf("%s settings: %w", protocol, err)
	}
	switch protocol {
	case "vmess":
		c.VMess = &panel.VMess{ID: panel.CanonicalUUID(s.ID)}
	case "vless":
		c.VLESS = &panel.VLESS{ID: panel.CanonicalUUID(s.ID), Flow: s.Flow}
	case "trojan":
		c.Trojan = &panel.Trojan{Password: secret.NewValue(s.Password), Flow: s.Flow}
	case "shadowsocks":
		c.Shadowsocks = &panel.Shadowsocks{Password: secret.NewValue(s.Password), Method: s.Method}
	default:
		return fmt.Errorf("unknown protocol %q", protocol)
	}
	return nil
}

// ProxySettings encodes the settings object for protocol from c, or nil if
// c has no such protocol. Every field is written explicitly, because Marzban
// fills omitted ones with new random values.
func ProxySettings(c panel.Credentials, protocol string) ([]byte, error) {
	var s any
	switch protocol {
	case "vmess":
		if c.VMess == nil {
			return nil, nil
		}
		s = map[string]string{"id": string(c.VMess.ID)}
	case "vless":
		if c.VLESS == nil {
			return nil, nil
		}
		s = map[string]string{"id": string(c.VLESS.ID), "flow": c.VLESS.Flow}
	case "trojan":
		if c.Trojan == nil {
			return nil, nil
		}
		s = map[string]string{"password": c.Trojan.Password.Reveal(), "flow": c.Trojan.Flow}
	case "shadowsocks":
		if c.Shadowsocks == nil {
			return nil, nil
		}
		s = map[string]string{"password": c.Shadowsocks.Password.Reveal(), "method": c.Shadowsocks.Method}
	default:
		return nil, fmt.Errorf("unknown protocol %q", protocol)
	}
	return json.Marshal(s)
}

// HasProtocol reports whether c carries protocol.
func HasProtocol(c panel.Credentials, protocol string) bool {
	switch strings.ToLower(protocol) {
	case "vmess":
		return c.VMess != nil
	case "vless":
		return c.VLESS != nil
	case "trojan":
		return c.Trojan != nil
	case "shadowsocks":
		return c.Shadowsocks != nil
	}
	return false
}
