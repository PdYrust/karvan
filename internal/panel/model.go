// Package panel is the one door to Marzban and PasarGuard panels. It holds
// the shared model and the interfaces each panel adapter implements, so the
// migration engine never depends on a specific panel.
//
// Version 1 reads a source panel through its API and writes the destination
// panel through its database.
package panel

import (
	"log/slog"
	"strings"
	"time"

	"github.com/PdYrust/karvan/internal/secret"
)

// Kind names a panel product.
type Kind string

// Supported panels.
const (
	Marzban    Kind = "marzban"
	PasarGuard Kind = "pasarguard"
)

// UUID is a canonical (lowercase, hyphenated) UUID string. Both panels
// canonicalize UUIDs, so comparing canonical forms finds real duplicates.
// For VMess and VLESS the UUID is the credential, so it is redacted in logs.
type UUID string

// CanonicalUUID lowercases and trims s. It does not validate.
func CanonicalUUID(s string) UUID { return UUID(strings.ToLower(strings.TrimSpace(s))) }

// LogValue implements slog.LogValuer.
func (UUID) LogValue() slog.Value { return slog.StringValue("[redacted]") }

// Status is a user's state. Limited and Expired are derived by the panels
// from traffic and expiry; they are carried so reports can show them.
type Status string

// User statuses.
const (
	StatusActive   Status = "active"
	StatusDisabled Status = "disabled"
	StatusLimited  Status = "limited"
	StatusExpired  Status = "expired"
	StatusOnHold   Status = "on_hold"
)

// ResetStrategy is how often a user's used traffic resets.
type ResetStrategy string

// Reset strategies shared by both panels.
const (
	ResetNever ResetStrategy = "no_reset"
	ResetDay   ResetStrategy = "day"
	ResetWeek  ResetStrategy = "week"
	ResetMonth ResetStrategy = "month"
	ResetYear  ResetStrategy = "year"
)

// Admin is a panel admin; in both panels a reseller is a non-owner admin.
type Admin struct {
	SourceID       int64 // id in the panel it was read from; 0 when the API has none
	Username       string
	IsOwner        bool // Marzban is_sudo, PasarGuard owner/administrator role
	TelegramID     *int64
	DiscordWebhook string
	Note           string
	UsedTraffic    int64 // Marzban users_usage, PasarGuard used_traffic
	// RoleID and RoleName are the admin's role on a PasarGuard source.
	RoleID   int64
	RoleName string
	// Overrides are the admin's own limits over its role (PasarGuard).
	Overrides *RoleLimits
}

// Credentials are the per-protocol secrets that live inside client configs.
// A nil protocol is absent; adapters must write every present protocol
// explicitly, because both panels fill omitted ones with new random values.
type Credentials struct {
	VMess       *VMess
	VLESS       *VLESS
	Trojan      *Trojan
	Shadowsocks *Shadowsocks
	WireGuard   *WireGuard
	Hysteria    *Hysteria
}

// VMess credentials.
type VMess struct{ ID UUID }

// VLESS credentials. Flow is per user only in Marzban.
type VLESS struct {
	ID   UUID
	Flow string
}

// Trojan credentials. Flow exists only in Marzban.
type Trojan struct {
	Password secret.Value
	Flow     string
}

// Shadowsocks credentials. PasarGuard requires at least 22 characters.
type Shadowsocks struct {
	Password secret.Value
	Method   string
}

// WireGuard credentials (PasarGuard only). Peer IPs are assigned by the
// destination and are informational here.
type WireGuard struct {
	PrivateKey secret.Value
	PublicKey  string
	PeerIPs    []string
}

// Hysteria credentials (PasarGuard only).
type Hysteria struct{ Auth secret.Value }

// NextPlan is what a user switches to when the current plan runs out.
type NextPlan struct {
	DataLimit           int64
	Expire              *time.Duration
	AddRemainingTraffic bool
	FireOnEither        bool // Marzban only
}

// User is one panel user with everything a migration carries.
type User struct {
	SourceID      int64
	Username      string
	OwnerUsername string // empty for orphan users
	Credentials   Credentials
	Status        Status
	Expire        *time.Time
	DataLimit     int64 // bytes; 0 is unlimited
	UsedTraffic   int64 // bytes
	LifetimeUsed  int64 // bytes
	ResetStrategy ResetStrategy
	Note          string
	OnHoldExpire  *time.Duration
	OnHoldTimeout *time.Time
	AutoDeleteIn  *int // days
	HWIDLimit     *int // PasarGuard only
	NextPlan      *NextPlan
	CreatedAt     time.Time
	OnlineAt      *time.Time
	SubRevokedAt  *time.Time // PasarGuard only; Marzban's API does not return it
	SubUpdatedAt  *time.Time
	SubUserAgent  string
	// Where the user sat on the source, for categories and the report:
	// Marzban protocol to inbound tags, PasarGuard group ids.
	SourceInbounds map[string][]string
	SourceGroups   []int64
	// SubscriptionURL is the link as the source panel shows it. It never
	// survives a move, but its token maps old links to new users.
	SubscriptionURL string
}

// SourceGroup is a group of an old PasarGuard panel with the protocols of
// its inbounds, so a user's current protocols can be told.
type SourceGroup struct {
	ID        int64
	Disabled  bool
	Protocols []string
}

// Target is where moved users are attached on the destination: a group in
// PasarGuard, or a protocol's inbound tags in Marzban.
type Target struct {
	ID       string // group id, or "<protocol>/<tag>"
	Name     string
	Protocol string   // set for Marzban inbounds
	Inbounds []string // a PasarGuard group's inbound tags
	Disabled bool     // a disabled PasarGuard group
}
