-- Subscriptions: domains and certificates become
-- rows of their own, and redirect rules get a table. Additive only: no
-- existing row is deleted or changed. Times are unix seconds, like
-- legacy_sites.
CREATE TABLE domains (
    id         INTEGER PRIMARY KEY,
    name       TEXT    NOT NULL UNIQUE CHECK (length(name) BETWEEN 1 AND 253), -- a host name, *.name or an IP address
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
) STRICT;
-- Every site domain gets a row. Site domains are already stored lower
-- case and checked, so they are valid names.
INSERT INTO domains (name, created_at, updated_at)
    SELECT j.value, min(s.created_at), min(s.created_at) FROM legacy_sites s, json_each(s.domains) j GROUP BY j.value;
-- A certificate is a file pair on this server (referenced, never copied)
-- or a pasted certificate whose key is sealed under
-- ('certificates', 'key_sealed', id).
CREATE TABLE certificates (
    id         INTEGER PRIMARY KEY,
    name       TEXT    NOT NULL UNIQUE CHECK (length(name) BETWEEN 1 AND 64),
    cert_file  TEXT    NOT NULL DEFAULT '',
    key_file   TEXT    NOT NULL DEFAULT '',
    cert_pem   TEXT    NOT NULL DEFAULT '',
    key_sealed BLOB,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    CHECK ((cert_file <> '' AND key_file <> '' AND cert_pem = '') OR (cert_file = '' AND key_file = '' AND cert_pem <> ''))
) STRICT;
-- NULL with empty own certificate columns is plain HTTP; NULL with own
-- columns set is a site the bridge has not adopted yet.
ALTER TABLE legacy_sites ADD COLUMN certificate_id INTEGER REFERENCES certificates (id);
-- Redirect rules: a request on a port, domain and path is sent to a URL.
CREATE TABLE link_redirects (
    id             INTEGER PRIMARY KEY,
    name           TEXT    NOT NULL UNIQUE,
    domains        TEXT    NOT NULL DEFAULT '[]',
    listen_port    INTEGER NOT NULL CHECK (listen_port BETWEEN 1 AND 65535),
    from_path      TEXT    NOT NULL,
    match          TEXT    NOT NULL CHECK (match IN ('exact', 'prefix')),
    to_url         TEXT    NOT NULL,
    certificate_id INTEGER REFERENCES certificates (id),
    created_at     INTEGER NOT NULL,
    updated_at     INTEGER NOT NULL
) STRICT;
