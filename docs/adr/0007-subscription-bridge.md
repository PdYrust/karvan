# ADR 0007: Old subscription links keep working through a bridge
## Status
Accepted, 2026-10-16
## Context
Every moved user has an old link that points at the old panel's domain and carries a token signed with the old panel's secret. Resellers cannot send thousands of new links at once. The old domain can be pointed at the new server, but the new panel does not accept the old tokens. Karvan never had the old panel's secret: the source is read with the reseller's login only.
## Decision
- **A legacy site** is an address the bridge listens on: a port, the old path, domain patterns, the old panel's address (with the IP and certificate pin the migration used) and an optional certificate (a file Karvan re-reads when it changes, or a pasted PEM whose key is stored sealed). It never sits in front of the new panel's own port. A site whose old panel address the dial guard forbids is refused.
- **Order for each GET or HEAD** (D59):
  1. The token's HMAC is in the verified set and the user passes the destination check: Karvan mints the new panel's own token for that user and proxies the request.
  2. Otherwise the request goes to the new panel's own subscription route with the same token. The path prefix is changed to the new panel's subscription path (the spec said "unchanged"; with another path the panel could never answer). Any answer except 404 goes back as is, so the panel judges its own tokens.
  3. On 404: the payload must name a user moved from this site's old panel, lazy checks must be on (and opted into for a plain-http old panel), and the destination check must pass. Then the old panel's `/{path}/{token}/info` is asked once. Its answer must be that exact user: name (and id on PasarGuard), `created_at` and a fingerprint of the source credentials. Only then is the token kept.
- **Only HMACs of tokens are stored**, with a key derived from Karvan's key file, so the database alone cannot rebuild a working link. Each stored token points at its ledger row; the user always comes from that row, never from the payload.
- **Owner rule:** if the new panel revokes a user's link or deletes the user, Karvan closes that user's bridge entry for good: their tokens go and no lazy check runs for them again, so fresh links from the old panel stop working too. This is checked on every request and by an hourly sweep.
- **Limits on lazy checks:** 10 per second per site with 4 at once, and 3 per 10 seconds per client address (IPv6 per /64). Over a limit, or when the old panel times out, errors or rate limits, the answer is 503 with `Retry-After` and nothing is cached. Only the old panel's real 404 is cached, for an hour.
- A verified token is asked about again once a day, in the background. A revoke on the old panel drops the token; an old panel that no longer answers keeps it.
- Token lists can be uploaded (text or CSV of old links) and are verified 4 at a time, so users never wait for a lazy check.
## Checked
Unit tests run every step and abuse case with fake panels (forged tokens, a reused name or id on the old panel, an answer for another user, floods from one address, one /64 and many addresses, an old panel that is down, slow or rate limiting). Against the running Marzban, an old link through the bridge returns the moved user's real subscription, and a revoke on the new panel closes it.
## Alternatives considered
- **Ask for the old panel's secret** (or its database) and accept old tokens offline. That is the database method of Phase 6; in Phase 3 only the reseller's login exists.
- **Accept any token whose payload names a moved user.** Anyone could forge one from a username.
## Consequences
- A user whose old link was never used or uploaded while the old panel was up cannot be bridged after it goes away; the new-links CSV covers them.
- The bridge listeners are public by design (they answer subscription clients). They serve only GET and HEAD on the old path and never expose the panel's API.
