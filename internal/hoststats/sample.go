package hoststats

import (
	"sort"
	"time"
)

// IfaceRate is one physical network interface's traffic.
type IfaceRate struct {
	Name    string  `json:"name"`
	Rx      float64 `json:"rx"` // bytes per second
	Tx      float64 `json:"tx"`
	TotalRx uint64  `json:"totalRx"` // bytes since boot
	TotalTx uint64  `json:"totalTx"`
}

// Sample is the server over the last few seconds. Rates are per second.
type Sample struct {
	Seq          int64       `json:"seq"`
	At           time.Time   `json:"at"`
	CPU          float64     `json:"cpu"` // busy share, 0 to 1
	Cores        []float64   `json:"cores"`
	Load         [3]float64  `json:"load"`
	Mem          Memory      `json:"mem"`
	NetRx        float64     `json:"netRx"`
	NetTx        float64     `json:"netTx"`
	Ifaces       []IfaceRate `json:"ifaces"`
	DiskRead     float64     `json:"diskRead"`
	DiskWrite    float64     `json:"diskWrite"`
	TCP          int         `json:"tcp"`
	TCPTimeWait  int         `json:"tcpTimeWait"`
	Procs        int         `json:"procs"`
	ProcsRunning int         `json:"procsRunning"`
	Uptime       float64     `json:"uptime"` // seconds since boot
}

// delta is b - a for counters, and 0 when a counter went backwards (an
// interface came back up, a counter wrapped).
func delta(a, b uint64) float64 {
	if b < a {
		return 0
	}
	return float64(b - a)
}

func share(a, b cpuTimes) float64 {
	total := delta(a.total, b.total)
	if total == 0 {
		return 0
	}
	return min(1, delta(a.busy, b.busy)/total)
}

// between turns two readings into a sample over the time between them.
func between(a, b raw) Sample {
	secs := b.at.Sub(a.at).Seconds()
	if secs <= 0 {
		secs = 1
	}
	s := Sample{
		At: b.at, CPU: share(a.cpu, b.cpu), Load: b.load.load, Mem: b.mem,
		DiskRead: delta(a.diskRead, b.diskRead) / secs, DiskWrite: delta(a.diskWrite, b.diskWrite) / secs,
		TCP: b.tcp, TCPTimeWait: b.tw, Procs: b.load.total, ProcsRunning: b.load.running, Uptime: b.uptime,
		Cores: make([]float64, 0, len(b.cores)), Ifaces: []IfaceRate{},
	}
	for i, c := range b.cores {
		if i < len(a.cores) {
			s.Cores = append(s.Cores, share(a.cores[i], c))
		}
	}
	for _, name := range names(b.ifaces) {
		if !b.physical[name] {
			continue
		}
		now, then := b.ifaces[name], a.ifaces[name]
		r := IfaceRate{Name: name, Rx: delta(then.rx, now.rx) / secs, Tx: delta(then.tx, now.tx) / secs, TotalRx: now.rx, TotalTx: now.tx}
		s.NetRx += r.Rx
		s.NetTx += r.Tx
		s.Ifaces = append(s.Ifaces, r)
	}
	// Busiest interface first: on most servers that is the one that matters.
	sort.SliceStable(s.Ifaces, func(i, j int) bool {
		return s.Ifaces[i].TotalRx+s.Ifaces[i].TotalTx > s.Ifaces[j].TotalRx+s.Ifaces[j].TotalTx
	})
	return s
}
