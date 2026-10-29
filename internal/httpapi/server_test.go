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

// A deep page under the right hidden path signs the operator in again when
// the session has ended, at the same address; everything else stays the one
// plain 404, so the hidden path is never revealed.
func TestDeepPagesWithoutSessionServeSignIn(t *testing.T) {
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
	serve := func(method, path string) *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(method, path, nil))
		return rec
	}
	signIn, notFound := serve(http.MethodGet, "/abcdefgh/"), serve(http.MethodGet, "/nothing-here")
	for _, p := range appPages {
		for _, path := range []string{"/abcdefgh/" + p, "/abcdefgh/" + p + "/rest"} {
			for _, method := range []string{http.MethodGet, http.MethodHead} {
				rec := serve(method, path)
				if rec.Code != http.StatusOK || rec.Header().Get("Cache-Control") != "no-store" || rec.Body.String() != signIn.Body.String() {
					t.Errorf("%s %s = %d %q %q, want the sign-in page", method, path, rec.Code, rec.Header().Get("Cache-Control"), rec.Body)
				}
			}
		}
	}
	for _, c := range [][2]string{{http.MethodGet, "/wrongpath/domains"}, {http.MethodGet, "/abcdefgh/nope"},
		{http.MethodPost, "/abcdefgh/domains"}, {http.MethodGet, "/abcdefgh/assets/app-1.js"}, {http.MethodGet, "/domains"}} {
		rec := serve(c[0], c[1])
		if rec.Code != notFound.Code || rec.Body.String() != notFound.Body.String() || rec.Header().Get("Content-Type") != notFound.Header().Get("Content-Type") {
			t.Errorf("%s %s = %d %q, want the plain 404", c[0], c[1], rec.Code, rec.Body)
		}
	}
}
