// Package reload makes a running destination panel pick up the users
// Karvan wrote into its database (D46), through the panel's own API on
// this server: a no-op edit per user ("touch") or one core restart.
package reload

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/PdYrust/karvan/internal/migration/dest"
)

// callTimeout bounds one API call. A core restart on a busy panel is the
// slowest call.
const callTimeout = 60 * time.Second

// client calls one panel as its full admin.
type client struct {
	api  dest.API
	http *http.Client
	now  func() time.Time
}

func newClient(api dest.API) (*client, error) {
	t := &http.Transport{MaxIdleConnsPerHost: workers, IdleConnTimeout: 30 * time.Second}
	if api.Socket != "" {
		var d net.Dialer
		t.DialContext = func(ctx context.Context, _, _ string) (net.Conn, error) {
			return d.DialContext(ctx, "unix", api.Socket)
		}
	}
	if strings.HasPrefix(api.BaseURL, "https://") {
		pin, err := leaf(api.CertFile)
		if err != nil {
			return nil, err
		}
		t.TLSClientConfig = &tls.Config{
			MinVersion: tls.VersionTLS12,
			// The panel's certificate is for its public name, not for
			// 127.0.0.1, so Karvan pins the exact certificate the panel
			// serves from its own ENV instead of checking the name.
			InsecureSkipVerify: true, //nolint:gosec // G402: replaced by the pin in VerifyConnection
			VerifyConnection: func(cs tls.ConnectionState) error {
				if len(cs.PeerCertificates) == 0 || !bytes.Equal(cs.PeerCertificates[0].Raw, pin) {
					return errors.New("the panel serves a different certificate than its ENV names")
				}
				return nil
			},
		}
	}
	return &client{api: api, http: &http.Client{Transport: t, Timeout: callTimeout}, now: time.Now}, nil
}

// leaf reads the first certificate in a PEM file.
func leaf(path string) ([]byte, error) {
	b, err := os.ReadFile(path) //nolint:gosec // G304: the path comes from the panel's ENV
	if err != nil {
		return nil, fmt.Errorf("read the panel's certificate: %w", err)
	}
	for {
		var blk *pem.Block
		blk, b = pem.Decode(b)
		if blk == nil {
			return nil, errors.New("the panel's certificate file has no certificate")
		}
		if blk.Type == "CERTIFICATE" {
			return blk.Bytes, nil
		}
	}
}

// do sends body as JSON and decodes a 2xx answer into out (when not nil).
func (c *client) do(ctx context.Context, method, path string, body, out any) error {
	var rd io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return err
		}
		rd = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.api.BaseURL+path, rd)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.api.Token(c.now()))
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return err
	}
	if resp.StatusCode/100 != 2 {
		return fmt.Errorf("%s %s: %d %s", method, path, resp.StatusCode, detail(raw))
	}
	if out == nil {
		return nil
	}
	return json.Unmarshal(raw, out)
}

// detail is the panel's error text, short.
func detail(raw []byte) string {
	var e struct {
		Detail any `json:"detail"`
	}
	s := strings.TrimSpace(string(raw))
	if json.Unmarshal(raw, &e) == nil && e.Detail != nil {
		if b, err := json.Marshal(e.Detail); err == nil {
			s = strings.Trim(string(b), `"`)
		}
	}
	if len(s) > 200 {
		s = s[:200]
	}
	return s
}
