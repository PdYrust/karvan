# ADR 0006: How Safe Mode takes a failed migration back
## Status
Accepted, 2026-10-16. Changed the same day: the rollback is a setting, and Karvan stops and starts the panel itself (owner).
## Context
The owner wants Safe Mode to show "backing up, applying, failed, restoring" and leave the panel as it was, all from the web panel: nobody should type `marzban down` or any other command. Some operators trust only a full restore of the backup. The destination keeps running during a migration: other admins edit users and the panel writes usage rows. A full restore takes those writes back too, and it must run with the panel stopped, or the panel's next writes mix into the old data.
## Decision
- **Safe Mode on (the default)** takes an encrypted backup before the first write. On any failure before reload the failing batch rolls back its own transaction, and then the **rollback setting** decides:
  - **undo (the default):** committed batches are removed newest first, each in a short transaction that locks the users' rows, deletes their children and then the users, matched by id, username and `created_at`. Ledger rows become `compensated` per batch, so a stopped undo resumes. Then a check that nothing of the job is left; only if the undo failed or rows are left is the backup put back.
  - **backup:** the whole backup is put back. If that fails (the panel would not stop, the restore broke), Karvan falls back to the undo, so the job's rows still go.
- **Karvan stops and starts the panel itself** (`internal/panelctl`), the way the official install scripts do: `docker compose -f <panel folder>/docker-compose.yml -p <folder name> stop|start <panel services>`. Only services that run the panel image stop; a database container in the same file keeps running, so the backup can go back into it. The panel is always started again once it was stopped.
- Karvan's config `panelControl` says how it may use Docker: `auto` (Docker itself, else `sudo -n`), `docker`, `sudo` or `off`. The Phase 5 installer gives the `karvan` user a sudo rule for exactly these commands, not Docker access in general, which would equal root.
- A run with the backup choice **does not start** when Karvan cannot stop and start that panel; the API answers `panel_control_unavailable` with the reason. The panel list shows the same status per panel, so the web panel can warn before the operator picks the choice.
- A backup can also be **put back on request** from the web panel (`POST /api/v1/safe-mode/backups/{id}/restore`, the backup id typed to confirm, audited). Karvan holds the destination's lock, stops the panel, restores, starts it, and marks the job rolled back. `karvan safe-mode restore <id>` does the same when Karvan's server is not running.
- Only the job's own `done` rows are removed by the undo, never `adopted` ones. A created admin is removed only if it owns nothing else.
- **Safe Mode off:** no backup and no rollback. A failing batch still rolls back; earlier batches stay and a re-run continues from the ledger.
- Turning Safe Mode off needs the typed confirmation `turn-off-safe-mode`. Every change to Safe Mode or the rollback choice is audited. A running job's backup cannot be deleted.
- **Shutdown** cancels running jobs between batches, so Safe Mode takes them back. A crash leaves the job interrupted; at the next start Karvan settles it from the destination and waits for the operator to roll back (with the job's own rollback choice) or run again.
## Checked
On the running Marzban 0.8.4 and PasarGuard 5.4.1 (`make test-panels`): Karvan stops and starts each panel through its compose file, and a migration that fails in its second batch with the backup choice stops Marzban, puts the backup back, starts it again and leaves the user count as it was.
## Changed while building
The spec (D40) planned to run the chosen reload again after a rollback, so the core drops removed users. The running panels showed that rows Karvan writes stay out of the core until a reload, and the rollback only runs before reload, so there is nothing to drop.
## Alternatives considered
- **Always restore the backup.** Loses other admins' and the panel's writes made during the job, and stops the panel for every failure. Kept as a choice, not the default.
- **Give Karvan Docker access.** Equal to root on the server; a narrow sudo rule does the same job.
- **One big undo transaction.** On SQLite it would block the panel's own writes for the whole undo.
## Consequences
- With the backup choice, users see the panel stopped for the seconds the restore takes, and everything written to the panel during the job is gone; the report says so (`restore_lost_panel_writes`).
- A panel not installed with its official script (no compose file next to its `.env`) cannot be stopped by Karvan; only the undo choice works for it.
- A restore on request also takes back later jobs on the same panel; their ledger rows then point at rows that are gone, which a re-run sees as moved users that left (C16b).
