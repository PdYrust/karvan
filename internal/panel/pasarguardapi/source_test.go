package pasarguardapi

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/PdYrust/karvan/internal/panel"
	"github.com/PdYrust/karvan/internal/panel/fakepanel"
	"github.com/PdYrust/karvan/internal/panel/source"
	"github.com/PdYrust/karvan/internal/secret"
)

var admin = fakepanel.PasarGuardAdmin{ID: 7, Username: "shop", Password: "pw", APIKey: "pg_key_0f0e", UsedTraffic: 99}

func client(t *testing.T, url string) *source.Client {
	t.Helper()
	c, err := source.New(source.Options{BaseURL: url, Sleep: func(context.Context, time.Duration) error { return nil }})
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func users(n int) []map[string]any {
	out := make([]map[string]any, n)
	for i := range out {
		out[i] = fakepanel.PasarGuardUser(int64(i+1), fmt.Sprintf("user%04d", i), nil)
	}
	return out
}

func TestPasswordAndAPIKeySignIn(t *testing.T) {
	f := fakepanel.NewPasarGuard(admin, users(3), fakepanel.Faults{})
	defer f.Close()
	for _, l := range []Login{
		{Username: "shop", Password: secret.NewValue("pw")},
		{APIKey: secret.NewValue("pg_key_0f0e")},
	} {
		a, err := New(client(t, f.URL), l).CurrentAdmin(context.Background())
		if err != nil || a.SourceID != 7 || a.IsOwner || a.UsedTraffic != 99 {
			t.Errorf("%+v %v", a, err)
		}
	}
	if _, err := New(client(t, f.URL), Login{APIKey: secret.NewValue("pg_key_bad")}).CurrentAdmin(context.Background()); source.CodeOf(err) != source.CodeUnauthorized {
		t.Errorf("bad key: %v", err)
	}
}

func TestReadsUsersWithSubscriptionLinks(t *testing.T) {
	f := fakepanel.NewPasarGuard(admin, users(1001), fakepanel.Faults{RateLimitFirst: 1, ExpireTokenAt: 4})
	defer f.Close()
	got, err := New(client(t, f.URL), Login{Username: "shop", Password: secret.NewValue("pw")}).Users(context.Background())
	if err != nil || len(got) != 1001 {
		t.Fatalf("%d %v", len(got), err)
	}
	u := got[0]
	c := u.Credentials
	if u.SourceID != 1 || c.Shadowsocks.Method != "chacha20-ietf-poly1305" || c.Hysteria.Auth.Reveal() != "hy-1" ||
		c.VLESS.ID != "22222222-0000-4000-8000-000000000001" {
		t.Fatalf("user: %+v", u)
	}
	if !strings.Contains(u.SubscriptionURL, "/sub/") {
		t.Fatalf("load_sub was not sent: %q", u.SubscriptionURL)
	}
	if u.Expire.Format(time.RFC3339) != "2026-12-31T20:30:00Z" || u.CreatedAt.Nanosecond() != 654321000 ||
		*u.HWIDLimit != 3 || len(u.SourceGroups) != 2 || u.ResetStrategy != panel.ResetMonth {
		t.Fatalf("fields: %+v", u)
	}
}

func TestOlderShapeAndUnixExpire(t *testing.T) {
	old := admin
	old.Shape = "is_sudo"
	list := []map[string]any{fakepanel.PasarGuardUser(5, "a", map[string]any{"expire": 1790000000, "data_limit_reset_strategy": nil})}
	f := fakepanel.NewPasarGuard(old, list, fakepanel.Faults{})
	defer f.Close()
	s := New(client(t, f.URL), Login{Username: "shop", Password: secret.NewValue("pw")})
	info, err := s.Info(context.Background())
	if err != nil || info.Capabilities.RBAC {
		t.Fatalf("%+v %v", info, err)
	}
	got, err := s.Users(context.Background())
	if err != nil || got[0].Expire.Unix() != 1790000000 || got[0].ResetStrategy != panel.ResetNever {
		t.Fatalf("%+v %v", got, err)
	}
}

func TestRefusesMalformedUsers(t *testing.T) {
	for name, set := range map[string]map[string]any{
		"no id":         {"id": nil},
		"bad settings":  {"proxy_settings": "x"},
		"bad expire":    {"expire": map[string]any{}},
		"string groups": {"group_ids": "1,2"},
	} {
		f := fakepanel.NewPasarGuard(admin, []map[string]any{fakepanel.PasarGuardUser(1, "a", set)}, fakepanel.Faults{})
		_, err := New(client(t, f.URL), Login{APIKey: secret.NewValue("pg_key_0f0e")}).Users(context.Background())
		if source.CodeOf(err) != source.CodeBadResponse {
			t.Errorf("%s: %v", name, err)
		}
		f.Close()
	}
}

var _ panel.Source = (*Source)(nil)
