-- The jobs list shows each job's counts and when it ended without reading
-- its whole report, which holds a line per user. Jobs from before this
-- take both from their report once. ended_at is unix milliseconds, 0 for
-- a job that has not ended.
ALTER TABLE jobs ADD COLUMN counts TEXT NOT NULL DEFAULT '';
ALTER TABLE jobs ADD COLUMN ended_at INTEGER NOT NULL DEFAULT 0;
UPDATE jobs SET
    counts = COALESCE(json_extract(report, '$.counts'), ''),
    ended_at = MAX(0, COALESCE(CAST(strftime('%s', json_extract(report, '$.endedAt')) AS INTEGER), 0) * 1000)
    WHERE report != '' AND json_valid(report);
CREATE INDEX jobs_created ON jobs (created_at, id);
