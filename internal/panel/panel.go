package panel

import (
	"context"
	"iter"

	"github.com/PdYrust/karvan/internal/secret"
)

// Capabilities are the features a panel version has. Adapters report them
// from the detected version, never from guesses.
type Capabilities struct {
	APIKey      bool // PasarGuard X-Api-Key login (from c9b48df42f10)
	RBAC        bool // PasarGuard admin roles (from 66c38b8a687a)
	Groups      bool // users attach to groups, not per-protocol inbounds
	WireGuard   bool
	Hysteria    bool
	PerUserFlow bool // Marzban per-user VLESS/Trojan flow
	HWIDLimit   bool
}

// Info identifies a connected panel.
type Info struct {
	Kind         Kind
	Version      string // release version when known
	Revision     string // alembic revision; empty when read through the API only
	Capabilities Capabilities
}

// Source is a panel Karvan reads from through its API with a reseller's
// credentials. Implementations never write to the source.
type Source interface {
	Info(ctx context.Context) (Info, error)
	// CurrentAdmin is the admin whose credentials were given.
	CurrentAdmin(ctx context.Context) (Admin, error)
	// Users yields every user the admin can see, paged with an explicit
	// sort, because neither panel orders its lists by default.
	Users(ctx context.Context) iter.Seq2[User, error]
}

// Rules are the destination's validation rules, which Karvan must enforce
// itself because database writes skip the panel's own API checks.
type Rules struct {
	UsernamePattern       string // regular expression
	UsernameMaxLen        int
	UsernameCaseSensitive bool
	MinShadowsocksLen     int
	ShadowsocksMethods    []string
	Protocols             []string // protocols the destination can serve
}

// LookupQuery asks the destination which of these already exist.
type LookupQuery struct {
	AdminUsernames []string
	UserUsernames  []string
	UUIDs          []UUID
	Passwords      []secret.Value // trojan, shadowsocks and hysteria secrets
}

// LookupResult lists the values from LookupQuery that already exist, with
// the username that holds each credential. Secrets are reported by their
// index in LookupQuery.Passwords, so the result never repeats them.
type LookupResult struct {
	Admins        []string
	Users         []string
	UUIDOwners    map[UUID]string
	PasswordOwner map[int]string
}

// Destination is a panel Karvan writes into through its database.
type Destination interface {
	Info(ctx context.Context) (Info, error)
	Rules() Rules
	Targets(ctx context.Context) ([]Target, error)
	Lookup(ctx context.Context, q LookupQuery) (LookupResult, error)
	// Begin starts a transaction. Nothing reaches the destination outside one.
	Begin(ctx context.Context) (Tx, error)
}

// Tx is one all-or-nothing write to a destination.
type Tx interface {
	// CreateAdmin creates admin with a password hash in the destination's
	// format and returns its new id.
	CreateAdmin(ctx context.Context, admin Admin, passwordHash string) (int64, error)
	// CreateUser creates u under ownerID, attached to targets, and returns
	// its new id.
	CreateUser(ctx context.Context, u User, ownerID int64, targets []Target) (int64, error)
	Commit() error
	Rollback() error
}
