package pasarguardapi

import (
	"context"
	"errors"
	"net/http"
	"slices"
	"testing"

	"github.com/PdYrust/karvan/internal/panel"
	"github.com/PdYrust/karvan/internal/panel/fakepanel"
	"github.com/PdYrust/karvan/internal/panel/source"
	"github.com/PdYrust/karvan/internal/secret"
)

var testGroups = []map[string]any{
	{"id": 1, "name": "wireguard", "inbound_tags": []string{"wg0"}, "is_disabled": false, "total_users": 3},
	{"id": 2, "name": "mixed", "inbound_tags": []string{"VLESS TCP", "VMESS WS", "gone"}, "is_disabled": true, "total_users": 0},
	{"id": 3, "name": "empty", "inbound_tags": nil, "is_disabled": false, "total_users": 0},
}

var testInbounds = []map[string]any{
	{"tag": "wg0", "protocol": "wireguard", "network": nil, "wireguard_private_key": "server-secret"},
	{"tag": "VLESS TCP", "protocol": "vless", "network": "tcp"},
	{"tag": "VMESS WS", "protocol": "vmess", "network": "ws"},
}

func groupsSource(t *testing.T, groups, inbounds []map[string]any, status int) *Source {
	t.Helper()
	f := fakepanel.NewPasarGuard(admin, users(1), fakepanel.Faults{})
	t.Cleanup(f.Close)
	f.ServeGroups(groups, inbounds, status)
	return New(client(t, f.URL), Login{Username: "shop", Password: secret.NewValue("pw")})
}

func TestReadsGroupsWithInboundProtocols(t *testing.T) {
	got, err := groupsSource(t, testGroups, testInbounds, 0).Groups(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	want := []panel.SourceGroup{
		{ID: 1, Protocols: []string{"wireguard"}},
		{ID: 2, Disabled: true, Protocols: []string{"vless", "vmess"}},
		{ID: 3},
	}
	if !slices.EqualFunc(got, want, func(a, b panel.SourceGroup) bool {
		return a.ID == b.ID && a.Disabled == b.Disabled && slices.Equal(a.Protocols, b.Protocols)
	}) {
		t.Errorf("groups = %+v, want %+v", got, want)
	}
}

func TestGroupsUnknownOnForbiddenOrMissingRoute(t *testing.T) {
	for name, s := range map[string]*Source{
		"forbidden":      groupsSource(t, testGroups, testInbounds, http.StatusForbidden),
		"no routes":      groupsSource(t, nil, nil, 0),
		"no inbound api": groupsSource(t, testGroups, nil, 0),
	} {
		if _, err := s.Groups(context.Background()); !errors.Is(err, ErrGroupsUnknown) {
			t.Errorf("%s: %v, want ErrGroupsUnknown", name, err)
		}
	}
}

func TestRefusesMalformedGroups(t *testing.T) {
	for name, groups := range map[string][]map[string]any{
		"no id":       {{"name": "x", "inbound_tags": []string{}}},
		"string tags": {{"id": 1, "inbound_tags": "wg0"}},
	} {
		if _, err := groupsSource(t, groups, testInbounds, 0).Groups(context.Background()); source.CodeOf(err) != source.CodeBadResponse {
			t.Errorf("%s: %v", name, err)
		}
	}
}
