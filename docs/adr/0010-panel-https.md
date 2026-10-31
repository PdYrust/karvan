# ADR 0010: The web panel serves HTTPS with its own certificate or a domain's

Status: accepted

## Context
The panel served plain HTTP only. Its sign-in, session cookie and every secret it shows crossed the network in the clear unless the operator put a proxy in front. Many installs have no domain, only an IP address. Others already have a domain with a certificate in Karvan's store (uploaded, from Let's Encrypt, or found in a panel's ENV). A wrong certificate or a closed port must never lock the operator out.

## Decision
- Three modes, stored in the settings table next to the address (`http.tls`, `http.tls_domain`, `http.tls_shared`), never in `config.json`: `off` (plain HTTP), `self` and `domain`. An installation made before this reads back `off`. `karvan init` starts a new one in `self`.
- `self`: Karvan makes an ECDSA P-256 certificate for this server's addresses (no link-local ones) and `localhost`, valid 397 days, and makes a new one 30 days before the end. Its key is sealed with the key file (AAD `settings/http.tls_self_key`). `init` and `reset` print its SHA-256 fingerprint so the operator can compare it with the browser's warning. A new one that cannot be stored leaves the old one in use and is tried again an hour later.
- `domain`: the operator picks a registered domain whose certificate covers its name. Each handshake reads the domain's certificates row and serves it through the bridge's certificate cache, which reads it again when the row or its file changes, so a renewal needs no restart.
- When the domain's certificate cannot be loaded (file gone, key mismatch), the error is logged once and the panel serves the last certificate it loaded for that domain; with none, its self-signed one, with a warning. A failed handshake is a debug line.
- A change of mode or port is an address change (ADR 0009): `PUT /api/v1/settings/https` asks for the password and code again, serves the new address next to the old one for 10 minutes, and only a confirm from the new address saves it. On the same port both schemes are served while the change waits: the first byte of each connection (0x16 for TLS) picks HTTP or TLS, and a connection silent for 10 seconds is closed without holding up others.
- HSTS (`max-age=86400`, no `includeSubDomains`) is sent only in `domain` mode, by the listener, for the panel's own paths. A trusted proxy that says `https` no longer brings HSTS.
- The listener speaks TLS 1.2 and up, HTTP/1.1 only.

## Alternatives considered
- **Certificate files named in `config.json`.** A second certificate system next to the store, and a restart for every change.
- **Only domain mode.** IP-only installs would stay on plain HTTP.
- **A new port for every change of scheme.** The operator's firewall often has only the panel's port open; the first-byte check keeps the same port usable.
- **HSTS for a year in every HTTPS mode.** A self-signed certificate under HSTS cannot be clicked through, and going back to HTTP would lock browsers out.

## Consequences
- New installs get encryption from the first sign-in, with a browser warning the fingerprint answers.
- Cookies follow the scheme (`__Host-s` over HTTPS), so the operator signs in again on the new address before confirming.
- A domain whose certificate breaks still reaches the panel, with a browser warning, instead of not at all.
- The shared bridge port (`shared`) is checked here (`port_mixed`) and served by the bridge (ADR 0011).
