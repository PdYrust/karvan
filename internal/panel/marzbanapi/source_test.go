package marzbanapi

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/PdYrust/karvan/internal/panel"
	"github.com/PdYrust/karvan/internal/panel/fakepanel"
	"github.com/PdYrust/karvan/internal/panel/source"
	"github.com/PdYrust/karvan/internal/secret"
)

var admin = fakepanel.MarzbanAdmin{Username: "reseller", Password: "pw", UsersUsage: 123}

func open(t *testing.T, f *fakepanel.Marzban, password string) *Source {
	t.Helper()
	c, err := source.New(source.Options{BaseURL: f.URL, Sleep: func(context.Context, time.Duration) error { return nil }})
	if err != nil {
		t.Fatal(err)
	}
	return New(c, admin.Username, secret.NewValue(password))
}

func users(n int) []map[string]any {
	out := make([]map[string]any, n)
	for i := range out {
		out[i] = fakepanel.MarzbanUser(fmt.Sprintf("user%04d", i), i, nil)
	}
	return out
}

func TestCurrentAdmin(t *testing.T) {
	f := fakepanel.NewMarzban(admin, nil, fakepanel.Faults{})
	defer f.Close()
	a, err := open(t, f, "pw").CurrentAdmin(context.Background())
	if err != nil || a.Username != "reseller" || a.IsOwner || a.UsedTraffic != 123 {
		t.Fatalf("admin = %+v, %v", a, err)
	}
}

func TestReadsEveryUserExactly(t *testing.T) {
	f := fakepanel.NewMarzban(admin, users(1203), fakepanel.Faults{RateLimitFirst: 2, ServerErrorAt: []int{5}})
	defer f.Close()
	got, err := open(t, f, "pw").Users(context.Background())
	if err != nil || len(got) != 1203 {
		t.Fatalf("%d users, %v", len(got), err)
	}
	u := got[7]
	if u.Username != "user0007" || u.Credentials.VLESS.ID != "00000000-0000-4000-8000-000000000007" ||
		u.Credentials.VLESS.Flow != "xtls-rprx-vision" || u.Credentials.Trojan.Password.Reveal() != "trojan-pass-7" {
		t.Fatalf("credentials: %+v", u.Credentials)
	}
	if u.UsedTraffic != 7000 || u.LifetimeUsed != 21000 || u.Expire.Unix() != 1790000000 ||
		u.CreatedAt.Nanosecond() != 123456000 || u.OwnerUsername != "reseller" {
		t.Fatalf("fields: %+v", u)
	}
	if u.SourceInbounds["vless"][0] != "VLESS TCP REALITY" {
		t.Fatalf("inbounds: %v", u.SourceInbounds)
	}
}

func TestLogsInAgainWhenTheTokenExpires(t *testing.T) {
	f := fakepanel.NewMarzban(admin, users(20), fakepanel.Faults{ExpireTokenAt: 3})
	defer f.Close()
	got, err := open(t, f, "pw").Users(context.Background())
	if err != nil || len(got) != 20 {
		t.Fatalf("%d, %v", len(got), err)
	}
}

func TestWrongPassword(t *testing.T) {
	f := fakepanel.NewMarzban(admin, users(1), fakepanel.Faults{})
	defer f.Close()
	if _, err := open(t, f, "nope").CurrentAdmin(context.Background()); source.CodeOf(err) != source.CodeUnauthorized {
		t.Fatalf("err = %v", err)
	}
}

func TestRefusesMalformedUsers(t *testing.T) {
	cases := map[string]map[string]any{
		"string traffic":  {"used_traffic": "12"},
		"unknown status":  {"status": "frozen"},
		"unknown proto":   {"proxies": map[string]any{"wireguard": map[string]any{}}},
		"no proxies":      {"proxies": map[string]any{}},
		"bad created_at":  {"created_at": "yesterday"},
		"negative usage":  {"used_traffic": -1},
		"null usage":      {"used_traffic": nil},
		"bad reset":       {"data_limit_reset_strategy": "hourly"},
		"settings string": {"proxies": map[string]any{"vmess": "id"}},
	}
	for name, set := range cases {
		list := users(3)
		list[1] = fakepanel.MarzbanUser("user0001", 1, set)
		f := fakepanel.NewMarzban(admin, list, fakepanel.Faults{})
		_, err := open(t, f, "pw").Users(context.Background())
		if source.CodeOf(err) != source.CodeBadResponse {
			t.Errorf("%s: err = %v", name, err)
		}
		f.Close()
	}
}

func TestCloseDropsCredentials(t *testing.T) {
	f := fakepanel.NewMarzban(admin, users(1), fakepanel.Faults{})
	defer f.Close()
	s := open(t, f, "pw")
	if _, err := s.CurrentAdmin(context.Background()); err != nil {
		t.Fatal(err)
	}
	s.Close()
	if !s.password.IsZero() || !s.token.IsZero() {
		t.Fatal("credentials kept after Close")
	}
	if _, err := s.CurrentAdmin(context.Background()); err == nil {
		t.Fatal("a closed source still works")
	}
}

var _ panel.Source = (*Source)(nil)
