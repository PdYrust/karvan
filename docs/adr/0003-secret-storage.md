# ADR 0003: How Karvan stores secrets

## Status
Accepted, 2026-10-16

## Context
Karvan will hold panel tokens, database passwords and SSH keys. A stolen database copy (a backup, a careless `scp`) must not be enough to read them.

## Decision
- **One 256-bit key in its own file** (`/etc/karvan/secret.key`, mode 0600, directory 0700), outside the data directory. Config validation refuses a key path inside the data directory. Karvan refuses to load a key file other users can read.
- **XChaCha20-Poly1305** for every stored secret. Nonces are random, 24 bytes, so there is no counter to keep. The associated data names `table/column/row`, so a ciphertext copied to another row or column fails to open. A one-byte version prefix leaves room for key rotation.
- **Passwords are hashed, not encrypted:** argon2id, 64 MiB, 3 passes, 2 lanes, stored as PHC strings so parameters can be raised. At most two hashes run at once.
- **Session tokens are never stored.** The database keeps their SHA-256; the CSRF token is an HMAC of the session token, so it is never stored either.
- **Secrets in memory are `secret.Value`**, which prints, logs and JSON-encodes as `[redacted]`.
- **One-time values** (the first password, the hidden path) are printed once by `karvan init` or `karvan reset` on stdout, never by `serve`, so they cannot land in the service journal.

## Alternatives considered
- **AES-256-GCM:** equally good with a careful nonce scheme; random 96-bit nonces would need a rotation policy at high volume. XChaCha's 192-bit nonce removes that concern.
- **Key in an environment variable:** ends up in process listings, crash dumps and systemd unit files.
- **Key derived from the operator's password:** the service would need the password to start after a reboot.

## Consequences
- Losing the key file makes every stored secret unreadable. Backups (phase 7) must save it separately from the database and say so plainly.
- `init` never overwrites an existing key file, and it removes what it created if it fails part way.
