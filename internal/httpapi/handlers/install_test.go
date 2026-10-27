package handlers_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"testing"
)

// installFiles are what Chrome fetches to install the panel as an app.
var installFiles = map[string]string{
	"manifest.webmanifest":  "application/manifest+json",
	"icon-192.png":          "image/png",
	"icon-512.png":          "image/png",
	"icon-maskable-512.png": "image/png",
	"apple-touch-icon.png":  "image/png",
}

// Chrome fetches the manifest and icons without the session cookie, so they
// are served signed out, under the hidden path only, with their real types.
func TestInstallFilesAreServedSignedOut(t *testing.T) {
	e := newEnv(t)
	for name, want := range installFiles {
		res := e.do(http.MethodGet, "/"+base+"/"+name, nil)
		if res.status != http.StatusOK || res.header.Get("Content-Type") != want {
			t.Errorf("%s: %d %q, want 200 %q", name, res.status, res.header.Get("Content-Type"), want)
		}
		if res.header.Get("Service-Worker-Allowed") != "" {
			t.Errorf("%s: Service-Worker-Allowed set", name)
		}
	}
	manifest := e.do(http.MethodGet, "/"+base+"/manifest.webmanifest", nil)
	if bytes.Contains(manifest.body, []byte(base)) {
		t.Errorf("manifest names the hidden path: %s", manifest.body)
	}
}

// Outside the hidden path the files do not exist, like any unknown path, so
// a caller without the path learns nothing from asking for them.
func TestInstallFilesAreNotFoundOutsideBasePath(t *testing.T) {
	e := newEnv(t)
	notFound := e.raw(http.MethodGet, "/nothing-here")
	for name := range installFiles {
		for _, path := range []string{"/" + name, "/x/" + name, "/" + base[:len(base)-1] + "/" + name} {
			if got := e.raw(http.MethodGet, path); !bytes.Equal(got, notFound) {
				t.Errorf("GET %s:\n%s", path, got)
			}
		}
	}
}

// The real manifest uses only relative URLs, so it works under any hidden
// path, and the path is never written into it.
func TestManifestIsRelative(t *testing.T) {
	raw, err := os.ReadFile("../../../web/public/manifest.webmanifest")
	if err != nil {
		t.Fatal(err)
	}
	var m struct {
		StartURL string `json:"start_url"`
		Scope    string `json:"scope"`
		Display  string `json:"display"`
		Icons    []struct {
			Src     string `json:"src"`
			Sizes   string `json:"sizes"`
			Type    string `json:"type"`
			Purpose string `json:"purpose"`
		} `json:"icons"`
	}
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	if m.StartURL != "./" || m.Scope != "./" || m.Display != "standalone" {
		t.Errorf("start_url %q scope %q display %q", m.StartURL, m.Scope, m.Display)
	}
	for _, icon := range m.Icons {
		if strings.Contains(icon.Src, "/") || installFiles[icon.Src] != icon.Type {
			t.Errorf("icon %+v is not one of the served files", icon)
		}
	}
}
