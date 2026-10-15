CREATE TABLE operators (
    id             INTEGER PRIMARY KEY,
    username       TEXT    NOT NULL UNIQUE,
    password_hash  TEXT    NOT NULL,
    totp_secret    BLOB,
    totp_pending   BLOB,
    totp_last_step INTEGER NOT NULL DEFAULT 0,
    created_at     INTEGER NOT NULL,
    updated_at     INTEGER NOT NULL
) STRICT;

CREATE TABLE sessions (
    token_hash   BLOB    PRIMARY KEY,
    operator_id  INTEGER NOT NULL REFERENCES operators (id) ON DELETE CASCADE,
    created_at   INTEGER NOT NULL,
    last_seen_at INTEGER NOT NULL,
    expires_at   INTEGER NOT NULL,
    ip           TEXT    NOT NULL,
    user_agent   TEXT    NOT NULL
) STRICT;

CREATE INDEX sessions_operator ON sessions (operator_id);

CREATE TABLE login_failures (
    scope        TEXT    NOT NULL CHECK (scope IN ('ip', 'account')),
    key          TEXT    NOT NULL,
    failures     INTEGER NOT NULL,
    window_start INTEGER NOT NULL,
    locked_until INTEGER NOT NULL DEFAULT 0,
    lock_level   INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (scope, key)
) STRICT;

CREATE TABLE settings (
    key   TEXT PRIMARY KEY,
    value TEXT NOT NULL
) STRICT;

CREATE TABLE audit_events (
    id      INTEGER PRIMARY KEY,
    at      INTEGER NOT NULL,
    actor   TEXT    NOT NULL,
    ip      TEXT    NOT NULL,
    action  TEXT    NOT NULL,
    outcome TEXT    NOT NULL CHECK (outcome IN ('success', 'failure')),
    detail  TEXT    NOT NULL DEFAULT '{}'
) STRICT;

CREATE INDEX audit_events_at ON audit_events (at);
