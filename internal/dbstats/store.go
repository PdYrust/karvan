package dbstats

import (
	"context"
	"database/sql"
	"fmt"
)

// Minute is one panel database over one minute, or a longer bucket in a
// range. CacheHit is nil where the engine has no buffer cache to ask.
type Minute struct {
	TS          int64    `json:"t"`
	Size        int64    `json:"size"`
	Connections int64    `json:"connections"`
	QPS         float64  `json:"qps"`
	CacheHit    *float64 `json:"cacheHit"`
	Users       int64    `json:"users"`
	Active      int64    `json:"active"`
	Online      int64    `json:"online"`
	UsedTraffic int64    `json:"usedTraffic"`
}

// Store keeps panel database minutes in Karvan's own database.
type Store struct{ db *sql.DB }

// NewStore returns a Store over Karvan's database.
func NewStore(db *sql.DB) *Store { return &Store{db: db} }

// Put saves one minute of one panel.
func (s *Store) Put(ctx context.Context, panelID int64, m Minute) error {
	_, err := s.db.ExecContext(ctx, `INSERT OR REPLACE INTO metrics_db
		(panel_id, ts, size, connections, qps, cache_hit, users, active, online, used_traffic) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		panelID, m.TS, m.Size, m.Connections, m.QPS, m.CacheHit, m.Users, m.Active, m.Online, m.UsedTraffic)
	if err != nil {
		return fmt.Errorf("save database minute: %w", err)
	}
	return nil
}

// Range returns one panel's minutes from (inclusive) to to (exclusive),
// averaged into buckets of bucket seconds. Counts are the bucket's last.
func (s *Store) Range(ctx context.Context, panelID, from, to, bucket int64) ([]Minute, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT (ts / ?1) * ?1 AS b, MAX(size), CAST(AVG(connections) AS INTEGER), AVG(qps), AVG(cache_hit),
		MAX(users), MAX(active), MAX(online), MAX(used_traffic)
		FROM metrics_db WHERE panel_id = ?2 AND ts >= ?3 AND ts < ?4 GROUP BY b ORDER BY b`, bucket, panelID, from, to)
	if err != nil {
		return nil, fmt.Errorf("read database minutes: %w", err)
	}
	defer rows.Close()
	out := []Minute{}
	for rows.Next() {
		var m Minute
		var hit sql.NullFloat64
		if err := rows.Scan(&m.TS, &m.Size, &m.Connections, &m.QPS, &hit, &m.Users, &m.Active, &m.Online, &m.UsedTraffic); err != nil {
			return nil, fmt.Errorf("read database minutes: %w", err)
		}
		if hit.Valid {
			m.CacheHit = &hit.Float64
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// Trim removes minutes before the given time.
func (s *Store) Trim(ctx context.Context, before int64) error {
	if _, err := s.db.ExecContext(ctx, "DELETE FROM metrics_db WHERE ts < ?", before); err != nil {
		return fmt.Errorf("trim database minutes: %w", err)
	}
	return nil
}
