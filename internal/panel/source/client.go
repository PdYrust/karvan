package source

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"strings"
	"sync/atomic"
	"time"
)

// Defaults for Options.
const (
	DefaultTimeout  = 30 * time.Second
	DefaultMaxBody  = 64 << 20
	DefaultAttempts = 6
)

// Options configure a Client.
type Options struct {
	BaseURL   string // the panel's address as the operator typed it
	PinSHA256 string // hex SHA-256 of the panel's certificate, to trust a self-signed one
	Resolver  Resolver
	Timeout   time.Duration
	MaxBody   int64
	Attempts  int
	// Sleep waits between retries; tests replace it.
	Sleep func(ctx context.Context, d time.Duration) error
}

// Client talks to one old panel.
type Client struct {
	base     *url.URL
	dialer   *guardedDialer
	http     *http.Client
	maxBody  int64
	attempts int
	sleep    func(ctx context.Context, d time.Duration) error
	skew     atomic.Int64 // nanoseconds, panel clock minus ours
}

// Response is a successful (2xx) answer.
type Response struct {
	Status int
	Header http.Header
	Body   []byte
}

// ParseBase validates a panel address: http or https, a host, no user
// info, no query. The path is kept (panels can sit under a prefix).
func ParseBase(raw string) (*url.URL, error) {
	u, err := url.Parse(strings.TrimSpace(raw))
	switch {
	case err != nil:
		return nil, fmt.Errorf("panel address: %w", err)
	case u.Scheme != "http" && u.Scheme != "https":
		return nil, errors.New("panel address must start with http:// or https://")
	case u.Host == "" || u.Hostname() == "":
		return nil, errors.New("panel address has no host")
	case u.User != nil:
		return nil, errors.New("panel address must not carry a username or password")
	case u.RawQuery != "" || u.Fragment != "":
		return nil, errors.New("panel address must not carry a query")
	}
	u.Path = strings.TrimRight(u.Path, "/")
	return u, nil
}

// New builds a client for o.BaseURL.
func New(o Options) (*Client, error) {
	base, err := ParseBase(o.BaseURL)
	if err != nil {
		return nil, err
	}
	tlsConf, err := tlsConfig(o.PinSHA256)
	if err != nil {
		return nil, err
	}
	dialer := newGuardedDialer(o.Resolver)
	tr := &http.Transport{
		DialContext:           dialer.DialContext,
		TLSClientConfig:       tlsConf,
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: orDefault(o.Timeout, DefaultTimeout),
		MaxIdleConnsPerHost:   4,
		ForceAttemptHTTP2:     true,
		Proxy:                 nil, // never through an environment proxy
	}
	c := &Client{
		base:   base,
		dialer: dialer,
		http: &http.Client{
			Transport:     tr,
			Timeout:       orDefault(o.Timeout, DefaultTimeout),
			CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
		},
		maxBody:  o.MaxBody,
		attempts: o.Attempts,
		sleep:    o.Sleep,
	}
	if c.maxBody <= 0 {
		c.maxBody = DefaultMaxBody
	}
	if c.attempts <= 0 {
		c.attempts = DefaultAttempts
	}
	if c.sleep == nil {
		c.sleep = sleepCtx
	}
	return c, nil
}

func orDefault(d, def time.Duration) time.Duration {
	if d <= 0 {
		return def
	}
	return d
}

// Base is the panel address.
func (c *Client) Base() *url.URL { u := *c.base; return &u }

// Addr is the checked address of the client's last connection to the
// panel; not valid before the first one.
func (c *Client) Addr() netip.Addr {
	if a := c.dialer.last.Load(); a != nil {
		return *a
	}
	return netip.Addr{}
}

// Skew is the panel's clock minus Karvan's, from the last Date header.
func (c *Client) Skew() time.Duration { return time.Duration(c.skew.Load()) }

// Get fetches path (relative to the base) with query.
func (c *Client) Get(ctx context.Context, path string, query url.Values, h http.Header) (*Response, error) {
	return c.do(ctx, http.MethodGet, path, query, nil, h)
}

// PostForm posts a form to path. Login is the only POST Karvan sends; it
// has no side effect on the panel, so it is safe to retry.
func (c *Client) PostForm(ctx context.Context, path string, form url.Values, h http.Header) (*Response, error) {
	if h == nil {
		h = http.Header{}
	}
	h.Set("Content-Type", "application/x-www-form-urlencoded")
	return c.do(ctx, http.MethodPost, path, nil, []byte(form.Encode()), h)
}

func (c *Client) do(ctx context.Context, method, path string, query url.Values, body []byte, h http.Header) (*Response, error) {
	u := c.Base()
	u.Path += path
	u.RawQuery = query.Encode()
	var last error
	for attempt := 0; attempt < c.attempts; attempt++ {
		resp, wait, err := c.once(ctx, method, u.String(), body, h)
		if err == nil {
			return resp, nil
		}
		last = err
		if wait < 0 || attempt == c.attempts-1 {
			break
		}
		if err := c.sleep(ctx, backoff(attempt, wait)); err != nil {
			return nil, &Error{Code: CodeTimeout, Err: err}
		}
	}
	return nil, last
}

// once sends one request. wait < 0 means do not retry; wait > 0 is the
// panel's Retry-After.
func (c *Client) once(ctx context.Context, method, u string, body []byte, h http.Header) (*Response, time.Duration, error) {
	req, err := http.NewRequestWithContext(ctx, method, u, bytesReader(body))
	if err != nil {
		return nil, -1, err
	}
	req.Header = h.Clone()
	if req.Header == nil {
		req.Header = http.Header{}
	}
	req.Header.Set("Accept", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, retryable(err), transportError(err)
	}
	defer resp.Body.Close()
	c.noteClock(resp.Header.Get("Date"))
	b, err := io.ReadAll(io.LimitReader(resp.Body, c.maxBody+1))
	if err != nil {
		return nil, 0, &Error{Code: CodeUnreachable, Status: resp.StatusCode, Err: err}
	}
	if int64(len(b)) > c.maxBody {
		return nil, -1, &Error{Code: CodeTooLarge, Status: resp.StatusCode}
	}
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return &Response{Status: resp.StatusCode, Header: resp.Header, Body: b}, 0, nil
	}
	return nil, statusWait(resp), &Error{Code: statusCode(resp.StatusCode), Status: resp.StatusCode}
}

func bytesReader(b []byte) io.Reader {
	if b == nil {
		return nil
	}
	return strings.NewReader(string(b))
}

func (c *Client) noteClock(date string) {
	if t, err := http.ParseTime(date); err == nil {
		c.skew.Store(int64(time.Until(t)))
	}
}

func statusCode(s int) string {
	switch {
	case s >= 300 && s < 400:
		return CodeRedirect
	case s == http.StatusUnauthorized:
		return CodeUnauthorized
	case s == http.StatusForbidden:
		return CodeForbidden
	case s == http.StatusNotFound:
		return CodeNotFound
	case s == http.StatusTooManyRequests:
		return CodeRateLimited
	case s >= 500:
		return CodeServerError
	}
	return CodeBadResponse
}

func statusWait(resp *http.Response) time.Duration {
	if resp.StatusCode != http.StatusTooManyRequests && resp.StatusCode < 500 {
		return -1
	}
	return retryAfter(resp.Header.Get("Retry-After"))
}

func transportError(err error) error {
	var se *Error
	if errors.As(err, &se) {
		return se
	}
	var ne net.Error
	var ce *tls.CertificateVerificationError
	switch {
	case errors.As(err, &ce):
		e := &Error{Code: CodeTLSUntrusted, Err: err}
		if len(ce.UnverifiedCertificates) > 0 {
			cert := certificateOf(ce.UnverifiedCertificates[0])
			e.Cert = &cert
		}
		return e
	case errors.Is(err, errPinMismatch):
		return &Error{Code: CodeTLS, Err: err}
	case strings.Contains(err.Error(), "tls:"):
		return &Error{Code: CodeTLS, Err: err}
	case errors.As(err, &ne) && ne.Timeout():
		return &Error{Code: CodeTimeout, Err: err}
	}
	return &Error{Code: CodeUnreachable, Err: err}
}

func retryable(err error) time.Duration {
	switch CodeOf(transportError(err)) {
	case CodeTimeout, CodeUnreachable:
		return 0
	}
	return -1
}
