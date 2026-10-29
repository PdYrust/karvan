//go:build dockertest

package panelctl

import (
	"context"
	"net/http"
	"os"
	"testing"
)

// scripts/dockertest/run.sh runs this inside Karvan's image, which has no
// Docker CLI, with the socket mounted and a test panel in
// KARVAN_SMOKE_PANEL: mode auto must reach it through the socket.
func TestSocketSmoke(t *testing.T) {
	ctx := context.Background()
	c, err := Opener{Mode: ModeAuto}.Open(ctx, os.Getenv("KARVAN_SMOKE_PANEL"))
	if err != nil {
		t.Fatal(err)
	}
	if c.engine == nil || len(c.ids) != 1 {
		t.Fatalf("not through the socket: services %v", c.Services)
	}
	if err := c.Stop(ctx); err != nil {
		t.Fatal(err)
	}
	if running(t, c) {
		t.Fatal("the panel still runs after stop")
	}
	if err := c.Start(ctx); err != nil {
		t.Fatal(err)
	}
	if !running(t, c) {
		t.Fatal("the panel did not start")
	}
}

func running(t *testing.T, c *Control) bool {
	t.Helper()
	var info struct {
		State struct{ Running bool }
	}
	if err := c.engine.do(context.Background(), http.MethodGet, "/containers/"+c.ids[0]+"/json", &info); err != nil {
		t.Fatal(err)
	}
	return info.State.Running
}
