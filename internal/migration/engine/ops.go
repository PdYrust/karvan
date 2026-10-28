package engine

import (
	"math"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/PdYrust/karvan/internal/migration/plan"
	"github.com/PdYrust/karvan/internal/panel"
)

// Ops lists every operation and its parameters (D26).
var Ops = map[string][]string{
	"traffic.add": {"gb"}, "traffic.set": {"gb"}, "traffic.unlimited": nil, "traffic.reset_used": nil,
	"traffic.carry_remaining": nil, "traffic.strategy": {"strategy"},
	"time.add": {"days"}, "time.set": {"date"}, "time.restart": {"days"}, "time.unlimited": nil, "time.on_hold": {"days"},
	"status.set": {"status"}, "status.auto_delete": {"days"}, "status.hwid": {"limit"},
	"name.prefix": {"text"}, "name.suffix": {"text"}, "name.replace": {"from", "to"},
	"note.set": {"text"}, "note.append": {"text"}, "note.clear": nil,
	"tag.add": {"tag"}, "plan.clear_next": nil, "skip": nil,
	"cred.new": {"protocols"},
}

// opContext is what operations need besides the item.
type opContext struct {
	now      time.Time
	generate func(key, protocol string) panel.Credentials
}

// checkOp validates an operation before any user sees it. place.targets
// is gone (D32): groups come from the role, Marzban gets every inbound.
func checkOp(o plan.Op) error {
	if o.Kind == "place.targets" {
		return &PlanError{Where: "ops", Problem: "place.targets"}
	}
	want, ok := Ops[o.Kind]
	if !ok {
		return planErr("operation", "unknown operation %q", o.Kind)
	}
	for _, p := range want {
		if _, ok := o.Params[p]; !ok {
			return planErr("operation "+o.Kind, "missing %q", p)
		}
	}
	return checkOpValues(o)
}

func checkOpValues(o plan.Op) error {
	p := o.Params
	switch o.Kind {
	case "traffic.add", "traffic.set":
		if _, err := strconv.ParseFloat(p["gb"], 64); err != nil {
			return planErr("operation "+o.Kind, "gb must be a number")
		}
	case "time.add", "time.restart", "time.on_hold", "status.auto_delete", "status.hwid":
		key := "days"
		if o.Kind == "status.hwid" {
			key = "limit"
		}
		if _, err := strconv.Atoi(p[key]); err != nil {
			return planErr("operation "+o.Kind, "%s must be a whole number", key)
		}
	case "time.set":
		if _, err := time.Parse(time.RFC3339, p["date"]); err != nil {
			return planErr("operation time.set", "date must be RFC 3339")
		}
	case "status.set":
		if p["status"] != "active" && p["status"] != "disabled" {
			return planErr("operation status.set", "status must be active or disabled")
		}
	case "traffic.strategy":
		if !slices.Contains([]string{"no_reset", "day", "week", "month", "year"}, p["strategy"]) {
			return planErr("operation traffic.strategy", "unknown strategy")
		}
	}
	return nil
}

// apply runs a checked operation on it.
func apply(it *Item, o plan.Op, c opContext) {
	u := &it.User
	p := o.Params
	it.note("op:" + o.Kind)
	switch o.Kind {
	case "traffic.add", "traffic.set", "traffic.unlimited", "traffic.reset_used", "traffic.carry_remaining", "traffic.strategy":
		applyTraffic(u, o.Kind, p)
	case "time.add", "time.set", "time.restart", "time.unlimited", "time.on_hold":
		applyTime(u, o.Kind, p, c.now)
	case "status.set":
		u.Status = panel.Status(p["status"])
	case "status.auto_delete":
		n, _ := strconv.Atoi(p["days"])
		u.AutoDeleteIn = &n
	case "status.hwid":
		n, _ := strconv.Atoi(p["limit"])
		u.HWIDLimit = &n
	default:
		applyOther(it, o, c)
	}
}

func applyOther(it *Item, o plan.Op, c opContext) {
	u := &it.User
	p := o.Params
	switch o.Kind {
	case "name.prefix":
		u.Username = p["text"] + u.Username
	case "name.suffix":
		u.Username += p["text"]
	case "name.replace":
		u.Username = strings.ReplaceAll(u.Username, p["from"], p["to"])
	case "note.set":
		u.Note = p["text"]
	case "note.append":
		u.Note = strings.TrimSpace(u.Note + " " + p["text"])
	case "note.clear":
		u.Note = ""
	case "tag.add":
		if !slices.Contains(it.Tags, p["tag"]) {
			it.Tags = append(it.Tags, p["tag"])
		}
	case "plan.clear_next":
		u.NextPlan = nil
	case "skip":
		it.Include = false
	case "cred.new":
		for _, proto := range splitList(p["protocols"]) {
			replaceCredential(it, proto, c.generate(it.Key, proto))
		}
	}
}

func applyTraffic(u *panel.User, kind string, p map[string]string) {
	gb, _ := strconv.ParseFloat(p["gb"], 64)
	bytes := int64(math.Round(gb * gib))
	switch kind {
	case "traffic.add":
		if u.DataLimit > 0 {
			// Never below one byte: 0 would mean unlimited.
			u.DataLimit = max(u.DataLimit+bytes, 1)
		}
	case "traffic.set":
		u.DataLimit = max(bytes, 0)
	case "traffic.unlimited":
		u.DataLimit = 0
	case "traffic.reset_used":
		u.UsedTraffic = 0
	case "traffic.carry_remaining":
		if u.DataLimit > 0 {
			u.DataLimit = max(u.DataLimit-u.UsedTraffic, 1)
		}
		u.UsedTraffic = 0
	case "traffic.strategy":
		u.ResetStrategy = panel.ResetStrategy(p["strategy"])
	}
}

func applyTime(u *panel.User, kind string, p map[string]string, now time.Time) {
	days, _ := strconv.Atoi(p["days"])
	d := time.Duration(days) * 24 * time.Hour
	switch kind {
	case "time.add":
		if u.Expire != nil {
			t := u.Expire.Add(d)
			u.Expire = &t
		}
	case "time.set":
		t, _ := time.Parse(time.RFC3339, p["date"])
		t = t.UTC()
		u.Expire = &t
	case "time.restart":
		t := now.Add(d)
		u.Expire = &t
	case "time.unlimited":
		u.Expire = nil
	case "time.on_hold":
		u.Status, u.Expire, u.OnHoldExpire = panel.StatusOnHold, nil, &d
	}
}

// deriveStatus recomputes what the panels derive from traffic and time
// (their review jobs do the same), so "add 10 days" wakes an expired user.
// Disabled and on-hold users keep their status.
func deriveStatus(u *panel.User, now time.Time) {
	switch u.Status {
	case panel.StatusActive, panel.StatusLimited, panel.StatusExpired:
	default:
		return
	}
	switch {
	case u.DataLimit > 0 && u.UsedTraffic >= u.DataLimit:
		u.Status = panel.StatusLimited
	case u.Expire != nil && !u.Expire.After(now):
		u.Status = panel.StatusExpired
	default:
		u.Status = panel.StatusActive
	}
}

func splitList(s string) []string {
	var out []string
	for _, p := range strings.Split(s, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}
