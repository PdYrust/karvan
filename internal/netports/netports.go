// Package netports reads the TCP ports this host listens on from
// /proc/net/tcp and /proc/net/tcp6 (D13), and proposes a bridge port
// (D14).
package netports

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
)

// stateListen is TCP_LISTEN in the st column.
const stateListen = "0A"

// Listening returns the TCP ports with a listening socket, sorted, each
// once. In a container it sees the container's network, the host's with
// host networking.
func Listening() ([]int, error) { return listening("/proc/net") }

func listening(dir string) ([]int, error) {
	var all []int
	for _, name := range []string{"tcp", "tcp6"} {
		f, err := os.Open(filepath.Join(dir, name)) //nolint:gosec // G304: a fixed /proc file, or a test folder.
		if name == "tcp6" && errors.Is(err, fs.ErrNotExist) {
			continue // no IPv6 in this kernel
		}
		if err != nil {
			return nil, fmt.Errorf("read listening ports: %w", err)
		}
		ports, err := Parse(f)
		_ = f.Close()
		if err != nil {
			return nil, fmt.Errorf("read listening ports from %s: %w", name, err)
		}
		all = append(all, ports...)
	}
	slices.Sort(all)
	return slices.Compact(all), nil
}

// Parse reads a /proc/net/tcp or tcp6 table and returns its listening
// ports, sorted, each once.
func Parse(r io.Reader) ([]int, error) {
	sc := bufio.NewScanner(r)
	var ports []int
	for first := true; sc.Scan(); first = false {
		if first {
			continue // the header
		}
		f := strings.Fields(sc.Text())
		if len(f) < 4 {
			return nil, fmt.Errorf("short line %q", sc.Text())
		}
		if f[3] != stateListen {
			continue
		}
		_, hex, ok := strings.Cut(f[1], ":")
		if !ok {
			return nil, fmt.Errorf("local address %q has no port", f[1])
		}
		p, err := strconv.ParseUint(hex, 16, 16)
		if err != nil {
			return nil, fmt.Errorf("local address %q: %w", f[1], err)
		}
		ports = append(ports, int(p))
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	slices.Sort(ports)
	return slices.Compact(ports), nil
}

// fallbacks are the HTTPS ports common CDNs also pass through, in the
// order they are proposed after 443.
var fallbacks = []int{8443, 2053, 2083, 2087, 2096}

// Propose is the bridge port for a new rule (D14): 443 when usable, else
// prefer (the old domain's own port) when set and usable, else the first
// usable fallback; 0 when none is.
func Propose(usable func(port int) bool, prefer int) int {
	if usable(443) {
		return 443
	}
	if prefer > 0 && prefer <= 65535 && usable(prefer) {
		return prefer
	}
	for _, p := range fallbacks {
		if usable(p) {
			return p
		}
	}
	return 0
}
