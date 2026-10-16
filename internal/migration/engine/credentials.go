package engine

import "github.com/PdYrust/karvan/internal/panel"

// replaceCredential swaps protocol's credential for the one in fresh.
// It always changes the client config, so the item is flagged.
func replaceCredential(it *Item, protocol string, fresh panel.Credentials) {
	c := &it.User.Credentials
	switch protocol {
	case "vmess":
		c.VMess = fresh.VMess
	case "vless":
		flow := ""
		if c.VLESS != nil {
			flow = c.VLESS.Flow
		}
		c.VLESS = fresh.VLESS
		if c.VLESS != nil {
			c.VLESS.Flow = flow
		}
	case "trojan":
		c.Trojan = fresh.Trojan
	case "shadowsocks":
		method := ""
		if c.Shadowsocks != nil {
			method = c.Shadowsocks.Method
		}
		c.Shadowsocks = fresh.Shadowsocks
		if c.Shadowsocks != nil && method != "" {
			c.Shadowsocks.Method = method
		}
	case "hysteria":
		c.Hysteria = fresh.Hysteria
	case "wireguard":
		c.WireGuard = fresh.WireGuard
	default:
		return
	}
	it.ChangesConfig = true
	it.note("new:" + protocol)
}

// dropProtocol removes protocol from the user. The client config for it
// stops working, so the item is flagged.
func dropProtocol(it *Item, protocol string) {
	c := &it.User.Credentials
	switch protocol {
	case "vmess":
		c.VMess = nil
	case "vless":
		c.VLESS = nil
	case "trojan":
		c.Trojan = nil
	case "shadowsocks":
		c.Shadowsocks = nil
	case "hysteria":
		c.Hysteria = nil
	case "wireguard":
		c.WireGuard = nil
	default:
		return
	}
	it.ChangesConfig = true
	it.note("drop:" + protocol)
}

// fitProtocols drops what the destination model cannot hold: Marzban has
// no WireGuard or Hysteria. Those come from PasarGuard, which fills every
// protocol for every user, so dropping them is noted but not flagged.
func fitProtocols(it *Item, dest panel.Kind) {
	if dest != panel.Marzban {
		return
	}
	c := &it.User.Credentials
	if c.WireGuard != nil {
		c.WireGuard = nil
		it.note("unsupported:wireguard")
	}
	if c.Hysteria != nil {
		c.Hysteria = nil
		it.note("unsupported:hysteria")
	}
}
