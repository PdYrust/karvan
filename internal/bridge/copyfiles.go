package bridge

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"strings"
)

// Why a copy was refused or failed, as codes the web panel puts in words.
const (
	CopyNotFound = "not_found"    // the base folder does not exist
	CopyLink     = "link"         // a folder or file on the way is a link
	CopyOutside  = "outside"      // the path leaves its base folder
	CopyForeign  = "foreign_file" // a file is there that Karvan did not write, or changed since
	CopyTaken    = "taken"        // another certificate is copied to the folder
	CopyNoKey    = "no_key"       // Karvan cannot read the certificate or its key
	CopyNoName   = "no_name"      // the certificate names no domain a folder can take
	CopyFailed   = "write_failed" // the files could not be written
)

// CopyError is a copy refused or failed: the reason and the path it is about.
type CopyError struct {
	Reason string `json:"reason"`
	Path   string `json:"path"`
}

func (e *CopyError) Error() string { return fmt.Sprintf("certificate copy %s: %s", e.Reason, e.Path) }

// The files of a copy, with their modes.
const (
	copyCertName = "fullchain.pem"
	copyKeyName  = "privkey.pem"
	copyDirMode  = 0o700
	copyCertMode = 0o644
	copyKeyMode  = 0o600
)

// copyFile is one file of a copy, by name in its folder.
type copyFile struct {
	name string
	data []byte
	mode os.FileMode
}

// writeCopy writes files into rel below base, all through an os.Root at
// base: no link is followed and nothing outside base is reached. Missing
// folders are made 0700. A file that is there already is replaced only
// while it holds what Karvan wrote (sums, by path); every file is checked
// before any is written, and each is written to a temporary file and
// renamed. It returns the sums written, by path.
func (b *Bridge) writeCopy(base, rel string, files []copyFile, sums map[string]string) (map[string]string, error) {
	if !fs.ValidPath(rel) || rel == "." || !strings.HasPrefix(base, "/") {
		return nil, &CopyError{Reason: CopyOutside, Path: path.Join(base, rel)}
	}
	root, err := openCopyBase(b.CopyBase+base, base)
	if err != nil {
		return nil, err
	}
	defer root.Close()
	if err := makeCopyDirs(root, base, rel); err != nil {
		return nil, err
	}
	written := map[string]string{}
	for _, f := range files {
		p := path.Join(base, rel, f.name)
		if err := checkCopyFile(root, path.Join(rel, f.name), p, sums[p], b.copySum); err != nil {
			return nil, err
		}
		written[p] = b.copySum(f.data)
	}
	for _, f := range files {
		if err := writeAtomic(root, path.Join(rel, f.name), f.data, f.mode); err != nil {
			return nil, &CopyError{Reason: CopyFailed, Path: path.Join(base, rel, f.name)}
		}
	}
	return written, nil
}

// openCopyBase opens dir, shown as base, as an os.Root: a folder, not a link.
func openCopyBase(dir, base string) (*os.Root, error) {
	info, err := os.Lstat(dir)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return nil, &CopyError{Reason: CopyNotFound, Path: base}
	case err != nil:
		return nil, &CopyError{Reason: CopyFailed, Path: base}
	case info.Mode()&fs.ModeSymlink != 0:
		return nil, &CopyError{Reason: CopyLink, Path: base}
	case !info.IsDir():
		return nil, &CopyError{Reason: CopyNotFound, Path: base}
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		return nil, &CopyError{Reason: CopyFailed, Path: base}
	}
	// The folder could have been swapped between the check and the open.
	if now, err := root.Stat("."); err != nil || !os.SameFile(now, info) {
		_ = root.Close()
		return nil, &CopyError{Reason: CopyLink, Path: base}
	}
	return root, nil
}

// makeCopyDirs makes each missing folder of rel 0700; one that is there
// must be a folder, not a link.
func makeCopyDirs(root *os.Root, base, rel string) error {
	at := ""
	for _, part := range strings.Split(rel, "/") {
		at = path.Join(at, part)
		info, err := root.Lstat(at)
		switch {
		case errors.Is(err, fs.ErrNotExist):
			if err := root.Mkdir(at, copyDirMode); err != nil {
				return copyPathError(err, path.Join(base, at))
			}
			if err := root.Chmod(at, copyDirMode); err != nil { // the umask may have taken bits
				return copyPathError(err, path.Join(base, at))
			}
		case err != nil:
			return copyPathError(err, path.Join(base, at))
		case info.Mode()&fs.ModeSymlink != 0:
			return &CopyError{Reason: CopyLink, Path: path.Join(base, at)}
		case !info.IsDir():
			return &CopyError{Reason: CopyForeign, Path: path.Join(base, at)}
		}
	}
	return nil
}

// checkCopyFile says whether rel may be written: it is not there, or it
// is a regular file that still holds what Karvan wrote (want, its sum).
func checkCopyFile(root *os.Root, rel, shown, want string, sum func([]byte) string) error {
	info, err := root.Lstat(rel)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return nil
	case err != nil:
		return copyPathError(err, shown)
	case info.Mode()&fs.ModeSymlink != 0:
		return &CopyError{Reason: CopyLink, Path: shown}
	case !info.Mode().IsRegular() || info.Size() > maxCertFileBytes || want == "":
		return &CopyError{Reason: CopyForeign, Path: shown}
	}
	data, err := root.ReadFile(rel)
	if err != nil {
		return copyPathError(err, shown)
	}
	if sum(data) != want {
		return &CopyError{Reason: CopyForeign, Path: shown}
	}
	return nil
}

// writeAtomic writes data to a new temporary file next to rel, then
// renames it over rel.
func writeAtomic(root *os.Root, rel string, data []byte, mode os.FileMode) error {
	var rnd [6]byte
	if _, err := rand.Read(rnd[:]); err != nil {
		return err
	}
	tmp := path.Join(path.Dir(rel), "."+path.Base(rel)+".karvan-"+hex.EncodeToString(rnd[:]))
	f, err := root.OpenFile(tmp, os.O_WRONLY|os.O_CREATE|os.O_EXCL, mode)
	if err != nil {
		return err
	}
	err = f.Chmod(mode)
	if err == nil {
		_, err = f.Write(data)
	}
	if err == nil {
		err = f.Sync()
	}
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err == nil {
		err = root.Rename(tmp, rel)
	}
	if err != nil {
		_ = root.Remove(tmp)
	}
	return err
}

// copyPathError is a refusal for an error os.Root gave on shown: a path
// that leaves the base, or a failed write.
func copyPathError(err error, shown string) error {
	if strings.Contains(err.Error(), "path escapes from parent") {
		return &CopyError{Reason: CopyOutside, Path: shown}
	}
	return &CopyError{Reason: CopyFailed, Path: shown}
}

// copyFolderName is the folder a copy of leaf's certificate goes in: its
// main domain (the subject's name when it is one of its names, else the
// first), without a leading wildcard, as Let's Encrypt names a live folder.
func copyFolderName(names []string, subject string) string {
	name := ""
	for _, n := range names {
		if strings.EqualFold(n, subject) {
			name = n
		}
	}
	if name == "" && len(names) > 0 {
		name = names[0]
	}
	name = normalDomain(strings.TrimPrefix(name, "*."))
	if strings.ContainsAny(name, "*/") {
		return ""
	}
	return name
}
