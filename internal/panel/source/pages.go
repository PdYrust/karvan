package source

import (
	"context"
	"errors"
	"fmt"
	"maps"
)

// ErrIncomplete means the panel's list kept changing (or kept lying about
// its total) across every pass, so Karvan refuses to move a partial set.
var ErrIncomplete = errors.New("the user list changed during every read")

// Page fetches items [offset, offset+limit) and the panel's total.
type Page[T any] func(ctx context.Context, offset, limit int) (items []T, total int, err error)

// maxPasses bounds ReadAll: two equal passes in a row are needed.
const maxPasses = 3

// Progress is how far a read of a list got: Read items of the panel's
// Total in Pass, counted from 1. A list is read until two passes agree,
// so a read takes two passes or three.
type Progress struct {
	Pass, Read, Total int
}

type progressKey struct{}

// WithProgress returns a context in which ReadAll calls fn after each
// page, the way net/http/httptrace carries its hooks.
func WithProgress(ctx context.Context, fn func(Progress)) context.Context {
	return context.WithValue(ctx, progressKey{}, fn)
}

func report(ctx context.Context, p Progress) {
	if fn, ok := ctx.Value(progressKey{}).(func(Progress)); ok {
		fn(p)
	}
}

// ReadAll reads every item through fetch, sorted by a unique key on the
// panel side, until two passes in a row return the same set of keys.
// Count checks alone miss a delete plus a create during paging. The last
// pass is returned, so values are the freshest.
func ReadAll[T any](ctx context.Context, limit int, fetch Page[T], key func(T) string) ([]T, error) {
	var prev map[string]struct{}
	for pass := 0; pass < maxPasses; pass++ {
		items, keys, complete, err := readPass(ctx, pass+1, limit, fetch, key)
		switch {
		case err != nil:
			return nil, err
		case !complete:
			// Fewer items than the panel's total: a page boundary moved
			// under a change. This pass cannot count as stable.
			prev = nil
		case prev != nil && maps.Equal(prev, keys):
			return items, nil
		default:
			prev = keys
		}
	}
	return nil, ErrIncomplete
}

// readPass reads one pass; complete is false when it saw fewer distinct
// items than the panel's last total.
func readPass[T any](ctx context.Context, pass, limit int, fetch Page[T], key func(T) string) ([]T, map[string]struct{}, bool, error) {
	var out []T
	keys := map[string]struct{}{}
	total := -1
	for offset, pages := 0, 0; total < 0 || offset < total; pages++ {
		if total >= 0 && pages > total/limit+16 {
			return nil, nil, false, fmt.Errorf("%w: paging does not end", ErrIncomplete)
		}
		items, t, err := fetch(ctx, offset, limit)
		if err != nil {
			return nil, nil, false, err
		}
		if t < 0 {
			return nil, nil, false, BadResponse("negative total %d", t)
		}
		total = t
		if len(items) == 0 {
			break
		}
		for _, it := range items {
			k := key(it)
			if _, dup := keys[k]; dup {
				continue
			}
			keys[k] = struct{}{}
			out = append(out, it)
		}
		report(ctx, Progress{Pass: pass, Read: len(out), Total: total})
		offset += len(items)
	}
	return out, keys, len(out) >= total, nil
}
