package middleware

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAccessLogOmitsHiddenPathAndQuery(t *testing.T) {
	var logs bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logs, &slog.HandlerOptions{Level: slog.LevelDebug}))
	rt := NewRouter()
	rt.HandleFunc("GET /api/v1/session", ok)
	h := BasePath(base, AccessLog(logger, rt))
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/"+base+"/api/v1/session?x=secret", nil))
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/"+base+"/missing", nil))
	out := logs.String()
	if strings.Contains(out, base) || strings.Contains(out, "secret") {
		t.Fatalf("log leaks the path or query: %s", out)
	}
	if !strings.Contains(out, "path=/api/v1/session") || !strings.Contains(out, "status=404") {
		t.Fatalf("log missing fields: %s", out)
	}
}
