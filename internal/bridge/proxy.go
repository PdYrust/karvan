package bridge

import (
	"io"
	"net"
	"net/http"
	"net/textproto"
	"strings"
)

// hopHeaders apply to one connection only and are never passed on
// (RFC 9110 section 7.6.1).
var hopHeaders = []string{"Connection", "Proxy-Connection", "Keep-Alive", "Proxy-Authenticate",
	"Proxy-Authorization", "Te", "Trailer", "Transfer-Encoding", "Upgrade"}

// copyHeaders copies every end-to-end header.
func copyHeaders(dst, src http.Header) {
	drop := map[string]bool{}
	for _, h := range hopHeaders {
		drop[h] = true
	}
	for _, v := range src.Values("Connection") {
		for _, f := range strings.Split(v, ",") {
			drop[textproto.CanonicalMIMEHeaderKey(strings.TrimSpace(f))] = true
		}
	}
	for k, vs := range src {
		if !drop[k] {
			dst[k] = append(dst[k], vs...)
		}
	}
}

// forward sends the client's request to the panel's own subscription
// route with token: same method, query and headers (User-Agent, Host and
// HWID headers decide what the panel answers), plus X-Forwarded-For.
func forward(r *http.Request, d Destination, token, rest string) (*http.Response, error) {
	u := d.BaseURL() + "/" + d.SubPath() + "/" + token + rest
	if r.URL.RawQuery != "" {
		u += "?" + r.URL.RawQuery
	}
	// The base is the panel's own listener from its ENV; token and rest
	// passed tokenRe and restRe in route.
	out, err := http.NewRequestWithContext(r.Context(), r.Method, u, nil) //nolint:gosec // G704: see above

	if err != nil {
		return nil, err
	}
	copyHeaders(out.Header, r.Header)
	out.Host = r.Host
	if ip, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		if prior := out.Header.Get("X-Forwarded-For"); prior != "" {
			ip = prior + ", " + ip
		}
		out.Header.Set("X-Forwarded-For", ip)
	}
	return d.Client().Do(out) //nolint:gosec // G704: the panel's own listener
}

// relay writes the panel's answer to the client unchanged, minus
// hop-by-hop headers.
func relay(w http.ResponseWriter, resp *http.Response) {
	defer resp.Body.Close()
	copyHeaders(w.Header(), resp.Header)
	w.WriteHeader(resp.StatusCode)
	_, _ = io.Copy(w, resp.Body)
}

// answer writes the panels' own error shape.
func answer(w http.ResponseWriter, status int) {
	w.Header().Set("Content-Type", "application/json")
	if status == http.StatusServiceUnavailable {
		w.Header().Set("Retry-After", retryAfter)
	}
	w.WriteHeader(status)
	_, _ = io.WriteString(w, `{"detail":"`+http.StatusText(status)+`"}`)
}
