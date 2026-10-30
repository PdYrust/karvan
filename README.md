<p align="center">
  <img src="docs/brand/karvan-mark.svg" width="112" height="112" alt="Karvan">
</p>

<h1 align="center">Karvan</h1>

<p align="center">Move and merge users between Marzban and PasarGuard panels.</p>

---

Karvan ships as one binary with a bilingual (Persian and English) web panel.

> Status: beta (0.5.0-beta). Try it on a test server or with Safe Mode on before you use it on a panel with real users.

## Install
On a Linux server (amd64 or arm64) with systemd:

**Newest release** (pre-releases included):
```sh
sudo bash -c "$(curl -fsSL https://github.com/PdYrust/karvan/raw/main/packaging/get.sh)" @ install
```
**A chosen version:**
```sh
sudo bash -c "$(curl -fsSL https://github.com/PdYrust/karvan/raw/main/packaging/get.sh)" @ install --version 0.5.0-beta
```
**Update** (newest release, or add `--version`):
```sh
sudo bash -c "$(curl -fsSL https://github.com/PdYrust/karvan/raw/main/packaging/get.sh)" @ update
```
**Remove** (keeps the data, config and backups; `--purge` also deletes them after you type to confirm):
```sh
sudo bash -c "$(curl -fsSL https://github.com/PdYrust/karvan/raw/main/packaging/get.sh)" @ uninstall
```
### After installation
- The install downloads the release archive, checks it against `SHA256SUMS` and installs Karvan as a systemd service. If the download or a check fails, nothing is installed.
- At the end it prints the panel address, username and password once. Save them.
- The panel starts on HTTPS with its own certificate. The browser warns about it once; go on only if it shows the fingerprint the install printed.
- Files: `/usr/local/lib/karvan`. Config: `/etc/karvan/config.json`. Data: `/var/lib/karvan`.

Karvan runs as root: it reads and writes the panels' databases and folders, and stops and starts their containers for a Safe Mode restore. Its data folder `/var/lib/karvan` is open to root only, and `/etc/karvan/config.json` and `secret.key` too.
## Update, roll back, remove with the karvan command
```sh
sudo karvan update        # newest release; your data and settings stay
sudo karvan rollback      # back to the version before the last update
sudo karvan uninstall     # removes Karvan, keeps its data, config and backups
sudo karvan uninstall -purge   # also deletes them, after you type to confirm
```
An update backs up the data first. If the new version does not start healthy within 60 seconds, it goes back to the old one by itself.
## Docker

The image `ghcr.io/pdyrust/karvan:0.5.0-beta` holds only the binary. Volumes, ports and the Docker socket are explained in [docs/docker.md](docs/docker.md). Read its warning about the socket before you start it.

## HTTPS for the panel

Settings, HTTPS: keep the self-signed certificate, pick one of your domains with a certificate (uploaded, from Let's Encrypt or found in a panel), or turn HTTPS off. The panel can also share a port with your subscription links, such as 443. Every change is tried at the new address first and saved only after you confirm it there, so a wrong choice does not lock you out.

## Check a download (optional)

The install checks the checksum by itself. To also check that a file was built by this repository's release workflow, with the [GitHub CLI](https://cli.github.com/):

```sh
gh attestation verify karvan_0.5.0-beta_linux_amd64.tar.gz -R PdYrust/karvan
```

More in [docs/release.md](docs/release.md).

## Build from source

Requirements: Go 1.27, Node.js 24, pnpm 12.

```sh
make build        # web panel + bin/karvan
make check        # lint, tests, coverage and floor rules
```

First run on a server:

```sh
karvan init       # prints the address, username and password once
karvan serve
```

Lost access? `karvan reset -password` (also `-user`, `-path`, `-port`, `-totp`).

## Contributing

Design decisions are recorded in [docs/adr](docs/adr).

## License

[GNU Affero General Public License v3.0](LICENSE)
