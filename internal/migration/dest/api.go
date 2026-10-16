package dest

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/PdYrust/karvan/internal/panel"
	"github.com/PdYrust/karvan/internal/panel/paneldb"
	"github.com/PdYrust/karvan/internal/panel/subtoken"
	"github.com/PdYrust/karvan/internal/secret"
)

// ErrNoFullAdmin means the panel has no admin with full rights Karvan can
// act as, so only the manual reload is possible.
var ErrNoFullAdmin = errors.New("the panel has no full admin: no Marzban sudo admin, PasarGuard owner or SUDO_USERNAME")

// API is how Karvan calls the destination panel's own API on this server,
// for the reload step only (D46). The panel signs its tokens with the DB
// jwt secret, so Karvan mints a short token for a full admin instead of
// asking for a password (owner decision Q6).
type API struct {
	Local
	Kind    panel.Kind
	Admin   string // the full admin the token is for
	AdminID int64  // 0 for the ENV sudoer
	secret  secret.Value
}

// tokenTTL is short (D47): one call never takes this long, and Token
// mints a fresh one per call.
const tokenTTL = 5 * time.Minute

// Token mints an admin token valid from now.
func (a API) Token(now time.Time) string {
	c := subtoken.AdminClaims{Sub: a.Admin, Access: "admin", AdminID: a.AdminID}
	if a.Kind == panel.Marzban {
		c.Access = "sudo" // an ENV sudoer needs it; a DB sudo admin accepts it
		c.AdminID = 0
	}
	return subtoken.AdminToken(c, a.secret.Reveal(), now, tokenTTL)
}

// API reads what calling the panel needs. It fails with ErrNoFullAdmin
// when there is nobody to act as.
func (p *Panel) API(ctx context.Context) (API, error) {
	s := p.Found.Settings
	a := API{Kind: p.Found.Kind, Local: p.Local()}
	r, err := paneldb.OpenReader(ctx, p.Target)
	if err != nil {
		return a, err
	}
	defer r.Close()
	err = r.View(ctx, func(q paneldb.Querier) error {
		var key string
		if err := q.QueryRowContext(ctx, "SELECT secret_key FROM jwt ORDER BY id LIMIT 1").Scan(&key); err != nil {
			return fmt.Errorf("read the panel's jwt secret: %w", err)
		}
		a.secret = secret.NewValue(key)
		a.Admin, a.AdminID, err = fullAdmin(ctx, q, a.Kind)
		return err
	})
	if errors.Is(err, sql.ErrNoRows) {
		if s.EnvSudoer == "" {
			return a, ErrNoFullAdmin
		}
		a.Admin, a.AdminID, err = s.EnvSudoer, 0, nil
	}
	return a, err
}

// fullAdmin picks the oldest admin with full rights: Marzban is_sudo,
// PasarGuard an active admin with an owner role.
func fullAdmin(ctx context.Context, q paneldb.Querier, kind panel.Kind) (string, int64, error) {
	query := "SELECT username, id FROM admins WHERE is_sudo = ? ORDER BY id LIMIT 1"
	args := []any{true}
	if kind == panel.PasarGuard {
		query = "SELECT a.username, a.id FROM admins a JOIN admin_roles r ON r.id = a.role_id " +
			"WHERE r.is_owner = ? AND a.status <> 'disabled' ORDER BY a.id LIMIT 1"
	}
	var name string
	var id int64
	err := q.QueryRowContext(ctx, query, args...).Scan(&name, &id)
	return name, id, err
}
