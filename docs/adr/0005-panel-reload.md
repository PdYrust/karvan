# ADR 0005: How a running panel picks up the users Karvan writes
## Context
Karvan writes users straight into the destination panel's database. A running panel does not see those rows: Marzban's Xray holds its users in memory and PasarGuard pushes users to its nodes only when they change through its own code. PasarGuard also gives a WireGuard user its peer IP only on its own create and edit path. The operator should see every way to fix this on the final step and pick one there.
## Decision
- **Three methods, all shown on the final step** with what each does on that panel (`run.ReloadOptions`):
  - **Manual.** Karvan shows the panel's restart command (`marzban restart`, `pasarguard restart`) and does nothing else.
  - **Touch.** One edit per written user through the panel API, with nothing changed. Marzban adds the user to Xray and its nodes; PasarGuard pushes it to its nodes and allocates WireGuard peer IPs. Both panels delete `next_plan` when an edit leaves it out, and PasarGuard allocates only when `group_ids` is set, so Karvan reads the user first and sends those two fields back. Four calls run at once.
  - **Core restart.** Marzban: `POST /api/core/restart`, which rebuilds the Xray config from the database and restarts the core and its nodes. PasarGuard: `POST /api/nodes/reconnect`, which pushes every user to every node again; WireGuard users get a touch first, because a reconnect does not give them IPs.
- **Karvan acts as a full admin it mints a token for.** Both panels sign admin tokens (HS256) with the `jwt.secret_key` in their database, which Karvan already reads. The token is for the oldest Marzban sudo admin or PasarGuard owner, else the `SUDO_USERNAME` ENV admin, lives 5 minutes, is minted per call and is never stored. With no full admin, only manual is offered.
- **Karvan calls the panel where its `main.py` binds:** the Unix socket when `UVICORN_UDS` is set; without a certificate, 127.0.0.1 (PasarGuard: its private host); with one, the configured host over HTTPS, pinned to the exact certificate file the ENV names instead of a name check.
- **A reload never undoes data**. Its result goes to the report, and a failed reload leaves the manual command there.
## Checked on the running panels
`make test-panels` runs Marzban 0.8.4 and PasarGuard 5.4.1 on SQLite (`test/panels`), moves users into them with Karvan and checks:
- Users Karvan wrote are missing from Marzban's running Xray until a reload; after touch or core restart they are there (read with `xray api inbounduser`).
- Touch sends one `user_updated` webhook per user on Marzban; core restart sends none. PasarGuard sends its "user modified" notification on each edit (read from `app/operation/user.py`, not counted).
- Touch keeps `next_plan` on both panels.
- PasarGuard WireGuard users have no peer IP after a manual reload and get one from the panel's own pool after touch or core restart.
- The moved admin logs in with the password Karvan hashed with Go's bcrypt, and the panel API reads every moved user back with the same credentials.
- The panel kept answering while Karvan wrote into its SQLite file (slowest call 37 ms on the test machine).
## Alternatives considered
- **Ask the operator for a panel admin password.** One more secret to type and hold, for something the database secret already allows.
- **Restart the panel container ourselves.** Needs Docker access and drops every user for longer than a core restart; it stays the manual method, and Karvan's panel control may offer it later.
- **Copy PasarGuard's WireGuard allocator into Karvan.** It works per core namespace under row locks; a copy would race the running panel.
## Consequences
- Touch on a large migration takes minutes and floods notification channels; the final step says both.
- Core restart drops every user on the panel for a few seconds.
- A newer panel version that changes these endpoints or the token claims needs a new check in `test/panels` before Karvan supports it.
