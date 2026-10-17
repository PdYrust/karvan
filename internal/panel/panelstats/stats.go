// Package panelstats reads analytics from a panel's own tables, read only:
// users by status, who is online, traffic, admins and nodes. Marzban and
// PasarGuard share these tables; each adapter describes the few places
// where they differ with a Schema.
package panelstats

import (
	"context"
	"fmt"
	"time"

	"github.com/PdYrust/karvan/internal/panel/paneldb"
)

// Schema is what differs between the panels for these queries.
type Schema struct {
	// ExpireUnix: users.expire holds unix seconds (Marzban), not a datetime.
	ExpireUnix bool
	// Updates: subscription apps come from user_subscription_updates
	// (PasarGuard); otherwise from each user's last user agent.
	Updates bool
}

// OnlineWindow is how recently a user must have connected to count as
// online now, as PasarGuard counts it (app/db/crud/user.py).
const OnlineWindow = 2 * time.Minute

// Data is a panel's users, admins and nodes at one moment.
type Data struct {
	Users        int64            `json:"users"`
	ByStatus     map[string]int64 `json:"byStatus"`
	OnlineNow    int64            `json:"onlineNow"`
	OnlineDay    int64            `json:"onlineDay"`
	NewDay       int64            `json:"newDay"`
	NewWeek      int64            `json:"newWeek"`
	ExpiringWeek int64            `json:"expiringWeek"`
	UsedTraffic  int64            `json:"usedTraffic"`
	Admins       int64            `json:"admins"`
	TopAdmins    []Admin          `json:"topAdmins"`
	Nodes        []Node           `json:"nodes"`
	Uplink       int64            `json:"uplink"` // the panel's own totals
	Downlink     int64            `json:"downlink"`
}

// Admin is one admin (a reseller) with the users it owns.
type Admin struct {
	Username    string `json:"username"`
	Users       int64  `json:"users"`
	Active      int64  `json:"active"`
	UsedTraffic int64  `json:"usedTraffic"`
}

// Node is one of the panel's nodes.
type Node struct {
	Name     string `json:"name"`
	Status   string `json:"status"`
	Uplink   int64  `json:"uplink"`
	Downlink int64  `json:"downlink"`
}

const topAdmins = 10
const maxNodes = 50

// Read takes the panel's numbers at now.
func Read(ctx context.Context, q paneldb.Querier, d paneldb.Dialect, s Schema, now time.Time) (Data, error) {
	out := Data{ByStatus: map[string]int64{}, TopAdmins: []Admin{}, Nodes: []Node{}}
	for _, step := range []func() error{
		func() error { return statuses(ctx, q, &out) },
		func() error { return activity(ctx, q, d, now, &out) },
		func() error { return expiring(ctx, q, d, s, now, &out) },
		func() error { return admins(ctx, q, &out) },
		func() error { return nodes(ctx, q, &out) },
		func() error {
			// SYSTEM is a reserved word in MySQL 8.
			return q.QueryRowContext(ctx, "SELECT COALESCE(SUM(uplink), 0), COALESCE(SUM(downlink), 0) FROM "+paneldb.Ident(d, "system")).
				Scan(&out.Uplink, &out.Downlink)
		},
	} {
		if err := step(); err != nil {
			return out, err
		}
	}
	return out, nil
}

func statuses(ctx context.Context, q paneldb.Querier, out *Data) error {
	rows, err := q.QueryContext(ctx, "SELECT status, COUNT(*), COALESCE(SUM(used_traffic), 0) FROM users GROUP BY status")
	if err != nil {
		return fmt.Errorf("count users: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var status string
		var n, traffic int64
		if err := rows.Scan(&status, &n, &traffic); err != nil {
			return fmt.Errorf("count users: %w", err)
		}
		out.ByStatus[status] = n
		out.Users += n
		out.UsedTraffic += traffic
	}
	return rows.Err()
}

func activity(ctx context.Context, q paneldb.Querier, d paneldb.Dialect, now time.Time, out *Data) error {
	t := func(ago time.Duration) any { return paneldb.TimeArg(d, now.Add(-ago)) }
	err := q.QueryRowContext(ctx, `SELECT
		COALESCE(SUM(CASE WHEN online_at >= ? THEN 1 ELSE 0 END), 0),
		COALESCE(SUM(CASE WHEN online_at >= ? THEN 1 ELSE 0 END), 0),
		COALESCE(SUM(CASE WHEN created_at >= ? THEN 1 ELSE 0 END), 0),
		COALESCE(SUM(CASE WHEN created_at >= ? THEN 1 ELSE 0 END), 0)
		FROM users`, t(OnlineWindow), t(24*time.Hour), t(24*time.Hour), t(7*24*time.Hour)).
		Scan(&out.OnlineNow, &out.OnlineDay, &out.NewDay, &out.NewWeek)
	if err != nil {
		return fmt.Errorf("count active users: %w", err)
	}
	return nil
}

func expiring(ctx context.Context, q paneldb.Querier, d paneldb.Dialect, s Schema, now time.Time, out *Data) error {
	from, to := paneldb.TimeArg(d, now), paneldb.TimeArg(d, now.Add(7*24*time.Hour))
	if s.ExpireUnix {
		from, to = now.Unix(), now.Add(7*24*time.Hour).Unix()
	}
	// The status is a literal so PostgreSQL compares it with its enum type.
	err := q.QueryRowContext(ctx, "SELECT COUNT(*) FROM users WHERE status = 'active' AND expire > ? AND expire <= ?", from, to).Scan(&out.ExpiringWeek)
	if err != nil {
		return fmt.Errorf("count expiring users: %w", err)
	}
	return nil
}

func admins(ctx context.Context, q paneldb.Querier, out *Data) error {
	if err := q.QueryRowContext(ctx, "SELECT COUNT(*) FROM admins").Scan(&out.Admins); err != nil {
		return fmt.Errorf("count admins: %w", err)
	}
	rows, err := q.QueryContext(ctx, `SELECT a.username, COUNT(u.id),
		COALESCE(SUM(CASE WHEN u.status = 'active' THEN 1 ELSE 0 END), 0), COALESCE(SUM(u.used_traffic), 0)
		FROM admins a LEFT JOIN users u ON u.admin_id = a.id
		GROUP BY a.id, a.username ORDER BY 2 DESC, a.username LIMIT ?`, topAdmins)
	if err != nil {
		return fmt.Errorf("rank admins: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var a Admin
		if err := rows.Scan(&a.Username, &a.Users, &a.Active, &a.UsedTraffic); err != nil {
			return fmt.Errorf("rank admins: %w", err)
		}
		out.TopAdmins = append(out.TopAdmins, a)
	}
	return rows.Err()
}

func nodes(ctx context.Context, q paneldb.Querier, out *Data) error {
	rows, err := q.QueryContext(ctx, `SELECT COALESCE(name, address), status, COALESCE(uplink, 0), COALESCE(downlink, 0)
		FROM nodes ORDER BY 1 LIMIT ?`, maxNodes)
	if err != nil {
		return fmt.Errorf("list nodes: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var n Node
		if err := rows.Scan(&n.Name, &n.Status, &n.Uplink, &n.Downlink); err != nil {
			return fmt.Errorf("list nodes: %w", err)
		}
		out.Nodes = append(out.Nodes, n)
	}
	return rows.Err()
}
