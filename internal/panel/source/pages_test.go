package source

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"testing"
)

// list is a fake panel list sorted by name; mutate runs before each page.
type list struct {
	names  []string
	calls  int
	mutate func(l *list)
}

func (l *list) page(_ context.Context, offset, limit int) ([]string, int, error) {
	l.calls++
	if l.mutate != nil {
		l.mutate(l)
	}
	slices.Sort(l.names)
	end := min(offset+limit, len(l.names))
	if offset > end {
		return nil, len(l.names), nil
	}
	return slices.Clone(l.names[offset:end]), len(l.names), nil
}

func names(n int) []string {
	out := make([]string, n)
	for i := range out {
		out[i] = fmt.Sprintf("u%04d", i)
	}
	return out
}

func id(s string) string { return s }

func TestReadAllStableList(t *testing.T) {
	l := &list{names: names(1234)}
	got, err := ReadAll(context.Background(), 500, l.page, id)
	if err != nil || len(got) != 1234 {
		t.Fatalf("%d %v", len(got), err)
	}
	if l.calls != 6 {
		t.Errorf("two passes of 3 pages expected, got %d calls", l.calls)
	}
}

// Each page read is reported with its pass, so the operator sees users
// read of total, and a second pass that makes sure nothing changed.
func TestReadAllReportsEachPage(t *testing.T) {
	l := &list{names: names(1234)}
	var got []Progress
	ctx := WithProgress(context.Background(), func(p Progress) { got = append(got, p) })
	if _, err := ReadAll(ctx, 500, l.page, id); err != nil {
		t.Fatal(err)
	}
	var want []Progress
	for pass := 1; pass <= 2; pass++ {
		for _, read := range []int{500, 1000, 1234} {
			want = append(want, Progress{Pass: pass, Read: read, Total: 1234})
		}
	}
	if !slices.Equal(got, want) {
		t.Fatalf("progress %v", got)
	}
}

func TestReadAllCatchesSwapDuringPaging(t *testing.T) {
	// During the first pass one user is deleted and one created before it,
	// so the count stays right but a page boundary shifts.
	l := &list{names: names(1000)}
	l.mutate = func(l *list) {
		if l.calls == 2 {
			l.names = slices.DeleteFunc(l.names, func(s string) bool { return s == "u0700" })
			l.names = append(l.names, "a-new")
		}
	}
	got, err := ReadAll(context.Background(), 500, l.page, id)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1000 || !slices.Contains(got, "a-new") || slices.Contains(got, "u0700") {
		t.Fatalf("returned a stale or partial pass: %d items", len(got))
	}
}

func TestReadAllGivesUpOnABusyList(t *testing.T) {
	n := 0
	l := &list{names: names(10)}
	l.mutate = func(l *list) { n++; l.names = append(l.names, fmt.Sprintf("z%04d", n)) }
	if _, err := ReadAll(context.Background(), 5, l.page, id); !errors.Is(err, ErrIncomplete) {
		t.Fatalf("err = %v", err)
	}
}

func TestReadAllRefusesAShortList(t *testing.T) {
	short := func(_ context.Context, offset, _ int) ([]string, int, error) {
		if offset > 0 {
			return nil, 10, nil
		}
		return []string{"a", "b"}, 10, nil
	}
	if _, err := ReadAll(context.Background(), 5, short, id); !errors.Is(err, ErrIncomplete) {
		t.Fatalf("err = %v", err)
	}
}

func TestReadAllDedupes(t *testing.T) {
	dup := func(_ context.Context, offset, _ int) ([]string, int, error) {
		if offset == 0 {
			return []string{"a", "b", "b"}, 3, nil
		}
		return nil, 3, nil
	}
	_, err := ReadAll(context.Background(), 3, dup, id)
	if !errors.Is(err, ErrIncomplete) {
		t.Fatalf("a duplicated row hides a missing one; err = %v", err)
	}
}

func TestReadAllEmpty(t *testing.T) {
	l := &list{}
	got, err := ReadAll(context.Background(), 500, l.page, id)
	if err != nil || len(got) != 0 {
		t.Fatalf("%v %v", got, err)
	}
}
