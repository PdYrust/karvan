package hoststats

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"
)

// Reader reads one server's counters. The roots are /proc and /sys on a
// real server and fixture folders in tests.
type Reader struct {
	Proc string
	Sys  string
}

// System is the server Karvan runs on.
var System = Reader{Proc: "/proc", Sys: "/sys"}

func (r Reader) file(name string) ([]byte, error) {
	return os.ReadFile(filepath.Join(r.Proc, name)) //nolint:gosec // G304: fixed names under the /proc root.
}

// virtualIface names interfaces that only carry traffic that also crosses
// a physical one (containers, bridges, tunnels), so counting them would
// count the same bytes twice.
func virtualIface(name string) bool {
	if name == "lo" {
		return true
	}
	for _, p := range []string{"veth", "docker", "br-", "virbr", "cni", "flannel", "cali", "vxlan", "tun", "tap", "wg", "kube"} {
		if strings.HasPrefix(name, p) {
			return true
		}
	}
	return strings.HasSuffix(name, "_tun")
}

// physicalIfaces are the network cards: interfaces with a device under
// /sys/class/net. Some virtual servers (OpenVZ's venet0) show none, so then
// every interface that is not known to be virtual counts.
func (r Reader) physicalIfaces(all map[string]ifaceCounters) map[string]bool {
	out := map[string]bool{}
	for name := range all {
		if _, err := os.Stat(filepath.Join(r.Sys, "class/net", name, "device")); err == nil {
			out[name] = true
		}
	}
	if len(out) > 0 {
		return out
	}
	for name := range all {
		if !virtualIface(name) {
			out[name] = true
		}
	}
	return out
}

// wholeDisks are block devices with a device under /sys/block: real disks,
// not partitions, loop devices or LVM volumes, which would count twice.
func (r Reader) wholeDisks() map[string]bool {
	out := map[string]bool{}
	entries, err := os.ReadDir(filepath.Join(r.Sys, "block"))
	if err != nil {
		return out
	}
	for _, e := range entries {
		name := e.Name()
		if strings.HasPrefix(name, "loop") || strings.HasPrefix(name, "ram") || strings.HasPrefix(name, "zram") {
			continue
		}
		if _, err := os.Stat(filepath.Join(r.Sys, "block", name, "device")); err == nil {
			out[name] = true
		}
	}
	return out
}

// raw is one reading of every counter at a moment.
type raw struct {
	at        time.Time
	cpu       cpuTimes
	cores     []cpuTimes
	ifaces    map[string]ifaceCounters
	physical  map[string]bool
	diskRead  uint64
	diskWrite uint64
	mem       Memory
	load      loadavg
	uptime    float64
	tcp, tw   int
}

// read takes one reading. CPU and memory must be readable; the rest is
// best effort, since containers and some kernels hide parts of /sys.
func (r Reader) read(at time.Time) (raw, error) {
	out := raw{at: at}
	stat, err := r.file("stat")
	if err != nil {
		return out, fmt.Errorf("read cpu times: %w", err)
	}
	if out.cpu, out.cores, err = parseStat(stat); err != nil {
		return out, err
	}
	mem, err := r.file("meminfo")
	if err != nil {
		return out, fmt.Errorf("read memory: %w", err)
	}
	if out.mem, err = parseMeminfo(mem); err != nil {
		return out, err
	}
	var errs []error
	if b, err := r.file("loadavg"); err == nil {
		out.load, err = parseLoadavg(b)
		errs = append(errs, err)
	}
	if b, err := r.file("uptime"); err == nil {
		out.uptime, err = parseUptime(b)
		errs = append(errs, err)
	}
	if b, err := r.file("net/dev"); err == nil {
		out.ifaces, err = parseNetDev(b)
		errs = append(errs, err)
		out.physical = r.physicalIfaces(out.ifaces)
	}
	if b, err := r.file("diskstats"); err == nil {
		out.diskRead, out.diskWrite = parseDiskstats(b, r.wholeDisks())
	}
	for _, name := range []string{"net/sockstat", "net/sockstat6"} {
		if b, err := r.file(name); err == nil {
			n, tw := parseSockstat(b)
			out.tcp, out.tw = out.tcp+n, out.tw+tw
		}
	}
	return out, errors.Join(errs...)
}

// names returns the map's keys in order.
func names[V any](m map[string]V) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	slices.Sort(out)
	return out
}
