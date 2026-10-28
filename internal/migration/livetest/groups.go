//go:build paneltest

package livetest

import (
	"crypto/ecdh"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// Groups are the test PasarGuard's groups, each served by its own core.
type Groups struct {
	WireGuard int64 // group "wireguard": WireGuard core "wg", inbound wg0
	VLESS     int64 // group "vless": xray core "vless", inbound "VLESS TCP"
}

// The VLESS inbound listens on loopback, next to Marzban's 18441 to 18444.
const vlessPort = 18451

// EnsureGroups gives the test PasarGuard a WireGuard group and a VLESS
// group, each with its core and a host so links render. Cores and hosts
// are found by inbound and groups by name, so a second call adds nothing.
func EnsureGroups(l *Live) Groups {
	l.t.Helper()
	return Groups{
		WireGuard: l.ensureGroup("wireguard", "wg0", 51820, l.wireguardCore),
		VLESS:     l.ensureGroup("vless", "VLESS TCP", vlessPort, vlessCore),
	}
}

type coreInfo struct {
	Config map[string]any `json:"config"`
}

// serves reports whether the core serves inbound: an xray core lists it
// in config.inbounds, a WireGuard core names it interface_name.
func (c coreInfo) serves(inbound string) bool {
	if c.Config["interface_name"] == inbound {
		return true
	}
	ins, _ := c.Config["inbounds"].([]any)
	for _, in := range ins {
		if m, _ := in.(map[string]any); m["tag"] == inbound {
			return true
		}
	}
	return false
}

type hostInfo struct {
	InboundTag string `json:"inbound_tag"`
}

// ensureGroup adds the core that core() makes, a host for inbound and
// the group named name, each only when missing.
func (l *Live) ensureGroup(name, inbound string, port int, core func() map[string]any) int64 {
	l.t.Helper()
	var cores struct {
		Cores []coreInfo `json:"cores"`
	}
	l.mustCall(http.MethodGet, "/api/cores", nil, &cores)
	if !anyOf(cores.Cores, func(c coreInfo) bool { return c.serves(inbound) }) {
		l.mustCall(http.MethodPost, "/api/core", core(), nil)
	}
	var hosts []hostInfo
	l.mustCall(http.MethodGet, "/api/hosts", nil, &hosts)
	if !anyOf(hosts, func(h hostInfo) bool { return h.InboundTag == inbound }) {
		l.mustCall(http.MethodPost, "/api/host/", map[string]any{"remark": name, "address": []string{"127.0.0.1"},
			"port": port, "inbound_tag": inbound, "priority": 1}, nil)
	}
	var groups struct {
		Groups []struct {
			ID   int64  `json:"id"`
			Name string `json:"name"`
		} `json:"groups"`
	}
	l.mustCall(http.MethodGet, "/api/groups", nil, &groups)
	for _, g := range groups.Groups {
		if g.Name == name {
			return g.ID
		}
	}
	var g struct {
		ID int64 `json:"id"`
	}
	l.mustCall(http.MethodPost, "/api/group", map[string]any{"name": name, "inbound_tags": []string{inbound}}, &g)
	return g.ID
}

// EnsureRole gives the test PasarGuard a role named name that allows only
// group, with the operator role's permissions, and returns its id. Users
// moved under it get only that group (D31). A second call adds nothing.
func EnsureRole(l *Live, name string, group int64) int64 {
	l.t.Helper()
	var roles struct {
		Roles []struct {
			ID   int64  `json:"id"`
			Name string `json:"name"`
		} `json:"roles"`
	}
	l.mustCall(http.MethodGet, "/api/admin-roles", nil, &roles)
	for _, r := range roles.Roles {
		if r.Name == name {
			return r.ID
		}
	}
	var operator map[string]any
	l.mustCall(http.MethodGet, "/api/admin-role/3", nil, &operator)
	body := map[string]any{"name": name, "permissions": operator["permissions"], "limits": operator["limits"],
		"features": operator["features"], "access": map[string]any{"allowed_group_ids": []int64{group}}}
	var r struct {
		ID int64 `json:"id"`
	}
	l.mustCall(http.MethodPost, "/api/admin-role", body, &r)
	return r.ID
}

func anyOf[T any](xs []T, ok func(T) bool) bool {
	for _, x := range xs {
		if ok(x) {
			return true
		}
	}
	return false
}

// wireguardCore is the WireGuard core body; its pool is 10.77.0.0/24.
func (l *Live) wireguardCore() map[string]any {
	k, err := ecdh.X25519().GenerateKey(rand.Reader)
	if err != nil {
		l.t.Fatal(err)
	}
	return map[string]any{"name": "wg", "type": "wg", "exclude_inbound_tags": []string{}, "fallbacks_inbound_tags": []string{},
		"config": map[string]any{"interface_name": "wg0", "private_key": base64.StdEncoding.EncodeToString(k.Bytes()),
			"listen_port": 51820, "address": []string{"10.77.0.1/24"}}}
}

// vlessCore is an xray core body (pasarguard.md, Cores) with one VLESS
// TCP inbound.
func vlessCore() map[string]any {
	return map[string]any{"name": "vless", "type": "xray", "exclude_inbound_tags": []string{}, "fallbacks_inbound_tags": []string{},
		"config": map[string]any{
			"log": map[string]any{"loglevel": "warning"},
			"inbounds": []any{map[string]any{"tag": "VLESS TCP", "listen": "127.0.0.1", "port": vlessPort, "protocol": "vless",
				"settings": map[string]any{"clients": []any{}, "decryption": "none"}, "streamSettings": map[string]any{"network": "tcp"}}},
			"outbounds": []any{map[string]any{"protocol": "freedom", "tag": "DIRECT"}},
		}}
}

func (l *Live) mustCall(method, path string, body, out any) {
	l.t.Helper()
	if code := l.Call(method, path, body, out); code/100 != 2 {
		l.t.Fatalf("%s %s: %d", method, path, code)
	}
}

// Links reads a PasarGuard user's subscription links.
func (l *Live) Links(id int64) []string {
	l.t.Helper()
	sub, _ := l.User(id, "")["subscription_url"].(string)
	if sub == "" {
		l.t.Fatalf("user %d has no subscription_url", id)
	}
	req, _ := http.NewRequest(http.MethodGet, l.API.BaseURL+sub+"/links", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		l.t.Fatal(err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		l.t.Fatalf("links of user %d: %d", id, resp.StatusCode)
	}
	return strings.Fields(string(raw))
}

// Setup counts the panel's cores, hosts and groups.
func (l *Live) Setup() string {
	l.t.Helper()
	var cores struct {
		Count int `json:"count"`
	}
	var hosts []any
	var groups struct {
		Total int `json:"total"`
	}
	l.mustCall(http.MethodGet, "/api/cores", nil, &cores)
	l.mustCall(http.MethodGet, "/api/hosts", nil, &hosts)
	l.mustCall(http.MethodGet, "/api/groups", nil, &groups)
	return fmt.Sprintf("%d cores, %d hosts, %d groups", cores.Count, len(hosts), groups.Total)
}
