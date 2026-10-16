package bridge

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"time"

	"github.com/PdYrust/karvan/internal/migration/dest"
	"github.com/PdYrust/karvan/internal/panel/paneldb"
)

// Row is a destination user as the bridge checks it.
type Row struct {
	Username     string
	CreatedAt    time.Time
	SubRevokedAt *time.Time
}

// Destination is the new panel the moved users live on.
type Destination interface {
	// Key is the ledger's name for it (Target.String()).
	Key() string
	// Rows reads the current rows of ids; a missing id is a deleted user.
	Rows(ctx context.Context, ids []int64) (map[int64]Row, error)
	// Token mints the panel's own subscription token for a user.
	Token(id int64, username string, now time.Time) string
	SubPath() string
	// BaseURL and Client reach the panel's own listener.
	BaseURL() string
	Client() *http.Client
	Close()
}

// panelDest is a discovered panel opened read only.
type panelDest struct {
	key    string
	reader *paneldb.Reader
	links  dest.Links
	path   string
	local  dest.Local
	client *http.Client
}

// openPanel opens the panel whose ENV is at env.
func openPanel(ctx context.Context, env string) (Destination, error) {
	d, err := dest.Open(ctx, env)
	if err != nil {
		return nil, err
	}
	links, err := d.Links(ctx, 0)
	if err != nil {
		return nil, err
	}
	local := d.Local()
	t, err := local.Transport(16)
	if err != nil {
		return nil, err
	}
	r, err := paneldb.OpenReader(ctx, d.Target)
	if err != nil {
		return nil, err
	}
	client := &http.Client{Transport: t, Timeout: 30 * time.Second,
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	return &panelDest{key: d.Target.String(), reader: r, links: links, path: d.Found.Settings.SubPath, local: local, client: client}, nil
}

func (p *panelDest) Key() string          { return p.key }
func (p *panelDest) SubPath() string      { return p.path }
func (p *panelDest) BaseURL() string      { return p.local.BaseURL }
func (p *panelDest) Client() *http.Client { return p.client }
func (p *panelDest) Close()               { _ = p.reader.Close() }

func (p *panelDest) Token(id int64, username string, now time.Time) string {
	return p.links.Token(id, username, now)
}

// rowChunk keeps IN lists well under every server's parameter limit.
const rowChunk = 500

func (p *panelDest) Rows(ctx context.Context, ids []int64) (map[int64]Row, error) {
	out := make(map[int64]Row, len(ids))
	err := p.reader.View(ctx, func(q paneldb.Querier) error {
		for start := 0; start < len(ids); start += rowChunk {
			marks, args := paneldb.In(ids[start:min(start+rowChunk, len(ids))])
			rows, err := q.QueryContext(ctx, "SELECT id, username, created_at, sub_revoked_at FROM users WHERE id IN ("+marks+")", args...)
			if err != nil {
				return fmt.Errorf("read users: %w", err)
			}
			if err := scanRows(rows, out); err != nil {
				return err
			}
		}
		return nil
	})
	return out, err
}

func scanRows(rows *sql.Rows, out map[int64]Row) error {
	defer rows.Close()
	for rows.Next() {
		var id int64
		var r Row
		var created, revoked paneldb.NullTime
		if err := rows.Scan(&id, &r.Username, &created, &revoked); err != nil {
			return err
		}
		r.CreatedAt, r.SubRevokedAt = created.Time, revoked.Ptr()
		out[id] = r
	}
	return rows.Err()
}
