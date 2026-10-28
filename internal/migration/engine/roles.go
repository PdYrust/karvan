package engine

import (
	"slices"
	"strconv"
	"time"

	"github.com/PdYrust/karvan/internal/migration/plan"
	"github.com/PdYrust/karvan/internal/panel"
)

// Role conflicts (owner, 2026-10-16; spec D65-D69). PasarGuard checks a
// role's limits only when the reseller creates or edits a user, never on
// rows already there, so a moved user beyond them works; the reseller's
// next edit of that value fails. The operator sees each case and decides.
const (
	RoleLimit     = "C31" // a user beyond the role's limits
	RoleFeature   = "C32" // a user setting the role does not allow
	RoleMaxUsers  = "C33" // the admin would own more users than the role allows
	RoleTemplates = "C34" // the role only lets the admin use templates
	RoleMissing   = "C35" // the chosen role is not on the destination
	RoleOwner     = "C37" // the owner role bypasses every check
)

// operatorRole is PasarGuard's built-in reseller role.
const operatorRole = 3

// chooseRole picks an admin's role on a PasarGuard destination (D65): an
// admin merged into keeps its own; else the operator's choice; else the
// role with the source role's name; else operator. The owner role is
// never picked by default.
func chooseRole(ap plan.AdminPlan, out *AdminOutcome, d Dest) []Conflict {
	if d.Kind != panel.PasarGuard {
		return nil
	}
	id, overrides := ap.RoleID, out.Source.Overrides
	if out.Merge != nil {
		id, overrides = out.Merge.RoleID, out.Merge.Overrides
	}
	if id == 0 {
		id = defaultRole(d.Roles, out.Source.RoleName)
	}
	i := slices.IndexFunc(d.Roles, func(r panel.Role) bool { return r.ID == id })
	if i < 0 {
		return []Conflict{{Type: RoleMissing, Severity: Blocker, Detail: map[string]string{"role": strconv.FormatInt(id, 10)}}}
	}
	r := d.Roles[i]
	out.role, out.limits, out.RoleID = &r, r.Limits.With(overrides), r.ID
	var c []Conflict
	if r.IsOwner {
		c = append(c, Conflict{Type: RoleOwner, Severity: Warning, Detail: map[string]string{"role": r.Name}})
	}
	if r.RequireTemplate {
		c = append(c, Conflict{Type: RoleTemplates, Severity: Warning, Detail: map[string]string{"role": r.Name}})
	}
	return c
}

func defaultRole(roles []panel.Role, sourceName string) int64 {
	for _, r := range roles {
		if sourceName != "" && r.Name == sourceName && !r.IsOwner {
			return r.ID
		}
	}
	for _, r := range roles {
		if r.ID == operatorRole {
			return r.ID
		}
	}
	for _, r := range roles {
		if !r.IsOwner {
			return r.ID
		}
	}
	return operatorRole
}

// defaultTargets is where an admin's users go (D31, D32): every enabled
// group its role allows on PasarGuard, every inbound on Marzban. There is
// no other choice.
func defaultTargets(d Dest, role *panel.Role) []string {
	var out []string
	for _, t := range d.Targets {
		if t.Disabled {
			continue
		}
		if role != nil && role.Groups != nil && !role.IsOwner && !slices.Contains(role.Groups, t.ID) {
			continue
		}
		out = append(out, t.ID)
	}
	return out
}

// maxUsers is C33: each admin's users after the move against its role.
// Admins merged into one existing admin count together.
func maxUsers(admins []AdminOutcome) []Conflict {
	var out []Conflict
	for _, a := range admins {
		if a.role == nil || a.role.IsOwner || a.limits.MaxUsers == nil {
			continue
		}
		n := a.Users
		if a.Merge != nil {
			n = a.Merge.Users
			for _, b := range admins {
				if b.Merge != nil && b.Merge.ID == a.Merge.ID {
					n += b.Users
				}
			}
		}
		if int64(n) <= *a.limits.MaxUsers {
			continue
		}
		out = append(out, Conflict{Type: RoleMaxUsers, Severity: Warning, Detail: map[string]string{
			"admin": a.Source.Username, "max": strconv.FormatInt(*a.limits.MaxUsers, 10), "users": strconv.Itoa(n)}})
	}
	return out
}

// roleChecks finds the user values the role's limits or features forbid.
func (d *detector) roleChecks(it *Item) {
	a := d.in.admins[it.Admin]
	if a.role == nil || a.role.IsOwner {
		return
	}
	if fields := beyondLimits(it.User, a.limits, d.in.now); len(fields) > 0 {
		it.add(RoleLimit, Warning, map[string]string{"fields": joinSorted(fields)})
	}
	if fields := forbidden(it.User, a.role.Features); len(fields) > 0 {
		it.add(RoleFeature, Warning, map[string]string{"fields": joinSorted(fields)})
	}
}

// beyondLimits names the values outside l, the way the panel's
// _enforce_user_limits judges them. An unlimited value breaks a maximum.
func beyondLimits(u panel.User, l panel.RoleLimits, now time.Time) []string {
	var out []string
	if over(u.DataLimit, l.DataLimitMin, l.DataLimitMax) {
		out = append(out, "data_limit")
	}
	if u.Status == panel.StatusOnHold && u.OnHoldExpire != nil {
		if over(int64(u.OnHoldExpire.Seconds()), l.ExpireMin, l.ExpireMax) {
			out = append(out, "on_hold")
		}
	} else if expireOver(u.Expire, l, now) {
		out = append(out, "expire")
	}
	if u.HWIDLimit != nil && *u.HWIDLimit > 0 && over(int64(*u.HWIDLimit), l.MinHWID, l.MaxHWID) {
		out = append(out, "hwid_limit")
	}
	return out
}

// over: v of 0 is unlimited, which only a maximum forbids.
func over(v int64, lo, hi *int64) bool {
	if v <= 0 {
		return hi != nil
	}
	return hi != nil && v > *hi || lo != nil && v < *lo
}

func expireOver(e *time.Time, l panel.RoleLimits, now time.Time) bool {
	if e == nil {
		return l.ExpireMax != nil
	}
	left := int64(e.Sub(now).Seconds())
	if left <= 0 {
		return false // already expired; any new expiry is the reseller's call
	}
	return l.ExpireMax != nil && left > *l.ExpireMax || l.ExpireMin != nil && left < *l.ExpireMin
}

func forbidden(u panel.User, f panel.RoleFeatures) []string {
	var out []string
	if !f.ResetStrategy && u.ResetStrategy != "" && u.ResetStrategy != panel.ResetNever {
		out = append(out, "reset_strategy")
	}
	if !f.NextPlan && u.NextPlan != nil {
		out = append(out, "next_plan")
	}
	return out
}

func joinSorted(s []string) string {
	slices.Sort(s)
	out := ""
	for i, v := range s {
		if i > 0 {
			out += ","
		}
		out += v
	}
	return out
}

// fitLimits moves the user's values to the nearest the role allows.
func fitLimits(u *panel.User, l panel.RoleLimits, now time.Time) bool {
	changed := false
	clamp := func(v int64, lo, hi *int64) int64 {
		if hi != nil && (v <= 0 || v > *hi) {
			changed = true
			return *hi
		}
		if lo != nil && v > 0 && v < *lo {
			changed = true
			return *lo
		}
		return v
	}
	u.DataLimit = clamp(u.DataLimit, l.DataLimitMin, l.DataLimitMax)
	if u.Status == panel.StatusOnHold && u.OnHoldExpire != nil {
		d := time.Duration(clamp(int64(u.OnHoldExpire.Seconds()), l.ExpireMin, l.ExpireMax)) * time.Second
		u.OnHoldExpire = &d
	} else if expireOver(u.Expire, l, now) {
		left := int64(0)
		if u.Expire != nil {
			left = int64(u.Expire.Sub(now).Seconds())
		}
		t := now.Add(time.Duration(clamp(left, l.ExpireMin, l.ExpireMax)) * time.Second).UTC().Truncate(time.Second)
		u.Expire = &t
	}
	if u.HWIDLimit != nil && *u.HWIDLimit > 0 {
		h := int(clamp(int64(*u.HWIDLimit), l.MinHWID, l.MaxHWID))
		u.HWIDLimit = &h
	}
	return changed
}

// removeForbidden drops the settings the role does not allow.
func removeForbidden(u *panel.User, f panel.RoleFeatures) bool {
	changed := false
	if !f.ResetStrategy && u.ResetStrategy != "" && u.ResetStrategy != panel.ResetNever {
		u.ResetStrategy, changed = panel.ResetNever, true
	}
	if !f.NextPlan && u.NextPlan != nil {
		u.NextPlan, changed = nil, true
	}
	return changed
}

// fillProtocols gives the user credentials for every protocol its targets
// serve that it lacks (D20, D67), so it reaches every inbound it is given.
// Old configs are untouched: these protocols were in none of them.
func fillProtocols(it *Item, dest Dest, gen func(key, protocol string) panel.Credentials) {
	for _, t := range it.Targets {
		for _, p := range dest.TargetProtocols[t] {
			if hasProtocol(it.User.Credentials, p) {
				continue
			}
			if setProtocol(&it.User.Credentials, p, gen(it.Key, p)) {
				it.note("added:" + p)
			}
		}
	}
}

func setProtocol(c *panel.Credentials, p string, fresh panel.Credentials) bool {
	switch {
	case p == "vmess" && fresh.VMess != nil:
		c.VMess = fresh.VMess
	case p == "vless" && fresh.VLESS != nil:
		c.VLESS = fresh.VLESS
	case p == "trojan" && fresh.Trojan != nil:
		c.Trojan = fresh.Trojan
	case p == "shadowsocks" && fresh.Shadowsocks != nil:
		c.Shadowsocks = fresh.Shadowsocks
	case p == "hysteria" && fresh.Hysteria != nil:
		c.Hysteria = fresh.Hysteria
	case p == "wireguard" && fresh.WireGuard != nil:
		c.WireGuard = fresh.WireGuard
	default:
		return false
	}
	return true
}
