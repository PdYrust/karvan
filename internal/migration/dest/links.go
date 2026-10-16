package dest

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/PdYrust/karvan/internal/panel"
	"github.com/PdYrust/karvan/internal/panel/paneldb"
	"github.com/PdYrust/karvan/internal/panel/subtoken"
	"github.com/PdYrust/karvan/internal/secret"
)

// Links mints new subscription links the way the destination panel does.
// It holds the panel's jwt secret in memory only.
type Links struct {
	kind   panel.Kind
	secret secret.Value
	prefix string // empty when the panel takes it from the request
	path   string
}

// Links reads what minting needs: the jwt secret and the URL prefix
// (Marzban: XRAY_SUBSCRIPTION_URL_PREFIX; PasarGuard: the admin's
// sub_domain, else settings.subscription.url_prefix).
func (p *Panel) Links(ctx context.Context, adminID int64) (Links, error) {
	r, err := paneldb.OpenReader(ctx, p.Target)
	if err != nil {
		return Links{}, err
	}
	defer r.Close()
	l := Links{kind: p.Found.Kind, path: p.Found.Settings.SubPath, prefix: p.Found.Settings.SubURLPrefix}
	err = r.View(ctx, func(q paneldb.Querier) error {
		var key string
		if err := q.QueryRowContext(ctx, "SELECT secret_key FROM jwt ORDER BY id LIMIT 1").Scan(&key); err != nil {
			return fmt.Errorf("read the panel's jwt secret: %w", err)
		}
		l.secret = secret.NewValue(key)
		if l.kind == panel.PasarGuard {
			l.prefix, err = pasarguardPrefix(ctx, q, adminID)
		}
		return err
	})
	return l, err
}

func pasarguardPrefix(ctx context.Context, q paneldb.Querier, adminID int64) (string, error) {
	var domain sql.NullString
	err := q.QueryRowContext(ctx, "SELECT sub_domain FROM admins WHERE id = ?", adminID).Scan(&domain)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return "", err
	}
	if domain.String != "" {
		return domain.String, nil
	}
	var raw []byte
	if err := q.QueryRowContext(ctx, "SELECT subscription FROM settings ORDER BY id LIMIT 1").Scan(&raw); err != nil && !errors.Is(err, sql.ErrNoRows) {
		return "", err
	}
	var s struct {
		URLPrefix string `json:"url_prefix"`
	}
	_ = json.Unmarshal(raw, &s)
	return s.URLPrefix, nil
}

// Link is a fresh link for a written user (id for PasarGuard, username for
// Marzban). A "*" in the prefix gets a random hex salt, like the panels.
func (l Links) Link(id int64, username string, now time.Time) string {
	var token string
	if l.kind == panel.PasarGuard {
		token = subtoken.PasarGuard(id, l.secret.Reveal(), now)
	} else {
		token = subtoken.Marzban(username, l.secret.Reveal(), now)
	}
	prefix := l.prefix
	if strings.Contains(prefix, "*") {
		b := make([]byte, 8)
		_, _ = rand.Read(b)
		prefix = strings.ReplaceAll(prefix, "*", hex.EncodeToString(b))
	}
	return strings.TrimRight(prefix, "/") + "/" + l.path + "/" + token
}

// Token mints a destination token for a user, for the bridge.
func (l Links) Token(id int64, username string, now time.Time) string {
	if l.kind == panel.PasarGuard {
		return subtoken.PasarGuard(id, l.secret.Reveal(), now)
	}
	return subtoken.Marzban(username, l.secret.Reveal(), now)
}
