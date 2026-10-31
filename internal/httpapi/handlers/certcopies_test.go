package handlers_test

import (
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

type copyOut struct {
	ID       int64  `json:"id"`
	Target   string `json:"target"`
	Dir      string `json:"dir"`
	CertFile string `json:"certFile"`
	KeyFile  string `json:"keyFile"`
	Panel    *struct {
		ID   int64  `json:"id"`
		Name string `json:"name"`
	} `json:"panel"`
}

// newCopyEnv is a signed-in env whose copies go under a temporary folder
// with /etc/letsencrypt and panel 7's /var/lib/marzban, and a pasted
// certificate for copy.example.com; it returns the folder and the
// certificate's copies path.
func newCopyEnv(t *testing.T) (*env, string, string) {
	t.Helper()
	e := newEnv(t)
	e.login("")
	base := t.TempDir()
	for _, d := range []string{"etc/letsencrypt", "var/lib/marzban"} {
		if err := os.MkdirAll(filepath.Join(base, d), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	e.bridge.CopyBase = base
	if _, err := e.db.Exec("INSERT INTO panels (id, env_path, name, kind, scanned_at) VALUES (7, '/opt/marzban/.env', 'marzban', 'marzban', 0)"); err != nil {
		t.Fatal(err)
	}
	cert, key := testCert(t, "copy.example.com")
	res := e.do(http.MethodPost, api+"/certificates", map[string]any{"name": "copy", "certPem": cert, "keyPem": key})
	var c struct {
		ID int64 `json:"id"`
	}
	res.json(t, &c)
	if res.status != http.StatusCreated {
		t.Fatalf("add certificate: %d %s", res.status, res.body)
	}
	return e, base, api + "/certificates/" + strconv.FormatInt(c.ID, 10) + "/copies"
}

// A copy goes to Let's Encrypt live or a panel's certificate folder, is
// listed on its certificate and audited.
func TestCopyCertificate(t *testing.T) {
	e, base, copies := newCopyEnv(t)
	var le, panel copyOut
	res := e.do(http.MethodPost, copies, map[string]any{"target": "letsencrypt"})
	res.json(t, &le)
	if res.status != http.StatusOK || le.Dir != "/etc/letsencrypt/live/copy.example.com" || le.Panel != nil {
		t.Fatalf("letsencrypt: %d %s", res.status, res.body)
	}
	res = e.do(http.MethodPost, copies, map[string]any{"target": "panel", "panelId": 7})
	res.json(t, &panel)
	if res.status != http.StatusOK || panel.Dir != "/var/lib/marzban/certs/copy.example.com" || panel.Panel == nil ||
		panel.Panel.Name != "marzban" {
		t.Fatalf("panel: %d %s", res.status, res.body)
	}
	for _, f := range []string{le.CertFile, le.KeyFile, panel.CertFile, panel.KeyFile} {
		if _, err := os.Stat(filepath.Join(base, f)); err != nil {
			t.Errorf("%s: %v", f, err)
		}
	}
	var list struct {
		Certificates []struct {
			Copies []copyOut `json:"copies"`
		} `json:"certificates"`
	}
	e.do(http.MethodGet, api+"/certificates", nil).json(t, &list)
	if len(list.Certificates) != 1 || len(list.Certificates[0].Copies) != 2 {
		t.Fatalf("certificates = %+v", list)
	}
	if n := len(auditRows(t, e, "certificate.copy")); n != 2 {
		t.Fatalf("certificate.copy audited %d times", n)
	}
}

// A refused copy says why and where, and a wrong target or certificate
// is refused before anything is written.
func TestCopyCertificateRefusals(t *testing.T) {
	e, base, copies := newCopyEnv(t)
	dir := filepath.Join(base, "etc/letsencrypt/live/copy.example.com")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "fullchain.pem"), []byte("certbot"), 0o644); err != nil {
		t.Fatal(err)
	}
	res := e.do(http.MethodPost, copies, map[string]any{"target": "letsencrypt"})
	var refused struct {
		Error struct {
			Code   string `json:"code"`
			Reason string `json:"reason"`
			Path   string `json:"path"`
		} `json:"error"`
	}
	res.json(t, &refused)
	if res.status != http.StatusConflict || refused.Error.Code != "copy_refused" || refused.Error.Reason != "foreign_file" ||
		refused.Error.Path != "/etc/letsencrypt/live/copy.example.com/fullchain.pem" {
		t.Fatalf("foreign file: %d %s", res.status, res.body)
	}
	for _, c := range []struct {
		path   string
		body   map[string]any
		status int
	}{
		{copies, map[string]any{"target": "elsewhere"}, http.StatusBadRequest},
		{copies, map[string]any{"target": "panel", "panelId": 99}, http.StatusNotFound},
		{api + "/certificates/99/copies", map[string]any{"target": "letsencrypt"}, http.StatusNotFound},
		{api + "/certificates/x/copies", map[string]any{"target": "letsencrypt"}, http.StatusBadRequest},
	} {
		if res := e.do(http.MethodPost, c.path, c.body); res.status != c.status {
			t.Errorf("%s %v: %d %s", c.path, c.body, res.status, res.body)
		}
	}
	if got, _ := os.ReadFile(filepath.Join(dir, "fullchain.pem")); string(got) != "certbot" {
		t.Fatalf("the foreign file changed: %q", got)
	}
}

// Stop copying forgets the copy and leaves its files.
func TestStopCertificateCopy(t *testing.T) {
	e, base, copies := newCopyEnv(t)
	var c copyOut
	e.do(http.MethodPost, copies, map[string]any{"target": "letsencrypt"}).json(t, &c)
	stop := copies + "/" + strconv.FormatInt(c.ID, 10)
	if res := e.do(http.MethodDelete, stop, nil); res.status != http.StatusNoContent {
		t.Fatalf("stop: %d %s", res.status, res.body)
	}
	if res := e.do(http.MethodDelete, stop, nil); res.status != http.StatusNotFound {
		t.Fatalf("stop again: %d %s", res.status, res.body)
	}
	if res := e.do(http.MethodDelete, copies+"/x", nil); res.status != http.StatusBadRequest {
		t.Fatalf("bad id: %d %s", res.status, res.body)
	}
	if _, err := os.Stat(filepath.Join(base, c.CertFile)); err != nil {
		t.Fatalf("the files went with the copy: %v", err)
	}
	if n := len(auditRows(t, e, "certificate.copy_stop")); n != 1 {
		t.Fatalf("certificate.copy_stop audited %d times", n)
	}
}
