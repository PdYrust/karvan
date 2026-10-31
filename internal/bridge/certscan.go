package bridge

import (
	"context"
	"crypto/tls"
	"encoding/pem"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/PdYrust/karvan/internal/discovery"
)

// Why a scan place was not scanned, as codes the web panel puts in words.
const (
	SkipNotFound   = "not_found"
	SkipUnreadable = "unreadable"
	SkipNotAllowed = "not_allowed" // it is, holds or is inside Karvan's data dir or key folder, or is too short
	SkipTooMany    = "too_many"
	SkipTime       = "time"
)

// Scan limits.
const (
	maxScanDepth   = 3 // folder levels below a place
	maxScanEntries = 2000
	maxScanFiles   = 200
	maxFolderKeys  = 20
	scanTimeout    = 10 * time.Second
)

// ErrScanBusy means a certificate scan is running already.
var ErrScanBusy = errors.New("a certificate scan is running")

// ScanFound is a certificate the scan found with the key that pairs with
// it. The paths are the walked ones (a Let's Encrypt live link, not its
// archive file), so a certificate added from them follows renewals.
type ScanFound struct {
	Place    string    `json:"place"`
	CertFile string    `json:"certFile"`
	KeyFile  string    `json:"keyFile"`
	Facts    CertFacts `json:"facts"`
	Added    bool      `json:"added"` // a stored certificate has these two paths
}

// ScanSkipped is a place the scan did not read, or did not finish.
type ScanSkipped struct {
	Place  string `json:"place"`
	Reason string `json:"reason"`
}

// ScanResult is what a certificate scan found. It holds facts and paths
// only, never a key.
type ScanResult struct {
	Found   []ScanFound   `json:"found"`
	Skipped []ScanSkipped `json:"skipped"`
}

// scanTest holds what tests change in a scan: a folder the fixed places
// are moved under, a shorter time limit, and hooks around each file open.
type scanTest struct {
	base       string
	timeout    time.Duration
	beforeOpen func(rel string)
	opened     func(info fs.FileInfo, read bool)
}

// scanPlace is one folder the scan reads. It is opened as an os.Root at
// boundary and walked from start, the place relative to the boundary.
type scanPlace struct {
	dir      string
	boundary string
	start    string
	depth    int
}

// ScanCertificates looks for certificates with their keys in the fixed
// places on this server. It only reads: every file is opened
// through its place's os.Root, so no link leads out of a place, and
// Karvan's data dir and key file are never read.
func (b *Bridge) ScanCertificates(ctx context.Context) (ScanResult, error) {
	if !b.scanMu.TryLock() {
		return ScanResult{}, ErrScanBusy
	}
	defer b.scanMu.Unlock()
	own, err := b.ownFiles()
	if err != nil {
		return ScanResult{}, err
	}
	places, err := b.scanPlaces(ctx)
	if err != nil {
		return ScanResult{}, err
	}
	added, err := b.storedPairs(ctx)
	if err != nil {
		return ScanResult{}, err
	}
	limit := scanTimeout
	if b.scanT.timeout != 0 {
		limit = b.scanT.timeout
	}
	ctx, cancel := context.WithTimeout(ctx, limit)
	defer cancel()
	s := &scan{b: b, ctx: ctx, own: own, added: added, seen: map[string]bool{},
		res: ScanResult{Found: []ScanFound{}, Skipped: []ScanSkipped{}}}
	for _, p := range places {
		s.place(p)
	}
	b.Log.Info("bridge: certificate scan", "found", len(s.res.Found), "skipped", len(s.res.Skipped))
	return s.res, nil
}

// ownFiles are Karvan's data dir and key folder: their paths after links,
// which no place may hold, be or be inside, and the folders themselves,
// which the walk never enters. key is the key file, never read.
type ownFiles struct {
	paths []string
	dirs  []os.FileInfo
	key   os.FileInfo
}

func (b *Bridge) ownFiles() (ownFiles, error) {
	var own ownFiles
	if b.DataDir == "" || b.KeyFile == "" {
		return own, errors.New("certificate scan: data dir or key file not set")
	}
	for _, p := range []string{b.DataDir, filepath.Dir(b.KeyFile)} {
		abs, err := filepath.Abs(p)
		if err != nil {
			return own, err
		}
		if r, err := filepath.EvalSymlinks(abs); err == nil {
			abs = r
		}
		own.paths = append(own.paths, abs)
		if info, err := os.Stat(abs); err == nil {
			own.dirs = append(own.dirs, info)
		}
	}
	var err error
	if own.key, err = os.Stat(b.KeyFile); err != nil {
		return own, fmt.Errorf("certificate scan: %w", err)
	}
	return own, nil
}

// isOwn reports whether info is Karvan's data dir or key folder.
func (o ownFiles) isOwn(info os.FileInfo) bool {
	return slices.ContainsFunc(o.dirs, func(d os.FileInfo) bool { return os.SameFile(d, info) })
}

// scanPlaces lists the scan places, each once: /root/cert, Let's
// Encrypt live, and for every panel ENV /var/lib/<folder>/certs and the
// folders of its certificate and key paths (those folders' own files).
func (b *Bridge) scanPlaces(ctx context.Context) ([]scanPlace, error) {
	base := b.scanT.base
	places := []scanPlace{
		{dir: base + "/root/cert", boundary: base + "/root/cert", start: ".", depth: maxScanDepth},
		{dir: base + "/etc/letsencrypt/live", boundary: base + "/etc/letsencrypt", start: "live", depth: maxScanDepth},
	}
	registered, err := b.panelEnvs(ctx)
	if err != nil {
		return nil, err
	}
	for _, env := range discovery.CertEnvs(b.EnvRoots, registered) {
		if f := env.Folder; f != "" && f != "." && f != ".." && !strings.Contains(f, "/") {
			lib := base + "/var/lib/" + f + "/certs"
			places = append(places, scanPlace{dir: lib, boundary: lib, start: ".", depth: maxScanDepth})
		}
		for _, f := range []string{env.CertFile, env.KeyFile} {
			if filepath.IsAbs(f) && filepath.Clean(f) == f {
				d := filepath.Dir(f)
				places = append(places, scanPlace{dir: d, boundary: d, start: "."})
			}
		}
	}
	seen := map[string]bool{}
	out := places[:0]
	for _, p := range places {
		if !seen[p.dir] {
			seen[p.dir] = true
			out = append(out, p)
		}
	}
	return out, nil
}

func (b *Bridge) panelEnvs(ctx context.Context) ([]string, error) {
	rows, err := b.DB.QueryContext(ctx, "SELECT env_path FROM panels ORDER BY id")
	if err != nil {
		return nil, fmt.Errorf("read panels: %w", err)
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var p string
		if err := rows.Scan(&p); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// storedPairs are the file pairs of the stored certificates.
func (b *Bridge) storedPairs(ctx context.Context) (map[[2]string]bool, error) {
	rows, err := b.DB.QueryContext(ctx, "SELECT cert_file, key_file FROM certificates WHERE cert_file <> ''")
	if err != nil {
		return nil, fmt.Errorf("read certificates: %w", err)
	}
	defer rows.Close()
	out := map[[2]string]bool{}
	for rows.Next() {
		var c, k string
		if err := rows.Scan(&c, &k); err != nil {
			return nil, err
		}
		out[[2]string{c, k}] = true
	}
	return out, rows.Err()
}

// scan is one run of ScanCertificates.
type scan struct {
	b     *Bridge
	ctx   context.Context
	own   ownFiles
	added map[[2]string]bool
	seen  map[string]bool // certificate paths found
	files int             // files opened
	stop  string          // set when the time or the file limit is reached
	res   ScanResult
}

func (s *scan) skip(place, reason string) {
	s.res.Skipped = append(s.res.Skipped, ScanSkipped{Place: place, Reason: reason})
}

// place walks one place and reads its candidates folder by folder.
func (s *scan) place(p scanPlace) {
	if s.stop == "" && s.ctx.Err() != nil {
		s.stop = SkipTime
	}
	if s.stop != "" {
		s.skip(p.dir, s.stop)
		return
	}
	root, reason := openPlace(p, s.own)
	if reason != "" {
		s.skip(p.dir, reason)
		return
	}
	defer root.Close()
	folders, reason := s.walk(root, p)
	for _, f := range folders {
		if s.stop != "" {
			break
		}
		s.folder(root, p, f)
	}
	switch {
	case s.stop != "":
		s.skip(p.dir, s.stop)
	case reason != "":
		s.skip(p.dir, reason)
	}
}

// openPlace checks p against the scan limits and opens its boundary as an os.Root,
// or says why it is skipped.
func openPlace(p scanPlace, own ownFiles) (*os.Root, string) {
	if tooShort(p.dir) {
		return nil, SkipNotAllowed
	}
	bound, err := filepath.EvalSymlinks(p.boundary)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return nil, SkipNotFound
	case err != nil:
		return nil, SkipUnreadable
	}
	if tooShort(bound) {
		return nil, SkipNotAllowed
	}
	for _, o := range own.paths {
		if within(bound, o) || within(o, bound) {
			return nil, SkipNotAllowed
		}
	}
	info, err := os.Stat(bound)
	switch {
	case err != nil:
		return nil, SkipUnreadable
	case !info.IsDir():
		return nil, SkipNotFound
	}
	root, err := os.OpenRoot(bound)
	if err != nil {
		return nil, SkipUnreadable
	}
	// The folder could have been swapped between the check and the open.
	if now, err := root.Stat("."); err != nil || !os.SameFile(now, info) {
		_ = root.Close()
		return nil, SkipUnreadable
	}
	if own.isOwn(info) {
		_ = root.Close()
		return nil, SkipNotAllowed
	}
	if reason := checkStart(root, p.start); reason != "" {
		_ = root.Close()
		return nil, reason
	}
	return root, ""
}

// checkStart says why the walk cannot start at start inside root: it must
// be a folder there, not a link.
func checkStart(root *os.Root, start string) string {
	if start == "." {
		return ""
	}
	info, err := root.Lstat(start)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return SkipNotFound
	case err != nil || !info.IsDir():
		return SkipUnreadable
	}
	return ""
}

// tooShort is / or a folder of one element, such as /etc.
func tooShort(dir string) bool {
	return len(strings.Split(strings.Trim(dir, "/"), "/")) < 2
}

// within reports whether p is dir or inside it.
func within(p, dir string) bool {
	rel, err := filepath.Rel(dir, p)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, "../")
}

// walk lists the candidates of p by folder, in walk order. It enters no
// folder reached through a link and goes at most p.depth folders deep.
func (s *scan) walk(root *os.Root, p scanPlace) ([][]string, string) {
	var folders [][]string
	index := map[string]int{}
	entries, reason := 0, ""
	_ = fs.WalkDir(root.FS(), p.start, func(rel string, d fs.DirEntry, err error) error {
		if s.ctx.Err() != nil {
			s.stop = SkipTime
			return fs.SkipAll
		}
		if err != nil {
			if rel == p.start {
				reason = SkipUnreadable
			}
			return nil
		}
		if rel == p.start {
			return nil
		}
		if entries++; entries > maxScanEntries {
			reason = SkipTooMany
			return fs.SkipAll
		}
		if d.IsDir() {
			// Karvan's own folders are not entered, even when a folder was
			// swapped while the place was checked.
			if info, err := d.Info(); err != nil || s.own.isOwn(info) || levels(p.start, rel) > p.depth {
				return fs.SkipDir
			}
			return nil
		}
		if !scanCandidate(d) {
			return nil
		}
		dir := path.Dir(rel)
		i, ok := index[dir]
		if !ok {
			i = len(folders)
			index[dir] = i
			folders = append(folders, nil)
		}
		folders[i] = append(folders[i], rel)
		return nil
	})
	return folders, reason
}

// levels is how many folders deep rel is below start.
func levels(start, rel string) int {
	if start != "." {
		rel = strings.TrimPrefix(rel, start+"/")
	}
	return strings.Count(rel, "/") + 1
}

// scanCandidate is a regular file or a link with a certificate or key
// name. Folders, devices, FIFOs, sockets and other special files are not.
func scanCandidate(d fs.DirEntry) bool {
	t := d.Type()
	if !t.IsRegular() && t != fs.ModeSymlink {
		return false
	}
	switch strings.ToLower(path.Ext(d.Name())) {
	case ".pem", ".crt", ".cer", ".cert", ".key":
		return true
	}
	return false
}

// folder reads the candidates of one folder and adds every certificate
// that pairs with a key of the same folder. The keys never leave it.
func (s *scan) folder(root *os.Root, p scanPlace, rels []string) {
	type file struct {
		rel  string
		data []byte
	}
	var certs, keys []file
	defer func() {
		for _, k := range keys {
			clear(k.data)
		}
	}()
	for _, rel := range rels {
		if s.ctx.Err() != nil {
			s.stop = SkipTime
			return
		}
		if s.files >= maxScanFiles {
			s.stop = SkipTooMany
			return
		}
		s.files++
		data, ok := s.read(root, rel)
		if !ok {
			continue
		}
		isCert, isKey := firstCert(data) != nil, hasPrivateKey(data)
		if isCert {
			certs = append(certs, file{rel, data})
		}
		if isKey && len(keys) < maxFolderKeys {
			keys = append(keys, file{rel, data})
		} else if !isCert {
			clear(data)
		}
	}
	for _, c := range certs {
		for _, k := range keys {
			if _, err := tls.X509KeyPair(c.data, k.data); err != nil {
				continue
			}
			cf, kf := filepath.Join(p.boundary, c.rel), filepath.Join(p.boundary, k.rel)
			if !s.seen[cf] {
				s.seen[cf] = true
				s.res.Found = append(s.res.Found, ScanFound{Place: p.dir, CertFile: cf, KeyFile: kf,
					Facts: inspect(c.data, k.data, nil, s.b.Now()), Added: s.added[[2]string{cf, kf}]})
			}
			break
		}
	}
}

// read opens rel through root and reads it, or leaves it out.
func (s *scan) read(root *os.Root, rel string) ([]byte, bool) {
	if s.b.scanT.beforeOpen != nil {
		s.b.scanT.beforeOpen(rel)
	}
	data, info, err := readSmallFileIn(root, rel, s.own.key)
	if s.b.scanT.opened != nil && info != nil {
		s.b.scanT.opened(info, err == nil)
	}
	return data, err == nil
}

// hasPrivateKey reports whether data holds a PEM private key block.
func hasPrivateKey(data []byte) bool {
	for len(data) > 0 {
		var block *pem.Block
		block, data = pem.Decode(data)
		if block == nil {
			return false
		}
		if strings.HasSuffix(block.Type, "PRIVATE KEY") {
			return true
		}
	}
	return false
}
