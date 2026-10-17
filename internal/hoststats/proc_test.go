package hoststats

import (
	"errors"
	"testing"
)

func TestParseStat(t *testing.T) {
	all, cores, err := parseStat([]byte("cpu  100 0 50 800 50 0 0 0 7 7\ncpu0 60 0 20 400 20 0 0 0 0 0\ncpu1 40 0 30 400 30 0 0 0 0 0\nintr 1 2\n"))
	if err != nil {
		t.Fatal(err)
	}
	// guest fields are left out: they are already in user and nice.
	if all != (cpuTimes{busy: 150, total: 1000}) || len(cores) != 2 || cores[1] != (cpuTimes{busy: 70, total: 500}) {
		t.Fatalf("all %+v cores %+v", all, cores)
	}
	if _, _, err := parseStat([]byte("intr 1\n")); !errors.Is(err, errFormat) {
		t.Fatalf("no cpu line: %v", err)
	}
	if _, _, err := parseStat([]byte("cpu  1 x 2 3 4\n")); !errors.Is(err, errFormat) {
		t.Fatalf("bad field: %v", err)
	}
}

func TestParseMeminfo(t *testing.T) {
	m, err := parseMeminfo([]byte("MemTotal: 1000 kB\nMemFree: 100 kB\nMemAvailable: 400 kB\nBuffers: 10 kB\nCached: 200 kB\nSReclaimable: 30 kB\nSwapTotal: 500 kB\nSwapFree: 300 kB\n"))
	if err != nil {
		t.Fatal(err)
	}
	want := Memory{Total: 1000 * 1024, Used: 600 * 1024, Cache: 240 * 1024, Available: 400 * 1024, SwapTotal: 500 * 1024, SwapUsed: 200 * 1024}
	if m != want {
		t.Fatalf("got %+v want %+v", m, want)
	}
	old, err := parseMeminfo([]byte("MemTotal: 1000 kB\nMemFree: 100 kB\nBuffers: 10 kB\nCached: 200 kB\n"))
	if err != nil || old.Available != 310*1024 {
		t.Fatalf("kernel without MemAvailable: %+v %v", old, err)
	}
	if _, err := parseMeminfo([]byte("Nope: 1 kB\nbroken line\n")); !errors.Is(err, errFormat) {
		t.Fatalf("no MemTotal: %v", err)
	}
}

func TestParseSmallFiles(t *testing.T) {
	l, err := parseLoadavg([]byte("0.52 0.58 0.59 2/1234 5678\n"))
	if err != nil || l.load != [3]float64{0.52, 0.58, 0.59} || l.running != 2 || l.total != 1234 {
		t.Fatalf("loadavg %+v %v", l, err)
	}
	for _, bad := range []string{"0.1 0.2", "x 0.2 0.3 1/2", "0.1 0.2 0.3 12"} {
		if _, err := parseLoadavg([]byte(bad)); err == nil {
			t.Errorf("loadavg %q accepted", bad)
		}
	}
	if up, err := parseUptime([]byte("12345.67 9999.00\n")); err != nil || up != 12345.67 {
		t.Fatalf("uptime %v %v", up, err)
	}
	if _, err := parseUptime(nil); err == nil {
		t.Fatal("empty uptime accepted")
	}
	in, tw := parseSockstat([]byte("sockets: used 1328\nTCP: inuse 78 orphan 0 tw 13 alloc 103 mem 348\nUDP: inuse 18 mem 254\n"))
	if in != 78 || tw != 13 {
		t.Fatalf("sockstat %d %d", in, tw)
	}
	if in, _ := parseSockstat([]byte("TCP6: inuse 4\n")); in != 4 {
		t.Fatalf("sockstat6 %d", in)
	}
}

func TestParseNetDevAndDiskstats(t *testing.T) {
	dev := "Inter-|   Receive\n face |bytes packets\n    lo: 900 5 0 0 0 0 0 0 900 5 0 0 0 0 0 0\n  eth0: 1000 10 0 0 0 0 0 0 2000 20 0 0 0 0 0 0\n"
	ifs, err := parseNetDev([]byte(dev))
	if err != nil || ifs["eth0"] != (ifaceCounters{rx: 1000, tx: 2000}) || ifs["lo"].rx != 900 {
		t.Fatalf("net/dev %+v %v", ifs, err)
	}
	if _, err := parseNetDev([]byte("eth0: x 1 2 3 4 5 6 7 8 9\n")); !errors.Is(err, errFormat) {
		t.Fatalf("bad net/dev: %v", err)
	}
	stats := "   8       0 sda 100 0 4 0 200 0 8 0 0 0 0\n   8       1 sda1 100 0 4 0 200 0 8 0 0 0 0\n 7 0 loop0 1 0 99 0 1 0 99 0 0 0 0\n"
	r, w := parseDiskstats([]byte(stats), map[string]bool{"sda": true})
	if r != 4*sectorBytes || w != 8*sectorBytes {
		t.Fatalf("diskstats %d %d", r, w)
	}
}

func TestParseFactsAndMounts(t *testing.T) {
	if got := osName([]byte("NAME=Debian\nPRETTY_NAME=\"Debian GNU/Linux 13 (trixie)\"\n")); got != "Debian GNU/Linux 13 (trixie)" {
		t.Fatalf("os %q", got)
	}
	if got := osName([]byte("NAME=x\n")); got != "" {
		t.Fatalf("os without pretty name %q", got)
	}
	if got := cpuModel([]byte("processor\t: 0\nmodel name\t: Intel(R) Pentium(R) CPU 2117U\n")); got != "Intel(R) Pentium(R) CPU 2117U" {
		t.Fatalf("x86 model %q", got)
	}
	if got := cpuModel([]byte("processor : 0\nHardware : BCM2835\n")); got != "BCM2835" {
		t.Fatalf("arm model %q", got)
	}
	if got := rss([]byte("Name:\tkarvan\nVmRSS:\t   2048 kB\n")); got != 2048*1024 {
		t.Fatalf("rss %d", got)
	}
	if rss([]byte("VmRSS:\n")) != 0 {
		t.Fatal("empty rss")
	}
	ms := parseMounts([]byte("/dev/sda1 / ext4 rw 0 0\nproc /proc proc rw 0 0\n/dev/sda1 /var/lib/docker/x ext4 rw 0 0\n/dev/sdb1 /mnt/my\\040data xfs rw 0 0\noverlay /x overlay rw 0 0\n"))
	if len(ms) != 2 || ms[0].path != "/" || ms[1].path != "/mnt/my data" || ms[1].fs != "xfs" {
		t.Fatalf("mounts %+v", ms)
	}
}

func TestVirtualIfaces(t *testing.T) {
	for _, n := range []string{"lo", "veth5a8984a", "docker0", "br-e4b598889ec6", "singbox_tun", "wg0", "tun0"} {
		if !virtualIface(n) {
			t.Errorf("%s should be virtual", n)
		}
	}
	for _, n := range []string{"eth0", "ens3", "venet0", "enp2s0f0"} {
		if virtualIface(n) {
			t.Errorf("%s should count", n)
		}
	}
}
