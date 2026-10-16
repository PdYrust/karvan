package run

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"

	"github.com/PdYrust/karvan/internal/panel"
	"github.com/PdYrust/karvan/internal/secret"
)

// generate makes fresh credentials for one protocol the way both panels'
// defaults do: uuid4 ids and secrets.token_urlsafe(24) passwords (32
// characters, long enough for PasarGuard's shadowsocks rule). Only used
// for fixes the operator chose (D20, D25).
func generate(protocol string) panel.Credentials {
	switch protocol {
	case "vmess":
		return panel.Credentials{VMess: &panel.VMess{ID: uuid4()}}
	case "vless":
		return panel.Credentials{VLESS: &panel.VLESS{ID: uuid4()}}
	case "trojan":
		return panel.Credentials{Trojan: &panel.Trojan{Password: token()}}
	case "shadowsocks":
		return panel.Credentials{Shadowsocks: &panel.Shadowsocks{Password: token(), Method: "chacha20-ietf-poly1305"}}
	case "hysteria":
		return panel.Credentials{Hysteria: &panel.Hysteria{Auth: token()}}
	}
	return panel.Credentials{}
}

func uuid4() panel.UUID {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	b[6] = b[6]&0x0f | 0x40
	b[8] = b[8]&0x3f | 0x80
	return panel.UUID(fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16]))
}

func token() secret.Value {
	b := make([]byte, 24)
	_, _ = rand.Read(b)
	return secret.NewValue(base64.RawURLEncoding.EncodeToString(b))
}
