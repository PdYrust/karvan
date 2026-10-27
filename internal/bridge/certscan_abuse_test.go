package bridge

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"syscall"
	"testing"
	"time"

	kstore "github.com/PdYrust/karvan/internal/store"
)

// A file swapped for a FIFO, a link re-pointed out of the place and a
// parent folder swapped for a link to Karvan's key folder, all between the
// walk and the open, are left out: the open resolves inside the place.
func TestScanSwapsBetweenTheWalkAndTheOpenAreLeftOut(t *testing.T) {
	e := newScanEnv(t)
	rootCert := filepath.Join(e.base, "root", "cert")
	keyDir := filepath.Dir(e.keyFile)
	shadow := filepath.Join(e.base, "outside", "shadow")
	writeFile(t, shadow, "root:secret")
	kcf, kkf := e.pair(keyDir, "c", "key-folder.example.com")
	e.pair(rootCert, "f", "f.example.com")
	rcf, rkf := e.pair(filepath.Join(rootCert, "real"), "l", "l.example.com")
	symlink(t, "real/l.crt", filepath.Join(rootCert, "l.crt"))
	symlink(t, "real/l.crt", filepath.Join(rootCert, "l2.crt"))
	e.pair(filepath.Join(rootCert, "sub"), "c", "sub.example.com")
	e.pair(filepath.Join(rootCert, "sub2"), "c", "sub2.example.com")
	swaps := map[string]func() error{
		"f.crt": func() error {
			if err := os.Remove(filepath.Join(rootCert, "f.crt")); err != nil {
				return err
			}
			return syscall.Mkfifo(filepath.Join(rootCert, "f.crt"), 0o600)
		},
		"l.crt":  func() error { return repoint(filepath.Join(rootCert, "l.crt"), shadow) },
		"l2.crt": func() error { return repoint(filepath.Join(rootCert, "l2.crt"), "../../outside/shadow") },
		"sub/c.crt": func() error {
			return swapForLink(filepath.Join(rootCert, "sub"), keyDir)
		},
		"sub2/c.crt": func() error {
			return swapForLink(filepath.Join(rootCert, "sub2"), "../../etc/karvan")
		},
	}
	e.b.scanT.beforeOpen = func(rel string) {
		if swap, ok := swaps[rel]; ok {
			delete(swaps, rel)
			if err := swap(); err != nil {
				t.Error(err)
			}
		}
	}
	done := make(chan ScanResult)
	go func() { done <- e.scan() }()
	var r ScanResult
	select {
	case r = <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("the scan blocked on a swapped file")
	}
	if len(swaps) != 0 {
		t.Fatalf("swaps not reached: %v", swaps)
	}
	if got := foundFiles(r); !slices.Equal(got, []string{rcf + " " + rkf}) {
		t.Errorf("found %v", got)
	}
	if p := e.openedAny(shadow, kcf, kkf, e.keyFile); p != "" {
		t.Errorf("opened %s", p)
	}
}

func repoint(link, target string) error {
	if err := os.Remove(link); err != nil {
		return err
	}
	return os.Symlink(target, link)
}

func swapForLink(dir, target string) error {
	if err := os.RemoveAll(dir); err != nil {
		return err
	}
	return os.Symlink(target, dir)
}

// ENV folders at /, too short, at Karvan's key folder, at or inside its
// data dir, above it, or a link to it are not scanned; a hard link to the
// key file inside a place is opened and closed unread.
func TestScanRefusesPlacesHoldingKarvansFiles(t *testing.T) {
	e := newScanEnv(t)
	data := e.b.DataDir
	linkData := filepath.Join(e.base, "srv", "data-link")
	symlink(t, data, linkData)
	e.panelEnv("p1", "UVICORN_SSL_KEYFILE="+e.keyFile+"\n")
	e.panelEnv("p2", "UVICORN_SSL_CERTFILE=/x.pem\n")
	e.panelEnv("p3", "UVICORN_SSL_CERTFILE=/etc/x.pem\n")
	e.panelEnv("p4", "UVICORN_SSL_CERTFILE="+filepath.Join(e.base, "var", "lib", "x.pem")+"\n")
	e.panelEnv("p5", "UVICORN_SSL_CERTFILE="+filepath.Join(data, "x.pem")+"\nUVICORN_SSL_KEYFILE="+
		filepath.Join(linkData, "x.key")+"\n")
	e.panelEnv("karvan", "")
	e.pair(filepath.Join(data, "certs"), "d", "data.example.com")
	rootCert := filepath.Join(e.base, "root", "cert")
	mkdirs(t, rootCert)
	if err := os.Link(e.keyFile, filepath.Join(rootCert, "k.key")); err != nil {
		t.Fatal(err)
	}

	r := e.scan()
	for _, place := range []string{filepath.Dir(e.keyFile), "/", "/etc", filepath.Join(e.base, "var", "lib"), data, linkData,
		filepath.Join(data, "certs")} {
		if got := skipReason(r, place); got != SkipNotAllowed {
			t.Errorf("%s: %q", place, got)
		}
	}
	if len(r.Found) != 0 {
		t.Errorf("found %v", foundFiles(r))
	}
	key, err := os.Stat(e.keyFile)
	if err != nil {
		t.Fatal(err)
	}
	if slices.ContainsFunc(e.read, func(o fs.FileInfo) bool { return os.SameFile(o, key) }) {
		t.Error("the key file was read")
	}
	if p := e.openedAny(kstore.Path(data), filepath.Join(data, "certs", "d.crt")); p != "" {
		t.Errorf("opened %s", p)
	}
}

// The walk goes three folders deep, a file over 1 MiB is left out unread,
// and a place with more than 2000 entries is cut short as too_many.
func TestScanLimits(t *testing.T) {
	e := newScanEnv(t)
	rootCert := filepath.Join(e.base, "root", "cert")
	cf, kf := e.pair(filepath.Join(rootCert, "a", "b", "c"), "x", "deep.example.com")
	dcf, _ := e.pair(filepath.Join(rootCert, "a", "b", "c", "d"), "y", "deeper.example.com")
	big := filepath.Join(rootCert, "big.crt")
	writeFile(t, big, strings.Repeat("A", 2<<20))
	e.panelEnv("many", "")
	many := filepath.Join(e.base, "var", "lib", "many", "certs")
	mkdirs(t, many)
	for i := range maxScanEntries + 1 {
		writeFile(t, filepath.Join(many, fmt.Sprintf("n%04d", i)), "")
	}

	r := e.scan()
	if got := foundFiles(r); !slices.Equal(got, []string{cf + " " + kf}) {
		t.Errorf("found %v", got)
	}
	if got := skipReason(r, many); got != SkipTooMany {
		t.Errorf("many: %q", got)
	}
	if p := e.openedAny(dcf); p != "" {
		t.Errorf("opened %s", p)
	}
	info, err := os.Stat(big)
	if err != nil {
		t.Fatal(err)
	}
	if slices.ContainsFunc(e.read, func(o fs.FileInfo) bool { return os.SameFile(o, info) }) {
		t.Error("a 2 MiB file was read")
	}
}

// At most 200 files are read in one scan; the place that reaches the
// limit and every place after it are too_many.
func TestScanReadsAtMost200Files(t *testing.T) {
	e := newScanEnv(t)
	rootCert := filepath.Join(e.base, "root", "cert")
	for i := range maxScanFiles + 1 {
		writeFile(t, filepath.Join(rootCert, fmt.Sprintf("j%03d.pem", i)), "junk")
	}
	e.pair(filepath.Join(e.base, "etc", "letsencrypt", "live", "x"), "x", "x.example.com")
	r := e.scan()
	if len(e.opened) != maxScanFiles || len(r.Found) != 0 {
		t.Errorf("opened %d, found %v", len(e.opened), foundFiles(r))
	}
	for _, place := range []string{rootCert, filepath.Join(e.base, "etc", "letsencrypt", "live")} {
		if got := skipReason(r, place); got != SkipTooMany {
			t.Errorf("%s: %q", place, got)
		}
	}
}

// A scan leaves a read-only tree byte for byte as it was, mtimes too.
func TestScanLeavesAReadOnlyTreeUnchanged(t *testing.T) {
	e := newScanEnv(t)
	rootCert := filepath.Join(e.base, "root", "cert")
	e.pair(rootCert, "a", "a.example.com")
	e.pair(filepath.Join(rootCert, "sub"), "b", "b.example.com")
	symlink(t, "sub/b.crt", filepath.Join(rootCert, "link.crt"))
	type state struct {
		mode  fs.FileMode
		mtime time.Time
		data  string
	}
	snapshot := func() map[string]state {
		out := map[string]state{}
		err := filepath.WalkDir(rootCert, func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			info, err := os.Lstat(p)
			if err != nil {
				return err
			}
			s := state{mode: info.Mode(), mtime: info.ModTime()}
			if info.Mode().IsRegular() {
				b, err := os.ReadFile(p)
				if err != nil {
					return err
				}
				s.data = string(b)
			}
			out[p] = s
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
		return out
	}
	chmodTree := func(dirMode, fileMode fs.FileMode) {
		var dirs []string
		_ = filepath.WalkDir(rootCert, func(p string, d fs.DirEntry, _ error) error {
			switch {
			case d.IsDir():
				dirs = append(dirs, p)
			case d.Type().IsRegular():
				_ = os.Chmod(p, fileMode)
			}
			return nil
		})
		for _, d := range slices.Backward(dirs) {
			_ = os.Chmod(d, dirMode)
		}
	}
	t.Cleanup(func() { chmodTree(0o700, 0o600) })
	chmodTree(0o500, 0o400)
	before := snapshot()

	if r := e.scan(); len(r.Found) != 2 {
		t.Errorf("found %v", foundFiles(r))
	}
	after := snapshot()
	if len(before) != len(after) {
		t.Fatalf("entries %d, then %d", len(before), len(after))
	}
	for p, s := range before {
		if after[p] != s {
			t.Errorf("%s changed: %+v to %+v", p, s, after[p])
		}
	}
}

// One scan runs at a time, and a scan out of time skips what is left.
func TestScanBusyAndTime(t *testing.T) {
	e := newScanEnv(t)
	e.b.scanMu.Lock()
	if _, err := e.b.ScanCertificates(context.Background()); !errors.Is(err, ErrScanBusy) {
		t.Errorf("busy: %v", err)
	}
	e.b.scanMu.Unlock()
	e.pair(filepath.Join(e.base, "root", "cert"), "a", "a.example.com")
	e.b.scanT.timeout = time.Nanosecond
	r, err := e.b.ScanCertificates(context.Background())
	if err != nil || len(r.Found) != 0 || len(r.Skipped) != 2 || r.Skipped[0].Reason != SkipTime || r.Skipped[1].Reason != SkipTime {
		t.Errorf("out of time: %+v %v", r, err)
	}
}

// Serving, inspect and add still follow a Let's Encrypt live link, and
// refuse a FIFO or a device without blocking.
func TestReadSmallFileFollowsLiveLinks(t *testing.T) {
	dir := t.TempDir()
	c, _ := longCert(t, "le.example.com")
	writeFile(t, filepath.Join(dir, "archive", "x", "fullchain1.pem"), c)
	live := filepath.Join(dir, "live", "x", "fullchain.pem")
	symlink(t, "../../archive/x/fullchain1.pem", live)
	if b, err := readSmallFile(live); err != nil || string(b) != c {
		t.Errorf("live link: %v", err)
	}
	fifo := filepath.Join(dir, "fifo.pem")
	if err := syscall.Mkfifo(fifo, 0o600); err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{fifo, "/dev/null", "relative.pem"} {
		if _, err := readSmallFile(p); err == nil {
			t.Errorf("%s was read", p)
		}
	}
}
