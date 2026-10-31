# Karvan releases
Each release on GitHub has, for amd64 and arm64:
- `karvan_<version>_linux_<arch>.tar.gz`: the binary with the web panel built in, `install.sh`, `get.sh`, the systemd unit and the docs.
- `SHA256SUMS`: the SHA-256 checksum of each archive.

The image is `ghcr.io/pdyrust/karvan:<version>` for linux/amd64 and linux/arm64 (see [docker.md](docker.md)). Downloads and image pulls need no GitHub login.

Every archive, `SHA256SUMS` and the image carry a GitHub artifact attestation from the release workflow. Why there is no signing key: [ADR 0012](adr/0012-release-integrity.md).
## Check a download
`get.sh` checks the checksum by itself. To check an archive you downloaded by hand, put it next to `SHA256SUMS` and run:
```sh
sha256sum -c --ignore-missing SHA256SUMS
```
It must print `OK` for the archive. Anything else means the file is broken or changed: do not install it.
## Check where it was built (optional)
With the [GitHub CLI](https://cli.github.com/) installed:
```sh
gh attestation verify karvan_0.5.0-beta_linux_amd64.tar.gz -R PdYrust/karvan
gh attestation verify oci://ghcr.io/pdyrust/karvan:0.5.0-beta -R PdYrust/karvan
```
Both must end with a verified attestation from `PdYrust/karvan`'s release workflow. This proves the file was built by that workflow from this repository, at the commit the attestation names.
## Build a release locally
```sh
make release-dry
```
It builds `0.5.0-beta-dry` into `dist/` (both archives, `SHA256SUMS`, both images loaded into the local Docker, nothing pushed), checks the checksums and that a changed byte fails them, checks the attestation subject list (`scripts/release/subjects.sh`), looks for tool names in the unpacked archives, then runs the image check and the install test on the host's architecture. It needs Docker. Attestations need GitHub's OIDC token, so only the release workflow makes them.

`VERSION=<version> scripts/release/build.sh` builds the release files alone; `scripts/release/verify.sh dist` checks them.
## The workflows
- `.github/workflows/ci.yml` runs every target of `make check-full` on every push to `main` and every pull request, split into parallel jobs (lint, Go tests, database tests, panel tests, Let's Encrypt tests, web); the job `check-full` passes when all of them passed. A run takes about 15 minutes, the length of the race test job. It can only read the repository.
- `.github/workflows/release.yml` runs when a `v*` tag is pushed, split into jobs: `version` picks the version and reads the notes, `web` builds the panel once, `build` packs the amd64 and arm64 archives in parallel from that panel, `install-test` installs the amd64 archive, `image` builds the two images in parallel, `index` joins them, and `release` writes `SHA256SUMS` and does the attestations and the release. A run takes about 10 minutes, most of it the install test. It builds the release files, runs the install test on the amd64 archive, pushes `ghcr.io/pdyrust/karvan:<version>` (plus `<version>-amd64` and `<version>-arm64`, the two images it joins), attests every file and image, checks the attestations with `gh attestation verify`, and creates the GitHub release with the two archives and `SHA256SUMS`. A tag with a `-`, such as `v0.5.0-beta`, makes a pre-release. The release notes are the tag's section of this file (`## Release notes 0.5.0-beta`, below); without that section the workflow stops before it pushes anything.
- Run it by hand from the Actions tab (Release, Run workflow) to try it: with `dry_run` on, the default, it builds, tests and attests the files, builds both images without pushing them, and makes no release. The run's summary lists the attested files and digests.
- `make ci-lint` checks both workflows with actionlint and every shell script with shellcheck, without pushing.
Every action in the workflows is pinned by its full commit SHA. No secret is stored: the workflows use the token GitHub gives each run.
After the first release, open the `karvan` package on GitHub (your profile, Packages, Package settings) and check that its visibility is public. A new package can start out private, and then `docker pull` asks for a login.

## Before the tag

- the release dry run on GitHub (Actions, Release, Run workflow) passes,
- `docs/benchmarks.md` holds a passing bridge run on a real server,
- a native Persian speaker has read every Persian string.

## Release notes 0.5.5-beta
Karvan 0.5.5-beta names PasarGuard 5.5.1 correctly and tidies a few pages.
- A panel on PasarGuard 5.5.1 showed "unsupported version" while it was ready. It now shows PasarGuard 5.5.1. A version that passes the schema check but has no known name shows "version not known by name"; "unsupported" is only for a database the check refuses.
- The footer shows the Karvan version and a Telegram link.
- Migrations and Batches with nothing in them show only the empty card, without the search and filters.
- A panel's page has the same "..." menu as its card on the Panels list, which now also has Check again.
- In the sidebar, the Analytics sub-pages show the server icon and each panel's logo.
## Release notes 0.5.4-beta
Karvan 0.5.4-beta fits wide screens. Every page now sits centered next to the sidebar and grows up to 1760 px, so a Full HD or QHD screen no longer leaves a large empty area on one side.
- On wide screens the analytics charts and the Overview cards fill more columns. Forms, Settings and the wizard keep a readable width and are centered too.
## Release notes 0.5.3-beta
Karvan 0.5.3-beta opens every page on a fresh install and is tested on PasarGuard 5.5.1.
### Fixes
- On an install with no domains or certificates yet, the Domains and certificates page showed an error instead of opening. Empty lists now come back as empty lists, never as null, and the other new pages (Rules, Batches, the HTTPS settings, Add domain) open on an empty install too.
- PasarGuard 5.5 ties each admin token to the admin's password. Karvan's tokens for the reload after a move now carry that tie, so PasarGuard 5.5 no longer refuses them.
- PasarGuard 5.5.1 is a known version: the write, check and undo tests run on it as well as on 5.4.1, on every database server, and the live panel tests run on it.
## Release notes 0.5.2-beta

Karvan 0.5.2-beta works with PasarGuard 5.5.

### Fixes
- PasarGuard 5.5 added passkey sign-in for admins, with the tables `admin_passkeys` and `passkey_challenges`. Karvan no longer reports them as an unknown add-on, and undoing a move removes the passkeys of the admins it created.

## Release notes 0.5.1-beta

Karvan 0.5.1-beta follows 0.5.0-beta with installer and discovery fixes.

### Install
- The installer asks how the panel is served: plain HTTP, HTTPS on the server address with Karvan's own certificate, or HTTPS on a domain with a Let's Encrypt certificate. Unattended installs take the same choice as flags (`--https`, `--domain`, `--email`, `--port`, `--yes`).
- The result shows the server's address instead of a placeholder, and the firewall command when a firewall is on.
- Install, update and uninstall show each step with its result, in colour only on a terminal.

### Fixes
- Plain HTTP on the panel's HTTPS port now sends you on to HTTPS under the hidden path.
- Copies of a panel folder, such as the ones a panel update leaves behind, are no longer listed as panels.
- Long names no longer make the sidebar scroll sideways.

### Panel versions
- Karvan now works on Marzban and PasarGuard versions it does not know by name when their database has every table and column it reads and writes. When something it needs is missing or changed, it stops and names the table and column.

## Release notes 0.5.0-beta

Beta. The first beta of Karvan. Try it on a test server or with Safe Mode on before you use it on a panel with real users.

### Install and run

- One-line install for amd64 and arm64: `sudo bash -c "$(curl -fsSL https://github.com/PdYrust/karvan/raw/main/packaging/get.sh)" @ install` takes the newest release (`--version` picks one; `@ update` and `@ uninstall [--purge]` too). It checks the archive against `SHA256SUMS`, installs Karvan as a systemd service and prints the address, username and password once.
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

- Every archive, `SHA256SUMS` and the image carry a GitHub artifact attestation; `gh attestation verify` checks them (optional).
- Benchmarks are in `docs/benchmarks.md`.
