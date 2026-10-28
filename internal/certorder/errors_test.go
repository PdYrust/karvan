package certorder

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"golang.org/x/crypto/acme"

	"github.com/PdYrust/karvan/internal/bridge"
)

func acmeErr(typ, detail string) *acme.Error {
	return &acme.Error{StatusCode: 403, ProblemType: "urn:ietf:params:acme:error:" + typ, Detail: detail}
}

// D71: every CA problem, network failure and port failure in words.
func TestClassify(t *testing.T) {
	now := time.Date(2026, 10, 28, 12, 0, 0, 0, time.UTC)
	limited := acmeErr("rateLimited", "too many certificates")
	limited.Header = http.Header{"Retry-After": {"3600"}}
	for _, c := range []struct {
		err  error
		code string
	}{
		{limited, CodeRateLimited},
		{acmeErr("unauthorized", ""), CodeChallengeFailed},
		{acmeErr("incorrectResponse", ""), CodeChallengeFailed},
		{acmeErr("connection", ""), CodeChallengeFailed},
		{acmeErr("dns", ""), CodeChallengeFailed},
		{acmeErr("tls", ""), CodeChallengeFailed},
		{acmeErr("caa", ""), CodeCAA},
		{acmeErr("rejectedIdentifier", ""), CodeRejected},
		{acmeErr("unsupportedIdentifier", ""), CodeRejected},
		{acmeErr("accountDoesNotExist", ""), CodeAccount},
		{acmeErr("serverInternal", ""), CodeFailed},
		{&acme.AuthorizationError{Errors: []error{acmeErr("caa", "")}}, CodeCAA},
		{&acme.AuthorizationError{}, CodeChallengeFailed},
		{&acme.OrderError{Status: acme.StatusExpired}, CodeExpired},
		{&acme.OrderError{Status: acme.StatusInvalid, Problem: acmeErr("rejectedIdentifier", "")}, CodeRejected},
		{&acme.OrderError{Status: acme.StatusInvalid}, CodeFailed},
		{errExpired, CodeExpired},
		{errAccount{errors.New("open")}, CodeAccount},
		{fmt.Errorf("x: %w", context.DeadlineExceeded), CodeUnreachable},
		{&net.OpError{Op: "dial", Err: errors.New("refused")}, CodeUnreachable},
		{fmt.Errorf("%w: port 80", bridge.ErrPortDenied), CodePortDenied},
		{fmt.Errorf("%w: port 80", bridge.ErrPortBusy), CodePortBusy},
		{bridge.ErrPortTLS, CodePortTLS},
		{errors.New("other"), CodeFailed},
	} {
		if e, _ := classify(c.err, now); e.Code != c.code {
			t.Errorf("%v: %s, want %s", c.err, e.Code, c.code)
		}
	}
	e, problem := classify(limited, now)
	if problem != "rateLimited" || e.RetryAt == nil || !e.RetryAt.Equal(now.Add(time.Hour)) || e.Detail != "too many certificates" {
		t.Fatalf("rate limit: %+v %s", e, problem)
	}
}

// The CA's detail is untrusted: control characters go, and it is cut to
// 300 bytes on a character boundary.
func TestCleanDetail(t *testing.T) {
	got := cleanDetail("bad\x00\x1b[31m\nline " + strings.Repeat("ا", 200) + "\xff")
	if strings.ContainsAny(got, "\x00\x1b\n") || len(got) > maxDetail || !utf8.ValidString(got) || !strings.HasPrefix(got, "bad[31mline ") {
		t.Fatalf("%q (%d bytes)", got, len(got))
	}
}

// Problems from the CA end the order with their code, detail and, for a
// rate limit, its Retry-After as retryAt.
func TestCAProblemsFailTheOrder(t *testing.T) {
	for _, c := range []struct {
		endpoint string
		p        problem
		code     string
	}{
		{"order", problem{status: 429, typ: "rateLimited", detail: "too many new orders", retryAfter: "120"}, CodeRateLimited},
		{"order", problem{status: 400, typ: "rejectedIdentifier", detail: "name not allowed"}, CodeRejected},
		{"account", problem{status: 400, typ: "invalidContact", detail: "bad contact"}, CodeAccount},
		{"finalize", problem{status: 403, typ: "caa", detail: "CAA forbids"}, CodeCAA},
		{"chal", problem{status: 500, typ: "serverInternal", detail: strings.Repeat("x", 400)}, CodeFailed},
	} {
		t.Run(c.endpoint+"/"+c.p.typ, func(t *testing.T) {
			e := newOrderEnv(t)
			e.ca.set(c.endpoint, c.p)
			d := e.addACME("e.example.com", HTTP01)
			o := e.waitState(d.ID, "failed")
			if o.Error == nil || o.Error.Code != c.code || len(o.Error.Detail) > maxDetail || o.Error.Detail == "" {
				t.Fatalf("%+v", o.Error)
			}
			if c.p.retryAfter != "" && (o.Error.RetryAt == nil || !o.Error.RetryAt.Equal(e.clock.now().Add(2*time.Minute))) {
				t.Fatalf("retry at %v", o.Error.RetryAt)
			}
			if c.p.status < 500 && e.ca.count(c.endpoint) != 1 { // a server error is tried again
				t.Errorf("%s asked %d times", c.endpoint, e.ca.count(c.endpoint))
			}
			if !strings.Contains(e.logs.String(), "code="+c.code) {
				t.Error("no Warn line with the code")
			}
		})
	}
}

// A CA that cannot be reached is acme_unreachable.
func TestUnreachableCA(t *testing.T) {
	e := newOrderEnv(t)
	e.svc.TestDirectory = "http://127.0.0.1:" + fmt.Sprint(freePort(t)) + "/dir"
	d := e.addACME("u.example.com", DNS01)
	if o := e.waitState(d.ID, "failed"); o.Error == nil || o.Error.Code != CodeUnreachable {
		t.Fatalf("%+v", o.Error)
	}
}
