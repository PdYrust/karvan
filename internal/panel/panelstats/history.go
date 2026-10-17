package panelstats

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/PdYrust/karvan/internal/panel/paneldb"
)

// Hour is the traffic of one hour through all of the panel's nodes.
type Hour struct {
	At       time.Time `json:"at"`
	Uplink   int64     `json:"uplink"`
	Downlink int64     `json:"downlink"`
}

// Traffic reads the panel's own hourly usage table (node_usages) since a
// time. Both panels write one row per node and hour.
func Traffic(ctx context.Context, q paneldb.Querier, d paneldb.Dialect, since time.Time) ([]Hour, error) {
	rows, err := q.QueryContext(ctx, `SELECT created_at, COALESCE(SUM(uplink), 0), COALESCE(SUM(downlink), 0)
		FROM node_usages WHERE created_at >= ? GROUP BY created_at ORDER BY created_at`, paneldb.TimeArg(d, since))
	if err != nil {
		return nil, fmt.Errorf("read traffic: %w", err)
	}
	defer rows.Close()
	out := []Hour{}
	for rows.Next() {
		var at paneldb.NullTime
		var h Hour
		if err := rows.Scan(&at, &h.Uplink, &h.Downlink); err != nil {
			return nil, fmt.Errorf("read traffic: %w", err)
		}
		h.At = at.Time
		out = append(out, h)
	}
	return out, rows.Err()
}

// maxCreated caps how many sign-up times one request reads.
const maxCreated = 200_000

// NewUsers counts users created per hour since a time.
func NewUsers(ctx context.Context, q paneldb.Querier, d paneldb.Dialect, since time.Time) ([]Count, error) {
	rows, err := q.QueryContext(ctx, "SELECT created_at FROM users WHERE created_at >= ? LIMIT ?", paneldb.TimeArg(d, since), maxCreated)
	if err != nil {
		return nil, fmt.Errorf("read sign-ups: %w", err)
	}
	defer rows.Close()
	perHour := map[int64]int64{}
	for rows.Next() {
		var at paneldb.NullTime
		if err := rows.Scan(&at); err != nil {
			return nil, fmt.Errorf("read sign-ups: %w", err)
		}
		perHour[at.Time.Unix()/3600*3600]++
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read sign-ups: %w", err)
	}
	out := make([]Count, 0, len(perHour))
	for h, n := range perHour {
		out = append(out, Count{At: time.Unix(h, 0).UTC(), N: n})
	}
	slices.SortFunc(out, func(a, b Count) int { return a.At.Compare(b.At) })
	return out, nil
}

// Count is a number of events at a time.
type Count struct {
	At time.Time `json:"at"`
	N  int64     `json:"n"`
}

// Client is a subscription app and how many updates (or users) use it.
type Client struct {
	Name  string `json:"name"`
	Count int64  `json:"count"`
}

const topClients = 6

// Clients ranks the apps that fetch subscriptions since a time, as their
// user agents name them.
func Clients(ctx context.Context, q paneldb.Querier, d paneldb.Dialect, s Schema, since time.Time) ([]Client, error) {
	query := `SELECT sub_last_user_agent, COUNT(*) FROM users WHERE sub_updated_at >= ? AND sub_last_user_agent IS NOT NULL
		GROUP BY sub_last_user_agent ORDER BY 2 DESC LIMIT 500`
	if s.Updates {
		query = "SELECT user_agent, COUNT(*) FROM user_subscription_updates WHERE created_at >= ? GROUP BY user_agent ORDER BY 2 DESC LIMIT 500"
	}
	rows, err := q.QueryContext(ctx, query, paneldb.TimeArg(d, since))
	if err != nil {
		return nil, fmt.Errorf("read subscription apps: %w", err)
	}
	defer rows.Close()
	byName := map[string]int64{}
	for rows.Next() {
		var ua string
		var n int64
		if err := rows.Scan(&ua, &n); err != nil {
			return nil, fmt.Errorf("read subscription apps: %w", err)
		}
		byName[ClientName(ua)] += n
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read subscription apps: %w", err)
	}
	return top(byName), nil
}

// top keeps the largest apps and folds the rest into "Other".
func top(byName map[string]int64) []Client {
	out := make([]Client, 0, len(byName))
	for name, n := range byName {
		out = append(out, Client{Name: name, Count: n})
	}
	slices.SortFunc(out, func(a, b Client) int {
		if a.Count != b.Count {
			return int(b.Count - a.Count)
		}
		return strings.Compare(a.Name, b.Name)
	})
	if len(out) <= topClients {
		return out
	}
	rest := Client{Name: OtherClient}
	for _, c := range out[topClients-1:] {
		rest.Count += c.Count
	}
	return append(out[:topClients-1], rest)
}

// OtherClient is the name for the apps outside the top few.
const OtherClient = "other"

// clientNames are the apps people use with both panels, by a word in
// their user agent. Order matters: v2rayNG before v2rayN.
var clientNames = [][2]string{
	{"v2rayng", "v2rayNG"}, {"v2rayn", "v2rayN"}, {"hiddify", "Hiddify"}, {"streisand", "Streisand"},
	{"nekobox", "NekoBox"}, {"nekoray", "NekoRay"}, {"shadowrocket", "Shadowrocket"}, {"v2box", "V2Box"},
	{"foxray", "FoXray"}, {"happ", "Happ"}, {"karing", "Karing"}, {"singbox", "sing-box"}, {"sing-box", "sing-box"},
	{"sfa", "sing-box"}, {"sfi", "sing-box"}, {"clash", "Clash"}, {"mihomo", "Clash"}, {"stash", "Stash"},
	{"outline", "Outline"}, {"loon", "Loon"}, {"quantumult", "Quantumult X"}, {"surge", "Surge"},
	{"mozilla", "Browser"}, {"curl", "curl"},
}

// ClientName names the app behind a user agent, or "other".
func ClientName(ua string) string {
	low := strings.ToLower(ua)
	for _, c := range clientNames {
		if strings.Contains(low, c[0]) {
			return c[1]
		}
	}
	return OtherClient
}
