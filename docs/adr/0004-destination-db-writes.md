# ADR 0004: Karvan writes moved users straight into the destination's database
## Context
The rule for a migration: read the old panel through its API with the reseller's login, write admins and users into the new panel through its database, and use the new panel's API only to make it notice (ADR 0005). Neither panel's API can create a user with a given `created_at`, UUIDs, used traffic and reset history at once, and per-user API calls fire notifications and take minutes for large moves.
## Decision
- **One writer per panel revision** (`panel/marzbandb`, `panel/pasarguarddb`). Each knows the exact columns, enum storage and JSON shape of Marzban 0.8.4 (`2b231de97dc3`) and PasarGuard 5.4.1 (`48a6bcb8bba1`). Any other revision is refused at inspection. Schema fixtures come from the panels' own migrations on every supported database (`scripts/testdb`).
- **Read only until the run.** The check stage uses handles that cannot write: no `Exec` in the type, and read-only sessions on the server. A test compares table checksums before and after a check.
- **Short chunks.** One transaction per chunk: 50 users on SQLite (the panel shares its single writer), 500 on MySQL and PostgreSQL. Ids come from the database, never assumed. Every datetime is written in UTC at the column's precision.
- **A ledger in Karvan's own database** records every row before and after its chunk commits, so a crash at any step is settled from what the destination holds, a re-run never creates a user twice, and a rollback removes exactly what one job wrote.
- **Write what the panel would write.** Where the panel adds rows as a side effect, Karvan does the same: a Marzban inbound row gets its default host, as `get_or_create_inbound` does, or that inbound would give no links in any subscription (found with the running panel). Where the panel's own logic is hard to copy safely, Karvan leaves it to the panel: PasarGuard WireGuard peer IPs are assigned by the panel's edit path during the reload.
- **One process per destination and per data directory**: a database lock on the destination (flock on SQLite, `GET_LOCK` on MySQL, an advisory lock on PostgreSQL) and a flock on Karvan's data directory.
## Alternatives considered
- **Create users through the destination's API.** No way to keep `created_at` and traffic history; one notification per user; slow.
- **Copy rows with the panels' own Python code.** Needs their runtime and their database driver stack inside Karvan.
## Consequences
- Every new panel revision needs its own fixture, writer check and test run before Karvan accepts it.
- A panel that caches rows in memory does not see new users until told (ADR 0005).
