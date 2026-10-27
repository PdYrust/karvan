-- Batches (phase 4.1 D1 to D8): a migration that ended done becomes a
-- named batch holding the users it moved. Membership lives on the ledger
-- row. Additive only: no existing row is deleted and no token is
-- re-pointed. Times are unix milliseconds, like jobs.
CREATE TABLE batches (
    id              INTEGER PRIMARY KEY,
    job_id          TEXT    NOT NULL UNIQUE REFERENCES jobs (id),
    panel_id        INTEGER NOT NULL REFERENCES panels (id),
    dest            TEXT    NOT NULL,
    source_kind     TEXT    NOT NULL,
    source_origin   TEXT    NOT NULL,
    source_ip       TEXT    NOT NULL DEFAULT '',
    source_pin      TEXT    NOT NULL DEFAULT '',
    old_path        TEXT    NOT NULL DEFAULT '',
    old_hosts       TEXT    NOT NULL DEFAULT '[]',
    old_port        INTEGER NOT NULL DEFAULT 0,
    name            TEXT    NOT NULL CHECK (length(name) BETWEEN 1 AND 64),
    state           TEXT    NOT NULL CHECK (state IN ('active', 'inactive', 'deleted')),
    inactive_reason TEXT    NOT NULL DEFAULT '' CHECK (inactive_reason IN ('', 'restoring', 'rolled_back', 'restore_failed')),
    inactive_at     INTEGER NOT NULL DEFAULT 0,
    deleted_at      INTEGER NOT NULL DEFAULT 0,
    created_at      INTEGER NOT NULL,
    updated_at      INTEGER NOT NULL
) STRICT;
CREATE INDEX batches_dest ON batches (dest, created_at);

-- One batch per done job with users, and one inactive batch per
-- rolled-back job whose done or compensated users still have tokens, so
-- the operator decides about those tokens. Names follow D6: the source
-- admin, or "batch", with " 2", " 3" on later batches of the same name and
-- old panel. Old link facts stay empty; the rule form asks for them.
INSERT INTO batches (job_id, panel_id, dest, source_kind, source_origin, name, state, inactive_reason, inactive_at, created_at, updated_at)
WITH found AS (
    SELECT j.id, j.panel_id, j.source_kind, j.source_origin,
        (SELECT min(l.dest) FROM ledger l WHERE l.job_id = j.id AND l.kind = 'user') AS dest,
        COALESCE(NULLIF(substr(trim(j.source_admin, char(32, 9, 10, 11, 12, 13)), 1, 58), ''), 'batch') AS base,
        CASE j.state WHEN 'done' THEN 'active' ELSE 'inactive' END AS state,
        COALESCE(NULLIF(j.ended_at, 0), j.updated_at) AS at
    FROM jobs j
    WHERE (j.state = 'done' AND EXISTS (SELECT 1 FROM ledger l
            WHERE l.job_id = j.id AND l.kind = 'user' AND l.state IN ('done', 'adopted')))
        OR (j.state = 'rolled_back' AND EXISTS (SELECT 1 FROM ledger l JOIN legacy_tokens t ON t.ledger_id = l.id
            WHERE l.job_id = j.id AND l.kind = 'user' AND l.state IN ('done', 'compensated')))
), numbered AS (
    SELECT found.*, row_number() OVER (PARTITION BY source_origin, base ORDER BY at, id) AS n FROM found
)
SELECT id, panel_id, dest, source_kind, source_origin,
    CASE WHEN n > 1 THEN base || ' ' || n ELSE base END,
    state,
    CASE state WHEN 'active' THEN '' ELSE 'rolled_back' END,
    CASE state WHEN 'active' THEN 0 ELSE at END,
    at, at
FROM numbered ORDER BY at, id;

-- Membership. adopted rows of rolled-back jobs keep batch_id NULL: the
-- restore left those users on the destination and the bridge serves them
-- today, so they stay in manual scope.
ALTER TABLE ledger ADD COLUMN batch_id INTEGER REFERENCES batches (id);
UPDATE ledger SET batch_id = b.id FROM batches b
    WHERE b.job_id = ledger.job_id AND ledger.kind = 'user'
        AND (ledger.state IN ('done', 'compensated') OR (ledger.state = 'adopted' AND b.state = 'active'));
CREATE INDEX ledger_batch ON ledger (batch_id) WHERE batch_id IS NOT NULL;

-- The batches a panel rule serves.
CREATE TABLE legacy_site_batches (
    site_id  INTEGER NOT NULL REFERENCES legacy_sites (id) ON DELETE CASCADE,
    batch_id INTEGER NOT NULL REFERENCES batches (id),
    PRIMARY KEY (site_id, batch_id)
) STRICT;
CREATE INDEX legacy_site_batches_batch ON legacy_site_batches (batch_id);

-- Every existing site is a manual rule with exactly today's scope.
ALTER TABLE legacy_sites ADD COLUMN mode TEXT NOT NULL DEFAULT 'manual' CHECK (mode IN ('batches', 'manual'));
