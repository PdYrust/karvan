-- Several admins in one job (phase 4.1 D34): a user row names the source
-- admin it moved with, so its admin row is the job's row for that admin.
-- Rows from before, and admin rows, keep '' and mean the job's only
-- admin. Additive only: no existing row changes.
ALTER TABLE ledger ADD COLUMN source_admin TEXT NOT NULL DEFAULT '';
