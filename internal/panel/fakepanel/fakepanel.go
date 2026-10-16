// Package fakepanel serves fake Marzban and PasarGuard APIs for tests,
// built from .tmp/specs (the routes, shapes and quirks Karvan relies on),
// with switches for the failures a real panel or its proxy can produce.
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
