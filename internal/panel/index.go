package panel

import (
	"log/slog"
	"time"
)

// ExistingAdmin is an admin already on the destination.
type ExistingAdmin struct {
	ID       int64
	Username string
	IsOwner  bool
	// RoleID, Overrides and Users matter on PasarGuard: an admin Karvan
	// merges into keeps its role, and its role's limits count its users.
	RoleID    int64
	Overrides *RoleLimits
	Users     int
}

// ExistingUser is a user already on the destination.
type ExistingUser struct {
	ID           int64
	Username     string
	AdminID      int64 // 0 when the user has no admin
	CreatedAt    time.Time
	SubRevokedAt *time.Time
}

// Row names one written row by what cannot change behind Karvan's back:
// its id, username and created_at (D33). A row only counts as Karvan's if
// all three still match.
type Row struct {
	ID        int64
	Username  string
	CreatedAt time.Time
}

// Index is what the destination already holds, read once per check. User
// keys follow Rules.UserKey, so lookups use the destination's collation.
type Index struct {
	Users       map[string]*ExistingUser // by Rules.UserKey(username)
	UsersByID   map[int64]*ExistingUser
	Credentials map[CredentialKey]int64 // owner user id
}

// NewIndex returns an empty index.
func NewIndex() *Index {
	return &Index{
		Users:       map[string]*ExistingUser{},
		UsersByID:   map[int64]*ExistingUser{},
		Credentials: map[CredentialKey]int64{},
	}
}

// Add records u under key.
func (x *Index) Add(key string, u *ExistingUser) {
	x.Users[key] = u
	x.UsersByID[u.ID] = u
}

// AddCredentials records every credential in c as owned by userID. The
// first owner wins, so a destination that already has duplicates keeps
// pointing at the oldest row.
func (x *Index) AddCredentials(c Credentials, userID int64) {
	for _, k := range c.Keys() {
		if _, ok := x.Credentials[k]; !ok {
			x.Credentials[k] = userID
		}
	}
}

// CredentialKey identifies one credential inside one protocol. Two users
// clash only when they share a value in the same protocol, because each
// protocol has its own inbounds. It holds the secret, so it never leaves
// memory and is never logged.
type CredentialKey string

// LogValue keeps keys out of logs.
func (CredentialKey) LogValue() slog.Value { return slog.StringValue("[redacted]") }

// Keys lists the credential keys of every protocol present in c.
func (c Credentials) Keys() []CredentialKey {
	var out []CredentialKey
	add := func(proto, v string) {
		if v != "" {
			out = append(out, CredentialKey(proto+":"+v))
		}
	}
	if c.VMess != nil {
		add("vmess", string(c.VMess.ID))
	}
	if c.VLESS != nil {
		add("vless", string(c.VLESS.ID))
	}
	if c.Trojan != nil {
		add("trojan", c.Trojan.Password.Reveal())
	}
	if c.Shadowsocks != nil {
		add("shadowsocks", c.Shadowsocks.Password.Reveal())
	}
	if c.Hysteria != nil {
		add("hysteria", c.Hysteria.Auth.Reveal())
	}
	if c.WireGuard != nil {
		add("wireguard", c.WireGuard.PublicKey)
	}
	return out
}
