package engine

import (
	"fmt"
	"math"
	"path"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/PdYrust/karvan/internal/migration/plan"
	"github.com/PdYrust/karvan/internal/panel"
)

const gib = 1 << 30

// PlanError is a plan the engine cannot run (unknown field, bad value).
type PlanError struct {
	Where, Problem string
}

func (e *PlanError) Error() string { return e.Where + ": " + e.Problem }

func planErr(where, f string, a ...any) error {
	return &PlanError{Where: where, Problem: fmt.Sprintf(f, a...)}
}

// Fields lists every condition field and the comparisons it takes.
var Fields = map[string][]string{
	"status": {"in"}, "reset_strategy": {"in"}, "users": {"in"},
	"source_group": {"in"}, "source_inbound": {"in"},
	"used_gb": {"gt", "lt"}, "left_gb": {"gt", "lt"}, "data_limit_gb": {"gt", "lt"},
	"used_percent": {"gte", "lte"},
	"expire":       {"before", "after"}, "expires_within_days": {"lte"}, "not_online_days": {"gte"},
	"expired": {"is"}, "unlimited_traffic": {"is"}, "unlimited_time": {"is"}, "never_connected": {"is"},
	"has_next_plan": {"is"}, "has_note": {"is"},
	"username": {"prefix", "suffix", "contains", "glob"}, "note": {"contains"}, "tag": {"has"},
}

// matcher is a parsed condition.
type matcher func(it *Item) bool

func compile(c plan.Condition, now time.Time) (matcher, error) {
	ops, ok := Fields[c.Field]
	if !ok {
		return nil, planErr("condition", "unknown field %q", c.Field)
	}
	if !slices.Contains(ops, c.Op) {
		return nil, planErr("condition "+c.Field, "unknown comparison %q", c.Op)
	}
	switch c.Op {
	case "in":
		return listMatcher(c), nil
	case "is":
		return boolMatcher(c, now)
	case "prefix", "suffix", "contains", "glob", "has":
		return textMatcher(c), nil
	case "before", "after":
		return dateMatcher(c)
	}
	return numberMatcher(c, now)
}

func listMatcher(c plan.Condition) matcher {
	return func(it *Item) bool {
		u := it.Source
		switch c.Field {
		case "status":
			return slices.Contains(c.List, string(u.Status))
		case "reset_strategy":
			return slices.Contains(c.List, string(u.ResetStrategy))
		case "users":
			return slices.Contains(c.List, it.Key)
		case "source_group":
			return slices.ContainsFunc(u.SourceGroups, func(g int64) bool { return slices.Contains(c.List, strconv.FormatInt(g, 10)) })
		}
		for _, tags := range u.SourceInbounds {
			if slices.ContainsFunc(tags, func(t string) bool { return slices.Contains(c.List, t) }) {
				return true
			}
		}
		return false
	}
}

func boolMatcher(c plan.Condition, now time.Time) (matcher, error) {
	want, err := strconv.ParseBool(c.Value)
	if err != nil {
		return nil, planErr("condition "+c.Field, "value must be true or false")
	}
	return func(it *Item) bool {
		u := it.Source
		var v bool
		switch c.Field {
		case "expired":
			v = u.Status == "expired" || u.Expire != nil && !u.Expire.After(now)
		case "unlimited_traffic":
			v = u.DataLimit == 0
		case "unlimited_time":
			v = u.Expire == nil
		case "never_connected":
			v = u.OnlineAt == nil
		case "has_next_plan":
			v = u.NextPlan != nil
		case "has_note":
			v = strings.TrimSpace(u.Note) != ""
		}
		return v == want
	}, nil
}

func textMatcher(c plan.Condition) matcher {
	v := c.Value
	return func(it *Item) bool {
		switch c.Field {
		case "tag":
			return slices.Contains(it.Tags, v)
		case "note":
			return strings.Contains(strings.ToLower(it.Source.Note), strings.ToLower(v))
		}
		name := it.Source.Username
		switch c.Op {
		case "prefix":
			return strings.HasPrefix(name, v)
		case "suffix":
			return strings.HasSuffix(name, v)
		case "contains":
			return strings.Contains(name, v)
		}
		ok, err := path.Match(v, name)
		return err == nil && ok
	}
}

func dateMatcher(c plan.Condition) (matcher, error) {
	t, err := time.Parse(time.RFC3339, c.Value)
	if err != nil {
		return nil, planErr("condition expire", "date must be RFC 3339")
	}
	return func(it *Item) bool {
		e := it.Source.Expire
		if e == nil {
			return false
		}
		if c.Op == "before" {
			return e.Before(t)
		}
		return e.After(t)
	}, nil
}

func numberMatcher(c plan.Condition, now time.Time) (matcher, error) {
	n, err := strconv.ParseFloat(c.Value, 64)
	if err != nil || n < 0 {
		return nil, planErr("condition "+c.Field, "value must be a number of zero or more")
	}
	cmp := comparer(c.Op, n)
	value := numberField(c.Field, now)
	return func(it *Item) bool {
		v, ok := value(it.Source)
		return ok && cmp(v)
	}, nil
}

func comparer(op string, n float64) func(float64) bool {
	switch op {
	case "gt":
		return func(v float64) bool { return v > n }
	case "lt":
		return func(v float64) bool { return v < n }
	case "gte":
		return func(v float64) bool { return v >= n }
	}
	return func(v float64) bool { return v <= n }
}

// numberField reads a numeric field; ok is false when the user has no
// such value (an unlimited user has no "left").
func numberField(field string, now time.Time) func(u panel.User) (float64, bool) {
	switch field {
	case "used_gb":
		return func(u panel.User) (float64, bool) { return float64(u.UsedTraffic) / gib, true }
	case "data_limit_gb":
		return func(u panel.User) (float64, bool) { return float64(u.DataLimit) / gib, u.DataLimit > 0 }
	case "left_gb":
		return func(u panel.User) (float64, bool) { return float64(u.DataLimit-u.UsedTraffic) / gib, u.DataLimit > 0 }
	case "used_percent":
		return func(u panel.User) (float64, bool) {
			return 100 * float64(u.UsedTraffic) / float64(max(u.DataLimit, 1)), u.DataLimit > 0
		}
	case "expires_within_days":
		return func(u panel.User) (float64, bool) {
			if u.Expire == nil || !u.Expire.After(now) {
				return 0, false
			}
			return u.Expire.Sub(now).Hours() / 24, true
		}
	}
	// not_online_days: never connected counts as forever.
	return func(u panel.User) (float64, bool) {
		if u.OnlineAt == nil {
			return math.MaxFloat64, true
		}
		return now.Sub(*u.OnlineAt).Hours() / 24, true
	}
}
