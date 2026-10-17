package hoststats

import (
	"bufio"
	"bytes"
	"strings"
)

// Disk is one mounted file system with its space in bytes.
type Disk struct {
	Mount string `json:"mount"`
	Type  string `json:"type"`
	Total uint64 `json:"total"`
	Used  uint64 `json:"used"`
	Free  uint64 `json:"free"` // available to Karvan's user, without the root reserve
}

// realFS are file systems that hold data; the rest (proc, tmpfs, overlay,
// squashfs of snaps) would only repeat or hide the real ones.
var realFS = map[string]bool{
	"ext2": true, "ext3": true, "ext4": true, "xfs": true, "btrfs": true, "zfs": true, "f2fs": true,
	"jfs": true, "reiserfs": true, "nfs": true, "nfs4": true, "cifs": true, "fuseblk": true, "vfat": true,
}

type mount struct{ device, path, fs string }

// parseMounts reads /proc/self/mounts and keeps one entry per real device.
func parseMounts(b []byte) []mount {
	var out []mount
	seen := map[string]bool{}
	sc := bufio.NewScanner(bytes.NewReader(b))
	for sc.Scan() {
		f := strings.Fields(sc.Text())
		if len(f) < 3 || !realFS[f[2]] || seen[f[0]] {
			continue
		}
		seen[f[0]] = true
		// The kernel writes spaces in paths as \040.
		out = append(out, mount{device: f[0], path: strings.ReplaceAll(f[1], `\040`, " "), fs: f[2]})
	}
	return out
}

// maxDisks keeps the list readable on servers with many small mounts.
const maxDisks = 8

// Disks lists the server's real file systems, root first.
func (r Reader) Disks() []Disk {
	b, err := r.file("self/mounts")
	if err != nil {
		return []Disk{}
	}
	out := []Disk{}
	for _, m := range parseMounts(b) {
		total, free, avail, err := statfs(m.path)
		if err != nil || total == 0 {
			continue
		}
		d := Disk{Mount: m.path, Type: m.fs, Total: total, Used: total - free, Free: avail}
		if m.path == "/" {
			out = append([]Disk{d}, out...)
		} else {
			out = append(out, d)
		}
		if len(out) == maxDisks {
			break
		}
	}
	return out
}
