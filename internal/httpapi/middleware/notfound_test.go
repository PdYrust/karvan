package middleware

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
)

const base = "abcDEF2345ghijkm"

func ok(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("X-Seen-Path", r.URL.Path)
	_, _ = w.Write([]byte("ok"))
}

func app() http.Handler {
	rt := NewRouter()
	rt.HandleFunc("GET /{$}", ok)
	rt.HandleFunc("GET /api/v1/session", ok)
	rt.HandleFunc("POST /api/v1/session", ok)
	return BasePath(base, rt)
}

func dump(t *testing.T, method, target string) []byte {
	t.Helper()
	rec := httptest.NewRecorder()
	app().ServeHTTP(rec, httptest.NewRequest(method, target, nil))
	var b bytes.Buffer
	b.WriteString(http.StatusText(rec.Code))
	_ = rec.Header().Write(&b)
	b.Write(rec.Body.Bytes())
	return b.Bytes()
}

func TestEveryMissIsTheSameBytes(t *testing.T) {
	want := dump(t, http.MethodGet, "/")
	misses := [][2]string{
		{http.MethodGet, "/wp-admin"},
		{http.MethodGet, "/" + base[:len(base)-1]},
		{http.MethodGet, "/" + base + "x/"},
		{http.MethodPut, "/" + base + "/api/v1/session"},
		{http.MethodDelete, "/" + base + "/"},
		{http.MethodGet, "/" + base + "/api/v1/nope"},
		{http.MethodGet, "/" + base + "/api//v1/session"},
		{http.MethodGet, "/" + base + "/api/v1/../v1/session"},
		{http.MethodGet, "/" + base + "/api%2Fv1/session"},
		{http.MethodPost, "/" + base},
		{http.MethodGet, "/" + base + "/assets/missing.js"},
	}
	for _, m := range misses {
		if got := dump(t, m[0], m[1]); !bytes.Equal(got, want) {
			t.Errorf("%s %s:\n%s\nwant:\n%s", m[0], m[1], got, want)
		}
	}
	if !bytes.Contains(want, []byte("Not Found")) || bytes.Contains(bytes.ToLower(want), []byte("karvan")) {
		t.Fatalf("404 body: %s", want)
	}
}

func TestGuardAnswersBeforeInnerLayers(t *testing.T) {
	rt := NewRouter()
	rt.HandleFunc("GET /x", ok)
	inner := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusTeapot) })
	h := rt.Guard(inner)
	for method, want := range map[string]int{http.MethodGet: http.StatusTeapot, http.MethodPost: http.StatusNotFound} {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(method, "/x", nil))
		if rec.Code != want {
			t.Errorf("%s /x = %d, want %d", method, rec.Code, want)
		}
	}
}

func TestNotFoundDropsEarlierHeaders(t *testing.T) {
	rec := httptest.NewRecorder()
	rec.Header().Set("Content-Security-Policy", "default-src 'none'")
	NotFound(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Header().Get("Content-Security-Policy") != "" {
		t.Fatal("NotFound kept a header from an earlier layer")
	}
}

func TestBasePathStripsPrefix(t *testing.T) {
	rec := httptest.NewRecorder()
	app().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/"+base+"/api/v1/session", nil))
	if rec.Code != 200 || rec.Header().Get("X-Seen-Path") != "/api/v1/session" {
		t.Fatalf("status %d, path %q", rec.Code, rec.Header().Get("X-Seen-Path"))
	}
}

func TestBareBasePathRedirectsToSlash(t *testing.T) {
	rec := httptest.NewRecorder()
	app().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/"+base+"?lang=fa", nil))
	if rec.Code != http.StatusPermanentRedirect || rec.Header().Get("Location") != "/"+base+"/?lang=fa" {
		t.Fatalf("status %d, location %q", rec.Code, rec.Header().Get("Location"))
	}
}
