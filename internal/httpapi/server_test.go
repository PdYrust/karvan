package httpapi

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"

	"github.com/PdYrust/karvan/internal/auth"
)

func TestNewNeedsTheWebBuild(t *testing.T) {
	_, err := New(Deps{BasePath: "abcdefgh", Web: fstest.MapFS{}, Logger: slog.New(slog.NewTextHandler(io.Discard, nil))})
	if err == nil {
		t.Fatal("New succeeded without a web build")
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
	for path, want := range map[string]int{"/abcdefgh/": http.StatusOK, "/": http.StatusNotFound, "/abcdefgh/nope": http.StatusNotFound} {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
		if rec.Code != want {
			t.Errorf("GET %s = %d, want %d", path, rec.Code, want)
		}
	}
}
