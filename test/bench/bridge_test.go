//go:build benchtest

package bench

import (
	"context"
	"fmt"
	"io"
	"math/rand/v2"
	"net"
	"net/http"
	"os"
	"slices"
	"strconv"
	"sync"
	"syscall"
	"testing"
	"time"
)

// The targets of D37. The bridge and the fake panel run in one container
// limited to 1 CPU and 1 GB (TestBridgeServer); the load generator runs
// in a second container with its own 1 CPU limit on a private internal
// Docker network (TestBridgeLoad), so the generator's own work is not
// counted against the bridge (owner reading of D37, 2026-10-29).
const (
	maxTokens   = 1_000_000 // the D37 store; KARVAN_BENCH_TOKENS may set fewer
	steadyRate  = 500.0     // requests per second
	steadyFor   = 60 * time.Second
	burstUsers  = 50_000
	burstFor    = 10 * time.Minute
	maxOverhead = 5 * time.Millisecond
	serverWait  = 60 * time.Minute // the server stops by itself after this
)

// Fixed ports inside the server container; nothing is published on the
// host. The generator finds the server at KARVAN_BENCH_SERVER (default
// 127.0.0.1, both tests in one network namespace).
const (
	bridgePort  = 18080
	panelPort   = 18081
	controlPort = 18082
)

func serverHost() string {
	if h := os.Getenv("KARVAN_BENCH_SERVER"); h != "" {
		return h
	}
	return "127.0.0.1"
}

func controlURL(path string) string {
	return fmt.Sprintf("http://%s:%d%s", serverHost(), controlPort, path)
}

// TestBridgeServer stores the tokens, serves the bridge and the fake
// panel, and answers /ready once the tokens are in. It ends on /stop or
// after serverWait and logs its peak RSS.
func TestBridgeServer(t *testing.T) {
	start := time.Now()
	newRig(t, int64(tokenCount(t)), bridgePort, fmt.Sprintf(":%d", panelPort))
	ready := time.Now()
	stop := make(chan struct{})
	var once sync.Once
	mux := http.NewServeMux()
	mux.HandleFunc("GET /ready", func(http.ResponseWriter, *http.Request) {})
	mux.HandleFunc("GET /stop", func(http.ResponseWriter, *http.Request) { once.Do(func() { close(stop) }) })
	l, err := net.Listen("tcp", fmt.Sprintf(":%d", controlPort))
	if err != nil {
		t.Fatal(err)
	}
	srv := &http.Server{Handler: mux, ReadHeaderTimeout: 10 * time.Second}
	go func() { _ = srv.Serve(l) }()
	t.Logf("ready after %.1f s", ready.Sub(start).Seconds())
	select {
	case <-stop:
	case <-time.After(serverWait):
		t.Log("no stop request: stopping after the time limit")
	}
	_ = srv.Close()
	t.Logf("peak RSS of the bridge and fake panel %d MiB", peakRSS()>>20)
}

// TestBridgeLoad waits for the server, then runs open-loop Poisson load
// against the bridge and, at the same time and rate, straight against
// the fake panel. The p99 the bridge adds is the bridge's p99 minus the
// direct p99. KARVAN_BENCH_SCALE shortens both runs (0.1 is a tenth).
func TestBridgeLoad(t *testing.T) {
	scale, tokens := benchScale(t), tokenCount(t)
	waitReady(t)
	defer func() { _ = get(controlURL("/stop")) }()
	h := serverHost()
	tg := target{bridge: fmt.Sprintf("%s:%d", h, bridgePort), panel: fmt.Sprintf("http://%s:%d", h, panelPort)}
	t.Run("steady", func(t *testing.T) {
		d := time.Duration(float64(steadyFor) * scale)
		tg.compare(t, steadyRate, d, func(int) int64 { return rand.Int64N(int64(tokens)) + 1 })
	})
	t.Run("burst", func(t *testing.T) {
		users := min(int(burstUsers*scale), tokens)
		d := time.Duration(float64(burstFor) * scale)
		perm := rand.Perm(tokens)[:users] // distinct users, each once
		tg.compare(t, float64(users)/d.Seconds(), d, func(i int) int64 { return int64(perm[i%users]) + 1 })
	})
}

func benchScale(t *testing.T) float64 {
	s := os.Getenv("KARVAN_BENCH_SCALE")
	if s == "" {
		return 1
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil || v <= 0 || v > 1 {
		t.Fatalf("KARVAN_BENCH_SCALE %q: want a number in (0, 1]", s)
	}
	return v
}

// tokenCount is KARVAN_BENCH_TOKENS, 1000 to maxTokens, default maxTokens.
func tokenCount(t *testing.T) int {
	s := os.Getenv("KARVAN_BENCH_TOKENS")
	if s == "" {
		return maxTokens
	}
	n, err := strconv.Atoi(s)
	if err != nil || n < 1000 || n > maxTokens {
		t.Fatalf("KARVAN_BENCH_TOKENS %q: want 1000 to %d", s, maxTokens)
	}
	return n
}

// waitReady polls the server's /ready for up to serverWait.
func waitReady(t *testing.T) {
	start := time.Now()
	for get(controlURL("/ready")) != nil {
		if time.Since(start) > serverWait {
			t.Fatal("the server did not get ready")
		}
		time.Sleep(2 * time.Second)
	}
	t.Logf("server ready after %.0f s of waiting", time.Since(start).Seconds())
}

func get(url string) error {
	c := &http.Client{Timeout: 5 * time.Second}
	resp, err := c.Get(url)
	if err != nil {
		return err
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("status %d", resp.StatusCode)
	}
	return nil
}

// target is where the bridge and the fake panel answer.
type target struct{ bridge, panel string }

// compare loads the bridge and the panel for d at rate each and checks
// the target. pick gives the user of the i-th request.
func (tg target) compare(t *testing.T, rate float64, d time.Duration, pick func(i int) int64) {
	client := &http.Client{Timeout: 10 * time.Second, Transport: &http.Transport{MaxIdleConnsPerHost: 1024, IdleConnTimeout: time.Minute}}
	var bridged, direct result
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		bridged = openLoop(client, rate, d, func(i int) *http.Request {
			req, _ := http.NewRequest(http.MethodGet, "http://"+tg.bridge+"/sub/"+oldToken(pick(i)), nil)
			req.Host = oldDomain
			req.Header.Set("User-Agent", "v2rayNG/1.9")
			return req
		})
	}()
	go func() {
		defer wg.Done()
		direct = openLoop(client, rate, d, func(i int) *http.Request {
			req, _ := http.NewRequest(http.MethodGet, tg.panel+"/sub/dest-"+strconv.FormatInt(pick(i), 10), nil)
			req.Header.Set("User-Agent", "v2rayNG/1.9")
			return req
		})
	}()
	wg.Wait()
	t.Logf("rate %.1f/s for %s", rate, d)
	t.Logf("bridge %s", bridged)
	t.Logf("direct %s", direct)
	added := bridged.pct(99) - direct.pct(99)
	t.Logf("p99 added by the bridge %.2f ms (target under %.0f ms)", ms(added), ms(maxOverhead))
	if bridged.errs > 0 || direct.errs > 0 {
		t.Errorf("errors: bridge %d, direct %d (first: %v)", bridged.errs, direct.errs, firstErr(bridged, direct))
	}
	if added >= maxOverhead {
		t.Errorf("the bridge adds %.2f ms at p99, the target is under %.0f ms", ms(added), ms(maxOverhead))
	}
}

// result is the latencies of one open-loop run, measured from each
// request's planned start, so a stalled server cannot hide its queue.
type result struct {
	lat   []time.Duration
	errs  int
	first error
}

func (r result) pct(p float64) time.Duration {
	if len(r.lat) == 0 {
		return 0
	}
	return r.lat[min(len(r.lat)-1, int(float64(len(r.lat))*p/100))]
}

func (r result) String() string {
	return fmt.Sprintf("%d requests, %d errors, p50 %.2f ms, p90 %.2f ms, p99 %.2f ms, max %.2f ms",
		len(r.lat)+r.errs, r.errs, ms(r.pct(50)), ms(r.pct(90)), ms(r.pct(99)), ms(r.pct(100)))
}

func firstErr(rs ...result) error {
	for _, r := range rs {
		if r.first != nil {
			return r.first
		}
	}
	return nil
}

func ms(d time.Duration) float64 { return float64(d) / float64(time.Millisecond) }

// openLoop sends requests with Poisson arrivals at rate for d, each on
// its own goroutine, and waits for the last answer.
func openLoop(c *http.Client, rate float64, d time.Duration, req func(i int) *http.Request) result {
	var mu sync.Mutex
	var out result
	var wg sync.WaitGroup
	start := time.Now()
	at := time.Duration(0)
	for i := 0; ; i++ {
		at += time.Duration(rand.ExpFloat64() / rate * float64(time.Second))
		if at >= d {
			break
		}
		time.Sleep(time.Until(start.Add(at)))
		planned := start.Add(at)
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := send(c, req(i))
			lat := time.Since(planned)
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				out.errs++
				if out.first == nil {
					out.first = err
				}
				return
			}
			out.lat = append(out.lat, lat)
		}()
	}
	wg.Wait()
	slices.Sort(out.lat)
	return out
}

func send(c *http.Client, req *http.Request) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	resp, err := c.Do(req.WithContext(ctx))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	n, err := io.Copy(io.Discard, resp.Body)
	switch {
	case err != nil:
		return err
	case resp.StatusCode != http.StatusOK:
		return fmt.Errorf("status %d", resp.StatusCode)
	case n != int64(len(body)):
		return fmt.Errorf("body of %d bytes, want %d", n, len(body))
	}
	return nil
}

// peakRSS is the most memory the process held, in bytes.
func peakRSS() int64 {
	var u syscall.Rusage
	if err := syscall.Getrusage(syscall.RUSAGE_SELF, &u); err != nil {
		return 0
	}
	return u.Maxrss << 10 // Linux reports KiB
}
