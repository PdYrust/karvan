-- Certificates on domains (phase 4.1 owner change 2026-10-28, D61 to
-- D70): a domain points at its certificate, a port can keep serving
-- another one through a pin, and Let's Encrypt orders get their tables.
-- Additive only: no existing value changes except cert_filled, which
-- marks the domains the bridge fills from their sites at start (D64).
-- Times are unix seconds, like domains.
ALTER TABLE domains ADD COLUMN certificate_id INTEGER REFERENCES certificates (id); -- NULL: plain HTTP
ALTER TABLE domains ADD COLUMN cert_filled INTEGER NOT NULL DEFAULT 1 CHECK (cert_filled IN (0, 1));
-- Rows from before wait for the fill; rows added later are filled when made.
UPDATE domains SET cert_filled = 0;
-- How a Let's Encrypt certificate was got, and its renewal (D70).
ALTER TABLE domains ADD COLUMN acme_method TEXT NOT NULL DEFAULT '' CHECK (acme_method IN ('', 'http-01', 'dns-01'));
ALTER TABLE domains ADD COLUMN renew_at INTEGER NOT NULL DEFAULT 0;
ALTER TABLE domains ADD COLUMN renew_tries INTEGER NOT NULL DEFAULT 0;
ALTER TABLE domains ADD COLUMN renew_error TEXT NOT NULL DEFAULT '';
ALTER TABLE certificates ADD COLUMN origin TEXT NOT NULL DEFAULT '' CHECK (origin IN ('', 'acme'));
-- A port that served another certificate for a domain before the move
-- keeps serving it (D62, D64).
CREATE TABLE domain_pins (
    port           INTEGER NOT NULL CHECK (port BETWEEN 1 AND 65535),
    domain_id      INTEGER NOT NULL REFERENCES domains (id) ON DELETE CASCADE,
    certificate_id INTEGER NOT NULL REFERENCES certificates (id),
    PRIMARY KEY (port, domain_id)
) STRICT;
-- One ACME account per directory; its key is sealed under
-- ('acme_accounts', 'key_sealed', id) after the insert, in the same
-- transaction. A row with a NULL key is never used (D66).
CREATE TABLE acme_accounts (
    id         INTEGER PRIMARY KEY,
    directory  TEXT    NOT NULL UNIQUE,
    url        TEXT    NOT NULL DEFAULT '',
    email      TEXT    NOT NULL DEFAULT '',
    key_sealed BLOB,
    created_at INTEGER NOT NULL
) STRICT;
-- Let's Encrypt orders (D69); the certificate key is sealed under
-- ('acme_orders', 'key_sealed', id) before finalize.
CREATE TABLE acme_orders (
    id            INTEGER PRIMARY KEY,
    domain_id     INTEGER NOT NULL REFERENCES domains (id) ON DELETE CASCADE,
    account_id    INTEGER NOT NULL REFERENCES acme_accounts (id),
    method        TEXT    NOT NULL CHECK (method IN ('http-01', 'dns-01')),
    renewal       INTEGER NOT NULL DEFAULT 0 CHECK (renewal IN (0, 1)),
    state         TEXT    NOT NULL CHECK (state IN ('starting', 'waiting_dns', 'validating', 'issuing', 'done', 'failed', 'cancelled')),
    order_url     TEXT    NOT NULL DEFAULT '',
    authz_url     TEXT    NOT NULL DEFAULT '',
    challenge_url TEXT    NOT NULL DEFAULT '',
    token         TEXT    NOT NULL DEFAULT '',
    dns_name      TEXT    NOT NULL DEFAULT '',
    dns_value     TEXT    NOT NULL DEFAULT '',
    key_sealed    BLOB,
    error_code    TEXT    NOT NULL DEFAULT '',
    error_detail  TEXT    NOT NULL DEFAULT '' CHECK (length(error_detail) <= 300),
    retry_at      INTEGER NOT NULL DEFAULT 0,
    created_at    INTEGER NOT NULL,
    updated_at    INTEGER NOT NULL,
    expires_at    INTEGER NOT NULL DEFAULT 0
) STRICT;
-- One running order per domain, so two starts cannot race.
CREATE UNIQUE INDEX acme_orders_running ON acme_orders (domain_id) WHERE state IN ('starting', 'waiting_dns', 'validating', 'issuing');
CREATE INDEX acme_orders_domain ON acme_orders (domain_id, created_at);
