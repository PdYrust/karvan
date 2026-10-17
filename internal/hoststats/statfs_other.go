//go:build !linux

package hoststats

import "errors"

// statfs is Linux only; Karvan ships for Linux servers.
func statfs(string) (total, free, avail uint64, err error) {
	return 0, 0, 0, errors.New("disk space is read on Linux only")
}
