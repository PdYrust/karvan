# Changelog

## 0.5.0-beta

Beta. The first beta of Karvan. Try it on a test server or with Safe Mode on before you use it on a panel with real users.

### Install and run

- One-line install from GitHub Releases for amd64 and arm64: `get.sh` checks the archive against `SHA256SUMS`, installs Karvan as a systemd service and prints the address, username and password once.
- `karvan update`, `karvan rollback` and `karvan uninstall` (with `-purge` to delete the data too). An update backs up first and goes back by itself when the new version is not healthy within 60 seconds.
- A Docker image, `ghcr.io/pdyrust/karvan:0.5.0-beta`, with only the binary in it. See `docs/docker.md`.
- Karvan runs as root. Its data folder, config and key are open to root only.

### Panel

- HTTPS for the panel: its own certificate (the default for new installs), a domain's certificate, or off. A change is tried at the new address and saved only after you confirm it there.
- The panel can share a port such as 443 with the subscription links.
- The rule form suggests a free port for old links, 443 when it is free, and never a panel's port. `karvan ports` lists the ports in use.
- After a session ends, reloading any page under the panel's path shows the sign-in page and returns you to that page after you sign in.

### Fixes

- A migration of more than 32766 users no longer fails its verify step on SQLite.
- Old links are answered faster under load: the bridge prepares its queries once and keeps database connections open.

### Checks

- Every archive, `SHA256SUMS`, `get.sh` and the image carry a GitHub artifact attestation; `gh attestation verify` checks them (optional).
- Benchmarks are in `docs/benchmarks.md`.
