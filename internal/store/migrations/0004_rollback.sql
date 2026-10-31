-- What Safe Mode does when this job fails: undo its own rows, or put the
-- whole backup back.
ALTER TABLE jobs ADD COLUMN rollback TEXT NOT NULL DEFAULT 'undo' CHECK (rollback IN ('undo', 'backup'));
