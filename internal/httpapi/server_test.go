package httpapi

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"

	"github.com/PdYrust/karvan/internal/auth"
	"github.com/PdYrust/karvan/internal/httpapi/address"
	"github.com/PdYrust/karvan/internal/migration/run"
)

func TestNewNeedsTheWebBuild(t *testing.T) {
	_, err := New(Deps{BasePath: "abcdefgh", Web: fstest.MapFS{}, Logger: slog.New(slog.NewTextHandler(io.Discard, nil))})
	if err == nil {
		t.Fatal("New succeeded without a web build")
	}
}

// Routes of optional services exist only with them, and like every API
// route they need a session.
func TestOptionalRoutesNeedTheirService(t *testing.T) {
	web := fstest.MapFS{
		"login.html":          {Data: []byte(`<base href="/" />login`)},
		"index.html":          {Data: []byte(`<base href="/" />app`)},
		".vite/manifest.json": {Data: []byte(`{"login.html":{"file":"assets/login.js"}}`)},
	}
	routes := []string{"/abcdefgh/api/v1/settings/address", "/abcdefgh/api/v1/panels", "/abcdefgh/api/v1/audit"}
	for _, withServices := range []bool{false, true} {
		d := Deps{BasePath: "abcdefgh", Web: web, Auth: &auth.Service{Sessions: auth.NewSessions(nil)},
			Logger: slog.New(slog.NewTextHandler(io.Discard, nil))}
		want := map[string]int{routes[0]: http.StatusNotFound, routes[1]: http.StatusNotFound, routes[2]: http.StatusUnauthorized}
		if withServices {
			d.Address, d.Runs = &address.Server{}, &run.Service{}
			want[routes[0]], want[routes[1]] = http.StatusUnauthorized, http.StatusUnauthorized
		}
		h, err := New(d)
		if err != nil {
			t.Fatal(err)
		}
		for path, code := range want {
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
			if rec.Code != code {
				t.Errorf("services %v: GET %s = %d, want %d", withServices, path, rec.Code, code)
			}
		}
	}
}

func TestNewServesOnlyUnderBasePath(t *testing.T) {
	web := fstest.MapFS{
		"login.html":          {Data: []byte(`<base href="/" />login`)},
		"index.html":          {Data: []byte(`<base href="/" />app`)},
		".vite/manifest.json": {Data: []byte(`{"login.html":{"file":"assets/login.js"}}`)},
	}
	h, err := New(Deps{
		BasePath: "abcdefgh", Web: web, Auth: &auth.Service{Sessions: auth.NewSessions(nil)},
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	})
	if err != nil {
		t.Fatal(err)
	}
	for path, want := range map[string]int{"/abcdefgh/": http.StatusOK, "/": http.StatusNotFound, "/abcdefgh/nope": http.StatusNotFound, "/abcdefgh/templates": http.StatusNotFound,
		"/manifest.webmanifest": http.StatusNotFound, "/icon-192.png": http.StatusNotFound} {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
		if rec.Code != want {
			t.Errorf("GET %s = %d, want %d", path, rec.Code, want)
		}
	}
}
