//go:build benchtest

package bench

import (
	"context"
	"fmt"
	"io"
	"math/rand/v2"
	"net/http"
	"os"
	"runtime"
	"slices"
	"strconv"
	"sync"
	"syscall"
	"testing"
	"time"
)

// The targets of D37. KARVAN_BENCH_SCALE shortens both runs for a quick
// look (0.1 is a tenth); the recorded numbers use 1.
const (
	tokenRows   = 1_000_000
	steadyRate  = 500.0 // requests per second
	steadyFor   = 60 * time.Second
	burstUsers  = 50_000
	burstFor    = 10 * time.Minute
	maxOverhead = 5 * time.Millisecond
)

// TestBridgeOverhead runs open-loop Poisson load against the bridge and,
// at the same time and rate, straight against the fake panel. The p99
// the bridge adds is the bridge's p99 minus the direct p99.
func TestBridgeOverhead(t *testing.T) {
	scale := 1.0
	if s := os.Getenv("KARVAN_BENCH_SCALE"); s != "" {
		v, err := strconv.ParseFloat(s, 64)
		if err != nil || v <= 0 || v > 1 {
			t.Fatalf("KARVAN_BENCH_SCALE %q: want a number in (0, 1]", s)
		}
		scale = v
	}
	r := newRig(t, tokenRows)
	t.Logf("host: %d CPUs for Go (GOMAXPROCS %d), %s", runtime.NumCPU(), runtime.GOMAXPROCS(0), runtime.Version())
	t.Run("steady", func(t *testing.T) {
		d := time.Duration(float64(steadyFor) * scale)
		r.compare(t, steadyRate, d, func(int) int64 { return rand.Int64N(r.users) + 1 })
	})
	t.Run("burst", func(t *testing.T) {
		users := int(burstUsers * scale)
		d := time.Duration(float64(burstFor) * scale)
		perm := rand.Perm(int(r.users))[:users] // distinct users, each once
		r.compare(t, float64(users)/d.Seconds(), d, func(i int) int64 { return int64(perm[i%users]) + 1 })
	})
	t.Logf("peak RSS of the process %d MiB", peakRSS()>>20)
}

// compare loads the bridge and the panel for d at rate each and checks
// the target. pick gives the user of the i-th request.
func (r *rig) compare(t *testing.T, rate float64, d time.Duration, pick func(i int) int64) {
	client := &http.Client{Timeout: 10 * time.Second, Transport: &http.Transport{MaxIdleConnsPerHost: 1024, IdleConnTimeout: time.Minute}}
	var bridged, direct result
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		bridged = openLoop(client, rate, d, func(i int) *http.Request {
			req, _ := http.NewRequest(http.MethodGet, "http://"+r.bridge+"/sub/"+oldToken(pick(i)), nil)
			req.Host = oldDomain
			req.Header.Set("User-Agent", "v2rayNG/1.9")
			return req
		})
	}()
	go func() {
		defer wg.Done()
		direct = openLoop(client, rate, d, func(i int) *http.Request {
			req, _ := http.NewRequest(http.MethodGet, r.panel.BaseURL()+"/sub/dest-"+strconv.FormatInt(pick(i), 10), nil)
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
