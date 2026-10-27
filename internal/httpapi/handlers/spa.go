package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"mime"
	"net/http"
	"path"

	"github.com/PdYrust/karvan/internal/httpapi/middleware"
)

const (
	loginEntry = "login.html"
	appEntry   = "index.html"
	manifest   = ".vite/manifest.json"
	// The HTML ships with this tag; the real hidden path is put in at serve time
	// so the path can change without a rebuild.
	basePlaceholder = `<base href="/" />`
)

// SPA serves the two HTML entries and their assets from the web build.
// Only what the sign-in page needs is served without a session; the app's
// own code is NotFound until the operator signs in.
type SPA struct {
	dist      fs.FS
	loginHTML []byte
	appHTML   []byte
	public    map[string]bool
	gz        gzipCache
}

type manifestChunk struct {
	File           string   `json:"file"`
	Imports        []string `json:"imports"`
	DynamicImports []string `json:"dynamicImports"`
	CSS            []string `json:"css"`
	Assets         []string `json:"assets"`
}

// NewSPA reads the web build in dist and prepares it for basePath.
func NewSPA(dist fs.FS, basePath string) (*SPA, error) {
	s := &SPA{dist: dist}
	base := []byte(`<base href="/` + basePath + `/" />`)
	for name, dst := range map[string]*[]byte{loginEntry: &s.loginHTML, appEntry: &s.appHTML} {
		html, err := fs.ReadFile(dist, name)
		if err != nil {
			return nil, fmt.Errorf("web build is missing %s (run: make web): %w", name, err)
		}
		if !bytes.Contains(html, []byte(basePlaceholder)) {
			return nil, fmt.Errorf("web build %s has no %s tag", name, basePlaceholder)
		}
		*dst = bytes.Replace(html, []byte(basePlaceholder), base, 1)
	}
	public, err := publicFiles(dist)
	if err != nil {
		return nil, err
	}
	s.public = public
	return s, nil
}

// publicFiles collects every file the login entry loads, directly or
// through dynamic imports.
func publicFiles(dist fs.FS) (map[string]bool, error) {
	raw, err := fs.ReadFile(dist, manifest)
	if err != nil {
		return nil, fmt.Errorf("web build is missing %s: %w", manifest, err)
	}
	var chunks map[string]manifestChunk
	if err := json.Unmarshal(raw, &chunks); err != nil {
		return nil, fmt.Errorf("parse %s: %w", manifest, err)
	}
	if _, ok := chunks[loginEntry]; !ok {
		return nil, errors.New("web manifest has no login entry")
	}
	public := map[string]bool{}
	seen := map[string]bool{}
	var walk func(key string)
	walk = func(key string) {
		if seen[key] {
			return
		}
		seen[key] = true
		c := chunks[key]
		public[c.File] = true
		for _, f := range append(c.CSS, c.Assets...) {
			public[f] = true
		}
		for _, k := range append(c.Imports, c.DynamicImports...) {
			walk(k)
		}
	}
	walk(loginEntry)
	return public, nil
}

// App serves the app for a deep link into one of its pages, so a reload or
// a shared link opens the same page. Without a session these paths do not
// exist, like any other unknown path.
func (s *SPA) App(w http.ResponseWriter, r *http.Request) {
	if _, ok := middleware.Session(r.Context()); !ok {
		middleware.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	s.gz.write(w, r, appEntry, s.appHTML)
}

// Page serves the sign-in page or the app, depending on the session.
func (s *SPA) Page(w http.ResponseWriter, r *http.Request) {
	name, html := loginEntry, s.loginHTML
	if _, ok := middleware.Session(r.Context()); ok {
		name, html = appEntry, s.appHTML
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	s.gz.write(w, r, name, html)
}

// Asset serves a hashed file from assets/.
func (s *SPA) Asset(w http.ResponseWriter, r *http.Request) {
	name := "assets/" + r.PathValue("file")
	_, signedIn := middleware.Session(r.Context())
	if !s.public[name] && !signedIn {
		middleware.NotFound(w, r)
		return
	}
	data, err := fs.ReadFile(s.dist, name)
	if err != nil {
		middleware.NotFound(w, r)
		return
	}
	if ct := mime.TypeByExtension(path.Ext(name)); ct != "" {
		w.Header().Set("Content-Type", ct)
	}
	// File names carry a content hash, so they can be cached for good.
	visibility := "private"
	if s.public[name] {
		visibility = "public"
	}
	w.Header().Set("Cache-Control", visibility+", max-age=31536000, immutable")
	s.gz.write(w, r, name, data)
}

// Static serves a top-level file from the build's public folder, such as
// the favicon. These are shown on the sign-in page, so they need no session.
func (s *SPA) Static(name string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		data, err := fs.ReadFile(s.dist, name)
		if err != nil {
			middleware.NotFound(w, r)
			return
		}
		ct := mime.TypeByExtension(path.Ext(name))
		if path.Ext(name) == ".webmanifest" {
			// Not in Go's built-in table, and the system's table may not have it.
			ct = "application/manifest+json"
		}
		if ct != "" {
			w.Header().Set("Content-Type", ct)
		}
		w.Header().Set("Cache-Control", "public, max-age=86400")
		s.gz.write(w, r, name, data)
	}
}
