-- Legacy subscription sites: old links answered for moved users (D55-D64).
-- Times are unix seconds.
CREATE TABLE legacy_sites (
    id            INTEGER PRIMARY KEY,
    name          TEXT    NOT NULL UNIQUE,
    panel_id      INTEGER NOT NULL REFERENCES panels (id),
    source_kind   TEXT    NOT NULL CHECK (source_kind IN ('marzban', 'pasarguard')),
    source_url    TEXT    NOT NULL,              -- the old panel, for lazy checks
    source_ip     TEXT    NOT NULL DEFAULT '',   -- the address the migration used
    source_pin    TEXT    NOT NULL DEFAULT '',   -- hex SHA-256 of its certificate
    old_path      TEXT    NOT NULL,
    domains       TEXT    NOT NULL DEFAULT '[]', -- exact names or *.example.com
    listen_port   INTEGER NOT NULL CHECK (listen_port BETWEEN 1 AND 65535),
    cert_file     TEXT    NOT NULL DEFAULT '',
    key_file      TEXT    NOT NULL DEFAULT '',
    cert_pem      TEXT    NOT NULL DEFAULT '',
    key_sealed    BLOB,
    lazy          INTEGER NOT NULL DEFAULT 1,
    plain_http_ok INTEGER NOT NULL DEFAULT 0,
    created_at    INTEGER NOT NULL,
    updated_at    INTEGER NOT NULL
) STRICT;

-- Old tokens proven to belong to a moved user. Only an HMAC of the token
-- is kept, so this table alone cannot rebuild a working link (D57).
CREATE TABLE legacy_tokens (
    mac         BLOB    PRIMARY KEY,
    site_id     INTEGER NOT NULL REFERENCES legacy_sites (id) ON DELETE CASCADE,
    ledger_id   INTEGER NOT NULL REFERENCES ledger (id),
    issued_at   INTEGER NOT NULL, -- the time inside the token
    verified_at INTEGER NOT NULL,
    checked_at  INTEGER NOT NULL
) STRICT;
CREATE INDEX legacy_tokens_ledger ON legacy_tokens (ledger_id);

-- Users whose old links stay closed for good: revoked or deleted on the
-- new panel (owner rule, D58).
CREATE TABLE legacy_closed (
    ledger_id INTEGER PRIMARY KEY REFERENCES ledger (id),
    reason    TEXT    NOT NULL,
    closed_at INTEGER NOT NULL
) STRICT;
