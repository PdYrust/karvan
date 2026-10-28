package engine

import (
	"slices"

	"github.com/PdYrust/karvan/internal/panel"
)

// Conflict types about the protocols a user's old config uses (D36 to D38).
const (
	WireGuardOnly = "C38" // PasarGuard to Marzban, only WireGuard or Hysteria
	// GroupsUnknown is the job-level info line when the old PasarGuard
	// panel's groups could not be read, so C10 and C38 are not raised.
	GroupsUnknown = "source_groups_unknown"
)

// currentProtocols are the protocols the user's old config uses (D36).
// Marzban: the protocols with at least one inbound for the user.
// PasarGuard: the protocols of the inbounds in the user's enabled groups
// that the user has credentials for, since the panel fills every
// protocol for every user. ok is false when they cannot be known.
func currentProtocols(src Source, u panel.User) (out []string, ok bool) {
	add := func(p string) {
		if !slices.Contains(out, p) {
			out = append(out, p)
		}
	}
	switch src.Kind {
	case panel.Marzban:
		for p, tags := range u.SourceInbounds {
			if len(tags) > 0 {
				add(p)
			}
		}
	case panel.PasarGuard:
		if !src.GroupsRead {
			return nil, false
		}
		for _, g := range src.Groups {
			if g.Disabled || !slices.Contains(u.SourceGroups, g.ID) {
				continue
			}
			for _, p := range g.Protocols {
				if hasProtocol(u.Credentials, p) {
					add(p)
				}
			}
		}
	default:
		return nil, false
	}
	slices.Sort(out)
	return out, true
}

// unsupportedOnMarzban are the protocols Marzban has no model for.
func unsupportedOnMarzban(p string) bool { return p == "wireguard" || p == "hysteria" }

// toMarzban reports a PasarGuard to Marzban move, where WireGuard and
// Hysteria configs stop working.
func toMarzban(in Inputs) bool {
	return in.Source.Kind == panel.PasarGuard && in.Dest.Kind == panel.Marzban
}

// served raises C38 or C10 when none of the user's current protocols
// will be served after the move. A user with no current protocol has no
// config to lose, and no targets is the job-level C09.
func (d *detector) served(it *Item) {
	cur, ok := currentProtocols(d.in.Source, it.Source)
	if !ok || len(cur) == 0 || len(it.Targets) == 0 {
		return
	}
	if toMarzban(d.in) && !slices.ContainsFunc(cur, func(p string) bool { return !unsupportedOnMarzban(p) }) {
		it.add(WireGuardOnly, Blocker, nil) // replaces C10: one user, one reason
		return
	}
	for _, t := range it.Targets {
		for _, p := range d.in.Dest.TargetProtocols[t] {
			if slices.Contains(cur, p) {
				return
			}
		}
	}
	it.add(NoProtocol, Blocker, nil)
}

// markLostProtocols flags a PasarGuard to Marzban user whose current
// protocols include WireGuard or Hysteria: that config stops working
// (D39).
func markLostProtocols(it *Item, in Inputs) {
	if !toMarzban(in) {
		return
	}
	if cur, _ := currentProtocols(in.Source, it.Source); slices.ContainsFunc(cur, unsupportedOnMarzban) {
		it.ChangesConfig = true
	}
}

func hasProtocol(c panel.Credentials, p string) bool {
	switch p {
	case "vmess":
		return c.VMess != nil
	case "vless":
		return c.VLESS != nil
	case "trojan":
		return c.Trojan != nil
	case "shadowsocks":
		return c.Shadowsocks != nil
	case "hysteria":
		return c.Hysteria != nil
	case "wireguard":
		return c.WireGuard != nil
	}
	return false
}
