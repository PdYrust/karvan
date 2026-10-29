package netports

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

const tcp4 = `  sl  local_address rem_address   st tx_queue rx_queue tr tm->when retrnsmt   uid  timeout inode
   0: 0100007F:2A3E 00000000:0000 0A 00000000:00000000 00:00000000 00000000     0        0 27680 1 0000000067a12f5f 100 0 0 10 0
   1: 00000000:01BB 00000000:0000 0A 00000000:00000000 00:00000000 00000000     0        0 36324 1 00000000a45abe80 100 0 0 10 0
   2: 0100007F:A1B2 0100007F:01BB 01 00000000:00000000 00:00000000 00000000  1000        0 146247 1 000000000ac70757 20 4 30 10 -1
   3: 00000000:01BB 00000000:0000 0A 00000000:00000000 00:00000000 00000000     0        0 36325 1 00000000a45abe81 100 0 0 10 0
`

const tcp6 = `  sl  local_address                         remote_address                        st tx_queue rx_queue tr tm->when retrnsmt   uid  timeout inode
   0: 00000000000000000000000000000000:0050 00000000000000000000000000000000:0000 0A 00000000:00000000 00:00000000 00000000     0        0 1029266 2 0000000071c11651 100 0 0 10 0
   1: 00000000000000000000000001000000:20FB 00000000000000000000000000000000:0000 0A 00000000:00000000 00:00000000 00000000 64677        0 8847 1 000000005d7e3c33 100 0 0 10 0
   2: 00000000000000000000000001000000:C350 00000000000000000000000001000000:0050 06 00000000:00000000 03:00000ab3 00000000     0        0 0 3 0000000000000000
`

func TestParseIPv4(t *testing.T) {
	got, err := Parse(strings.NewReader(tcp4))
	if err != nil {
		t.Fatal(err)
	}
	// Only LISTEN rows count; 443 twice (SO_REUSEPORT) is one port.
	if want := []int{443, 10814}; !slices.Equal(got, want) {
		t.Fatalf("ports = %v, want %v", got, want)
	}
}

func TestParseIPv6(t *testing.T) {
	got, err := Parse(strings.NewReader(tcp6))
	if err != nil {
		t.Fatal(err)
	}
	if want := []int{80, 8443}; !slices.Equal(got, want) {
		t.Fatalf("ports = %v, want %v", got, want)
	}
}

func TestParseRejectsGarbage(t *testing.T) {
	for _, in := range []string{
		"header\n   0: 0100007F 00000000:0000 0A\n",
		"header\n   0: 0100007F:XYZ 00000000:0000 0A\n",
		"header\n   0: 0100007F:2A3E\n",
	} {
		if _, err := Parse(strings.NewReader(in)); err == nil {
			t.Errorf("%q parsed", in)
		}
	}
}

func TestListeningReadsBothFiles(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "tcp"), []byte(tcp4), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "tcp6"), []byte(tcp6), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := listening(dir)
	if err != nil {
		t.Fatal(err)
	}
	if want := []int{80, 443, 8443, 10814}; !slices.Equal(got, want) {
		t.Fatalf("ports = %v, want %v", got, want)
	}
	// A kernel without IPv6 has no tcp6.
	if err := os.Remove(filepath.Join(dir, "tcp6")); err != nil {
		t.Fatal(err)
	}
	if got, err = listening(dir); err != nil || !slices.Equal(got, []int{443, 10814}) {
		t.Fatalf("without tcp6 = %v, %v", got, err)
	}
	if _, err := listening(filepath.Join(dir, "missing")); err == nil {
		t.Fatal("no tcp file read as no ports")
	}
}

func TestPropose(t *testing.T) {
	usable := func(free ...int) func(int) bool {
		return func(p int) bool { return slices.Contains(free, p) }
	}
	for _, c := range []struct {
		name   string
		usable func(int) bool
		prefer int
		want   int
	}{
		{"443 free", usable(443, 2096, 8443), 2096, 443},
		{"old port", usable(2096, 8443), 2096, 2096},
		{"old port taken", usable(8443, 2053), 2096, 8443},
		{"no old port", usable(2053, 2083), 0, 2053},
		{"last of the list", usable(2096), 0, 2096},
		{"none free", usable(), 2096, 0},
	} {
		if got := Propose(c.usable, c.prefer); got != c.want {
			t.Errorf("%s: %d, want %d", c.name, got, c.want)
		}
	}
}

func TestPickFree(t *testing.T) {
	at := func(v int) func(int) (int, error) { return func(int) (int, error) { return v, nil } }
	if p, err := pickFree(nil, at(5)); err != nil || p != FreeLow+5 {
		t.Errorf("empty host: %d, %v", p, err)
	}
	if p, err := pickFree([]int{80, FreeLow + 5, FreeLow + 6}, at(5)); err != nil || p != FreeLow+7 {
		t.Errorf("taken start: %d, %v; want the next free one", p, err)
	}
	if p, err := pickFree([]int{FreeHigh}, at(FreeHigh-FreeLow)); err != nil || p != FreeLow {
		t.Errorf("wrap: %d, %v; want %d", p, err, FreeLow)
	}
	all := make([]int, 0, FreeHigh-FreeLow+1)
	for p := FreeLow; p <= FreeHigh; p++ {
		all = append(all, p)
	}
	if _, err := pickFree(all, at(0)); err == nil {
		t.Error("a full range gave a port")
	}
	if p, err := Free(); err != nil || p < FreeLow || p > FreeHigh {
		t.Errorf("Free = %d, %v", p, err)
	}
}
