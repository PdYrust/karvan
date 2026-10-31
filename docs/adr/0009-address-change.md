# ADR 0009: The panel's address changes only once it is confirmed from the new one

## Context
The panel lives at a random port and a hidden path. Changing either used to need `karvan reset` and a restart. The web panel must offer both with one click, but a new port the firewall blocks, a proxy that only forwards the old port, or a new path the operator did not write down would lock the operator out of their own panel.

## Decision
- `serve` puts every listener behind a handler that picks the site by its hidden path, and can open a second listener. The address of a request (its listener's port and the path it matched) travels with it.
- `POST /api/v1/settings/address` asks for the password again, and a code when two-step sign-in is on, with the same lockout as sign-in. It takes a new random path, a new port (typed or random), or both. Karvan then serves the new address next to the old one for 10 minutes: a second listener for a new port, a second path on the same listener for a new path only. A port that cannot be opened answers 409 `port_unavailable` and nothing changes.
- `POST /api/v1/settings/address/confirm` must come through the new address. It saves the address in the settings, and only then stops the old one: the old listener closes, or the old path answers the plain 404. From the old address it answers 409 `not_new_address`.
- No confirm, no change: after 10 minutes the new address goes away. It lives in memory only, so a restart drops it too. A newer change replaces the one waiting.
- Over plain HTTP the session cookie belongs to the path, so a new path asks for a sign-in there first. Cookies do not depend on the port, so a new port keeps the session.
- The new path is in the response to the request that made it and nowhere else (ADR 0003). The audit log records the port, which is not secret, and whether the path changed, never the path. The server log names ports only.

## Alternatives considered
- **Save at once and restart.** One mistyped port or a closed firewall and the panel is gone until someone reaches the shell.
- **A timer that rolls back unless confirmed, after saving.** A crash during the 10 minutes would leave the untested address saved.
- **Confirm with a token in the new address instead of a session.** The token would sit in the URL, the browser history and any proxy log.

## Consequences
- The operator proves the new address works from their own browser before it counts: a blocked port or a proxy that does not forward it simply expires.
- Behind a reverse proxy, a new port only works once the proxy forwards it; the panel says so next to the port field.
- Tabs open on the old address stop working after the confirm.
- If the confirmed address is lost anyway, `karvan reset -path -port` on the server still gives a new one.
