// Package fakepanel serves fake Marzban and PasarGuard APIs for tests,
// built from the routes, shapes and quirks of the real panels that Karvan
// relies on, with switches for the failures a real panel or its proxy can produce.
package fakepanel

import (
	crand "crypto/rand"
	"encoding/hex"
	"encoding/json"
	"math/rand/v2"
	"net/http"
	"net/http/httptest"
	"slices"
	"strconv"
	"strings"
	"sync"
)

// Faults are injected failures. Counters count API requests (not logins).
type Faults struct {
	RateLimitFirst int // answer the first N requests with 429 + Retry-After: 1
	ServerErrorAt  []int
	ExpireTokenAt  int // the token stops working at this request
	// BeforePage runs before each users page; it may change the users.
	BeforePage func(call int, users *[]map[string]any)
}

// panelState is what both fakes share.
type panelState struct {
	mu       sync.Mutex
	username string
	password string
	apiKey   string
	tokens   map[string]bool
	calls    int
	pages    int
	users    []map[string]any
	faults   Faults
	subs     map[string]string // subscription token -> username
	subCalls int
	subFail  int // answer /info with this status when set
}

// subInfo serves GET /sub/{token}/info: the user the token names, or the
// panels' own 404 for an unknown token. It needs no login.
func (p *panelState) subInfo(shape func(map[string]any) map[string]any) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		p.subCalls++
		if p.subFail != 0 {
			http.Error(w, `{"detail":"busy"}`, p.subFail)
			return
		}
		name, ok := p.subs[r.PathValue("token")]
		i := slices.IndexFunc(p.users, func(u map[string]any) bool { return u["username"] == name })
		if !ok || i < 0 {
			writeJSON(w, http.StatusNotFound, map[string]string{"detail": "Not Found"})
			return
		}
		out := shape(p.users[i])
		delete(out, "admin")
		writeJSON(w, http.StatusOK, out)
	}
}

// Subscribe makes token a valid subscription token for username; an
// empty username revokes it.
func (p *panelState) Subscribe(token, username string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if username == "" {
		delete(p.subs, token)
		return
	}
	p.subs[token] = username
}

// SubCalls is how many /info requests the fake answered.
func (p *panelState) SubCalls() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.subCalls
}

// FailSub makes /info answer status (0: answer normally).
func (p *panelState) FailSub(status int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.subFail = status
}

func (p *panelState) newToken() string {
	b := make([]byte, 16)
	_, _ = crand.Read(b)
	t := hex.EncodeToString(b)
	p.tokens[t] = true
	return t
}

// gate applies faults and auth. It returns false when it already answered.
func (p *panelState) gate(w http.ResponseWriter, r *http.Request) bool {
	p.calls++
	n := p.calls
	if n <= p.faults.RateLimitFirst {
		w.Header().Set("Retry-After", "1")
		http.Error(w, `{"detail":"slow down"}`, http.StatusTooManyRequests)
		return false
	}
	if slices.Contains(p.faults.ServerErrorAt, n) {
		http.Error(w, "upstream", http.StatusBadGateway)
		return false
	}
	if p.faults.ExpireTokenAt > 0 && n == p.faults.ExpireTokenAt {
		clear(p.tokens)
	}
	if p.apiKey != "" && (r.Header.Get("X-Api-Key") == p.apiKey || r.Header.Get("Authorization") == "ApiKey "+p.apiKey) {
		return true
	}
	tok := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	if !p.tokens[tok] {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"detail": "Could not validate credentials"})
		return false
	}
	return true
}

func (p *panelState) login(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil || r.PostForm.Get("username") != p.username || r.PostForm.Get("password") != p.password {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"detail": "Incorrect username or password"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"access_token": p.newToken(), "token_type": "bearer"})
}

// page serves users sorted by username when asked; without a sort the
// order is shuffled, as neither panel orders its lists by default.
func (p *panelState) page(r *http.Request, shape func(map[string]any) map[string]any) map[string]any {
	p.pages++
	if p.faults.BeforePage != nil {
		p.faults.BeforePage(p.pages, &p.users)
	}
	list := slices.Clone(p.users)
	if r.URL.Query().Get("sort") == "username" {
		slices.SortFunc(list, func(a, b map[string]any) int { return strings.Compare(a["username"].(string), b["username"].(string)) })
	} else {
		rand.Shuffle(len(list), func(i, j int) { list[i], list[j] = list[j], list[i] }) //nolint:gosec // only an order
	}
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	limit, err := strconv.Atoi(r.URL.Query().Get("limit"))
	if err != nil || limit <= 0 {
		limit = len(list)
	}
	start, end := min(offset, len(list)), min(offset+limit, len(list))
	out := make([]map[string]any, 0, end-start)
	for _, u := range list[start:end] {
		out = append(out, shape(u))
	}
	return map[string]any{"users": out, "total": len(list)}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func serve(p *panelState, routes map[string]http.HandlerFunc) *httptest.Server {
	mux := http.NewServeMux()
	for pattern, h := range routes {
		mux.HandleFunc(pattern, func(w http.ResponseWriter, r *http.Request) {
			p.mu.Lock()
			defer p.mu.Unlock()
			h(w, r)
		})
	}
	return httptest.NewServer(mux)
}

// Mutate changes the users under the panel's lock.
func (p *panelState) Mutate(fn func(users *[]map[string]any)) {
	p.mu.Lock()
	defer p.mu.Unlock()
	fn(&p.users)
}
