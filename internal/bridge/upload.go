package bridge

import (
	"context"
	"errors"
	"io"
	"net/http"
	"regexp"
	"strings"
	"sync"

	"github.com/PdYrust/karvan/internal/panel/subtoken"
)

// ErrPlainHTTP means the site reads a plain-http old panel and the
// operator has not accepted that anyone on the path could answer for it.
var ErrPlainHTTP = errors.New("the old panel is plain http; turn on the plain http option for this site first")

// maxUpload caps a token list; a list of 100k old links is about 10 MB.
const maxUpload = 16 << 20

// uploadWorkers is how many tokens are checked against the old panel at
// once (D64).
const uploadWorkers = 4

// UploadResult counts what happened to an uploaded token list.
type UploadResult struct {
	Found    int  `json:"found"`    // distinct tokens in the upload
	Stored   int  `json:"stored"`   // verified and kept
	Known    int  `json:"known"`    // already verified before
	NoUser   int  `json:"noUser"`   // no moved user of this site matches
	Refused  int  `json:"refused"`  // the old or new panel said no
	Failed   int  `json:"failed"`   // the old panel did not answer; try again later
	TooLarge bool `json:"tooLarge"` // the upload was cut at maxUpload
}

// Upload reads old links or bare tokens (text or CSV, any separator),
// verifies each one against the old panel and stores the ones that
// belong to a moved user. The upload itself is not kept.
func (b *Bridge) Upload(ctx context.Context, siteID int64, r io.Reader) (UploadResult, error) {
	var res UploadResult
	u, err := b.prepareUpload(ctx, siteID, r)
	if err != nil {
		return res, err
	}
	res.Found, res.TooLarge = len(u.tokens), u.tooLarge
	b.checkTokens(ctx, u, func(o uploadOutcome) { count(&res, o) })
	return res, ctx.Err()
}

// upload is a token list ready to check.
type upload struct {
	site     Site
	dest     Destination
	tokens   []string
	tooLarge bool
}

func (b *Bridge) prepareUpload(ctx context.Context, siteID int64, r io.Reader) (upload, error) {
	s, ok := b.site(siteID)
	if !ok {
		return upload{}, ErrNotFound
	}
	if plainHTTPRefused(s) {
		return upload{}, ErrPlainHTTP
	}
	d, err := b.dest(ctx, s.PanelID)
	if err != nil {
		return upload{}, err
	}
	raw, err := io.ReadAll(io.LimitReader(r, maxUpload+1))
	if err != nil {
		return upload{}, err
	}
	u := upload{site: s, dest: d}
	if len(raw) > maxUpload {
		raw, u.tooLarge = raw[:maxUpload], true
	}
	u.tokens = extractTokens(string(raw), s.OldPath)
	return u, nil
}

// checkTokens checks u's tokens uploadWorkers at a time and reports each
// outcome to done, which is never called by two workers at once.
func (b *Bridge) checkTokens(ctx context.Context, u upload, done func(uploadOutcome)) {
	jobs := make(chan string)
	var mu sync.Mutex
	var wg sync.WaitGroup
	for range uploadWorkers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for t := range jobs {
				out := b.uploadOne(ctx, request{site: u.site, dest: u.dest, token: t, mac: mac(b.macKey, t)})
				mu.Lock()
				done(out)
				mu.Unlock()
			}
		}()
	}
	for _, t := range u.tokens {
		if ctx.Err() != nil {
			break
		}
		jobs <- t
	}
	close(jobs)
	wg.Wait()
}

type uploadOutcome int

const (
	outStored uploadOutcome = iota
	outKnown
	outNoUser
	outRefused
	outFailed
)

func count(res *UploadResult, o uploadOutcome) {
	switch o {
	case outStored:
		res.Stored++
	case outKnown:
		res.Known++
	case outNoUser:
		res.NoUser++
	case outRefused:
		res.Refused++
	default:
		res.Failed++
	}
}

// uploadOne runs the lazy check's steps for one uploaded token, without
// the lazy switch and limits: the operator asked for these checks.
func (b *Bridge) uploadOne(ctx context.Context, q request) uploadOutcome {
	if _, row, ok, err := b.st.token(ctx, q.mac, q.site, q.dest.Key()); err == nil && ok && row.inScope {
		return outKnown
	}
	p, ok := subtoken.Parse(q.token)
	if !ok {
		return outNoUser
	}
	cands, status := b.candidates(ctx, q, p)
	if status == 0 {
		_, status = b.verify(ctx, q, cands)
	}
	switch {
	case status == 0:
		return outStored
	case status == http.StatusNotFound && len(cands) == 0:
		return outNoUser // none in the rule's scope, or held (D22)
	case status == http.StatusNotFound:
		return outRefused
	}
	return outFailed
}

// tokenInText finds tokens after the old path in links, and bare tokens
// on their own.
var tokenInText = regexp.MustCompile(`[A-Za-z0-9_\-.=]{15,1000}`)

// extractTokens returns the distinct tokens in text, in order.
func extractTokens(text, oldPath string) []string {
	seen := map[string]bool{}
	var out []string
	for _, field := range strings.FieldsFunc(text, func(r rune) bool {
		return r == ',' || r == ';' || r == '"' || r == '\'' || r == '\t' || r == '\n' || r == '\r' || r == ' '
	}) {
		cand := field
		if i := strings.Index(field, "/"+oldPath+"/"); i >= 0 {
			cand, _, _ = strings.Cut(field[i+len(oldPath)+2:], "/")
		} else if strings.Contains(field, "/") {
			continue // a URL on another path
		}
		cand, _, _ = strings.Cut(cand, "?")
		if tokenInText.FindString(cand) == cand && tokenRe.MatchString(cand) && !seen[cand] {
			if _, ok := subtoken.Parse(cand); ok {
				seen[cand] = true
				out = append(out, cand)
			}
		}
	}
	return out
}
