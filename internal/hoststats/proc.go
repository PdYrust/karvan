// Package hoststats reads this server's CPU, memory, disks, network and
// sockets from /proc and /sys, samples them every few seconds, and keeps
// minute averages for the analytics pages. It only reads; it needs no
// privileges beyond what any user has.
package hoststats

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// cpuTimes are cumulative jiffies: busy is everything but idle and iowait.
type cpuTimes struct{ busy, total uint64 }

// errFormat means a /proc file did not look like the kernel's format.
var errFormat = errors.New("unexpected /proc format")

// parseStat reads /proc/stat: the "cpu" line and one "cpuN" line per core.
func parseStat(b []byte) (cpuTimes, []cpuTimes, error) {
	var all cpuTimes
	var cores []cpuTimes
	found := false
	sc := bufio.NewScanner(bytes.NewReader(b))
	for sc.Scan() {
		f := strings.Fields(sc.Text())
		if len(f) < 5 || !strings.HasPrefix(f[0], "cpu") {
			continue
		}
		var t cpuTimes
		for i, v := range f[1:] {
			n, err := strconv.ParseUint(v, 10, 64)
			if err != nil {
				return all, nil, fmt.Errorf("%w: cpu field %q", errFormat, v)
			}
			// guest and guest_nice are already counted in user and nice.
			if i >= 8 {
				break
			}
			t.total += n
			if i != 3 && i != 4 { // idle, iowait
				t.busy += n
			}
		}
		if f[0] == "cpu" {
			all, found = t, true
		} else {
			cores = append(cores, t)
		}
	}
	if !found {
		return all, nil, fmt.Errorf("%w: no cpu line", errFormat)
	}
	return all, cores, nil
}

// Memory is the server's memory in bytes. Used is what applications hold:
// total minus what the kernel says is available.
type Memory struct {
	Total     uint64 `json:"total"`
	Used      uint64 `json:"used"`
	Cache     uint64 `json:"cache"`
	Available uint64 `json:"available"`
	SwapTotal uint64 `json:"swapTotal"`
	SwapUsed  uint64 `json:"swapUsed"`
}

func parseMeminfo(b []byte) (Memory, error) {
	kb := map[string]uint64{}
	sc := bufio.NewScanner(bytes.NewReader(b))
	for sc.Scan() {
		name, rest, ok := strings.Cut(sc.Text(), ":")
		if !ok {
			continue
		}
		f := strings.Fields(rest)
		if len(f) == 0 {
			continue
		}
		if n, err := strconv.ParseUint(f[0], 10, 64); err == nil {
			kb[name] = n * 1024
		}
	}
	total, ok := kb["MemTotal"]
	if !ok {
		return Memory{}, fmt.Errorf("%w: no MemTotal", errFormat)
	}
	avail, ok := kb["MemAvailable"]
	if !ok { // kernels before 3.14
		avail = kb["MemFree"] + kb["Buffers"] + kb["Cached"]
	}
	m := Memory{Total: total, Available: min(avail, total), SwapTotal: kb["SwapTotal"]}
	m.Used = total - m.Available
	m.Cache = kb["Buffers"] + kb["Cached"] + kb["SReclaimable"]
	if free := kb["SwapFree"]; free <= m.SwapTotal {
		m.SwapUsed = m.SwapTotal - free
	}
	return m, nil
}

// loadavg is /proc/loadavg: three averages and running/total tasks.
type loadavg struct {
	load           [3]float64
	running, total int
}

func parseLoadavg(b []byte) (loadavg, error) {
	f := strings.Fields(string(b))
	if len(f) < 4 {
		return loadavg{}, fmt.Errorf("%w: loadavg %q", errFormat, b)
	}
	var l loadavg
	for i := range 3 {
		v, err := strconv.ParseFloat(f[i], 64)
		if err != nil {
			return loadavg{}, fmt.Errorf("%w: load %q", errFormat, f[i])
		}
		l.load[i] = v
	}
	run, all, ok := strings.Cut(f[3], "/")
	if !ok {
		return loadavg{}, fmt.Errorf("%w: tasks %q", errFormat, f[3])
	}
	l.running, _ = strconv.Atoi(run)
	l.total, _ = strconv.Atoi(all)
	return l, nil
}

// parseUptime reads /proc/uptime: seconds since boot.
func parseUptime(b []byte) (float64, error) {
	f := strings.Fields(string(b))
	if len(f) == 0 {
		return 0, fmt.Errorf("%w: uptime %q", errFormat, b)
	}
	return strconv.ParseFloat(f[0], 64)
}

// ifaceCounters are cumulative bytes through one network interface.
type ifaceCounters struct{ rx, tx uint64 }

// parseNetDev reads /proc/net/dev.
func parseNetDev(b []byte) (map[string]ifaceCounters, error) {
	out := map[string]ifaceCounters{}
	sc := bufio.NewScanner(bytes.NewReader(b))
	for sc.Scan() {
		name, rest, ok := strings.Cut(sc.Text(), ":")
		if !ok {
			continue
		}
		f := strings.Fields(rest)
		if len(f) < 9 {
			continue
		}
		rx, err1 := strconv.ParseUint(f[0], 10, 64)
		tx, err2 := strconv.ParseUint(f[8], 10, 64)
		if err1 != nil || err2 != nil {
			return nil, fmt.Errorf("%w: net/dev line %q", errFormat, sc.Text())
		}
		out[strings.TrimSpace(name)] = ifaceCounters{rx: rx, tx: tx}
	}
	return out, nil
}

// sectorBytes is the unit of /proc/diskstats, whatever the disk's own.
const sectorBytes = 512

// parseDiskstats sums bytes read and written by the named whole disks.
func parseDiskstats(b []byte, disks map[string]bool) (read, written uint64) {
	sc := bufio.NewScanner(bytes.NewReader(b))
	for sc.Scan() {
		f := strings.Fields(sc.Text())
		if len(f) < 10 || !disks[f[2]] {
			continue
		}
		r, _ := strconv.ParseUint(f[5], 10, 64)
		w, _ := strconv.ParseUint(f[9], 10, 64)
		read += r * sectorBytes
		written += w * sectorBytes
	}
	return read, written
}

// parseSockstat reads the TCP line of /proc/net/sockstat or sockstat6:
// sockets in use and in TIME_WAIT.
func parseSockstat(b []byte) (inuse, timeWait int) {
	sc := bufio.NewScanner(bytes.NewReader(b))
	for sc.Scan() {
		f := strings.Fields(sc.Text())
		if len(f) == 0 || (f[0] != "TCP:" && f[0] != "TCP6:") {
			continue
		}
		for i := 1; i+1 < len(f); i += 2 {
			n, _ := strconv.Atoi(f[i+1])
			switch f[i] {
			case "inuse":
				inuse = n
			case "tw":
				timeWait = n
			}
		}
	}
	return inuse, timeWait
}
