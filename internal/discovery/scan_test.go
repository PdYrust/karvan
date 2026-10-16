package discovery

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/PdYrust/karvan/internal/panel"
	"github.com/PdYrust/karvan/internal/panel/marzbandb"
	"github.com/PdYrust/karvan/internal/panel/paneldb/paneldbtest"
	"github.com/PdYrust/karvan/internal/panel/pasarguarddb"
)

func write(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}

func codes(p Panel) []string {
	var out []string
	for _, f := range p.Findings {
		out = append(out, f.Code)
	}
	return out
}

func TestScan(t *testing.T) {
	root := t.TempDir()
	m := paneldbtest.Load(t, paneldbtest.Fixture{Flavor: "sqlite", Path: "../panel/marzbandb/testdata/" + marzbandb.Revision + ".sqlite.sql"})
	p := paneldbtest.Load(t, paneldbtest.Fixture{Flavor: "sqlite", Path: "../panel/pasarguarddb/testdata/" + pasarguarddb.Revision + ".sqlite.sql"})
	xray := filepath.Join(root, "marzban", "xray_config.json")
	write(t, xray, `{"inbounds": [{"tag": "VLESS", "protocol": "vless"}]}`)
	write(t, filepath.Join(root, "marzban", ".env"), `SQLALCHEMY_DATABASE_URL = "sqlite:///`+m.Path+`"
XRAY_JSON = "`+xray+`"
UVICORN_PORT = 8443
XRAY_SUBSCRIPTION_PATH = "/links/"
`)
	write(t, filepath.Join(root, "pg-two", ".env"), "SQLALCHEMY_DATABASE_URL=sqlite+aiosqlite:///"+p.Path+"\n")
	write(t, filepath.Join(root, "copy", ".env"), "SQLALCHEMY_DATABASE_URL=sqlite+aiosqlite:///"+p.Path+"\n")
	write(t, filepath.Join(root, "relative", ".env"), "SQLALCHEMY_DATABASE_URL=sqlite:///db.sqlite3\n")
	write(t, filepath.Join(root, "relative", "docker-compose.yml"), "services:\n  marzban:\n    image: gozargah/marzban:latest\n")
	write(t, filepath.Join(root, "other-app", ".env"), "DEBUG=1\n")

	got := Scan(context.Background(), []string{root}, nil)
	byName := map[string]Panel{}
	for _, x := range got {
		byName[x.Name] = x
	}
	if _, ok := byName["other-app"]; ok || len(got) != 4 {
		t.Fatalf("scan found %d panels: %+v", len(got), got)
	}
	mz := byName["marzban"]
	if mz.Kind != panel.Marzban || !mz.Ready() || mz.Settings.Port != 8443 || mz.Settings.SubPath != "links" || mz.Revision != marzbandb.Revision {
		t.Errorf("marzban: %+v", mz)
	}
	// Folders are sorted, so "copy" is seen first and "pg-two" is the duplicate.
	if c := byName["copy"]; c.Kind != panel.PasarGuard || !c.Ready() || c.Settings.SubPath != "sub" {
		t.Errorf("pasarguard: %+v", c)
	}
	if pg := byName["pg-two"]; !slices.Contains(codes(pg), DBShared) {
		t.Errorf("shared database not flagged: %v", codes(pg))
	}
	if r := byName["relative"]; r.Hint != panel.Marzban || !slices.Contains(codes(r), DBRelativePath) {
		t.Errorf("relative: %+v", r)
	}
}

func TestInspectReportsMissingPieces(t *testing.T) {
	root := t.TempDir()
	m := paneldbtest.Load(t, paneldbtest.Fixture{Flavor: "sqlite", Path: "../panel/marzbandb/testdata/" + marzbandb.Revision + ".sqlite.sql"})
	write(t, filepath.Join(root, "m", ".env"), "SQLALCHEMY_DATABASE_URL=sqlite:///"+m.Path+"\nUVICORN_SSL_CERTFILE=/nonexistent/fullchain.pem\n")
	got := codes(Inspect(context.Background(), filepath.Join(root, "m", ".env")))
	for _, want := range []string{PathRelative, CertUnreadable} {
		if !slices.Contains(got, want) {
			t.Errorf("missing %s in %v", want, got)
		}
	}
	if got := codes(Inspect(context.Background(), filepath.Join(root, "none", ".env"))); !slices.Contains(got, EnvUnreadable) {
		t.Errorf("unreadable env: %v", got)
	}
}
