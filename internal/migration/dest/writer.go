package dest

import (
	"context"
	"slices"
	"strconv"
	"time"

	"github.com/PdYrust/karvan/internal/migration/engine"
	"github.com/PdYrust/karvan/internal/panel"
	"github.com/PdYrust/karvan/internal/panel/marzbandb"
	"github.com/PdYrust/karvan/internal/panel/paneldb"
	"github.com/PdYrust/karvan/internal/panel/pasarguarddb"
)

// UsageLog is the one usage log row a moved user gets (D19).
type UsageLog struct {
	Traffic int64
	ResetAt time.Time
}

// ReadBack is a written user as the destination holds it.
type ReadBack struct {
	panel.Row
	AdminID int64
	Keys    []panel.CredentialKey // sorted
	Targets []string              // sorted
}

// Writer hides which panel the destination is.
type Writer interface {
	Prepare(ctx context.Context, x paneldb.Execer) error
	InsertAdmin(ctx context.Context, x paneldb.Execer, a panel.Admin, hash string, roleID int64, now time.Time) (int64, error)
	InsertUser(ctx context.Context, x paneldb.Execer, it *engine.Item, adminID int64, log *UsageLog, now time.Time) (int64, error)
	ReadBack(ctx context.Context, q paneldb.Querier, ids []int64) (map[int64]ReadBack, error)
	RemoveUsers(ctx context.Context, x paneldb.Execer, rows []panel.Row) (removed []int64, skipped []panel.Row, err error)
	RemoveAdmin(ctx context.Context, x paneldb.Execer, a panel.Row) (bool, error)
}

// NewWriter returns the writer for a destination kind.
func NewWriter(kind panel.Kind, d paneldb.Dialect, facts engine.Dest) Writer {
	if kind == panel.Marzban {
		return marzbanWriter{w: marzbandb.NewWriter(d, facts.Targets), d: d}
	}
	return pasarguardWriter{w: pasarguarddb.Writer{Dialect: d}, d: d}
}

type marzbanWriter struct {
	w marzbandb.Writer
	d paneldb.Dialect
}

func (m marzbanWriter) Prepare(ctx context.Context, x paneldb.Execer) error {
	return m.w.EnsureInbounds(ctx, x)
}

func (m marzbanWriter) InsertAdmin(ctx context.Context, x paneldb.Execer, a panel.Admin, hash string, _ int64, now time.Time) (int64, error) {
	return m.w.InsertAdmin(ctx, x, a, hash, now)
}

func (m marzbanWriter) InsertUser(ctx context.Context, x paneldb.Execer, it *engine.Item, adminID int64, l *UsageLog, now time.Time) (int64, error) {
	row := marzbandb.UserRow{User: it.User, AdminID: adminID, Targets: it.Targets}
	if l != nil {
		row.UsageLog = &marzbandb.UsageLog{Traffic: l.Traffic, ResetAt: l.ResetAt}
	}
	return m.w.InsertUser(ctx, x, row, now)
}

func (m marzbanWriter) ReadBack(ctx context.Context, q paneldb.Querier, ids []int64) (map[int64]ReadBack, error) {
	got, err := marzbandb.ReadUsers(ctx, q, m.d, ids)
	if err != nil {
		return nil, err
	}
	out := map[int64]ReadBack{}
	for id, w := range got {
		var targets []string
		for proto, tags := range m.w.Inbounds {
			for _, tag := range tags {
				if marzbanHas(w.Credentials, proto) && !slices.Contains(w.Excluded[proto], tag) {
					targets = append(targets, proto+"/"+tag)
				}
			}
		}
		out[id] = readBack(w.Row, w.AdminID, w.Credentials, targets)
	}
	return out, nil
}

func marzbanHas(c panel.Credentials, proto string) bool {
	switch proto {
	case "vmess":
		return c.VMess != nil
	case "vless":
		return c.VLESS != nil
	case "trojan":
		return c.Trojan != nil
	}
	return c.Shadowsocks != nil
}

func (m marzbanWriter) RemoveUsers(ctx context.Context, x paneldb.Execer, rows []panel.Row) ([]int64, []panel.Row, error) {
	return marzbandb.RemoveUsers(ctx, x, m.d, rows)
}

func (m marzbanWriter) RemoveAdmin(ctx context.Context, x paneldb.Execer, a panel.Row) (bool, error) {
	return marzbandb.RemoveAdmin(ctx, x, m.d, a)
}

type pasarguardWriter struct {
	w pasarguarddb.Writer
	d paneldb.Dialect
}

func (pasarguardWriter) Prepare(context.Context, paneldb.Execer) error { return nil }

func (p pasarguardWriter) InsertAdmin(ctx context.Context, x paneldb.Execer, a panel.Admin, hash string, roleID int64, now time.Time) (int64, error) {
	if roleID == 0 {
		roleID = pasarguarddb.OperatorRole
	}
	return p.w.InsertAdmin(ctx, x, a, hash, roleID, now)
}

func (p pasarguardWriter) InsertUser(ctx context.Context, x paneldb.Execer, it *engine.Item, adminID int64, l *UsageLog, _ time.Time) (int64, error) {
	groups, err := pasarguarddb.GroupIDs(it.Targets)
	if err != nil {
		return 0, err
	}
	row := pasarguarddb.UserRow{User: it.User, AdminID: adminID, GroupIDs: groups}
	if l != nil {
		row.UsageLog = &pasarguarddb.UsageLog{Traffic: l.Traffic, ResetAt: l.ResetAt}
	}
	return p.w.InsertUser(ctx, x, row)
}

func (p pasarguardWriter) ReadBack(ctx context.Context, q paneldb.Querier, ids []int64) (map[int64]ReadBack, error) {
	got, err := pasarguarddb.ReadUsers(ctx, q, ids)
	if err != nil {
		return nil, err
	}
	out := map[int64]ReadBack{}
	for id, w := range got {
		var targets []string
		for _, g := range w.GroupIDs {
			targets = append(targets, strconv.FormatInt(g, 10))
		}
		out[id] = readBack(w.Row, w.AdminID, w.Credentials, targets)
	}
	return out, nil
}

func (p pasarguardWriter) RemoveUsers(ctx context.Context, x paneldb.Execer, rows []panel.Row) ([]int64, []panel.Row, error) {
	return pasarguarddb.RemoveUsers(ctx, x, p.d, rows)
}

func (p pasarguardWriter) RemoveAdmin(ctx context.Context, x paneldb.Execer, a panel.Row) (bool, error) {
	return pasarguarddb.RemoveAdmin(ctx, x, p.d, a)
}

func readBack(r panel.Row, admin int64, c panel.Credentials, targets []string) ReadBack {
	keys := c.Keys()
	slices.Sort(keys)
	slices.Sort(targets)
	return ReadBack{Row: r, AdminID: admin, Keys: keys, Targets: targets}
}
