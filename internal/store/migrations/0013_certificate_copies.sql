-- Folders Karvan keeps a certificate copied to, and every file it wrote
-- there. Additive only: two new tables, no existing row changes.
-- Times are unix seconds.
CREATE TABLE certificate_copies (
    id             INTEGER PRIMARY KEY,
    certificate_id INTEGER NOT NULL REFERENCES certificates (id) ON DELETE CASCADE,
    panel_id       INTEGER REFERENCES panels (id) ON DELETE CASCADE, -- NULL: Let's Encrypt live
    base           TEXT    NOT NULL, -- the folder the copy stays inside, such as /var/lib/marzban
    rel            TEXT    NOT NULL, -- the copy's folder below base, such as certs/example.com
    -- Keyed sums of the certificate and key last copied or tried, so a
    -- change copies again and an unchanged one does not.
    cert_sum       TEXT    NOT NULL DEFAULT '',
    key_sum        TEXT    NOT NULL DEFAULT '',
    error_reason   TEXT    NOT NULL DEFAULT '', -- why the last copy failed; '' when it worked
    error_path     TEXT    NOT NULL DEFAULT '',
    copied_at      INTEGER NOT NULL DEFAULT 0,  -- the last copy that worked
    created_at     INTEGER NOT NULL,
    UNIQUE (base, rel)
) STRICT;
CREATE INDEX certificate_copies_certificate ON certificate_copies (certificate_id);
-- A file is overwritten only while it still holds what Karvan wrote.
CREATE TABLE certificate_copy_files (
    path       TEXT    PRIMARY KEY,
    sum        TEXT    NOT NULL, -- keyed sum of what Karvan wrote
    written_at INTEGER NOT NULL
) STRICT;
