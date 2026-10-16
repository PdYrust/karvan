# ADR 0006: Undo a failed migration row by row; restore the backup only as a last resort
## Status
Accepted, 2026-10-16
## Context
The owner wants Safe Mode to show "backing up, applying, failed, restoring" and leave the panel as it was. The destination keeps running during a migration: other admins edit users and the panel writes usage rows. Restoring a full backup under a running panel would throw those writes away and mix the panel's next writes into old data.
## Decision
- **Safe Mode on (the default):** an encrypted backup before the first write, then on any failure before reload:
  1. the failing batch rolls back its own transaction;
  2. committed batches are removed newest first, each in a short transaction that locks the users' rows, deletes their children and then the users, matched by id, username and `created_at`. Ledger rows become `compensated` per batch, so a stopped undo resumes;
  3. a check that nothing of the job is left;
  4. only if the undo failed or rows are left: restore the backup with the panel stopped. Stopping the panel needs a permission the Phase 5 installer grants per panel. Without it Karvan gives the command (`karvan safe-mode restore -panel-stopped <id>`).
- Only the job's own `done` rows are removed, never `adopted` ones. A created admin is removed only if it owns nothing else.
- **Safe Mode off:** no backup and no undo. A failing batch still rolls back; earlier batches stay and a re-run continues from the ledger.
- Turning Safe Mode off needs the typed confirmation `turn-off-safe-mode` and is audited. A running job's backup cannot be deleted.
- **Shutdown** cancels running jobs between batches, so Safe Mode undoes them. A crash leaves the job interrupted; at the next start Karvan settles it from the destination and waits for the operator to roll back or run again.
## Changed while building
The spec (D40) planned to run the chosen reload again after a rollback, so the core drops removed users. The running panels showed that rows Karvan writes stay out of the core until a reload, and the undo only runs before reload, so there is nothing to drop. The report still lists "the core may have loaded users" for a panel someone restarted during the job.
## Alternatives considered
- **Always restore the backup on failure.** Loses other admins' and the panel's writes made during the job, and needs the panel stopped for every failure.
- **One big undo transaction.** On SQLite it would block the panel's own writes for the whole undo.
## Consequences
- An undo can leave rows when the panel itself changed them meanwhile (a child the panel added holds a lock, a user was deleted); the report names them and the backup restore covers them.
