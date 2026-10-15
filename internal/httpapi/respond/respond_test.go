package respond

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestErrorShape(t *testing.T) {
	rec := httptest.NewRecorder()
	Error(rec, APIError{Status: http.StatusTooManyRequests, Code: "locked", Extra: map[string]any{"retryAfter": 120}})
	if rec.Code != 429 || rec.Header().Get("Content-Type") != "application/json; charset=utf-8" {
		t.Fatalf("status %d, type %q", rec.Code, rec.Header().Get("Content-Type"))
	}
	var body struct {
		Error struct {
			Code       string `json:"code"`
			RetryAfter int    `json:"retryAfter"`
		} `json:"error"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil || body.Error.Code != "locked" || body.Error.RetryAfter != 120 {
		t.Fatalf("body %s, %v", rec.Body, err)
	}
}

func TestInternalHidesTheCause(t *testing.T) {
	rec := httptest.NewRecorder()
	Internal(rec, httptest.NewRequest(http.MethodGet, "/", nil), errors.New("disk on fire at /var/lib/karvan"))
	if rec.Code != 500 || strings.Contains(rec.Body.String(), "disk") {
		t.Fatalf("status %d body %s", rec.Code, rec.Body)
	}
}

func TestNoContentAndBadRequest(t *testing.T) {
	rec := httptest.NewRecorder()
	NoContent(rec)
	if rec.Code != 204 || rec.Body.Len() != 0 {
		t.Fatalf("NoContent: %d %q", rec.Code, rec.Body)
	}
	rec = httptest.NewRecorder()
	BadRequest(rec)
	if rec.Code != 400 || !strings.Contains(rec.Body.String(), CodeBadRequest) {
		t.Fatalf("BadRequest: %d %s", rec.Code, rec.Body)
	}
}

func TestDecode(t *testing.T) {
	type in struct {
		Name string `json:"name"`
	}
	for name, tc := range map[string]struct {
		ctype, body string
		ok          bool
	}{
		"valid":         {"application/json", `{"name":"a"}`, true},
		"charset":       {"application/json; charset=utf-8", `{"name":"a"}`, true},
		"wrong type":    {"text/plain", `{"name":"a"}`, false},
		"unknown field": {"application/json", `{"nam":"a"}`, false},
		"trailing":      {"application/json", `{"name":"a"}{}`, false},
		"too big":       {"application/json", `{"name":"` + strings.Repeat("a", MaxBodyBytes) + `"}`, false},
		"not json":      {"application/json", `name=a`, false},
	} {
		t.Run(name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(tc.body))
			r.Header.Set("Content-Type", tc.ctype)
			var v in
			err := Decode(httptest.NewRecorder(), r, &v)
			if (err == nil) != tc.ok {
				t.Fatalf("err = %v, want ok=%v", err, tc.ok)
			}
		})
	}
}
