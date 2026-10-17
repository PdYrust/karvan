package hoststats

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/PdYrust/karvan/internal/store"
)

// fixture is a fake server: /proc and /sys files under a temp dir.
type fixture struct {
	t    *testing.T
	root string
	r    Reader
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	root := t.TempDir()
	f := &fixture{t: t, root: root, r: Reader{Proc: filepath.Join(root, "proc"), Sys: filepath.Join(root, "sys")}}
	for _, dir := range []string{"sys/class/net/eth0/device", "sys/class/net/docker0", "sys/block/sda/device", "sys/block/loop0/device", "etc"} {
		if err := os.MkdirAll(filepath.Join(root, dir), 0o700); err != nil {
			t.Fatal(err)
		}
	}
	f.write("etc/os-release", "PRETTY_NAME=\"Test OS 1\"\n")
	f.write("proc/sys/kernel/osrelease", "6.1.0-test\n")
	f.write("proc/cpuinfo", "model name\t: Test CPU\n")
	f.write("proc/self/status", "VmRSS:\t1024 kB\n")
	f.write("proc/self/mounts", fmt.Sprintf("/dev/sda1 %s ext4 rw 0 0\nproc /proc proc rw 0 0\n", root))
	f.counters(0)
	return f
}

func (f *fixture) write(name, body string) {
	f.t.Helper()
	p := filepath.Join(f.root, name)
	if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
		f.t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		f.t.Fatal(err)
	}
}

// counters writes counters that grow by step: per step, 50 busy jiffies of
// 100, 5000 bytes in and 1000 out on eth0, 10 sectors read.
func (f *fixture) counters(step uint64) {
	busy, idle := 1000+50*step, 1000+50*step
	f.write("proc/stat", fmt.Sprintf("cpu  %d 0 0 %d 0 0 0 0 0 0\ncpu0 %d 0 0 %d 0 0 0 0 0 0\n", busy, idle, busy, idle))
	f.write("proc/meminfo", "MemTotal: 1000 kB\nMemAvailable: 250 kB\nCached: 100 kB\nSwapTotal: 0 kB\nSwapFree: 0 kB\n")
	f.write("proc/loadavg", "1.50 1.00 0.50 3/200 999\n")
	f.write("proc/uptime", "3600.00 100.00\n")
	f.write("proc/net/dev", fmt.Sprintf("h\nh\n    lo: 9 0 0 0 0 0 0 0 9 0 0 0 0 0 0 0\n  eth0: %d 0 0 0 0 0 0 0 %d 0 0 0 0 0 0 0\ndocker0: %d 0 0 0 0 0 0 0 %d 0 0 0 0 0 0 0\n",
		5000*step, 1000*step, 99999*step, 99999*step))
	f.write("proc/diskstats", fmt.Sprintf("8 0 sda 1 0 %d 0 1 0 %d 0 0 0 0\n7 0 loop0 1 0 %d 0 1 0 0 0 0 0 0\n", 10*step, 20*step, 999*step))
	f.write("proc/net/sockstat", "TCP: inuse 40 orphan 0 tw 5 alloc 50 mem 1\n")
	f.write("proc/net/sockstat6", "TCP6: inuse 2\n")
}

func openStore(t *testing.T) *Store {
	t.Helper()
	db, err := store.Open(context.Background(), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return NewStore(db)
}

func TestSamplerTurnsCountersIntoRates(t *testing.T) {
	f := newFixture(t)
	now := time.Date(2026, 10, 17, 12, 0, 0, 0, time.UTC)
	s := &Sampler{Reader: f.r, Now: func() time.Time { return now }}
	ctx := context.Background()
	s.Tick(ctx)
	if _, ok, _ := s.Latest(); ok {
		t.Fatal("one reading gave a sample")
	}
	f.counters(1)
	now = now.Add(5 * time.Second)
	s.Tick(ctx)
	got, ok, err := s.Latest()
	if !ok || err != nil {
		t.Fatalf("no sample: %v", err)
	}
	expect(t, []want{
		{"cpu", got.CPU, 0.5},
		{"cores", got.Cores, []float64{0.5}},
		// docker0 has no device, so only eth0 counts.
		{"net in", got.NetRx, 1000.0},
		{"net out", got.NetTx, 200.0},
		{"interfaces", len(got.Ifaces), 1},
		// loop0 is not a disk; 10 sectors in 5 s.
		{"disk read", got.DiskRead, float64(10 * sectorBytes / 5)},
		{"disk write", got.DiskWrite, float64(20 * sectorBytes / 5)},
		{"tcp", got.TCP, 42},
		{"time wait", got.TCPTimeWait, 5},
		{"procs", got.Procs, 200},
		{"load", got.Load[0], 1.5},
		{"memory used", got.Mem.Used, uint64(750 * 1024)},
		{"uptime", got.Uptime, 3600.0},
		{"since 0", len(s.Since(0)), 1},
		{"since latest", len(s.Since(got.Seq)), 0},
	})
}

type want struct {
	name      string
	got, want any
}

// expect reports every mismatch at once.
func expect(t *testing.T, cases []want) {
	t.Helper()
	for _, c := range cases {
		if !reflect.DeepEqual(c.got, c.want) {
			t.Errorf("%s: got %v, want %v", c.name, c.got, c.want)
		}
	}
}

func TestSamplerSavesMinutesAndTrims(t *testing.T) {
	f := newFixture(t)
	st := openStore(t)
	start := time.Date(2026, 10, 17, 12, 0, 0, 0, time.UTC)
	now := start
	s := &Sampler{Reader: f.r, Store: st, Now: func() time.Time { return now }}
	ctx := context.Background()
	for i := range 30 { // 150 s: two whole minutes and a half
		f.counters(uint64(i))
		s.Tick(ctx)
		now = now.Add(5 * time.Second)
	}
	mins, err := st.Range(ctx, start.Unix(), start.Add(time.Hour).Unix(), 60)
	if err != nil {
		t.Fatal(err)
	}
	if len(mins) != 2 || mins[0].TS != start.Unix() || mins[0].CPU != 0.5 || mins[0].NetRx != 1000 || mins[0].TCP != 42 {
		t.Fatalf("minutes %+v", mins)
	}
	buckets, err := st.Range(ctx, start.Unix(), start.Add(time.Hour).Unix(), 300)
	if err != nil || len(buckets) != 1 || buckets[0].CPUMax != 0.5 || buckets[0].Minutes != 2 {
		t.Fatalf("5-minute buckets %+v %v", buckets, err)
	}
	n, err := st.Trim(ctx, start.Add(90*time.Second).Unix())
	if err != nil || n != 2 {
		t.Fatalf("trim removed %d: %v", n, err)
	}
	if len(s.Since(0)) != 29 {
		t.Fatalf("ring holds %d", len(s.Since(0)))
	}
}

func TestReadFailsWithoutCPU(t *testing.T) {
	f := newFixture(t)
	if err := os.Remove(filepath.Join(f.root, "proc/stat")); err != nil {
		t.Fatal(err)
	}
	s := &Sampler{Reader: f.r}
	s.Tick(context.Background())
	if _, ok, err := s.Latest(); ok || err == nil || !strings.Contains(err.Error(), "cpu") {
		t.Fatalf("missing /proc/stat: ok %v err %v", ok, err)
	}
}

func TestFactsAndDisks(t *testing.T) {
	f := newFixture(t)
	now := time.Date(2026, 10, 17, 12, 0, 0, 0, time.UTC)
	got := f.r.Facts(now)
	if got.OS != "Test OS 1" || got.Kernel != "6.1.0-test" || got.CPUModel != "Test CPU" || got.KarvanRSS != 1024*1024 ||
		!got.BootedAt.Equal(now.Add(-time.Hour)) || got.Cores < 1 {
		t.Fatalf("facts %+v", got)
	}
	disks := f.r.Disks()
	if len(disks) != 1 || disks[0].Mount != f.root || disks[0].Total == 0 || disks[0].Type != "ext4" {
		t.Fatalf("disks %+v", disks)
	}
	if got := (Reader{Proc: t.TempDir()}).Disks(); len(got) != 0 {
		t.Fatalf("no mounts file: %+v", got)
	}
}
