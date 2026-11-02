package handlers

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

// DefaultReleasesURL lists the newest Karvan release on GitHub, pre-releases included.
const DefaultReleasesURL = "https://api.github.com/repos/PdYrust/karvan/releases?per_page=5"

const (
	releaseTimeout  = 3 * time.Second
	releaseCacheTTL = 15 * time.Minute
	releaseRetryTTL = 2 * time.Minute
)

// ReleaseChecker reads and caches the newest published Karvan release tag.
type ReleaseChecker struct {
	url    string
	client *http.Client

	mu     sync.Mutex
	latest string
	until  time.Time
}

// NewReleaseChecker builds a checker for the given GitHub releases URL.
func NewReleaseChecker(url string) *ReleaseChecker {
	if url == "" {
		url = DefaultReleasesURL
	}
	return &ReleaseChecker{
		url:    url,
		client: &http.Client{Timeout: releaseTimeout},
	}
}

// Latest returns the newest release version when it is newer than current, or "".
func (c *ReleaseChecker) Latest(ctx context.Context, current string) string {
	if c == nil {
		return ""
	}
	now := time.Now()
	c.mu.Lock()
	if now.Before(c.until) {
		tag := c.latest
		c.mu.Unlock()
		return newerVersion(current, tag)
	}
	c.mu.Unlock()

	tag, ok := c.fetch(ctx)
	c.mu.Lock()
	if ok {
		c.latest = tag
		c.until = time.Now().Add(releaseCacheTTL)
	} else {
		c.until = time.Now().Add(releaseRetryTTL)
	}
	cached := c.latest
	c.mu.Unlock()
	return newerVersion(current, cached)
}

type githubRelease struct {
	TagName string `json:"tag_name"`
	Draft   bool   `json:"draft"`
}

func (c *ReleaseChecker) fetch(ctx context.Context) (string, bool) {
	reqCtx, cancel := context.WithTimeout(ctx, releaseTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, c.url, nil)
	if err != nil {
		return "", false
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "karvan")
	res, err := c.client.Do(req)
	if err != nil {
		return "", false
	}
	defer func() { _ = res.Body.Close() }()
	if res.StatusCode != http.StatusOK {
		return "", false
	}
	var list []githubRelease
	if err := json.NewDecoder(io.LimitReader(res.Body, 64<<10)).Decode(&list); err != nil {
		return "", false
	}
	for _, r := range list {
		if !r.Draft && r.TagName != "" {
			return strings.TrimPrefix(strings.TrimSpace(r.TagName), "v"), true
		}
	}
	return "", true
}

func newerVersion(current, latest string) string {
	if !IsNewerRelease(current, latest) {
		return ""
	}
	clean := strings.TrimPrefix(strings.TrimSpace(latest), "v")
	if strings.HasPrefix(strings.TrimSpace(current), "v") {
		return "v" + clean
	}
	return clean
}

// IsNewerRelease reports whether latest is a newer SemVer release than current.
func IsNewerRelease(current, latest string) bool {
	majA, minA, patA, preA, okA := parseSemver(current)
	majB, minB, patB, preB, okB := parseSemver(latest)
	if !okA || !okB {
		return false
	}
	if majA != majB {
		return majB > majA
	}
	if minA != minB {
		return minB > minA
	}
	if patA != patB {
		return patB > patA
	}
	return comparePrerelease(preA, preB)
}

func parseSemver(v string) (major, minor, patch int, pre string, ok bool) {
	s := strings.TrimPrefix(strings.TrimSpace(v), "v")
	if i := strings.IndexByte(s, '+'); i >= 0 {
		s = s[:i]
	}
	s = strings.TrimSuffix(s, "-dirty")
	s = trimGitDescribe(s)
	core := s
	if i := strings.IndexByte(s, '-'); i >= 0 {
		core, pre = s[:i], s[i+1:]
	}
	parts := strings.Split(core, ".")
	if len(parts) != 3 {
		return 0, 0, 0, "", false
	}
	var err error
	if major, err = strconv.Atoi(parts[0]); err != nil || major < 0 {
		return 0, 0, 0, "", false
	}
	if minor, err = strconv.Atoi(parts[1]); err != nil || minor < 0 {
		return 0, 0, 0, "", false
	}
	if patch, err = strconv.Atoi(parts[2]); err != nil || patch < 0 {
		return 0, 0, 0, "", false
	}
	return major, minor, patch, pre, true
}

// trimGitDescribe strips a trailing "-<N>-g<hex>" added by git describe.
func trimGitDescribe(s string) string {
	g := strings.LastIndex(s, "-g")
	if g <= 0 || g+2 >= len(s) {
		return s
	}
	for i := g + 2; i < len(s); i++ {
		c := s[i]
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') {
			return s
		}
	}
	dash := strings.LastIndexByte(s[:g], '-')
	if dash <= 0 {
		return s
	}
	if n, err := strconv.Atoi(s[dash+1 : g]); err == nil && n >= 0 {
		return s[:dash]
	}
	return s
}

func comparePrerelease(preA, preB string) bool {
	if preA == preB || preA == "" {
		return false
	}
	if preB == "" {
		return true
	}
	a := strings.FieldsFunc(preA, func(r rune) bool { return r == '.' || r == '-' })
	b := strings.FieldsFunc(preB, func(r rune) bool { return r == '.' || r == '-' })
	for i := 0; i < len(a) && i < len(b); i++ {
		if a[i] == b[i] {
			continue
		}
		na, errA := strconv.Atoi(a[i])
		nb, errB := strconv.Atoi(b[i])
		if errA == nil && errB == nil {
			return nb > na
		}
		if errA == nil {
			return true
		}
		if errB == nil {
			return false
		}
		return b[i] > a[i]
	}
	return len(b) > len(a)
}
