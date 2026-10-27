-- A run writes its users in chunks, one destination transaction each, so
-- "batch" means only the operator's batch of moved users. SQLite 3.25 and
-- later rewrite the ledger_job index with the column.
ALTER TABLE ledger RENAME COLUMN batch TO chunk;
