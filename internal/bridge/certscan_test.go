package bridge

import (
	"context"
	"encoding/json"
	"errors"
	"io/fs"
	"log/slog"
	"net"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/PdYrust/karvan/internal/secret"
	kstore "github.com/PdYrust/karvan/internal/store"
)

// scanEnv is a bridge whose fixed scan places live under base, with
// Karvan's data dir at base/var/lib/karvan and its key file at
// base/etc/karvan/secret.key, and a record of every file the scan opened.
type scanEnv struct {
	t       *testing.T
	b       *Bridge
	base    string
	keyFile string
	opened  []fs.FileInfo
	read    []fs.FileInfo
}

func newScanEnv(t *testing.T) *scanEnv {
	t.Helper()
	base := t.TempDir()
	data := filepath.Join(base, "var", "lib", "karvan")
	keyFile := filepath.Join(base, "etc", "karvan", "secret.key")
	mkdirs(t, data, filepath.Dir(keyFile))
	db, err := kstore.Open(context.Background(), data)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := secret.CreateKeyFile(keyFile); err != nil {
		t.Fatal(err)
	}
	key, err := secret.LoadKeyFile(keyFile)
	if err != nil {
		t.Fatal(err)
	}
	e := &scanEnv{t: t, base: base, keyFile: keyFile}
	e.b = New(Deps{DB: db, Key: key, Log: slog.New(slog.DiscardHandler), DataDir: data, KeyFile: keyFile,
		EnvRoots: []string{filepath.Join(base, "opt")}})
	e.b.scanT.base = base
	e.b.scanT.opened = func(info fs.FileInfo, read bool) {
		e.opened = append(e.opened, info)
		if read {
			e.read = append(e.read, info)
		}
	}
	return e
}

func mkdirs(t *testing.T, dirs ...string) {
	t.Helper()
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0o700); err != nil {
			t.Fatal(err)
		}
	}
}

func writeFile(t *testing.T, path, data string) {
	t.Helper()
	mkdirs(t, filepath.Dir(path))
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}
}

func symlink(t *testing.T, target, link string) {
	t.Helper()
	mkdirs(t, filepath.Dir(link))
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
}

// pair writes a certificate for name and its key as dir/file+".crt" and
// dir/file+".key".
func (e *scanEnv) pair(dir, file, name string) (string, string) {
	e.t.Helper()
	c, k := longCert(e.t, name)
	cf, kf := filepath.Join(dir, file+".crt"), filepath.Join(dir, file+".key")
	writeFile(e.t, cf, c)
	writeFile(e.t, kf, k)
	return cf, kf
}

// panelEnv writes a panel ENV file under base/opt/folder.
func (e *scanEnv) panelEnv(folder, body string) {
	e.t.Helper()
	writeFile(e.t, filepath.Join(e.base, "opt", folder, ".env"), "SQLALCHEMY_DATABASE_URL=sqlite:////none\n"+body)
}

func (e *scanEnv) scan() ScanResult {
	e.t.Helper()
	r, err := e.b.ScanCertificates(context.Background())
	if err != nil {
		e.t.Fatal(err)
	}
	return r
}

// openedAny reports whether the scan opened any of paths.
func (e *scanEnv) openedAny(paths ...string) string {
	for _, p := range paths {
		info, err := os.Stat(p)
		if err != nil {
			e.t.Fatal(err)
		}
		if slices.ContainsFunc(e.opened, func(o fs.FileInfo) bool { return os.SameFile(o, info) }) {
			return p
		}
	}
	return ""
}

func foundFiles(r ScanResult) []string {
	out := []string{}
	for _, f := range r.Found {
		out = append(out, f.CertFile+" "+f.KeyFile)
	}
	return out
}

func skipReason(r ScanResult, place string) string {
	for _, s := range r.Skipped {
		if s.Place == place {
			return s.Reason
		}
	}
	return ""
}

// The scan finds pairs in /root/cert, under Let's Encrypt live (reported
// by the live links, so an added certificate follows renewals), in
// /var/lib/<panel>/certs and in the folders of the panel ENV certificate
// paths. Missing places are skipped, keys never leave, and a stored
// certificate with the same paths is marked added.
func TestScanFindsPairsInThePlaces(t *testing.T) {
	e := newScanEnv(t)
	rootCert := filepath.Join(e.base, "root", "cert")
	cf, kf := e.pair(rootCert, "a", "a.example.com")
	writeFile(t, filepath.Join(rootCert, "notes.txt"), "not a certificate")
	le := filepath.Join(e.base, "etc", "letsencrypt")
	c, k := longCert(t, "le.example.com")
	writeFile(t, filepath.Join(le, "archive", "le.example.com", "fullchain1.pem"), c)
	writeFile(t, filepath.Join(le, "archive", "le.example.com", "privkey1.pem"), k)
	symlink(t, "../../archive/le.example.com/fullchain1.pem", filepath.Join(le, "live", "le.example.com", "fullchain.pem"))
	symlink(t, "../../archive/le.example.com/privkey1.pem", filepath.Join(le, "live", "le.example.com", "privkey.pem"))
	acct := filepath.Join(le, "accounts", "acme", "private_key.pem")
	writeFile(t, acct, k)
	lib := filepath.Join(e.base, "var", "lib", "pg", "certs")
	lcf, lkf := e.pair(lib, "l", "l.example.com")
	envDir := filepath.Join(t.TempDir(), "ssl", "panel")
	ecf, ekf := e.pair(envDir, "e", "e.example.com")
	e.panelEnv("pg", "")
	e.panelEnv("gone", "")
	reg := filepath.Join(e.base, "srv", "reg", ".env")
	writeFile(t, reg, "SQLALCHEMY_DATABASE_URL=sqlite:////none\nUVICORN_SSL_CERTFILE="+ecf+"\nUVICORN_SSL_KEYFILE="+ekf+"\n")
	if _, err := e.b.DB.Exec("INSERT INTO panels (id, env_path, name, kind, scanned_at) VALUES (1, ?, 'reg', 'marzban', 0)",
		reg); err != nil {
		t.Fatal(err)
	}
	if _, err := e.b.SaveCertificate(context.Background(), 0, CertificateInput{Name: "a", CertFile: cf, KeyFile: kf}); err != nil {
		t.Fatal(err)
	}

	r := e.scan()
	live := filepath.Join(le, "live", "le.example.com")
	want := []string{cf + " " + kf, filepath.Join(live, "fullchain.pem") + " " + filepath.Join(live, "privkey.pem"),
		lcf + " " + lkf, ecf + " " + ekf}
	if got := foundFiles(r); !slices.Equal(got, want) {
		t.Fatalf("found %v\nwant %v", got, want)
	}
	if !r.Found[0].Added || r.Found[1].Added || r.Found[0].Place != rootCert || r.Found[1].Place != filepath.Join(le, "live") {
		t.Errorf("found %+v", r.Found)
	}
	if f := r.Found[1].Facts; !slices.Equal(f.Domains, []string{"le.example.com"}) || len(f.Warnings) != 0 {
		t.Errorf("facts %+v", f)
	}
	if got := skipReason(r, filepath.Join(e.base, "var", "lib", "gone", "certs")); got != SkipNotFound {
		t.Errorf("missing place: %q in %+v", got, r.Skipped)
	}
	if p := e.openedAny(acct, filepath.Join(rootCert, "notes.txt")); p != "" {
		t.Errorf("opened %s", p)
	}
	body, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(body), "PRIVATE KEY") || !strings.Contains(string(body), `"skipped":[`) {
		t.Errorf("answer: %s", body)
	}
}

// A link out of the place, a link to a folder, an absolute link (even to
// a file inside the place) and a relative link that climbs out are all
// left out, and nothing outside the place is opened.
func TestScanFollowsNoLinkOutOfThePlace(t *testing.T) {
	e := newScanEnv(t)
	rootCert := filepath.Join(e.base, "root", "cert")
	cf, kf := e.pair(rootCert, "a", "a.example.com")
	outside := filepath.Join(e.base, "outside")
	shadow := filepath.Join(outside, "shadow")
	writeFile(t, shadow, "root:secret")
	ocf, okf := e.pair(outside, "o", "o.example.com")
	symlink(t, shadow, filepath.Join(rootCert, "shadow.pem"))
	symlink(t, "../../outside/shadow", filepath.Join(rootCert, "climb.pem"))
	symlink(t, "/etc/shadow", filepath.Join(rootCert, "etc-shadow.pem"))
	symlink(t, outside, filepath.Join(rootCert, "dir"))
	symlink(t, "../../outside", filepath.Join(rootCert, "reldir"))
	symlink(t, cf, filepath.Join(rootCert, "abs.crt"))

	r := e.scan()
	if got := foundFiles(r); !slices.Equal(got, []string{cf + " " + kf}) {
		t.Errorf("found %v", got)
	}
	if p := e.openedAny(shadow, ocf, okf, e.keyFile); p != "" {
		t.Errorf("opened %s", p)
	}
}

// FIFOs, sockets and devices are skipped by type in the walk; one swapped
// in later is opened without blocking and closed unread.
func TestScanSkipsSpecialFiles(t *testing.T) {
	e := newScanEnv(t)
	rootCert := filepath.Join(e.base, "root", "cert")
	cf, kf := e.pair(rootCert, "a", "a.example.com")
	if err := syscall.Mkfifo(filepath.Join(rootCert, "fifo.pem"), 0o600); err != nil {
		t.Fatal(err)
	}
	ln, err := net.Listen("unix", filepath.Join(rootCert, "s.pem"))
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	done := make(chan ScanResult)
	go func() { done <- e.scan() }()
	select {
	case r := <-done:
		if got := foundFiles(r); !slices.Equal(got, []string{cf + " " + kf}) {
			t.Errorf("found %v", got)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("the scan blocked on a special file")
	}
	for _, m := range []fs.FileMode{fs.ModeDevice, fs.ModeDevice | fs.ModeCharDevice, fs.ModeNamedPipe, fs.ModeSocket, fs.ModeIrregular, fs.ModeDir} {
		if scanCandidate(fakeEntry{name: "x.pem", mode: m}) {
			t.Errorf("mode %v is a candidate", m)
		}
	}
	if !scanCandidate(fakeEntry{name: "x.pem", mode: fs.ModeSymlink}) || !scanCandidate(fakeEntry{name: "x.CRT"}) ||
		scanCandidate(fakeEntry{name: "x.txt"}) {
		t.Error("candidate names")
	}
	f, err := os.Open("/dev/null")
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := readOpened(f, nil); err == nil {
		t.Error("a device was read")
	}
}

type fakeEntry struct {
	name string
	mode fs.FileMode
}

func (f fakeEntry) Name() string               { return f.name }
func (f fakeEntry) IsDir() bool                { return f.mode.IsDir() }
func (f fakeEntry) Type() fs.FileMode          { return f.mode.Type() }
func (f fakeEntry) Info() (fs.FileInfo, error) { return nil, errors.New("no info") }
