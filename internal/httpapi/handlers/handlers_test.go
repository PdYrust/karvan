package handlers_test

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestSignInOverviewSignOut(t *testing.T) {
	e := newEnv(t)
	if res := e.do(http.MethodGet, "/"+base+"/api/v1/overview", nil); res.status != http.StatusUnauthorized {
		t.Fatalf("overview before sign-in: %d", res.status)
	}
	res := e.login("")
	if res.status != http.StatusOK {
		t.Fatalf("sign-in: %d %s", res.status, res.body)
	}
	if !strings.Contains(res.header.Get("Set-Cookie"), "HttpOnly") {
		t.Fatalf("session cookie flags: %q", res.header.Values("Set-Cookie"))
	}

	var ov struct {
		Version  string `json:"version"`
		SafeMode bool   `json:"safeMode"`
		Security struct {
			TOTPEnabled    bool `json:"totpEnabled"`
			PreviousSignIn *struct {
				At time.Time `json:"at"`
			} `json:"previousSignIn"`
		} `json:"security"`
		Migrations struct {
			Jobs map[string]int64 `json:"jobs"`
		} `json:"migrations"`
		Activity []struct {
			Action  string `json:"action"`
			Outcome string `json:"outcome"`
		} `json:"activity"`
	}
	res = e.do(http.MethodGet, "/"+base+"/api/v1/overview", nil)
	if res.status != http.StatusOK {
		t.Fatalf("overview: %d %s", res.status, res.body)
	}
	res.json(t, &ov)
	if ov.Version != "test" || !ov.SafeMode || ov.Migrations.Jobs == nil || len(ov.Activity) != 1 || ov.Activity[0].Action != "login" {
		t.Fatalf("overview = %+v", ov)
	}
	if ov.Security.PreviousSignIn != nil {
		t.Fatalf("first sign-in has a previous sign-in: %+v", ov.Security.PreviousSignIn)
	}

	if res := e.do(http.MethodDelete, "/"+base+"/api/v1/session", nil); res.status != http.StatusNoContent {
		t.Fatalf("sign-out: %d %s", res.status, res.body)
	}
	if res := e.do(http.MethodGet, "/"+base+"/api/v1/overview", nil); res.status != http.StatusUnauthorized {
		t.Fatalf("overview after sign-out: %d", res.status)
	}

	e.login("")
	e.do(http.MethodGet, "/"+base+"/api/v1/overview", nil).json(t, &ov)
	if ov.Security.PreviousSignIn == nil {
		t.Fatal("second sign-in does not show the previous one")
	}
}

func TestSignInErrors(t *testing.T) {
	e := newEnv(t)
	e.fetchCSRF()
	bad := map[string]string{"username": username, "password": "wrong password!"}
	res := e.do(http.MethodPost, "/"+base+"/api/v1/session", bad)
	if res.status != http.StatusUnauthorized || res.errorCode(t) != "invalid_credentials" {
		t.Fatalf("wrong password: %d %s", res.status, res.body)
	}
	if res := e.do(http.MethodPost, "/"+base+"/api/v1/session", map[string]string{"username": ""}); res.status != http.StatusBadRequest {
		t.Fatalf("empty body: %d", res.status)
	}
	for range 4 {
		e.do(http.MethodPost, "/"+base+"/api/v1/session", bad)
	}
	res = e.do(http.MethodPost, "/"+base+"/api/v1/session", map[string]string{"username": username, "password": password})
	if res.status != http.StatusTooManyRequests || res.errorCode(t) != "locked" || res.header.Get("Retry-After") == "" {
		t.Fatalf("locked: %d %s", res.status, res.body)
	}
}

func TestSignInNeedsCSRFToken(t *testing.T) {
	e := newEnv(t)
	e.fetchCSRF()
	e.csrf = "forged"
	res := e.do(http.MethodPost, "/"+base+"/api/v1/session", map[string]string{"username": username, "password": password})
	if res.status != http.StatusForbidden || res.errorCode(t) != "csrf" {
		t.Fatalf("forged token: %d %s", res.status, res.body)
	}
}

func TestTwoStepSignIn(t *testing.T) {
	e := newEnv(t)
	e.login("")
	var setup struct {
		Secret string `json:"secret"`
		URI    string `json:"uri"`
	}
	res := e.do(http.MethodPost, "/"+base+"/api/v1/account/totp", nil)
	res.json(t, &setup)
	if res.status != http.StatusOK || setup.Secret == "" || !strings.HasPrefix(setup.URI, "otpauth://") || res.header.Get("Cache-Control") != "no-store" {
		t.Fatalf("begin: %d %s", res.status, res.body)
	}
	if res := e.do(http.MethodPost, "/"+base+"/api/v1/account/totp/confirm", map[string]string{"code": "000000"}); res.status != http.StatusForbidden || res.errorCode(t) != "totp_invalid" {
		t.Fatalf("wrong confirm: %d %s", res.status, res.body)
	}
	if res := e.do(http.MethodPost, "/"+base+"/api/v1/account/totp/confirm", map[string]string{"code": totpNow(setup.Secret)}); res.status != http.StatusNoContent {
		t.Fatalf("confirm: %d %s", res.status, res.body)
	}
	if res := e.do(http.MethodPost, "/"+base+"/api/v1/account/totp/confirm", map[string]string{"code": "123456"}); res.status != http.StatusConflict || res.errorCode(t) != "totp_already_on" {
		t.Fatalf("confirm while on: %d %s", res.status, res.body)
	}
	if res := e.do(http.MethodPost, "/"+base+"/api/v1/account/totp", nil); res.status != http.StatusConflict || res.errorCode(t) != "totp_already_on" {
		t.Fatalf("new setup while on: %d %s", res.status, res.body)
	}

	e.do(http.MethodDelete, "/"+base+"/api/v1/session", nil)
	if res := e.login(""); res.status != http.StatusUnauthorized || res.errorCode(t) != "totp_required" {
		t.Fatalf("sign-in without code: %d %s", res.status, res.body)
	}

	res = e.do(http.MethodDelete, "/"+base+"/api/v1/account/totp", map[string]string{"password": "wrong password!", "code": "000000"})
	if res.status != http.StatusUnauthorized {
		t.Fatalf("disable while signed out: %d", res.status)
	}
}

func TestDisableTwoStepWhenOff(t *testing.T) {
	e := newEnv(t)
	e.login("")
	res := e.do(http.MethodDelete, "/"+base+"/api/v1/account/totp", map[string]string{"password": password, "code": "000000"})
	if res.status != http.StatusConflict || res.errorCode(t) != "totp_not_on" {
		t.Fatalf("disable while off: %d %s", res.status, res.body)
	}
	if res := e.do(http.MethodDelete, "/"+base+"/api/v1/account/totp", map[string]string{"password": "x"}); res.status != http.StatusBadRequest {
		t.Fatalf("bad body: %d", res.status)
	}
}

func TestPagesAndAssets(t *testing.T) {
	e := newEnv(t)
	page := e.do(http.MethodGet, "/"+base+"/", nil)
	if !bytes.Contains(page.body, []byte("login")) || !bytes.Contains(page.body, []byte(`<base href="/`+base+`/" />`)) {
		t.Fatalf("signed-out page: %s", page.body)
	}
	if page.header.Get("Cache-Control") != "no-store" || page.header.Get("Content-Security-Policy") == "" {
		t.Fatalf("page headers: %v", page.header)
	}
	for _, f := range []string{"login-1.js", "shared-1.js", "logo3d-1.js", "login-1.css", "font-1.woff2"} {
		if res := e.do(http.MethodGet, "/"+base+"/assets/"+f, nil); res.status != http.StatusOK || !strings.HasPrefix(res.header.Get("Cache-Control"), "public") {
			t.Errorf("public asset %s: %d %q", f, res.status, res.header.Get("Cache-Control"))
		}
	}
	notFound := e.raw(http.MethodGet, "/nothing-here")
	for _, f := range []string{"app-1.js", "overview-1.js", "not-in-manifest-1.js", "missing.js"} {
		if got := e.raw(http.MethodGet, "/"+base+"/assets/"+f); !bytes.Equal(got, notFound) {
			t.Errorf("private asset %s before sign-in:\n%s", f, got)
		}
	}
	if res := e.do(http.MethodGet, "/"+base+"/favicon.svg", nil); res.status != http.StatusOK {
		t.Errorf("favicon: %d", res.status)
	}
	if got := e.raw(http.MethodGet, "/"+base+"/boot.js"); !bytes.Equal(got, notFound) {
		t.Errorf("missing static file: %s", got)
	}

	e.login("")
	if page := e.do(http.MethodGet, "/"+base+"/", nil); !bytes.Contains(page.body, []byte("app")) {
		t.Fatalf("signed-in page: %s", page.body)
	}
	if res := e.do(http.MethodGet, "/"+base+"/assets/app-1.js", nil); res.status != http.StatusOK || !strings.HasPrefix(res.header.Get("Cache-Control"), "private") {
		t.Fatalf("private asset after sign-in: %d %q", res.status, res.header.Get("Cache-Control"))
	}
}

func TestDeepLinksOpenTheApp(t *testing.T) {
	e := newEnv(t)
	e.login("")
	for _, deep := range []string{"/settings", "/settings/safe-mode", "/migrations/abc/live", "/migrations/abc/report", "/migrations/abc/report/print",
		"/batches", "/batches/3", "/rules", "/rules/new", "/rules/3", "/rules/redirects/3", "/domains", "/domains/new", "/domains/3/certificate", "/domains/certificates/3"} {
		page := e.do(http.MethodGet, "/"+base+deep, nil)
		if page.status != http.StatusOK || !bytes.Contains(page.body, []byte("app")) || page.header.Get("Cache-Control") != "no-store" {
			t.Fatalf("deep link %s: %d %v %s", deep, page.status, page.header, page.body)
		}
	}
}

func TestRemovedPagesAnswerPlainNotFound(t *testing.T) {
	e := newEnv(t)
	e.login("")
	for _, gone := range []string{"/reports", "/reports/abc", "/reports/abc/print", "/templates", "/links", "/links/new", "/links/3"} {
		page := e.do(http.MethodGet, "/"+base+gone, nil)
		if page.status != http.StatusNotFound || !strings.HasPrefix(page.header.Get("Content-Type"), "text/plain") {
			t.Fatalf("removed page %s: %d %q", gone, page.status, page.header.Get("Content-Type"))
		}
	}
}

func TestBuildFilesAreGzippedWhenAccepted(t *testing.T) {
	e := newEnv(t)
	get := func(path, accept string) (http.Header, []byte) {
		req, _ := http.NewRequest(http.MethodGet, e.srv.URL+"/"+base+path, nil)
		if accept != "" {
			req.Header.Set("Accept-Encoding", accept)
		}
		res, err := http.DefaultTransport.RoundTrip(req)
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		body, _ := io.ReadAll(res.Body)
		return res.Header, body
	}

	h, body := get("/assets/login-1.js", "gzip, br")
	if h.Get("Content-Encoding") != "gzip" || h.Get("Vary") != "Accept-Encoding" {
		t.Fatalf("headers %v", h)
	}
	zr, err := gzip.NewReader(bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	if plain, _ := io.ReadAll(zr); string(plain) != "login()" {
		t.Fatalf("body %q", plain)
	}
	for _, tc := range []struct{ path, accept, want string }{
		{"/assets/login-1.js", "", ""},
		{"/assets/login-1.js", "gzip;q=0", ""},
		{"/assets/font-1.woff2", "gzip", ""},
		{"/", "gzip", "gzip"},
		{"/api/v1/session", "gzip", ""},
	} {
		if h, _ := get(tc.path, tc.accept); h.Get("Content-Encoding") != tc.want {
			t.Errorf("%s with %q: Content-Encoding %q, want %q", tc.path, tc.accept, h.Get("Content-Encoding"), tc.want)
		}
	}
}

func TestMissesLookIdentical(t *testing.T) {
	e := newEnv(t)
	want := e.raw(http.MethodGet, "/")
	for _, m := range [][2]string{
		{http.MethodGet, "/.env"},
		{http.MethodPost, "/" + base + "/"},
		{http.MethodPut, "/" + base + "/api/v1/session"},
		{http.MethodGet, "/" + base + "/api/v1/users"},
		{http.MethodGet, "/" + base + "/overview"},
		// App pages sign in again without a session, but only under the
		// hidden path and only for GET and HEAD.
		{http.MethodGet, "/wrongpath/domains"},
		{http.MethodGet, "/domains"},
		{http.MethodGet, "/" + base + "/nope"},
		{http.MethodPost, "/" + base + "/domains"},
	} {
		if got := e.raw(m[0], m[1]); !bytes.Equal(got, want) {
			t.Errorf("%s %s:\n%s\nwant\n%s", m[0], m[1], got, want)
		}
	}
}
