package handlers

import (
	"bytes"
	"compress/gzip"
	"net/http"
	"path"
	"strings"
	"sync"
)

// Only static, public build files are compressed. API responses are not,
// so a response that carries a token never gets compressed next to
// attacker-chosen input (the BREACH attack).
var compressible = map[string]bool{".js": true, ".css": true, ".html": true, ".svg": true, ".json": true}

// gzipCache holds the compressed form of each embedded file, made on first use.
type gzipCache struct{ m sync.Map }

func (c *gzipCache) get(name string, data []byte) []byte {
	if v, ok := c.m.Load(name); ok {
		return v.([]byte)
	}
	var buf bytes.Buffer
	zw, _ := gzip.NewWriterLevel(&buf, gzip.BestCompression)
	_, _ = zw.Write(data)
	_ = zw.Close()
	v, _ := c.m.LoadOrStore(name, buf.Bytes())
	return v.([]byte)
}

func acceptsGzip(r *http.Request) bool {
	for _, part := range strings.Split(r.Header.Get("Accept-Encoding"), ",") {
		enc, q, _ := strings.Cut(strings.TrimSpace(part), ";")
		if strings.EqualFold(enc, "gzip") && strings.TrimSpace(q) != "q=0" {
			return true
		}
	}
	return false
}

// write sends data for the build file name, gzipped when that helps and the
// client accepts it.
func (c *gzipCache) write(w http.ResponseWriter, r *http.Request, name string, data []byte) {
	if compressible[path.Ext(name)] {
		w.Header().Add("Vary", "Accept-Encoding")
		if acceptsGzip(r) {
			w.Header().Set("Content-Encoding", "gzip")
			data = c.get(name, data)
		}
	}
	_, _ = w.Write(data) //nolint:gosec // G705: bytes come from the embedded build, not from the request.
}
