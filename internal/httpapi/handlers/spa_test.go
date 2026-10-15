package handlers_test

import (
	"maps"
	"testing"
	"testing/fstest"

	"github.com/PdYrust/karvan/internal/httpapi/handlers"
)

func TestNewSPARejectsBrokenBuilds(t *testing.T) {
	without := func(name string) fstest.MapFS {
		m := maps.Clone(webBuild)
		delete(m, name)
		return m
	}
	with := func(name, data string) fstest.MapFS {
		m := maps.Clone(webBuild)
		m[name] = &fstest.MapFile{Data: []byte(data)}
		return m
	}
	for name, dist := range map[string]fstest.MapFS{
		"no login page":    without("login.html"),
		"no app page":      without("index.html"),
		"no base tag":      with("login.html", "<html></html>"),
		"no manifest":      without(".vite/manifest.json"),
		"bad manifest":     with(".vite/manifest.json", "{"),
		"no login in json": with(".vite/manifest.json", `{"index.html":{"file":"assets/app-1.js"}}`),
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := handlers.NewSPA(dist, base); err == nil {
				t.Fatal("NewSPA accepted a broken build")
			}
		})
	}
}
