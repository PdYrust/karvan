package engine

import (
	"strconv"

	"github.com/PdYrust/karvan/internal/migration/plan"
	"github.com/PdYrust/karvan/internal/panel"
	"github.com/PdYrust/karvan/internal/panel/marzban"
	"github.com/PdYrust/karvan/internal/panel/pasarguard"
)

// resolver applies fixes to the conflicts of one pass.
type resolver struct {
	p     plan.Plan
	in    Inputs
	rc    rankContext
	taken map[string]bool // destination keys claimed in this build
	gen   func(key, protocol string) panel.Credentials
}

// choose picks the fix for a conflict: the user's own choice, then the
// "apply to all" choice for the type, then the safe default.
func (r *resolver) choose(it *Item, c Conflict) (plan.Resolution, bool) {
	if d, ok := r.p.Users[it.Key]; ok {
		if res, ok := d.Resolutions[c.Type]; ok {
			return res, true
		}
	}
	if res, ok := r.p.Resolutions[c.Type]; ok {
		return res, true
	}
	if o, ok := defaultFix(c.Type, r.rc); ok {
		return plan.Resolution{Option: o.ID}, true
	}
	return plan.Resolution{}, false
}

// resolve applies one fix per conflict. It returns how many changed the
// item, so the build knows whether another pass is needed.
func (r *resolver) resolve(it *Item) int {
	changed := 0
	for i := range it.Conflicts {
		c := &it.Conflicts[i]
		res, ok := r.choose(it, *c)
		if !ok {
			c.Severity = Blocker
			continue
		}
		if !allowed(c.Type, res.Option) {
			c.Severity = Blocker
			c.Detail = merge(c.Detail, map[string]string{"invalidFix": res.Option})
			continue
		}
		c.Fix = res.Option
		if c.Severity == Blocker {
			c.Severity = Warning
		}
		switch {
		case r.apply(it, *c, res):
			it.note("fix:" + c.Type + ":" + res.Option)
			it.Resolved = append(it.Resolved, *c)
			changed++
		case mustChange[res.Option]:
			// The fix could not be carried out (a typed name that is taken
			// or invalid, nothing left after sanitizing).
			c.Severity, c.Fix = Blocker, ""
			c.Detail = merge(c.Detail, map[string]string{"fixFailed": res.Option})
		}
	}
	return changed
}

// mustChange are fixes that fail when they leave the user as it was.
var mustChange = map[string]bool{
	OptPrefix: true, OptSuffix: true, OptRename: true, OptSanitize: true,
	OptNewCredential: true, OptDropProtocol: true, OptChangeMethod: true,
}

func allowed(t, option string) bool {
	for _, o := range optionTable[t] {
		if o.ID == option {
			return true
		}
	}
	return false
}

func merge(a, b map[string]string) map[string]string {
	out := map[string]string{}
	for k, v := range a {
		out[k] = v
	}
	for k, v := range b {
		out[k] = v
	}
	return out
}

// apply carries out res on it and reports whether the item changed.
func (r *resolver) apply(it *Item, c Conflict, res plan.Resolution) bool {
	u := &it.User
	switch res.Option {
	case OptSkip:
		if !it.Include {
			return false
		}
		it.Include = false
		return true
	case OptPrefix, OptSuffix, OptRename, OptSanitize:
		return r.rename(it, res)
	case OptNewCredential:
		proto := protocolFor(c)
		replaceCredential(it, proto, r.gen(it.Key, proto))
		return true
	case OptDropProtocol:
		dropProtocol(it, protocolFor(c))
		return true
	case OptChangeMethod:
		if u.Credentials.Shadowsocks == nil || u.Credentials.Shadowsocks.Method == res.Params["method"] {
			return false
		}
		u.Credentials.Shadowsocks.Method = res.Params["method"]
		it.ChangesConfig = true
		return true
	case OptAccept:
		return accept(it, c)
	case OptAdopt:
		it.Include = false
		it.Adopt = r.in.Dest.Index.Users[r.in.Dest.Rules.UserKey(it.Source.Username)]
		return false
	case OptNormalize:
		return normalizeOnHold(u)
	case OptFit:
		return fitLimits(u, r.in.limits, r.in.now)
	case OptRemove:
		return r.in.role != nil && removeForbidden(u, r.in.role.Features)
	}
	// keep, move_again: nothing to change.
	return false
}

func protocolFor(c Conflict) string {
	if p := c.Detail["protocol"]; p != "" {
		return p
	}
	if c.Type == WireGuardIPs {
		return "wireguard"
	}
	return "shadowsocks"
}

func accept(it *Item, c Conflict) bool {
	creds := &it.User.Credentials
	switch c.Type {
	case FlowDropped:
		if creds.VLESS == nil || creds.VLESS.Flow == "" {
			return false
		}
		creds.VLESS.Flow = ""
	case WireGuardIPs:
		if creds.WireGuard == nil || creds.WireGuard.PeerIPs == nil {
			return false
		}
		creds.WireGuard.PeerIPs = nil // reassigned on write (D21)
	default:
		return false
	}
	it.ChangesConfig = true
	return true
}

func normalizeOnHold(u *panel.User) bool {
	if u.OnHoldExpire == nil || *u.OnHoldExpire <= 0 {
		u.Status = panel.StatusActive
		u.OnHoldExpire = nil
		return true
	}
	if u.Expire != nil {
		u.Expire = nil
		return true
	}
	return false
}

// rename gives the user a free, valid name for the destination.
func (r *resolver) rename(it *Item, res plan.Resolution) bool {
	base := it.User.Username
	switch res.Option {
	case OptPrefix:
		base = res.Params["text"] + base
	case OptSuffix:
		if t := res.Params["text"]; t != "" {
			base += t
		}
	case OptRename:
		if n := res.Params["username"]; n != "" {
			base = n
		}
	case OptSanitize:
		base = r.sanitize(base)
		if base == "" {
			return false
		}
	}
	if res.Option == OptRename && !r.usable(base) {
		return false
	}
	name := r.free(base, res.Option == OptSuffix && res.Params["text"] == "")
	if name == "" || name == it.User.Username {
		return false
	}
	it.User.Username = name
	it.ChangesLink = true
	r.taken[r.in.Dest.Rules.UserKey(name)] = true
	return true
}

func (r *resolver) sanitize(name string) string {
	if r.in.Dest.Kind == panel.Marzban {
		return marzban.Sanitize(name)
	}
	return pasarguard.Sanitize(name)
}

// free returns base if it is valid and unclaimed, else base with the
// smallest number suffix that is (when numbered is set, it always adds one).
// usable reports whether name is valid and claimed by nobody.
func (r *resolver) usable(name string) bool {
	k := r.in.Dest.Rules.UserKey(name)
	_, onDest := r.in.Dest.Index.Users[k]
	return !onDest && !r.taken[k] && r.in.Dest.Rules.ValidateUsername(name) == nil
}

func (r *resolver) free(base string, numbered bool) string {
	ok := r.usable
	if !numbered && ok(base) {
		return base
	}
	sep := "_"
	if l := len(base); l > 0 && (base[l-1] == '_' || base[l-1] == '-' || base[l-1] == '.' || base[l-1] == '@') {
		sep = ""
	}
	for n := 2; n < 1000; n++ {
		if cand := base + sep + strconv.Itoa(n); ok(cand) {
			return cand
		}
	}
	return ""
}
