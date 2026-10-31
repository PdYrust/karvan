<p align="center">
  <img src="docs/brand/karvan-mark.svg" width="112" height="112" alt="Karvan">
</p>

<h1 align="center">Karvan</h1>

<p align="center">Move and merge users between Marzban and PasarGuard panels.</p>

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
**Without questions** (the installer asks how the panel is served; `--https ip` or `domain` answers it):
```sh
sudo bash -c "$(curl -fsSL https://github.com/PdYrust/karvan/raw/main/packaging/get.sh)" @ install --https domain --domain panel.example.com --yes
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

The panel always answers HTTPS on its port. On the server's IP it uses Karvan's own certificate.

To open it on a domain with that domain's certificate, add the domain on the Domains page with its certificate (from Let's Encrypt, a file on the server, a pasted one, or one found in a panel), then choose Use for Karvan in the domain's "..." menu. A browser asking for that name gets the domain's certificate at once, and renewals reach it without a restart. Every other name and the server's IP keep Karvan's own certificate, so the IP address always opens. Karvan only reads the certificate, so other services that use it keep working. One domain at a time is used for Karvan; Stop using for Karvan goes back to the IP alone.

Domain settings, on the same page, say what the Karvan domain does while it has no certificate: Karvan's own certificate (the default), or plain HTTP for that name only.

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
