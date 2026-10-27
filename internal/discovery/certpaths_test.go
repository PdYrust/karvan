package discovery

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"syscall"
	"testing"
)

// The certificate paths come from every panel ENV under the roots and the
// extra ENV files, with no database probe: the database URL does not have
// to work. Files that are not a panel's, cannot be read or are too large
// give nothing.
func TestCertEnvs(t *testing.T) {
	root := t.TempDir()
	write(t, filepath.Join(root, "marzban", ".env"), "SQLALCHEMY_DATABASE_URL=mysql://u:p@127.0.0.1:1/none\n"+
		"UVICORN_SSL_CERTFILE=/var/lib/marzban/certs/fullchain.pem\nUVICORN_SSL_KEYFILE = \"/var/lib/marzban/certs/key.pem\"\n")
	write(t, filepath.Join(root, "compose-only", ".env"), "UVICORN_SSL_CERTFILE=/etc/ssl/a.pem\n")
	write(t, filepath.Join(root, "compose-only", "docker-compose.yml"), "services:\n  p:\n    image: pasarguard/panel:latest\n")
	write(t, filepath.Join(root, "other-app", ".env"), "UVICORN_SSL_CERTFILE=/etc/other/a.pem\n")
	write(t, filepath.Join(root, "big", ".env"), "SQLALCHEMY_DATABASE_URL=sqlite:////x\n"+strings.Repeat("#", maxEnvFileSize))
	extra := filepath.Join(t.TempDir(), "elsewhere", ".env")
	write(t, extra, "SQLALCHEMY_DATABASE_URL=sqlite:////x\nUVICORN_SSL_KEYFILE=relative/key.pem\n")
	gone := filepath.Join(root, "gone", ".env")

	got := CertEnvs([]string{root}, []string{extra, gone, filepath.Join(root, "marzban", ".env")})
	want := []CertEnv{
		{Folder: "compose-only", CertFile: "/etc/ssl/a.pem"},
		{Folder: "elsewhere", KeyFile: "relative/key.pem"},
		{Folder: "marzban", CertFile: "/var/lib/marzban/certs/fullchain.pem", KeyFile: "/var/lib/marzban/certs/key.pem"},
	}
	slices.SortFunc(got, func(a, b CertEnv) int { return strings.Compare(a.Folder, b.Folder) })
	if !slices.Equal(got, want) {
		t.Errorf("got %+v\nwant %+v", got, want)
	}
}

// A FIFO named .env is not opened, so the scan cannot hang on it.
func TestCertEnvsSkipAFIFO(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "p"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := syscall.Mkfifo(filepath.Join(root, "p", ".env"), 0o600); err != nil {
		t.Fatal(err)
	}
	if got := CertEnvs([]string{root}, nil); len(got) != 0 {
		t.Errorf("got %+v", got)
	}
}
