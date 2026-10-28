package handlers_test

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/PdYrust/karvan/internal/migration/migrationtest"
	"github.com/PdYrust/karvan/internal/panel/fakepanel"
	"github.com/PdYrust/karvan/internal/panel/subtoken"
)

type lockedBuffer struct {
	mu sync.Mutex
	b  bytes.Buffer
}

func (l *lockedBuffer) Write(p []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.b.Write(p)
}

func (l *lockedBuffer) String() string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.b.String()
}

// D51: after a whole run through the API (with Safe Mode, the downloads
// and a bridge token upload), no secret used anywhere is in the logs or
// in any file under Karvan's data directory: the database, its WAL and
// the Safe Mode backup.
func TestNoSecretLeavesTheRun(t *testing.T) {
	logs := &lockedBuffer{}
	testLog = logs
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(logs, &slog.HandlerOptions{Level: slog.LevelDebug})))
	t.Cleanup(func() { testLog = io.Discard; slog.SetDefault(prev) })

	m := newMigrationEnv(t, 0)
	secrets := []string{migrationtest.JWTSecret}
	m.src, secrets = secretSource(t, secrets)
	id := m.create()
	v := m.plan(id)
	if res := m.do(http.MethodPost, api+"/migrations/"+id+"/run", map[string]any{"planVersion": v, "reload": "manual"}); res.status != http.StatusAccepted {
		t.Fatalf("run: %d %s", res.status, res.body)
	}
	m.runs.Wait()
	secrets = append(secrets, csvColumn(t, m.do(http.MethodGet, api+"/migrations/"+id+"/links.csv", nil).body, 2, true)...)
	secrets = append(secrets, uploadOldToken(t, m))
	if len(secrets) < 10 {
		t.Fatalf("only %d secrets to look for", len(secrets))
	}
	scanFor(t, "the logs", []byte(logs.String()), secrets)
	if !strings.Contains(logs.String(), "/migrations/") {
		t.Fatal("the access log is empty: the scan would prove nothing")
	}
	seen := false // a control: the scan does read what Karvan stored
	err := filepath.WalkDir(m.data, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		scanFor(t, p, b, secrets)
		seen = seen || bytes.Contains(b, []byte("user001"))
		return nil
	})
	if err != nil || !seen {
		t.Fatalf("data scan: %v, saw usernames: %v", err, seen)
	}
}

// secretSource is an old Marzban whose password and user secrets are long
// enough to find anywhere.
func secretSource(t *testing.T, secrets []string) (migrationtest.Source, []string) {
	t.Helper()
	pw := "old-panel-login-password-0123456789"
	var users []map[string]any
	for i := range 3 {
		vless, vmess := fmt.Sprintf("5ec2e7a1-0000-4000-8000-%012d", i), fmt.Sprintf("5ec2e7a1-0000-4000-9000-%012d", i)
		trojan, ss := fmt.Sprintf("trojan-secret-%d-abcdefghij", i), fmt.Sprintf("shadowsocks-secret-%d-abcdefghij", i)
		users = append(users, fakepanel.MarzbanUser(fmt.Sprintf("user%03d", i), i, map[string]any{"proxies": map[string]any{
			"vless": map[string]any{"id": vless, "flow": ""}, "vmess": map[string]any{"id": vmess},
			"trojan": map[string]any{"password": trojan, "flow": ""}, "shadowsocks": map[string]any{"password": ss, "method": "aes-256-gcm"},
		}, "inbounds": map[string]any{"shadowsocks": []string{"Shadowsocks TCP"}}}))
		secrets = append(secrets, vless, vmess, trojan, ss)
	}
	f := fakepanel.NewMarzban(fakepanel.MarzbanAdmin{Username: "reseller", Password: pw}, users, fakepanel.Faults{})
	t.Cleanup(f.Close)
	return migrationtest.Source{URL: f.URL, Username: "reseller", Password: pw, Close: f.Close, Subscribe: f.Subscribe}, append(secrets, pw)
}

// csvColumn reads one column of a downloaded CSV; links keep only the
// token at the end.
func csvColumn(t *testing.T, body []byte, col int, lastSegment bool) []string {
	t.Helper()
	rows, err := csv.NewReader(bytes.NewReader(bytes.TrimPrefix(body, []byte("\ufeff")))).ReadAll()
	if err != nil || len(rows) < 2 {
		t.Fatalf("csv %q: %v", body, err)
	}
	var out []string
	for _, r := range rows[1:] {
		v := r[col]
		if lastSegment {
			v = v[strings.LastIndexByte(v, '/')+1:]
		}
		out = append(out, v)
	}
	return out
}

// uploadOldToken adds a legacy site and uploads one old link to it.
func uploadOldToken(t *testing.T, m *migrationEnv) string {
	t.Helper()
	tok := subtoken.Marzban("user001", "old-secret", time.Now().Add(-time.Minute))
	m.src.Subscribe(tok, "user001")
	created := m.do(http.MethodPost, api+"/legacy-sites", map[string]any{"name": "old", "panelId": m.panelID, "sourceKind": "marzban",
		"sourceUrl": m.src.URL, "sourceIp": "127.0.0.1", "oldPath": "sub", "port": freePort(t), "lazy": true, "plainHttpOk": true})
	var s struct{ ID int64 }
	created.json(t, &s)
	t.Cleanup(func() { _ = m.bridge.DeleteSite(t.Context(), s.ID) })
	path := fmt.Sprintf("%s/legacy-sites/%d/tokens", api, s.ID)
	if res := uploadText(t, m, path, "https://old.example.com/sub/"+tok); !strings.Contains(res, `"found":1`) {
		t.Fatalf("upload: %s", res)
	}
	if res := waitUpload(t, m, path); !strings.Contains(string(res.body), `"stored":1`) {
		t.Fatalf("checked: %s", res.body)
	}
	return tok
}

func uploadText(t *testing.T, m *migrationEnv, path, text string) string {
	t.Helper()
	req, _ := http.NewRequest(http.MethodPost, m.srv.URL+path, strings.NewReader(text))
	req.Header.Set("Content-Type", "text/plain")
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	req.Header.Set("X-CSRF-Token", m.csrf)
	resp, err := m.client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var b bytes.Buffer
	_, _ = b.ReadFrom(resp.Body)
	return b.String()
}

func scanFor(t *testing.T, where string, data []byte, secrets []string) {
	t.Helper()
	for _, s := range secrets {
		if bytes.Contains(data, []byte(s)) {
			t.Errorf("%s holds a secret: %.12s...", where, s)
		}
	}
}
