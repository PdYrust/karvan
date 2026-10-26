# ADR 0002: What a visitor can see before signing in

## Status
Accepted, 2026-10-16

## Context
The panel is reachable from the internet. Scanners probe every address; a leaked URL can reach the sign-in page. The owner wants a branded sign-in page with the logo and the name. (A 3D mark was planned at first and later dropped; the page now shows a wall of hexagonal columns. The decision below does not change.)

## Decision
- **Hidden base path.** Everything lives under `/<16 random characters>/`. Anything else, a wrong method on a real route, an unclean or encoded path, `OPTIONS *`, or a missing asset gets the same `404 Not Found` bytes, with every earlier header removed. Go's default 404 body is not used because it identifies the server.
- **Two HTML entries.** `login.html` and its assets (found by walking the Vite manifest from the login entry) are public. `index.html` and the app's own chunks answer the plain 404 until a session exists.
- **The sign-in page is branded.** The owner chose this (phase 2 spec, D15). The trade-off is accepted: someone with a leaked path learns the product. The hidden path, argon2id, persisted lockout, the bounded hash queue and two-step sign-in still stand between them and the panel.
- **Referrer-Policy: no-referrer** keeps the hidden path out of Referer headers, and the access log only records paths with the prefix removed.

## Alternatives considered
- **Brand-free sign-in page:** hides the product if the path leaks, but the owner wants the sign-in page to look finished and to carry the identity.
- **Path as the only protection:** rejected. The path is a secret that leaks through screenshots and shoulder-surfing; it is not authentication.

## Consequences
- Responses Go writes before any handler runs (a `400` for a malformed request, the `Date` header) fall outside the single 404. They are known residue.
- A new public file must be listed in `httpapi/server.go` (`staticFiles`) or reached from the login entry, or it is a 404 before sign-in.
