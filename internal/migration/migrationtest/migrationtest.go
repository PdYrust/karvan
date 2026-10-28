// Package migrationtest sets up destination panels and fake sources for
// migration tests: an ENV file and (for Marzban) an Xray config pointing
// at a fixture database, and fake old panels with generated users.
package migrationtest

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"testing"

	"github.com/PdYrust/karvan/internal/panel"
	"github.com/PdYrust/karvan/internal/panel/fakepanel"
	"github.com/PdYrust/karvan/internal/panel/paneldb"
)

// XrayInbounds are the Marzban test inbounds; MarzbanTargets picks two.
const XrayInbounds = `{"inbounds": [{"tag": "VLESS TCP", "protocol": "vless"}, {"tag": "VMess WS", "protocol": "vmess"},
 {"tag": "Trojan WS", "protocol": "trojan"}, {"tag": "Shadowsocks TCP", "protocol": "shadowsocks"}]}`

// MarzbanTargets and PasarGuardTargets are what tests move users into.
var (
	MarzbanTargets    = []string{"vless/VLESS TCP", "vmess/VMess WS", "trojan/Trojan WS", "shadowsocks/Shadowsocks TCP"}
	PasarGuardTargets = []string{"1"}
)

// URL turns a test target into the SQLAlchemy URL a panel ENV holds.
func URL(tg paneldb.Target, kind panel.Kind) string {
	switch tg.Dialect {
	case paneldb.SQLite:
		return "sqlite:///" + tg.Path
	case paneldb.MySQL:
		drv := "mysql+pymysql"
		if kind == panel.PasarGuard {
			drv = "mysql+asyncmy"
		}
		return fmt.Sprintf("%s://%s:%s@%s:%d/%s", drv, tg.User, url.QueryEscape(tg.Password.Reveal()), tg.Host, tg.Port, tg.Database)
	}
	return fmt.Sprintf("postgresql+asyncpg://%s:%s@%s:%d/%s", tg.User, url.QueryEscape(tg.Password.Reveal()), tg.Host, tg.Port, tg.Database)
}

// PanelEnv writes a panel folder for tg and returns its ENV path. For
// PasarGuard it also adds group 1 with a shadowsocks inbound.
func PanelEnv(t *testing.T, tg paneldb.Target, kind panel.Kind) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), string(kind))
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	env := "SQLALCHEMY_DATABASE_URL = \"" + URL(tg, kind) + "\"\n"
	execSQL(t, tg, "INSERT INTO jwt (id, secret_key) VALUES (?, ?)", 1, JWTSecret)
	if kind == panel.Marzban {
		xray := filepath.Join(dir, "xray_config.json")
		if err := os.WriteFile(xray, []byte(XrayInbounds), 0o600); err != nil {
			t.Fatal(err)
		}
		env += "XRAY_JSON = \"" + xray + "\"\n"
	} else {
		seedGroup(t, tg)
	}
	p := filepath.Join(dir, ".env")
	if err := os.WriteFile(p, []byte(env), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

// JWTSecret is the destination's jwt.secret_key in tests.
const JWTSecret = "karvan-test-jwt-secret-not-used-anywhere-else-0123456789abcdef" //nolint:gosec // G101: a test value for test databases

func execSQL(t *testing.T, tg paneldb.Target, q string, args ...any) {
	t.Helper()
	ctx := context.Background()
	w, err := paneldb.OpenWriter(ctx, tg)
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	if err := w.Chunk(ctx, func(x paneldb.Execer) error { _, err := x.ExecContext(ctx, q, args...); return err }); err != nil {
		t.Fatal(err)
	}
}

func seedGroup(t *testing.T, tg paneldb.Target) {
	t.Helper()
	ctx := context.Background()
	w, err := paneldb.OpenWriter(ctx, tg)
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	err = w.Chunk(ctx, func(x paneldb.Execer) error {
		for _, s := range []struct {
			q    string
			args []any
		}{
			{"INSERT INTO inbounds (id, tag) VALUES (?, ?)", []any{1, "Shadowsocks TCP"}},
			{"INSERT INTO " + paneldb.Ident(tg.Dialect, "groups") + " (id, name, is_disabled) VALUES (?, ?, ?)", []any{1, "Main", false}},
			{"INSERT INTO inbounds_groups_association (inbound_id, group_id) VALUES (?, ?)", []any{1, 1}},
		} {
			if _, err := x.ExecContext(ctx, s.q, s.args...); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

// Source is a fake old panel and the login for it.
type Source struct {
	URL      string
	Kind     panel.Kind
	Username string
	Password string
	APIKey   string // PasarGuard only
	Close    func()
	// Mutate changes the panel's users between reads.
	Mutate func(fn func(users *[]map[string]any))
	// Subscribe makes a subscription token valid for a user ("" revokes).
	Subscribe func(token, username string)
	// ServeGroups makes a PasarGuard fake answer its groups and inbound
	// details (fakepanel.PasarGuard.ServeGroups); nil on Marzban.
	ServeGroups func(groups, inbounds []map[string]any, status int)
}

// NewSource starts a fake old panel of kind with n users.
func NewSource(t *testing.T, kind panel.Kind, n int) Source {
	t.Helper()
	users := make([]map[string]any, n)
	if kind == panel.Marzban {
		for i := range users {
			users[i] = fakepanel.MarzbanUser(fmt.Sprintf("user%03d", i), i, map[string]any{
				"proxies": map[string]any{
					"vless":       map[string]any{"id": fmt.Sprintf("00000000-0000-4000-8000-%012d", i), "flow": ""},
					"vmess":       map[string]any{"id": fmt.Sprintf("00000000-0000-4000-9000-%012d", i)},
					"trojan":      map[string]any{"password": fmt.Sprintf("trojan-pass-%d", i), "flow": ""},
					"shadowsocks": map[string]any{"password": fmt.Sprintf("shadowsocks-password-%04d", i), "method": "aes-256-gcm"},
				},
				"inbounds": map[string]any{"vless": []string{"VLESS TCP"}, "vmess": []string{"VMess WS"}, "trojan": []string{"Trojan WS"},
					"shadowsocks": []string{"Shadowsocks TCP"}},
			})
		}
		f := fakepanel.NewMarzban(fakepanel.MarzbanAdmin{Username: "reseller", Password: "pw"}, users, fakepanel.Faults{})
		return Source{URL: f.URL, Kind: kind, Username: "reseller", Password: "pw", Close: f.Close, Mutate: f.Mutate, Subscribe: f.Subscribe}
	}
	for i := range users {
		users[i] = fakepanel.PasarGuardUser(int64(i+1), fmt.Sprintf("user%03d", i), nil)
	}
	f := fakepanel.NewPasarGuard(fakepanel.PasarGuardAdmin{ID: 7, Username: "shop", Password: "pw", APIKey: testAPIKey}, users, fakepanel.Faults{})
	return Source{URL: f.URL, Kind: kind, Username: "shop", Password: "pw", APIKey: testAPIKey, Close: f.Close, Mutate: f.Mutate, Subscribe: f.Subscribe,
		ServeGroups: f.ServeGroups}
}

// testAPIKey is the fake PasarGuard admin's API key.
const testAPIKey = "shop-api-key-for-tests" //nolint:gosec // G101: a test value for the fake panel
