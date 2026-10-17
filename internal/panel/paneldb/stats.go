package paneldb

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"strconv"
)

// EngineStats is what a database engine says about itself. Counters are
// cumulative since the server started; callers turn two readings into rates.
type EngineStats struct {
	Flavor         string `json:"flavor"`
	Version        string `json:"version"`
	Size           int64  `json:"size"` // bytes on disk, data and indexes
	Connections    int64  `json:"connections"`
	MaxConnections int64  `json:"maxConnections"`
	Running        int64  `json:"running"` // statements running now
	Uptime         int64  `json:"uptime"`  // seconds the server has run
	Queries        int64  `json:"-"`
	CacheReads     int64  `json:"-"` // block requests
	CacheMisses    int64  `json:"-"` // of those, read from disk
	PageSize       int64  `json:"pageSize,omitempty"`
	Pages          int64  `json:"pages,omitempty"`
	FreePages      int64  `json:"freePages,omitempty"`
	WALSize        int64  `json:"walSize,omitempty"`
	JournalMode    string `json:"journalMode,omitempty"`
	Timescale      string `json:"timescale,omitempty"` // TimescaleDB version, if installed
}

// Size is how big the database is: the file and its WAL for SQLite, data
// and indexes for the servers.
func Size(ctx context.Context, q Querier, t Target) (int64, error) {
	switch t.Dialect {
	case SQLite:
		var total int64
		for _, p := range []string{t.Path, t.Path + "-wal"} {
			if info, err := os.Stat(p); err == nil {
				total += info.Size()
			}
		}
		return total, nil
	case MySQL:
		var n int64
		err := q.QueryRowContext(ctx, "SELECT COALESCE(SUM(data_length + index_length), 0) FROM information_schema.TABLES WHERE table_schema = DATABASE()").Scan(&n)
		return n, err
	}
	var n int64
	err := q.QueryRowContext(ctx, "SELECT pg_database_size(current_database())").Scan(&n)
	return n, err
}

// Engine reads the engine's own statistics. Everything here is readable
// by the panel's own database user; nothing needs extra privileges.
func Engine(ctx context.Context, q Querier, t Target, srv Server) (EngineStats, error) {
	s := EngineStats{Flavor: srv.Flavor, Version: srv.Version}
	var err error
	if s.Size, err = Size(ctx, q, t); err != nil {
		return s, fmt.Errorf("database size: %w", err)
	}
	switch t.Dialect {
	case SQLite:
		err = sqliteStats(ctx, q, t, &s)
	case MySQL:
		err = mysqlStats(ctx, q, &s)
	default:
		err = postgresStats(ctx, q, &s)
	}
	return s, err
}

func sqliteStats(ctx context.Context, q Querier, t Target, s *EngineStats) error {
	for pragma, dst := range map[string]*int64{"page_size": &s.PageSize, "page_count": &s.Pages, "freelist_count": &s.FreePages} {
		if err := q.QueryRowContext(ctx, "PRAGMA "+pragma).Scan(dst); err != nil {
			return fmt.Errorf("sqlite %s: %w", pragma, err)
		}
	}
	if err := q.QueryRowContext(ctx, "PRAGMA journal_mode").Scan(&s.JournalMode); err != nil {
		return fmt.Errorf("sqlite journal mode: %w", err)
	}
	if info, err := os.Stat(t.Path + "-wal"); err == nil {
		s.WALSize = info.Size()
	}
	return nil
}

func mysqlStats(ctx context.Context, q Querier, s *EngineStats) error {
	rows, err := q.QueryContext(ctx, `SHOW GLOBAL STATUS WHERE Variable_name IN
		('Threads_connected', 'Threads_running', 'Questions', 'Uptime', 'Innodb_buffer_pool_read_requests', 'Innodb_buffer_pool_reads')`)
	if err != nil {
		return fmt.Errorf("mysql status: %w", err)
	}
	defer rows.Close()
	dst := map[string]*int64{
		"Threads_connected": &s.Connections, "Threads_running": &s.Running, "Questions": &s.Queries, "Uptime": &s.Uptime,
		"Innodb_buffer_pool_read_requests": &s.CacheReads, "Innodb_buffer_pool_reads": &s.CacheMisses,
	}
	for rows.Next() {
		var name, value string
		if err := rows.Scan(&name, &value); err != nil {
			return fmt.Errorf("mysql status: %w", err)
		}
		if p, ok := dst[name]; ok {
			*p, _ = strconv.ParseInt(value, 10, 64)
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("mysql status: %w", err)
	}
	if err := q.QueryRowContext(ctx, "SELECT @@max_connections").Scan(&s.MaxConnections); err != nil {
		return fmt.Errorf("mysql max_connections: %w", err)
	}
	return nil
}

func postgresStats(ctx context.Context, q Querier, s *EngineStats) error {
	var reads, hits int64
	err := q.QueryRowContext(ctx, `SELECT numbackends, xact_commit + xact_rollback, blks_read, blks_hit,
		CAST(EXTRACT(EPOCH FROM now() - pg_postmaster_start_time()) AS BIGINT)
		FROM pg_stat_database WHERE datname = current_database()`).Scan(&s.Connections, &s.Queries, &reads, &hits, &s.Uptime)
	if err != nil {
		return fmt.Errorf("postgres stats: %w", err)
	}
	s.CacheReads, s.CacheMisses = reads+hits, reads
	if err := q.QueryRowContext(ctx, "SELECT COUNT(*) FROM pg_stat_activity WHERE datname = current_database() AND state = 'active'").Scan(&s.Running); err != nil {
		return fmt.Errorf("postgres activity: %w", err)
	}
	var maxConn string
	if err := q.QueryRowContext(ctx, "SHOW max_connections").Scan(&maxConn); err != nil {
		return fmt.Errorf("postgres max_connections: %w", err)
	}
	s.MaxConnections, _ = strconv.ParseInt(maxConn, 10, 64)
	err = q.QueryRowContext(ctx, "SELECT extversion FROM pg_extension WHERE extname = 'timescaledb'").Scan(&s.Timescale)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("postgres extensions: %w", err)
	}
	return nil
}

// TableSize is one table with its rows and bytes (data and indexes).
type TableSize struct {
	Name  string `json:"name"`
	Rows  int64  `json:"rows"`
	Bytes int64  `json:"bytes"`
}

// scanTables runs the catalog query; the rows are closed before Tables
// counts SQLite rows on the same transaction.
func scanTables(ctx context.Context, q Querier, query string, limit int) ([]TableSize, error) {
	rows, err := q.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []TableSize{}
	for rows.Next() {
		var t TableSize
		if err := rows.Scan(&t.Name, &t.Rows, &t.Bytes); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// Tables lists the largest tables. MySQL's row counts are its own
// estimate; SQLite has no catalog of rows, so those are counted.
func Tables(ctx context.Context, q Querier, d Dialect, limit int) ([]TableSize, error) {
	var query string
	switch d {
	case SQLite:
		query = `SELECT m.tbl_name, 0, SUM(s.pgsize) FROM dbstat s JOIN sqlite_master m ON m.name = s.name
			WHERE m.tbl_name NOT LIKE 'sqlite_%' GROUP BY m.tbl_name ORDER BY 3 DESC LIMIT ?`
	case MySQL:
		query = `SELECT table_name, COALESCE(table_rows, 0), COALESCE(data_length + index_length, 0)
			FROM information_schema.TABLES WHERE table_schema = DATABASE() ORDER BY 3 DESC LIMIT ?`
	default:
		query = "SELECT relname, n_live_tup, pg_total_relation_size(relid) FROM pg_stat_user_tables ORDER BY 3 DESC LIMIT ?"
	}
	out, err := scanTables(ctx, q, query, limit)
	if err != nil {
		return nil, fmt.Errorf("list tables: %w", err)
	}
	if d == SQLite {
		for i := range out {
			if err := q.QueryRowContext(ctx, "SELECT COUNT(*) FROM "+Ident(d, out[i].Name)).Scan(&out[i].Rows); err != nil {
				return nil, fmt.Errorf("count %s: %w", out[i].Name, err)
			}
		}
	}
	return out, nil
}
