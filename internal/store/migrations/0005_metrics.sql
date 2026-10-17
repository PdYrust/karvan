-- One row per minute for the analytics pages, kept 30 days. ts is unix
-- seconds at the start of the minute. Rates are bytes per second.
CREATE TABLE metrics_host (
    ts         INTEGER PRIMARY KEY,
    cpu        REAL    NOT NULL,
    cpu_max    REAL    NOT NULL,
    load1      REAL    NOT NULL,
    mem_used   INTEGER NOT NULL,
    mem_cache  INTEGER NOT NULL,
    mem_total  INTEGER NOT NULL,
    swap_used  INTEGER NOT NULL,
    net_rx     REAL    NOT NULL,
    net_tx     REAL    NOT NULL,
    net_rx_max REAL    NOT NULL,
    net_tx_max REAL    NOT NULL,
    disk_read  REAL    NOT NULL,
    disk_write REAL    NOT NULL,
    tcp        INTEGER NOT NULL,
    procs      INTEGER NOT NULL
) STRICT;

-- Each panel database once a minute. cache_hit is NULL where the engine
-- has no buffer cache to ask about (SQLite).
CREATE TABLE metrics_db (
    panel_id     INTEGER NOT NULL REFERENCES panels (id) ON DELETE CASCADE,
    ts           INTEGER NOT NULL,
    size         INTEGER NOT NULL,
    connections  INTEGER NOT NULL,
    qps          REAL    NOT NULL,
    cache_hit    REAL,
    users        INTEGER NOT NULL,
    active       INTEGER NOT NULL,
    online       INTEGER NOT NULL,
    used_traffic INTEGER NOT NULL,
    PRIMARY KEY (panel_id, ts)
) STRICT;
