package bridge

import (
	"context"
	"strconv"
	"time"

	"github.com/PdYrust/karvan/internal/migration/ledger"
	"github.com/PdYrust/karvan/internal/panel"
)

// Reasons a user's old links close for good.
const (
	ClosedDeleted = "deleted" // the user is gone from the new panel, or its row is another user now
	ClosedRevoked = "revoked" // the new panel revoked the user's link
)

// verdict is the destination check's answer (D58).
type verdict struct {
	ok    bool
	close string // set when the user's old links must close for good
}

// check runs the destination check for ledger row e and a token issued
// at issued (zero: no token, as in the sweep).
func (b *Bridge) check(ctx context.Context, d Destination, e ledger.Entry, issued time.Time) (verdict, error) {
	if closed, err := b.st.closed(ctx, e.ID); err != nil || closed {
		return verdict{}, err
	}
	rows, err := d.Rows(ctx, []int64{e.Row.ID})
	if err != nil {
		return verdict{}, err
	}
	r, found := rows[e.Row.ID]
	return judge(e, r, found, issued), nil
}

// judge applies the destination's own token rules, plus the owner rule:
// a revoke or delete on the new panel closes the old links too.
func judge(e ledger.Entry, r Row, found bool, issued time.Time) verdict {
	switch {
	case !found || r.Username != e.Username || !r.CreatedAt.Equal(e.CreatedAt):
		return verdict{close: ClosedDeleted}
	case r.SubRevokedAt != nil && r.SubRevokedAt.After(e.Written):
		return verdict{close: ClosedRevoked}
	case issued.IsZero():
		return verdict{ok: true}
	case r.CreatedAt.After(issued), r.SubRevokedAt != nil && r.SubRevokedAt.After(issued):
		return verdict{}
	}
	return verdict{ok: true}
}

// apply closes the user when the verdict says so, and reports whether
// it did.
func (b *Bridge) apply(ctx context.Context, e ledger.Entry, v verdict) bool {
	if v.close == "" {
		return false
	}
	closed, err := b.st.close(ctx, e.ID, v.close)
	if err != nil {
		b.Log.Warn("bridge: close user", "ledger", e.ID, "err", err)
		return false
	}
	if closed {
		b.Log.Info("bridge: old links closed", "ledger", e.ID, "user", e.Username, "reason", v.close)
	}
	return closed
}

// sameUser reports whether the old panel's answer is the user the ledger
// recorded: same name (and id), same creation time, same credentials. A
// new user reusing a name or id on the old panel fails here (D60).
func sameUser(kind panel.Kind, u panel.User, e ledger.Entry) bool {
	if kind == panel.PasarGuard && strconv.FormatInt(u.SourceID, 10) != e.SourceKey {
		return false
	}
	return u.Username == e.SourceUsername && u.CreatedAt.Equal(e.SourceCreated) &&
		ledger.Fingerprint(u.Credentials) == e.CredPrint
}
