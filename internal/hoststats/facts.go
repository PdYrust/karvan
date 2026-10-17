package hoststats

import (
	"bufio"
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// Facts describe the server and Karvan's own process; they change rarely.
type Facts struct {
	Hostname   string    `json:"hostname"`
	OS         string    `json:"os"`
	Kernel     string    `json:"kernel"`
	Arch       string    `json:"arch"`
	CPUModel   string    `json:"cpuModel"`
	Cores      int       `json:"cores"`
	BootedAt   time.Time `json:"bootedAt"`
	GoVersion  string    `json:"goVersion"`
	KarvanRSS  uint64    `json:"karvanRss"` // bytes of memory Karvan holds
	StartedAt  time.Time `json:"startedAt"`
	Goroutines int       `json:"goroutines"`
}

var processStart = time.Now()

// Facts reads the server's facts. Missing pieces stay empty.
func (r Reader) Facts(now time.Time) Facts {
	f := Facts{Arch: runtime.GOARCH, Cores: runtime.NumCPU(), GoVersion: runtime.Version(),
		StartedAt: processStart.UTC(), Goroutines: runtime.NumGoroutine()}
	f.Hostname, _ = os.Hostname()
	if b, err := r.file("sys/kernel/osrelease"); err == nil {
		f.Kernel = strings.TrimSpace(string(b))
	}
	// /proc/../etc is /etc on a server and the fixture's etc in tests.
	if b, err := os.ReadFile(filepath.Join(r.Proc, "..", "etc", "os-release")); err == nil {
		f.OS = osName(b)
	}
	if b, err := r.file("cpuinfo"); err == nil {
		f.CPUModel = cpuModel(b)
	}
	if b, err := r.file("uptime"); err == nil {
		if up, err := parseUptime(b); err == nil {
			f.BootedAt = now.Add(-time.Duration(up * float64(time.Second))).UTC().Truncate(time.Second)
		}
	}
	if b, err := r.file("self/status"); err == nil {
		f.KarvanRSS = rss(b)
	}
	return f
}

// osName is PRETTY_NAME from os-release ("Debian GNU/Linux 13 (trixie)").
func osName(b []byte) string {
	sc := bufio.NewScanner(bytes.NewReader(b))
	for sc.Scan() {
		if v, ok := strings.CutPrefix(sc.Text(), "PRETTY_NAME="); ok {
			return strings.Trim(v, `"'`)
		}
	}
	return ""
}

// cpuModel is the first "model name" (x86) or "Hardware"/"CPU part" (ARM).
func cpuModel(b []byte) string {
	var fallback string
	sc := bufio.NewScanner(bytes.NewReader(b))
	for sc.Scan() {
		k, v, ok := strings.Cut(sc.Text(), ":")
		if !ok {
			continue
		}
		k, v = strings.TrimSpace(k), strings.TrimSpace(v)
		switch k {
		case "model name":
			return v
		case "Hardware", "Model":
			fallback = v
		}
	}
	return fallback
}

// rss is VmRSS from /proc/self/status, in bytes.
func rss(b []byte) uint64 {
	sc := bufio.NewScanner(bytes.NewReader(b))
	for sc.Scan() {
		v, ok := strings.CutPrefix(sc.Text(), "VmRSS:")
		if !ok {
			continue
		}
		if f := strings.Fields(v); len(f) > 0 {
			if n, err := strconv.ParseUint(f[0], 10, 64); err == nil {
				return n * 1024
			}
		}
	}
	return 0
}
