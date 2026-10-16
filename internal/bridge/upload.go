package bridge

import (
	"context"
	"errors"
	"io"
	"net/http"
	"regexp"
	"strings"
	"sync"

	"github.com/PdYrust/karvan/internal/migration/ledger"
	"github.com/PdYrust/karvan/internal/panel/subtoken"
)

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
	s, ok := b.site(siteID)
	if !ok {
		return res, ErrNotFound
	}
	d, err := b.dest(ctx, s.PanelID)
	if err != nil {
		return res, err
	}
	raw, err := io.ReadAll(io.LimitReader(r, maxUpload+1))
	if err != nil {
		return res, err
	}
	if len(raw) > maxUpload {
		raw, res.TooLarge = raw[:maxUpload], true
	}
	tokens := extractTokens(string(raw), s.OldPath)
	res.Found = len(tokens)
	jobs := make(chan string)
	var mu sync.Mutex
	var wg sync.WaitGroup
	for range uploadWorkers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for t := range jobs {
				out := b.uploadOne(ctx, request{site: s, dest: d, token: t, mac: mac(b.macKey, t)})
				mu.Lock()
				count(&res, out)
				mu.Unlock()
			}
		}()
	}
	for _, t := range tokens {
		if ctx.Err() != nil {
			break
		}
		jobs <- t
	}
	close(jobs)
	wg.Wait()
	return res, ctx.Err()
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
	if v, ok, err := b.st.token(ctx, q.mac); err == nil && ok && v.SiteID == q.site.ID {
		return outKnown
	}
	p, ok := subtoken.Parse(q.token)
	if !ok {
		return outNoUser
	}
	e, err := b.claimed(ctx, q, p)
	switch {
	case errors.Is(err, ledger.ErrNotFound):
		return outNoUser
	case err != nil:
		return outFailed
	}
	verd, err := b.check(ctx, q.dest, e, p.Issued)
	if err != nil {
		return outFailed
	}
	b.apply(ctx, e, verd)
	if !verd.ok {
		return outRefused
	}
	switch b.verify(ctx, q, e) {
	case 0:
		return outStored
	case http.StatusNotFound:
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
