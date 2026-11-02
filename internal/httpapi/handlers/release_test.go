package handlers_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/PdYrust/karvan/internal/httpapi/handlers"
)

func TestIsNewerRelease(t *testing.T) {
	cases := []struct {
		current string
		latest  string
		want    bool
	}{
		{"0.5.15-beta", "v0.5.15-beta", false},
		{"v0.5.15-beta", "v0.5.15-beta", false},
		{"v0.5.15-beta-dirty", "v0.5.15-beta", false},
		{"v0.5.15-beta-2-g0352240", "v0.5.15-beta", false},
		{"0.5.14-beta", "v0.5.15-beta", true},
		{"v0.5.14-beta", "0.5.15-beta", true},
		{"0.5.15-beta", "v0.5.15", true},
		{"0.5.15", "v0.5.15-beta", false},
		{"0.5.15-beta.1", "v0.5.15-beta.2", true},
		{"0.5.15-beta.10", "v0.5.15-beta.2", false},
		{"0.5.15-alpha", "v0.5.15-beta", true},
		{"0.5.15", "v0.6.0", true},
		{"0.5.15", "v1.0.0-beta", true},
		{"1.2.0", "v1.1.9", false},
		{"dev", "v0.5.15-beta", false},
		{"0.5.15-beta", "not-a-version", false},
	}
	for _, tc := range cases {
		if got := handlers.IsNewerRelease(tc.current, tc.latest); got != tc.want {
			t.Errorf("IsNewerRelease(%q, %q) = %v, want %v", tc.current, tc.latest, got, tc.want)
		}
	}
}

func TestReleaseChecker(t *testing.T) {
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits.Add(1)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"tag_name":"v0.5.16-beta","draft":true},{"tag_name":"v0.5.15-beta","draft":false}]`))
	}))
	defer srv.Close()

	c := handlers.NewReleaseChecker(srv.URL)
	if got := c.Latest(context.Background(), "0.5.14-beta"); got != "0.5.15-beta" {
		t.Fatalf("Latest(0.5.14-beta) = %q, want 0.5.15-beta", got)
	}
	if got := c.Latest(context.Background(), "v0.5.14-beta"); got != "v0.5.15-beta" {
		t.Fatalf("Latest(v0.5.14-beta) = %q, want v0.5.15-beta", got)
	}
	if got := c.Latest(context.Background(), "0.5.15-beta"); got != "" {
		t.Fatalf("Latest(0.5.15-beta) = %q, want empty", got)
	}
	if n := hits.Load(); n != 1 {
		t.Fatalf("server hits = %d, want 1 (cached)", n)
	}
}
