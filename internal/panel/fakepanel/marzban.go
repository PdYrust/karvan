package fakepanel

import (
	"fmt"
	"maps"
	"net/http"
	"net/http/httptest"
)

// Marzban is a fake Marzban 0.8.4 API for one reseller.
type Marzban struct {
	*httptest.Server
	state *panelState
	mints int
}

// MarzbanAdmin is GET /api/admin for the reseller.
type MarzbanAdmin struct {
	Username, Password string
	UsersUsage         int64
}

// NewMarzban serves admin and users (built with MarzbanUser) until Close.
func NewMarzban(admin MarzbanAdmin, users []map[string]any, f Faults) *Marzban {
	p := &panelState{username: admin.Username, password: admin.Password, tokens: map[string]bool{}, users: users, faults: f, subs: map[string]string{}}
	m := &Marzban{state: p}
	m.Server = serve(p, map[string]http.HandlerFunc{
		"POST /api/admin/token": p.login,
		"GET /api/admin": func(w http.ResponseWriter, r *http.Request) {
			if p.gate(w, r) {
				writeJSON(w, http.StatusOK, map[string]any{"username": admin.Username, "is_sudo": false,
					"telegram_id": nil, "discord_webhook": nil, "users_usage": admin.UsersUsage})
			}
		},
		"GET /api/users": func(w http.ResponseWriter, r *http.Request) {
			if p.gate(w, r) {
				writeJSON(w, http.StatusOK, p.page(r, func(u map[string]any) map[string]any { return m.response(u) }))
			}
		},
		"GET /sub/{token}/info": p.subInfo(m.response),
	})
	return m
}

// Calls is how many API requests (not logins) the fake answered.
func (m *Marzban) Calls() int {
	m.state.mu.Lock()
	defer m.state.mu.Unlock()
	return m.state.calls
}

// response mints a fresh subscription URL per read, like the panel.
func (m *Marzban) response(u map[string]any) map[string]any {
	out := maps.Clone(u)
	m.mints++
	out["subscription_url"] = fmt.Sprintf("%s/sub/%s-%d", m.URL, u["username"], m.mints)
	return out
}

// MarzbanUser is a UserResponse with sensible defaults; set overrides keys.
func MarzbanUser(username string, n int, set map[string]any) map[string]any {
	u := map[string]any{
		"username": username, "status": "active", "used_traffic": 1000 * n, "lifetime_used_traffic": 3000 * n,
		"created_at":                "2025-01-02T03:04:05.123456",
		"data_limit":                int64(50) << 30,
		"data_limit_reset_strategy": "no_reset",
		"expire":                    1790000000,
		"note":                      nil, "sub_updated_at": nil, "sub_last_user_agent": nil, "online_at": "2026-10-01T10:00:00",
		"on_hold_expire_duration": nil, "on_hold_timeout": nil, "auto_delete_in_days": nil, "next_plan": nil,
		"proxies": map[string]any{
			"vless":  map[string]any{"id": fmt.Sprintf("00000000-0000-4000-8000-%012d", n), "flow": "xtls-rprx-vision"},
			"vmess":  map[string]any{"id": fmt.Sprintf("00000000-0000-4000-9000-%012d", n)},
			"trojan": map[string]any{"password": fmt.Sprintf("trojan-pass-%d", n), "flow": ""},
		},
		"inbounds":          map[string]any{"vless": []string{"VLESS TCP REALITY"}, "vmess": []string{"VMess WS"}, "trojan": []string{"Trojan WS"}},
		"excluded_inbounds": map[string]any{},
		"links":             []string{},
		"admin":             map[string]any{"username": "reseller", "is_sudo": false},
	}
	maps.Copy(u, set)
	return u
}

// Mutate changes the fake's users between reads.
func (m *Marzban) Mutate(fn func(users *[]map[string]any)) { m.state.Mutate(fn) }

// Subscribe makes token valid for username at /sub/{token}/info.
func (m *Marzban) Subscribe(token, username string) { m.state.Subscribe(token, username) }

// SubCalls is how many /info requests the fake answered.
func (m *Marzban) SubCalls() int { return m.state.SubCalls() }

// FailSub makes /info answer status (0: normally).
func (m *Marzban) FailSub(status int) { m.state.FailSub(status) }
