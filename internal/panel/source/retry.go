package source

import (
	"context"
	"math/rand/v2"
	"net/http"
	"strconv"
	"time"
)

const (
	backoffBase = 500 * time.Millisecond
	backoffCap  = 30 * time.Second
	maxWait     = 2 * time.Minute
)

// backoff is full jitter over an exponential window, or the panel's own
// Retry-After when it sent one (capped, so a hostile panel cannot park a
// job for hours).
func backoff(attempt int, retryAfter time.Duration) time.Duration {
	if retryAfter > 0 {
		return min(retryAfter, maxWait)
	}
	window := min(backoffBase<<attempt, backoffCap)
	return time.Duration(rand.Int64N(int64(window)) + 1) //nolint:gosec // jitter, not a secret
}

// retryAfter reads a Retry-After header: seconds or an HTTP date.
func retryAfter(v string) time.Duration {
	if v == "" {
		return 0
	}
	if s, err := strconv.Atoi(v); err == nil && s > 0 {
		return time.Duration(s) * time.Second
	}
	if t, err := http.ParseTime(v); err == nil {
		if d := time.Until(t); d > 0 {
			return d
		}
	}
	return 0
}

func sleepCtx(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}
