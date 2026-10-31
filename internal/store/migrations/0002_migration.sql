CREATE TABLE panels (
    id         INTEGER PRIMARY KEY,
    env_path   TEXT    NOT NULL UNIQUE,
    name       TEXT    NOT NULL,
    kind       TEXT    NOT NULL DEFAULT '',
    scan       TEXT    NOT NULL DEFAULT '{}',
    scanned_at INTEGER NOT NULL
) STRICT;

CREATE TABLE jobs (
    id            TEXT    PRIMARY KEY,
    state         TEXT    NOT NULL,
    source_kind   TEXT    NOT NULL,
    source_origin TEXT    NOT NULL,
    source_admin  TEXT    NOT NULL DEFAULT '',
    panel_id      INTEGER NOT NULL REFERENCES panels (id),
    plan          TEXT    NOT NULL DEFAULT '{}',
    plan_version  INTEGER NOT NULL DEFAULT 0,
    safe_mode     INTEGER NOT NULL,
    backup_id     TEXT    NOT NULL DEFAULT '',
    report        TEXT    NOT NULL DEFAULT '',
    created_at    INTEGER NOT NULL,
    updated_at    INTEGER NOT NULL
) STRICT;
CREATE INDEX jobs_panel_state ON jobs (panel_id, state);

CREATE TABLE job_events (
    job_id TEXT    NOT NULL REFERENCES jobs (id) ON DELETE CASCADE,
    seq    INTEGER NOT NULL,
    at     INTEGER NOT NULL,
    stage  TEXT    NOT NULL,
    type   TEXT    NOT NULL,
    data   TEXT    NOT NULL DEFAULT '{}',
    PRIMARY KEY (job_id, seq)
) STRICT;

-- One row per admin or user Karvan moved, adopted or merged into.
-- Times are unix microseconds.
CREATE TABLE ledger (
    id               INTEGER PRIMARY KEY,
    kind             TEXT    NOT NULL CHECK (kind IN ('admin', 'user')),
    job_id           TEXT    NOT NULL,
    state            TEXT    NOT NULL CHECK (state IN ('pending', 'done', 'adopted', 'merged', 'unknown', 'compensated', 'superseded')),
    batch            INTEGER NOT NULL DEFAULT 0,
    source_origin    TEXT    NOT NULL,
    source_key       TEXT    NOT NULL,
    source_created   INTEGER NOT NULL,
    source_username  TEXT    NOT NULL,
    cred_fingerprint TEXT    NOT NULL DEFAULT '',
    dest             TEXT    NOT NULL,
    dest_id          INTEGER NOT NULL DEFAULT 0,
    dest_username    TEXT    NOT NULL,
    dest_created     INTEGER NOT NULL,
    created_at       INTEGER NOT NULL,
    updated_at       INTEGER NOT NULL
) STRICT;
CREATE UNIQUE INDEX ledger_live ON ledger (kind, source_origin, source_key, source_created, dest)
    WHERE state IN ('pending', 'done', 'adopted', 'unknown');
CREATE INDEX ledger_job ON ledger (job_id, state, batch);
CREATE INDEX ledger_dest ON ledger (dest, source_origin);

CREATE TABLE templates (
    id         INTEGER PRIMARY KEY,
    name       TEXT    NOT NULL UNIQUE,
    body       TEXT    NOT NULL,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
) STRICT;

CREATE TABLE safe_mode_backups (
    id         TEXT    PRIMARY KEY,
    job_id     TEXT    NOT NULL,
    dest       TEXT    NOT NULL,
    path       TEXT    NOT NULL,
    size       INTEGER NOT NULL,
    method     TEXT    NOT NULL,
    created_at INTEGER NOT NULL
) STRICT;
