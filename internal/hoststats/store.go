package hoststats

import (
	"context"
	"database/sql"
	"fmt"
)

// Minute is the server's average over one minute, with the peaks that an
// average would hide. In a range it is the same over a longer bucket.
type Minute struct {
	TS        int64   `json:"t"` // unix seconds at the start
	CPU       float64 `json:"cpu"`
	CPUMax    float64 `json:"cpuMax"`
	Load1     float64 `json:"load1"`
	MemUsed   int64   `json:"memUsed"`
	MemCache  int64   `json:"memCache"`
	MemTotal  int64   `json:"memTotal"`
	SwapUsed  int64   `json:"swapUsed"`
	NetRx     float64 `json:"netRx"`
	NetTx     float64 `json:"netTx"`
	NetRxMax  float64 `json:"netRxMax"`
	NetTxMax  float64 `json:"netTxMax"`
	DiskRead  float64 `json:"diskRead"`
	DiskWrite float64 `json:"diskWrite"`
	TCP       int64   `json:"tcp"`
	Procs     int64   `json:"procs"`
	// Minutes is how many saved minutes a bucket holds; a bucket the
	// server was down for part of holds fewer than its width.
	Minutes int64 `json:"minutes"`
}

// Store keeps minutes in Karvan's own database.
type Store struct{ db *sql.DB }

// NewStore returns a Store over Karvan's database.
func NewStore(db *sql.DB) *Store { return &Store{db: db} }

// Put saves one minute; a minute written twice keeps the last values.
func (s *Store) Put(ctx context.Context, m Minute) error {
	_, err := s.db.ExecContext(ctx, `INSERT OR REPLACE INTO metrics_host
		(ts, cpu, cpu_max, load1, mem_used, mem_cache, mem_total, swap_used, net_rx, net_tx, net_rx_max, net_tx_max, disk_read, disk_write, tcp, procs)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		m.TS, m.CPU, m.CPUMax, m.Load1, m.MemUsed, m.MemCache, m.MemTotal, m.SwapUsed,
		m.NetRx, m.NetTx, m.NetRxMax, m.NetTxMax, m.DiskRead, m.DiskWrite, m.TCP, m.Procs)
	if err != nil {
		return fmt.Errorf("save server minute: %w", err)
	}
	return nil
}

// Range returns minutes from (inclusive) to to (exclusive), averaged into
// buckets of bucket seconds; peaks stay peaks.
func (s *Store) Range(ctx context.Context, from, to, bucket int64) ([]Minute, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT (ts / ?1) * ?1 AS b,
		AVG(cpu), MAX(cpu_max), AVG(load1), CAST(AVG(mem_used) AS INTEGER), CAST(AVG(mem_cache) AS INTEGER),
		MAX(mem_total), CAST(AVG(swap_used) AS INTEGER), AVG(net_rx), AVG(net_tx), MAX(net_rx_max), MAX(net_tx_max),
		AVG(disk_read), AVG(disk_write), CAST(AVG(tcp) AS INTEGER), CAST(AVG(procs) AS INTEGER), COUNT(*)
		FROM metrics_host WHERE ts >= ?2 AND ts < ?3 GROUP BY b ORDER BY b`, bucket, from, to)
	if err != nil {
		return nil, fmt.Errorf("read server minutes: %w", err)
	}
	defer rows.Close()
	out := []Minute{}
	for rows.Next() {
		var m Minute
		if err := rows.Scan(&m.TS, &m.CPU, &m.CPUMax, &m.Load1, &m.MemUsed, &m.MemCache, &m.MemTotal, &m.SwapUsed,
			&m.NetRx, &m.NetTx, &m.NetRxMax, &m.NetTxMax, &m.DiskRead, &m.DiskWrite, &m.TCP, &m.Procs, &m.Minutes); err != nil {
			return nil, fmt.Errorf("read server minutes: %w", err)
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// Trim removes minutes before the given time and says how many went.
func (s *Store) Trim(ctx context.Context, before int64) (int64, error) {
	res, err := s.db.ExecContext(ctx, "DELETE FROM metrics_host WHERE ts < ?", before)
	if err != nil {
		return 0, fmt.Errorf("trim server minutes: %w", err)
	}
	return res.RowsAffected()
}
