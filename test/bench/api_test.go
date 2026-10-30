//go:build benchtest

package bench

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"os"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"
)

// The HTTP API part of D37: 50 clients at once, each signed in on its
// own, against a running Karvan. KARVAN_BENCH_URL is the panel address
// with its hidden path; KARVAN_BENCH_USER and KARVAN_BENCH_PASSWORD
// sign in.
const (
	apiClients = 50
	apiRounds  = 40 // GET requests per client and endpoint
	apiLogins  = 4  // sign-ins per client
)

var apiGets = []string{"overview", "migrations", "domains"}

var errBusy = errors.New("busy")

func TestAPILatency(t *testing.T) {
	base := os.Getenv("KARVAN_BENCH_URL")
	if base == "" {
		t.Fatal("set KARVAN_BENCH_URL, KARVAN_BENCH_USER and KARVAN_BENCH_PASSWORD")
	}
	base = strings.TrimSuffix(base, "/") + "/api/v1/"
	login, _ := json.Marshal(map[string]string{"username": os.Getenv("KARVAN_BENCH_USER"), "password": os.Getenv("KARVAN_BENCH_PASSWORD")})
	times := map[string][]time.Duration{}
	var mu sync.Mutex
	add := func(name string, d time.Duration) {
		mu.Lock()
		times[name] = append(times[name], d)
		mu.Unlock()
	}
	errs := make(chan error, apiClients)
	var wg sync.WaitGroup
	for range apiClients {
		wg.Add(1)
		go func() {
			defer wg.Done()
			c := &apiClient{base: base, login: login, add: add}
			errs <- c.run()
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	for _, name := range append(slices.Clone(apiGets), "POST session", "POST session busy") {
		l := times[name]
		slices.Sort(l)
		r := result{lat: l}
		t.Logf("%-12s %4d requests, p50 %.1f ms, p99 %.1f ms", name, len(l), ms(r.pct(50)), ms(r.pct(99)))
	}
}

// apiClient is one browser: its own cookies and CSRF token.
type apiClient struct {
	base  string
	login []byte
	add   func(string, time.Duration)
	http  *http.Client
	csrf  string
}

func (c *apiClient) run() error {
	for range apiLogins {
		if err := c.signIn(); err != nil {
			return err
		}
	}
	for range apiRounds {
		for _, p := range apiGets {
			start := time.Now()
			if _, err := c.do(http.MethodGet, p, nil); err != nil {
				return err
			}
			c.add(p, time.Since(start))
		}
	}
	return nil
}

// signIn starts a fresh browser session and times the POST alone.
func (c *apiClient) signIn() error {
	jar, _ := cookiejar.New(nil)
	c.http = &http.Client{Jar: jar, Timeout: 30 * time.Second}
	b, err := c.do(http.MethodGet, "session", nil)
	if err != nil {
		return err
	}
	var s struct {
		CSRFToken string `json:"csrfToken"`
	}
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	c.csrf = s.CSRFToken
	var start time.Time
	for {
		start = time.Now()
		b, err = c.do(http.MethodPost, "session", c.login)
		if !errors.Is(err, errBusy) {
			break
		}
		// Two password checks run at a time and a third waits up to 10 s,
		// then gets 503 busy (auth.hashSlots): counted, then tried again.
		c.add("POST session busy", time.Since(start))
		time.Sleep(time.Second)
	}
	if err != nil {
		return err
	}
	c.add("POST session", time.Since(start))
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	c.csrf = s.CSRFToken
	return nil
}

func (c *apiClient) do(method, path string, body []byte) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, method, c.base+path, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-CSRF-Token", c.csrf)
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	out, err := io.ReadAll(resp.Body)
	if err == nil && resp.StatusCode == http.StatusServiceUnavailable {
		return out, errBusy
	}
	if err == nil && resp.StatusCode != http.StatusOK {
		err = fmt.Errorf("%s %s: status %d", method, path, resp.StatusCode)
	}
	return out, err
}
