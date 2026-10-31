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

## Release notes 0.5.10-beta
Karvan 0.5.10-beta fixes the panel switcher at the top of the sidebar, which left out All panels when it was the current choice and only one panel was added; it now always lists All panels and every panel, with the current one checked.
## Release notes 0.5.9-beta
Karvan 0.5.9-beta can make the Karvan domain the only way into the panel, serves every run from one old panel with one rule, opens without a flash of the sign-in page, and tidies the domain page and the panel cards on phones.
- Domain settings have a new switch, Only open Karvan on its domain, on by default. While a domain is used for Karvan, only that name opens the panel, with whatever certificate the domain has: a handshake for another name or for the server's IP is refused, and a request for another host gets the plain 404. The subscription bridge is not affected.
- The switch holds only after your first sign-in through the domain since the domain or the switch last changed. Until then the server's IP keeps opening the panel, and the domain's row says it waits for that sign-in.
- If the domain's certificate ends or its files go missing, the server's IP opens the panel again with Karvan's own certificate. Karvan logs and audits why, and the overview and the domain say so. The overview also warns when the Karvan domain's certificate ends within 14 days and nothing renews it.
- On the server itself, 127.0.0.1 and ::1 always open the panel, so an SSH tunnel works. `karvan access open-ip` turns the switch off from the server, and a running Karvan takes it within a few seconds. The install result names it.
- Domain settings save each change at once, with no Save button. The choice for a Karvan domain without a certificate is one switch, Plain HTTP for a Karvan domain without a certificate, off by default (Karvan's own certificate).
- A domain's page, and the certificate of a rule, no longer say "No problems found" next to a state that already says it. Problems still show when there are some.
- On phones, a panel's card on Panels reads like the other list cards: the logo, the name and the "..." menu, then the version and the database on one quiet line, then its status. The database's address is on the panel's page.
- Opening Karvan as an installed app, or reloading a page, no longer shows the sign-in page for a moment before the panel. The page stays in the panel's background colour until the session answers, and only a signed-out answer shows the sign-in form. Signing in still returns to the page you asked for.
- Batches shows each old panel once, with its runs inside: the old address, the new panel, the number of runs and users, the latest run, and the rules that serve it, or Set up old links. Show all opens every run of that old panel. Each run stays its own batch, with its own page, so a backup restore still stops only the runs after its backup.
- A rule made from an old panel serves every run from it, the runs after the rule was saved too, with no change to the rule. Rules made from batches before this version now serve every active run of their old panel. Users in an inactive or deleted run still get the plain 404, and another old panel's users are never served.
- The rule form picks one old panel instead of batches, and a rule's card links to its old panel's runs. When a finished migration's old panel already has a rule, the result offers to open that rule instead of a new one. The last run of an old panel that a rule serves cannot be deleted until the rule is.
- Connect offers the old panel addresses used before, so the next run of the same panel is typed the same way. An address with a trailing dot, leading zeros in the port or repeated slashes now names the same old panel, and names stored before are renamed at start.
- The Summary step no longer shows the old panel's protocols, which could be wrong for custom inbound settings.
- When the old panel's groups cannot be read, Compare skips the protocol checks without a notice. The server logs it.
## Release notes 0.5.8-beta
Karvan 0.5.8-beta picks the panel's certificate by the name the browser asks for, so the server's IP always opens.
- Settings no longer has an HTTPS tab. A domain's "..." menu on the Domains list has Use for Karvan, or Stop using for Karvan while Karvan uses that domain, and the list marks it with Karvan. One domain is used at a time; using another switches to it. The change applies at once, without a restart or a confirm, and is audited.
- The panel's port always answers HTTPS. A browser that asks for the Karvan domain's name gets that domain's certificate, of any kind: Let's Encrypt, a file on the server, a pasted one, one found in a panel, or a self-signed one given to the domain. Renewals reach it without a restart. Every other name and the server's IP get Karvan's own certificate.
- In 0.5.7, a domain used for the panel had its certificate served to every visitor, the server's IP included, so the address by IP showed a certificate for another name. Now the IP keeps Karvan's own certificate.
- Domain settings, the former Let's Encrypt settings, say next to Keep port 80 open what the Karvan domain does while it has no certificate: Karvan's own certificate (the default) or plain HTTP for that name only. HTTPS on the same port keeps working either way.
- The panel no longer sends HSTS, so a domain name that falls back to Karvan's own certificate, and other services on the same name over plain HTTP, still open in the browser.
- Plain HTTP on the panel's port is sent on to HTTPS on the same port, except for the Karvan domain with plain HTTP chosen, and for requests from the server itself or a trusted proxy, where TLS ends in front of the panel.
- An update keeps what was set: a panel on a domain's certificate has that domain used for Karvan, one on Karvan's own certificate uses none, and one on plain HTTP answers HTTPS on the same port, where its old address is sent on.
- The domain used for Karvan can be deleted once Karvan stops using it.
- Install: `--https domain` adds the domain, gets its Let's Encrypt certificate and uses it for Karvan, also when the certificate comes later. `--https none` is gone.
- The overview's plain HTTP warning links to the same address over HTTPS.

## Release notes 0.5.7-beta
Karvan 0.5.7-beta can serve its own panel with a domain's certificate and copy certificates to a panel.
- The "..." menu of a domain has Use for the Karvan panel, or Stop using for the Karvan panel while the panel uses that domain. It starts the same HTTPS change as the settings: the new address waits 10 minutes for a confirm from there. Karvan only reads the certificate, so other services that use it keep working, and renewals reach the panel as before. A domain without a certificate for its name cannot be picked, and the menu says why.
- A domain's name on the Domains list opens its own page: the certificate (issuer, names, end date, problems and where it came from), the next renewal and the last error for Let's Encrypt, the rules that use it, and whether it serves the Karvan panel. Renew and Change certificate stay in the "..." menu of the list.
- The "..." menu of a certificate no longer has Edit; Change certificate on its domain does the same. It has Copy to a panel: the Let's Encrypt live folder or a Marzban or PasarGuard panel. Karvan copies (never moves) fullchain.pem and privkey.pem into a new folder named after the main domain, /var/lib/<panel folder>/certs/<domain>/ or /etc/letsencrypt/live/<domain>/, then shows the two paths. The panel's own settings must point to them; Karvan never changes a panel's ENV.
- After each renewal or change of the certificate, Karvan copies it to the same folders again, and logs and audits each copy. Each copy is listed on its certificate with Stop copying. Karvan replaces only files it wrote itself and refuses links, so a certbot folder or a file of the panel stays as it is.
- The systemd unit lets Karvan write to /etc/letsencrypt, for copies to the live folder.
- During a change between two HTTPS modes on the same port, a browser asking for the domain's name reaches the address with that domain's certificate, so the change can be confirmed there.
- A ready panel card shows only Ready. Panel control shows as a warning only when it is not available.
- Safe Mode now puts a backup back on TimescaleDB reliably.

## Release notes 0.5.6-beta
- Migrations and Batches no longer show the search and filters for a moment before the empty card. They show the loading skeleton until the list arrives, as Domains and Rules do.
- A panel's page no longer has a "..." menu. Its actions are in the "..." menu on the panel's card on the Panels list.
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
