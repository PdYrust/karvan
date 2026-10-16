package engine

import (
	"errors"
	"time"

	"github.com/PdYrust/karvan/internal/panel"
)

// Conflict types. Numbers follow panel-diff.md section 3 and phase-3 D24.
const (
	AdminTaken     = "C01"
	NameTaken      = "C02"
	CaseOnly       = "C03"
	NameInvalid    = "C04"
	NameTwice      = "C05"
	CredTwice      = "C06"
	CredTaken      = "C07"
	NoTargets      = "C09"
	NoProtocol     = "C10"
	FlowDropped    = "C11"
	ShadowsocksBad = "C12"
	AlreadyMoved   = "C16"
	MovedGone      = "C16b"
	OnHoldInvalid  = "C18"
	WireGuardIPs   = "C19"
	SourceInvalid  = "C24"
	ResetEstimated = "C26"
	ClockSkew      = "C27"
	AlreadyHere    = "C28"
	Unsettled      = "C29" // the plan keeps changing after five passes
	AdminInvalid   = "C30"
)

// Severities.
const (
	Blocker = "blocker" // needs the operator's choice before running
	Warning = "warning" // a safe default is applied; the operator may change it
	Info    = "info"
)

// Conflict is one problem, for one user or (User == "") for the job.
type Conflict struct {
	Type     string            `json:"type"`
	User     string            `json:"user,omitempty"`
	Severity string            `json:"severity"`
	Detail   map[string]string `json:"detail,omitempty"`
	Fix      string            `json:"fix,omitempty"` // the option applied
}

// detector finds the per-user conflicts of one pass.
type detector struct {
	in    Inputs
	items []*Item
	// names and creds hold what included users of this build already
	// claim, in sort order, so the later user of a pair is the one flagged.
	names map[string]string
	creds map[panel.CredentialKey]string
}

func detect(in Inputs, items []*Item) {
	d := &detector{in: in, items: items, names: map[string]string{}, creds: map[panel.CredentialKey]string{}}
	for _, it := range items {
		it.Conflicts = it.Conflicts[:0]
		if !it.Include && it.Adopt == nil {
			continue
		}
		if d.history(it) {
			continue
		}
		d.naming(it)
		d.credentials(it)
		d.shape(it)
	}
}

func (it *Item) add(t, sev string, detail map[string]string) {
	it.Conflicts = append(it.Conflicts, Conflict{Type: t, User: it.Key, Severity: sev, Detail: detail})
}

// history finds users moved before (C16, C16b) or already present (C28).
// It returns true when nothing else needs checking.
func (d *detector) history(it *Item) bool {
	if m, ok := d.in.Moved[it.Key]; ok {
		if present(d.in.Dest.Index, m) && m.State != "unknown" {
			it.add(AlreadyMoved, Info, map[string]string{"username": m.Username})
			return true
		}
		it.add(MovedGone, Warning, map[string]string{"username": m.Username})
		return false
	}
	if u := d.alreadyHere(it); u != nil {
		it.add(AlreadyHere, Warning, map[string]string{"username": u.Username})
		return true
	}
	return false
}

func present(x *panel.Index, m Moved) bool {
	u, ok := x.UsersByID[m.DestID]
	return ok && u.Username == m.Username && u.CreatedAt.Equal(m.Created)
}

// alreadyHere is C28: the same name, created time and every credential
// are on one destination user, with no ledger row (a lost ledger, another
// Karvan, a hand copy).
func (d *detector) alreadyHere(it *Item) *panel.ExistingUser {
	x := d.in.Dest.Index
	u := x.Users[d.in.Dest.Rules.UserKey(it.Source.Username)]
	if u == nil || !sameTime(u.CreatedAt, it.Source.CreatedAt) {
		return nil
	}
	keys := it.Source.Credentials.Keys()
	for _, k := range keys {
		if x.Credentials[k] != u.ID {
			return nil
		}
	}
	if len(keys) == 0 {
		return nil
	}
	return u
}

// sameTime compares at the coarsest precision a backend keeps (seconds on
// MySQL), so a row written by an earlier run still matches.
func sameTime(a, b time.Time) bool { return a.Unix() == b.Unix() }

func (d *detector) naming(it *Item) {
	name := it.User.Username
	if err := d.in.Dest.Rules.ValidateUsername(name); err != nil {
		var re *panel.RuleError
		code := "invalid"
		if errors.As(err, &re) {
			code = re.Code
		}
		it.add(NameInvalid, Blocker, map[string]string{"code": code})
		return
	}
	key := d.in.Dest.Rules.UserKey(name)
	if u, ok := d.in.Dest.Index.Users[key]; ok {
		t := NameTaken
		if u.Username != name {
			t = CaseOnly
		}
		it.add(t, Blocker, map[string]string{"username": u.Username})
		return
	}
	if other, ok := d.names[key]; ok {
		it.add(NameTwice, Blocker, map[string]string{"other": other})
		return
	}
	if it.Include {
		d.names[key] = it.Key
	}
}

func (d *detector) credentials(it *Item) {
	for _, k := range it.User.Credentials.Keys() {
		proto := protocolOf(k)
		if owner, ok := d.in.Dest.Index.Credentials[k]; ok {
			name := ""
			if u := d.in.Dest.Index.UsersByID[owner]; u != nil {
				name = u.Username
			}
			it.add(CredTaken, Blocker, map[string]string{"protocol": proto, "username": name})
			continue
		}
		if other, ok := d.creds[k]; ok {
			it.add(CredTwice, Blocker, map[string]string{"protocol": proto, "other": other})
			continue
		}
		if it.Include {
			d.creds[k] = it.Key
		}
	}
}

func protocolOf(k panel.CredentialKey) string {
	s := string(k)
	for i := range len(s) {
		if s[i] == ':' {
			return s[:i]
		}
	}
	return s
}

// shape finds values the destination cannot hold as they are.
func (d *detector) shape(it *Item) {
	d.credentialShape(it)
	u := it.User
	if !reachable(it, d.in.Dest) {
		it.add(NoProtocol, Warning, nil)
	}
	if u.Status == panel.StatusOnHold && (u.Expire != nil || u.OnHoldExpire == nil || *u.OnHoldExpire <= 0) {
		it.add(OnHoldInvalid, Warning, nil)
	}
	if u.ResetStrategy != panel.ResetNever && u.ResetStrategy != "" {
		it.add(ResetEstimated, Info, map[string]string{"strategy": string(u.ResetStrategy)})
	}
}

func (d *detector) credentialShape(it *Item) {
	c, dest := it.User.Credentials, d.in.Dest
	if err := panel.ValidateUUIDs(c); err != nil || emptySecret(c) {
		it.add(SourceInvalid, Blocker, nil)
	}
	var re *panel.RuleError
	if err := dest.Rules.ValidateCredentials(c); errors.As(err, &re) && (re.Code == panel.CodePasswordShort || re.Code == panel.CodeMethod) {
		it.add(ShadowsocksBad, Blocker, map[string]string{"code": re.Code})
	}
	if dest.Kind != panel.PasarGuard {
		return
	}
	if c.VLESS != nil && c.VLESS.Flow != "" {
		it.add(FlowDropped, Blocker, map[string]string{"flow": c.VLESS.Flow})
	}
	if c.WireGuard != nil {
		it.add(WireGuardIPs, Blocker, nil)
	}
}

func emptySecret(c panel.Credentials) bool {
	return c.Trojan != nil && c.Trojan.Password.IsZero() ||
		c.Shadowsocks != nil && c.Shadowsocks.Password.IsZero() ||
		c.Hysteria != nil && c.Hysteria.Auth.IsZero()
}

// reachable reports whether the user keeps at least one protocol its
// targets serve; otherwise it could not connect after the move.
func reachable(it *Item, dest Dest) bool {
	for _, t := range it.Targets {
		for _, p := range dest.TargetProtocols[t] {
			if hasProtocol(it.User.Credentials, p) {
				return true
			}
		}
	}
	return len(it.Targets) == 0 // no targets is the job-level C09
}

func hasProtocol(c panel.Credentials, p string) bool {
	switch p {
	case "vmess":
		return c.VMess != nil
	case "vless":
		return c.VLESS != nil
	case "trojan":
		return c.Trojan != nil
	case "shadowsocks":
		return c.Shadowsocks != nil
	case "hysteria":
		return c.Hysteria != nil
	case "wireguard":
		return c.WireGuard != nil
	}
	return false
}
