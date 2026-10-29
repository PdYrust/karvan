// Package engine turns a source snapshot, the destination's facts and the
// operator's plan into the exact outcome per user: operations applied,
// conflicts found and resolved, until nothing changes. It is pure: no I/O,
// no clock, no randomness except through Inputs, so the same inputs always
// give the same result (D23).
package engine

import (
	"slices"
	"strconv"
	"time"

	"github.com/PdYrust/karvan/internal/panel"
)

// Source is what was read from the old panel.
type Source struct {
	Kind   panel.Kind
	Origin string      // normalized panel address, the ledger's source panel
	Admin  panel.Admin // the login's admin
	// Admins are the other admins the login can read; a user's owner
	// missing here moves by its name only (D34).
	Admins []panel.Admin
	Users  []panel.User
	Skew   time.Duration // the panel's clock minus ours
	// Groups are an old PasarGuard panel's groups; GroupsRead is false
	// when the login could not read them (D38).
	Groups     []panel.SourceGroup
	GroupsRead bool
}

// Dest is what the destination already holds.
type Dest struct {
	Kind   panel.Kind
	Rules  panel.Rules
	Index  *panel.Index
	Admins map[string]panel.ExistingAdmin // existing admins, by the names asked for
	// Asked are the admin names looked up, found or not; a name outside
	// it may exist on the destination.
	Asked   []string
	Targets []panel.Target
	// TargetProtocols lists the protocols each target serves.
	TargetProtocols map[string][]string
	// TargetHosts counts the enabled hosts behind each target, for the
	// operator; links come from hosts.
	TargetHosts map[string]int
	// Roles are PasarGuard's admin roles; nil on Marzban.
	Roles []panel.Role
}

// Moved is a ledger entry for a source user moved earlier.
type Moved struct {
	State    string // "done", "adopted", "unknown"
	DestID   int64
	Username string
	Created  time.Time
}

// Inputs are everything a build reads.
type Inputs struct {
	Source Source
	Dest   Dest
	// Moved are ledger entries for this source and destination, by Key.
	Moved map[string]Moved
	// MovedAdmins are the destination admins earlier runs created or
	// merged into, by source admin; a re-run merges into them by default.
	MovedAdmins map[string]string
	// Generate makes new credentials when a resolution asks for them.
	// Values it returns are kept in Generated, so a rebuild reuses them.
	Generate func(protocol string) panel.Credentials
	// Generated holds what Generate returned, by "<key>/<protocol>". The
	// job keeps it in memory between builds; it holds secrets.
	Generated map[string]panel.Credentials

	// Set by Build: where each source admin lands, and the plan's "now".
	admins map[string]*AdminOutcome
	now    time.Time
}

// Item is one user on its way through the build.
type Item struct {
	Key     string     // source identity: PasarGuard id or Marzban username
	Admin   string     // the source admin it belongs to (D34)
	Source  panel.User // as read; never changed
	User    panel.User // the working copy that will be written
	Include bool
	Tags    []string
	Targets []string
	Nodes   []string // categories the user belongs to, in tree order
	Applied []string // what changed, in order ("op:time.add", "fix:C02:suffix")

	Conflicts     []Conflict // open after the last pass
	Resolved      []Conflict // fixed in an earlier pass, for the report
	ChangesConfig bool
	ChangesLink   bool
	Adopt         *panel.ExistingUser // C28: already on the destination

	steady []string // protocols whose credentials Diff compares; nil for all
}

// KeyOf is the source identity of u for a source kind (D31).
func KeyOf(k panel.Kind, u panel.User) string {
	if k == panel.PasarGuard {
		return strconv.FormatInt(u.SourceID, 10)
	}
	return u.Username
}

func newItems(in Inputs) []*Item {
	out := make([]*Item, 0, len(in.Source.Users))
	for _, u := range in.Source.Users {
		owner := ownerOf(u, in.Source.Admin.Username)
		out = append(out, &Item{
			Key:     KeyOf(in.Source.Kind, u),
			Admin:   owner,
			Source:  u,
			User:    clone(u),
			Include: true,
			Targets: slices.Clone(in.admins[owner].Targets),
			steady:  steadyProtocols(in.Source, u),
		})
	}
	slices.SortFunc(out, func(a, b *Item) int { return compareKeys(a.Key, b.Key) })
	return out
}

// compareKeys orders numeric keys numerically and the rest as text, so
// "user10" sorts after "user9" only where the panel would too.
func compareKeys(a, b string) int {
	na, ea := strconv.ParseInt(a, 10, 64)
	nb, eb := strconv.ParseInt(b, 10, 64)
	if ea == nil && eb == nil {
		switch {
		case na < nb:
			return -1
		case na > nb:
			return 1
		}
		return 0
	}
	switch {
	case a < b:
		return -1
	case a > b:
		return 1
	}
	return 0
}

func (it *Item) note(s string) { it.Applied = append(it.Applied, s) }

// AlreadyMoved reports whether an earlier migration moved the user (C16),
// so a re-run leaves it as it is (phase 4.1 D9).
func (it *Item) AlreadyMoved() bool {
	is := func(c Conflict) bool { return c.Type == AlreadyMoved }
	return slices.ContainsFunc(it.Conflicts, is) || slices.ContainsFunc(it.Resolved, is)
}

// clone deep-copies the parts of u that operations change.
func clone(u panel.User) panel.User {
	c := u
	c.Credentials = cloneCreds(u.Credentials)
	if u.Expire != nil {
		t := *u.Expire
		c.Expire = &t
	}
	if u.NextPlan != nil {
		np := *u.NextPlan
		c.NextPlan = &np
	}
	if u.OnHoldExpire != nil {
		d := *u.OnHoldExpire
		c.OnHoldExpire = &d
	}
	return c
}

func cloneCreds(c panel.Credentials) panel.Credentials {
	out := panel.Credentials{}
	if c.VMess != nil {
		v := *c.VMess
		out.VMess = &v
	}
	if c.VLESS != nil {
		v := *c.VLESS
		out.VLESS = &v
	}
	if c.Trojan != nil {
		v := *c.Trojan
		out.Trojan = &v
	}
	if c.Shadowsocks != nil {
		v := *c.Shadowsocks
		out.Shadowsocks = &v
	}
	if c.WireGuard != nil {
		v := *c.WireGuard
		v.PeerIPs = slices.Clone(c.WireGuard.PeerIPs)
		out.WireGuard = &v
	}
	if c.Hysteria != nil {
		v := *c.Hysteria
		out.Hysteria = &v
	}
	return out
}
