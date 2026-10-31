package certorder

import (
	"context"
	"errors"
	"net"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"golang.org/x/crypto/acme"

	"github.com/PdYrust/karvan/internal/bridge"
)

// Order error codes. Each has words in the web panel.
const (
	CodeRateLimited     = "acme_rate_limited"
	CodeChallengeFailed = "acme_challenge_failed"
	CodeCAA             = "acme_caa"
	CodeRejected        = "acme_rejected"
	CodeUnreachable     = "acme_unreachable"
	CodeExpired         = "acme_expired"
	CodeAccount         = "acme_account"
	CodeFailed          = "acme_failed"
	CodePortDenied      = "acme_port_denied"
	CodePortBusy        = "acme_port_busy"
	CodePortTLS         = "acme_port_tls"
)

// maxDetail is the most of the CA's detail that is kept.
const maxDetail = 300

// errExpired means the CA expired the order or authorization, or no
// longer knows it.
var errExpired = errors.New("the order expired")

// problemCodes maps the CA's problem types to codes.
var problemCodes = map[string]string{
	"rateLimited":             CodeRateLimited,
	"unauthorized":            CodeChallengeFailed,
	"incorrectResponse":       CodeChallengeFailed,
	"connection":              CodeChallengeFailed,
	"dns":                     CodeChallengeFailed,
	"tls":                     CodeChallengeFailed,
	"caa":                     CodeCAA,
	"rejectedIdentifier":      CodeRejected,
	"unsupportedIdentifier":   CodeRejected,
	"accountDoesNotExist":     CodeAccount,
	"badPublicKey":            CodeAccount,
	"invalidContact":          CodeAccount,
	"unsupportedContact":      CodeAccount,
	"userActionRequired":      CodeAccount,
	"externalAccountRequired": CodeAccount,
}

// classify is err in words: its code, the CA's cleaned detail and, for a
// rate limit, when to try again; problem is the CA's problem type, for
// the log.
func classify(err error, now time.Time) (e bridge.ACMEError, problem string) {
	ae, code := caProblem(err)
	if ae == nil {
		return bridge.ACMEError{Code: code}, ""
	}
	problem = strings.TrimPrefix(ae.ProblemType, "urn:ietf:params:acme:error:")
	e = bridge.ACMEError{Code: CodeFailed, Detail: cleanDetail(ae.Detail)}
	if c, ok := problemCodes[problem]; ok {
		e.Code = c
	}
	if d, ok := acme.RateLimit(ae); ok && d > 0 {
		at := now.Add(d).UTC().Truncate(time.Second)
		e.RetryAt = &at
	}
	return e, problem
}

// caProblem is the CA's problem in err, or else the code of err.
func caProblem(err error) (*acme.Error, string) {
	var authz *acme.AuthorizationError
	var order *acme.OrderError
	var ae *acme.Error
	switch {
	case errors.Is(err, bridge.ErrPortDenied):
		return nil, CodePortDenied
	case errors.Is(err, bridge.ErrPortBusy):
		return nil, CodePortBusy
	case errors.Is(err, bridge.ErrPortTLS):
		return nil, CodePortTLS
	case errors.Is(err, errExpired):
		return nil, CodeExpired
	case errors.As(err, &errAccount{}):
		return nil, CodeAccount
	case errors.As(err, &authz):
		for _, sub := range authz.Errors {
			if errors.As(sub, &ae) {
				return ae, ""
			}
		}
		return nil, CodeChallengeFailed
	case errors.As(err, &order):
		return orderProblem(order)
	case errors.As(err, &ae):
		return ae, ""
	case errors.Is(err, context.DeadlineExceeded), isNetError(err):
		return nil, CodeUnreachable
	}
	return nil, CodeFailed
}

func orderProblem(o *acme.OrderError) (*acme.Error, string) {
	switch {
	case o.Status == acme.StatusExpired:
		return nil, CodeExpired
	case o.Problem == nil:
		return nil, CodeFailed
	}
	return o.Problem, ""
}

func isNetError(err error) bool {
	var ne net.Error
	return errors.As(err, &ne)
}

// cleanDetail is the CA's detail without control characters, cut to
// maxDetail bytes on a character boundary: it is untrusted text.
func cleanDetail(s string) string {
	s = strings.ToValidUTF8(s, "")
	s = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, s)
	for len(s) > maxDetail {
		_, n := utf8.DecodeLastRuneInString(s)
		s = s[:len(s)-n]
	}
	return s
}
