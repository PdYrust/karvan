//go:build linux

package hoststats

import "syscall"

// statfs returns a file system's size, free bytes and bytes free for
// unprivileged users.
func statfs(path string) (total, free, avail uint64, err error) {
	var s syscall.Statfs_t
	if err := syscall.Statfs(path, &s); err != nil {
		return 0, 0, 0, err
	}
	bs := uint64(s.Bsize) //nolint:gosec // G115: block sizes are small positive numbers.
	return s.Blocks * bs, s.Bfree * bs, s.Bavail * bs, nil
}
