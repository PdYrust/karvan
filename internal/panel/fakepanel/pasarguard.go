package fakepanel

import (
	"fmt"
	"maps"
	"net/http"
	"net/http/httptest"
)

// PasarGuard is a fake PasarGuard 5.4.1 API for one reseller.
type PasarGuard struct {
	*httptest.Server
	state *panelState
	mints int
}

// PasarGuardAdmin is the reseller. APIKey enables X-Api-Key sign-in;
// Shape is "rbac" (5.x, role object) or "is_sudo" (older).
type PasarGuardAdmin struct {
	ID                         int64
	Username, Password, APIKey string
	UsedTraffic                int64
	Shape                      string
}

// NewPasarGuard serves admin and users (built with PasarGuardUser).
func NewPasarGuard(admin PasarGuardAdmin, users []map[string]any, f Faults) *PasarGuard {
	p := &panelState{username: admin.Username, password: admin.Password, apiKey: admin.APIKey,
		tokens: map[string]bool{}, users: users, faults: f}
	g := &PasarGuard{state: p}
	g.Server = serve(p, map[string]http.HandlerFunc{
		"POST /api/admin/token": p.login,
		"GET /api/admin": func(w http.ResponseWriter, r *http.Request) {
			if !p.gate(w, r) {
				return
			}
			a := map[string]any{"id": admin.ID, "username": admin.Username, "used_traffic": admin.UsedTraffic,
				"status": "active", "total_users": len(p.users), "telegram_id": nil, "note": nil}
			if admin.Shape == "is_sudo" {
				a["is_sudo"] = false
			} else {
				a["role"] = map[string]any{"id": 3, "name": "operator", "is_owner": false}
			}
			writeJSON(w, http.StatusOK, a)
		},
		"GET /api/users": func(w http.ResponseWriter, r *http.Request) {
			if p.gate(w, r) {
				loadSub := r.URL.Query().Get("load_sub") == "true"
				writeJSON(w, http.StatusOK, p.page(r, func(u map[string]any) map[string]any { return g.response(u, loadSub) }))
			}
		},
	})
	return g
}

func (g *PasarGuard) response(u map[string]any, loadSub bool) map[string]any {
	out := maps.Clone(u)
	out["subscription_url"] = ""
	if loadSub {
		g.mints++
		out["subscription_url"] = fmt.Sprintf("%s/sub/%v-%d", g.URL, u["id"], g.mints)
	}
	return out
}

// PasarGuardUser is a UserResponse with sensible defaults.
func PasarGuardUser(id int64, username string, set map[string]any) map[string]any {
	u := map[string]any{
		"id": id, "username": username, "status": "active", "used_traffic": 1000 * id, "lifetime_used_traffic": 4000 * id,
		"created_at": "2025-02-03T04:05:06.654321Z", "edit_at": nil, "online_at": "2026-10-01T10:00:00+00:00",
		"expire": "2026-12-31T20:30:00+00:00", "data_limit": int64(30) << 30, "data_limit_reset_strategy": "month",
		"note": "vip", "on_hold_expire_duration": nil, "on_hold_timeout": nil, "group_ids": []int64{1, 2},
		"auto_delete_in_days": nil, "hwid_limit": 3, "next_plan": nil,
		"admin": map[string]any{"id": 7, "username": "shop"},
		"proxy_settings": map[string]any{
			"vmess":       map[string]any{"id": fmt.Sprintf("11111111-0000-4000-8000-%012d", id)},
			"vless":       map[string]any{"id": fmt.Sprintf("22222222-0000-4000-8000-%012d", id)},
			"trojan":      map[string]any{"password": fmt.Sprintf("tp-%d", id)},
			"shadowsocks": map[string]any{"password": fmt.Sprintf("ss-password-long-enough-%d", id), "method": "chacha20-ietf-poly1305"},
			"hysteria":    map[string]any{"auth": fmt.Sprintf("hy-%d", id)},
		},
	}
	maps.Copy(u, set)
	return u
}
