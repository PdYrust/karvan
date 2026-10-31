package bridge

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"testing"

	"github.com/PdYrust/karvan/internal/audit"
)

// events collects the audit events the bridge records.
type events struct {
	mu   sync.Mutex
	list []audit.Event
}

func (e *events) Record(_ context.Context, ev audit.Event) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.list = append(e.list, ev)
	return nil
}

func (e *events) all() []audit.Event {
	e.mu.Lock()
	defer e.mu.Unlock()
	return slices.Clone(e.list)
}

// newCopyEnv is a cert env whose copies go under a temporary folder with
// /etc/letsencrypt and /var/lib/new (panel 1's) in it, and a pasted
// certificate for copy.example.com.
func newCopyEnv(t *testing.T) (*env, string, Certificate, *events) {
	t.Helper()
	e := newCertEnv(t)
	base := t.TempDir()
	for _, d := range []string{"etc/letsencrypt", "var/lib/new"} {
		if err := os.MkdirAll(filepath.Join(base, d), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	rec := &events{}
	e.b.CopyBase, e.b.Audit = base, rec
	c, k := longCert(t, "copy.example.com")
	return e, base, e.addCert(CertificateInput{Name: "copy", CertPEM: c, KeyPEM: k}), rec
}

func wantCopyError(t *testing.T, err error, reason, path string) {
	t.Helper()
	var ce *CopyError
	if !errors.As(err, &ce) || ce.Reason != reason || ce.Path != path {
		t.Fatalf("err = %v, want %s at %s", err, reason, path)
	}
}

func mode(t *testing.T, p string) os.FileMode {
	t.Helper()
	info, err := os.Lstat(p)
	if err != nil {
		t.Fatal(err)
	}
	return info.Mode().Perm()
}

// A copy writes fullchain.pem 0644 and privkey.pem 0600 in a new 0700
// folder named after the main domain, and the certificate lists it.
func TestCopyWritesTheFilesWithTheirModes(t *testing.T) {
	e, base, cert, _ := newCopyEnv(t)
	ctx := context.Background()
	le, err := e.b.CopyCertificate(ctx, cert.ID, CopyTarget{Target: CopyLetsEncrypt})
	if err != nil {
		t.Fatal(err)
	}
	if le.Dir != "/etc/letsencrypt/live/copy.example.com" || le.Panel != nil {
		t.Fatalf("copy = %+v", le)
	}
	panel, err := e.b.CopyCertificate(ctx, cert.ID, CopyTarget{Target: CopyPanel, PanelID: 1})
	if err != nil {
		t.Fatal(err)
	}
	if panel.Dir != "/var/lib/new/certs/copy.example.com" || panel.Panel == nil || panel.Panel.Name != "new" {
		t.Fatalf("panel copy = %+v", panel)
	}
	certPEM, keyPEM := e.b.material(certRow{pem: cert.CertPEM, id: cert.ID, sealed: e.sealedKey(cert.ID)})
	for _, c := range []CertCopy{le, panel} {
		wantCopied(t, base, c, certPEM, keyPEM)
	}
	if m := mode(t, filepath.Join(base, "var/lib/new/certs")); m != 0o700 {
		t.Errorf("a made certs folder: %v", m)
	}
	list, err := e.b.Certificates(ctx)
	if err != nil || len(list) != 1 || len(list[0].Copies) != 2 || list[0].Copies[0].ID != le.ID || list[0].Copies[1].ID != panel.ID {
		t.Fatalf("certificates = %+v, %v", list, err)
	}
}

// wantCopied checks c's folder is 0700 and holds the certificate 0644 and
// the key 0600.
func wantCopied(t *testing.T, base string, c CertCopy, certPEM, keyPEM []byte) {
	t.Helper()
	if m := mode(t, filepath.Join(base, c.Dir)); m != 0o700 {
		t.Errorf("%s: %v", c.Dir, m)
	}
	for _, f := range []struct {
		path string
		mode os.FileMode
		data []byte
	}{{c.CertFile, 0o644, certPEM}, {c.KeyFile, 0o600, keyPEM}} {
		got, err := os.ReadFile(filepath.Join(base, f.path))
		if m := mode(t, filepath.Join(base, f.path)); err != nil || string(got) != string(f.data) || m != f.mode {
			t.Errorf("%s: %v, %v", f.path, m, err)
		}
	}
}

// wantFile checks the file at path holds want.
func wantFile(t *testing.T, path, want, what string) {
	t.Helper()
	if got, _ := os.ReadFile(path); string(got) != want {
		t.Fatal(what)
	}
}

// sealedKey is certificate id's sealed key.
func (e *env) sealedKey(id int64) []byte {
	e.t.Helper()
	var k []byte
	if err := e.b.DB.QueryRow("SELECT key_sealed FROM certificates WHERE id = ?", id).Scan(&k); err != nil {
		e.t.Fatal(err)
	}
	return k
}

// A copy never follows a link and never replaces a file Karvan did not
// write; the file stays as it was.
func TestCopyRefusesLinksAndForeignFiles(t *testing.T) {
	e, base, cert, _ := newCopyEnv(t)
	ctx := context.Background()
	le := CopyTarget{Target: CopyLetsEncrypt}
	elsewhere := t.TempDir()
	if err := os.Symlink(elsewhere, filepath.Join(base, "etc/letsencrypt/live")); err != nil {
		t.Fatal(err)
	}
	_, err := e.b.CopyCertificate(ctx, cert.ID, le)
	wantCopyError(t, err, CopyLink, "/etc/letsencrypt/live")
	if err := os.Remove(filepath.Join(base, "etc/letsencrypt/live")); err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(base, "etc/letsencrypt/live/copy.example.com")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("../../archive/cert1.pem", filepath.Join(dir, "fullchain.pem")); err != nil {
		t.Fatal(err)
	}
	_, err = e.b.CopyCertificate(ctx, cert.ID, le)
	wantCopyError(t, err, CopyLink, "/etc/letsencrypt/live/copy.example.com/fullchain.pem")
	if err := os.Remove(filepath.Join(dir, "fullchain.pem")); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "privkey.pem"), []byte("theirs"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err = e.b.CopyCertificate(ctx, cert.ID, le)
	wantCopyError(t, err, CopyForeign, "/etc/letsencrypt/live/copy.example.com/privkey.pem")
	if got, _ := os.ReadFile(filepath.Join(dir, "privkey.pem")); string(got) != "theirs" {
		t.Fatalf("a foreign file was changed: %q", got)
	}
	if _, err := os.Lstat(filepath.Join(dir, "fullchain.pem")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("a file was written before the refusal: %v", err)
	}
	if n := e.count("SELECT count(*) FROM certificate_copies"); n != 0 {
		t.Fatalf("a refused copy is remembered: %d", n)
	}
	if err := os.RemoveAll(filepath.Join(base, "var/lib/new")); err != nil {
		t.Fatal(err)
	}
	_, err = e.b.CopyCertificate(ctx, cert.ID, CopyTarget{Target: CopyPanel, PanelID: 1})
	wantCopyError(t, err, CopyNotFound, "/var/lib/new")
}

// A file Karvan wrote is replaced on the next copy, but not once someone
// else has changed it.
func TestCopyReplacesOnlyWhatItWrote(t *testing.T) {
	e, base, cert, _ := newCopyEnv(t)
	ctx := context.Background()
	le := CopyTarget{Target: CopyLetsEncrypt}
	c, err := e.b.CopyCertificate(ctx, cert.ID, le)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := e.b.CopyCertificate(ctx, cert.ID, le); err != nil {
		t.Fatalf("copy again over its own files: %v", err)
	}
	if err := os.WriteFile(filepath.Join(base, c.CertFile), []byte("edited"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err = e.b.CopyCertificate(ctx, cert.ID, le)
	wantCopyError(t, err, CopyForeign, c.CertFile)
	other, k := longCert(t, "copy.example.com")
	second := e.addCert(CertificateInput{Name: "second", CertPEM: other, KeyPEM: k})
	_, err = e.b.CopyCertificate(ctx, second.ID, CopyTarget{Target: CopyPanel, PanelID: 1})
	if err != nil {
		t.Fatal(err)
	}
	_, err = e.b.CopyCertificate(ctx, cert.ID, CopyTarget{Target: CopyPanel, PanelID: 1})
	wantCopyError(t, err, CopyTaken, "/var/lib/new/certs/copy.example.com")
}

// No path leaves its base folder, through dots or a link inside it.
func TestCopyStaysInsideItsBase(t *testing.T) {
	e, base, _, _ := newCopyEnv(t)
	files := []copyFile{{name: copyCertName, data: []byte("x"), mode: copyCertMode}}
	for _, rel := range []string{"../escape", "live/../../escape", "/abs", "", "."} {
		_, err := e.b.writeCopy("/etc/letsencrypt", rel, files, nil)
		var ce *CopyError
		if !errors.As(err, &ce) || ce.Reason != CopyOutside {
			t.Errorf("%q: %v", rel, err)
		}
	}
	if err := os.Symlink(t.TempDir(), filepath.Join(base, "etc/letsencrypt/out")); err != nil {
		t.Fatal(err)
	}
	_, err := e.b.writeCopy("/etc/letsencrypt", "out/x", files, nil)
	wantCopyError(t, err, CopyLink, "/etc/letsencrypt/out")
	if _, err := os.Stat(filepath.Join(base, "escape")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("written outside: %v", err)
	}
}

// After a change of the certificate every copy is written again, logged
// and audited; a stopped copy is not.
func TestCopyFollowsChanges(t *testing.T) {
	e, base, cert, rec := newCopyEnv(t)
	ctx := context.Background()
	c, err := e.b.CopyCertificate(ctx, cert.ID, CopyTarget{Target: CopyLetsEncrypt})
	if err != nil {
		t.Fatal(err)
	}
	if err := e.b.Reload(ctx); err != nil {
		t.Fatal(err)
	}
	if n := len(rec.all()); n != 0 {
		t.Fatalf("an unchanged certificate was copied again: %d events", n)
	}
	next, k := longCert(t, "copy.example.com")
	if _, err := e.b.SaveCertificate(ctx, cert.ID, CertificateInput{Name: "copy", CertPEM: next, KeyPEM: k}); err != nil {
		t.Fatal(err)
	}
	wantFile(t, filepath.Join(base, c.CertFile), next, "the copy was not written again")
	wantFile(t, filepath.Join(base, c.KeyFile), k, "the key was not written again")
	ev := rec.all()
	if len(ev) != 1 || ev[0].Action != audit.ActionCertCopy || ev[0].Outcome != audit.Success || ev[0].Actor != audit.ActorSystem ||
		ev[0].Detail["dir"] != c.Dir {
		t.Fatalf("events = %+v", ev)
	}
	if _, err := e.b.StopCopy(ctx, cert.ID, c.ID); err != nil {
		t.Fatal(err)
	}
	last, k2 := longCert(t, "copy.example.com")
	if _, err := e.b.SaveCertificate(ctx, cert.ID, CertificateInput{Name: "copy", CertPEM: last, KeyPEM: k2}); err != nil {
		t.Fatal(err)
	}
	wantFile(t, filepath.Join(base, c.CertFile), next, "a stopped copy was written")
	if _, err := e.b.StopCopy(ctx, cert.ID, c.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("stop twice: %v", err)
	}
}

// A failed copy again is recorded on the copy and audited as a failure.
func TestFailedCopyAgainIsRecorded(t *testing.T) {
	e, base, cert, rec := newCopyEnv(t)
	ctx := context.Background()
	c, err := e.b.CopyCertificate(ctx, cert.ID, CopyTarget{Target: CopyLetsEncrypt})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(base, c.KeyFile), []byte("theirs"), 0o600); err != nil {
		t.Fatal(err)
	}
	next, k := longCert(t, "copy.example.com")
	if _, err := e.b.SaveCertificate(ctx, cert.ID, CertificateInput{Name: "copy", CertPEM: next, KeyPEM: k}); err != nil {
		t.Fatal(err)
	}
	list, err := e.b.Certificates(ctx)
	if err != nil {
		t.Fatal(err)
	}
	got := list[0].Copies[0]
	if got.Error == nil || got.Error.Reason != CopyForeign || got.Error.Path != c.KeyFile || got.CopiedAt == nil {
		t.Fatalf("copy = %+v", got)
	}
	if ev := rec.all(); len(ev) != 1 || ev[0].Outcome != audit.Failure || ev[0].Detail["reason"] != CopyForeign {
		t.Fatalf("events = %+v", ev)
	}
}

func TestCopyFolderName(t *testing.T) {
	for _, c := range []struct {
		names   []string
		subject string
		want    string
	}{
		{[]string{"a.example.com", "b.example.com"}, "b.example.com", "b.example.com"},
		{[]string{"a.example.com", "b.example.com"}, "", "a.example.com"},
		{[]string{"*.example.com", "example.com"}, "*.example.com", "example.com"},
		{[]string{"192.0.2.1"}, "", "192.0.2.1"},
		{[]string{"bad/name"}, "", ""},
		{nil, "", ""},
	} {
		if got := copyFolderName(c.names, c.subject); got != c.want {
			t.Errorf("%v %q: %q, want %q", c.names, c.subject, got, c.want)
		}
	}
}

func (e *env) count(q string) int {
	e.t.Helper()
	var n int
	if err := e.b.DB.QueryRow(q).Scan(&n); err != nil {
		e.t.Fatal(err)
	}
	return n
}
