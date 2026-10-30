# Karvan releases
Each release on GitHub has, for amd64 and arm64:
- `karvan_<version>_linux_<arch>.tar.gz`: the binary with the web panel built in, `install.sh`, `get.sh`, the systemd unit and the docs.
- `get.sh`: the one-liner that downloads the archive, checks it and installs it.
- `SHA256SUMS`: the SHA-256 checksum of each archive and of `get.sh`.

The image is `ghcr.io/pdyrust/karvan:<version>` for linux/amd64 and linux/arm64 (see [docker.md](docker.md)). Downloads and image pulls need no GitHub login.

Every archive, `SHA256SUMS`, `get.sh` and the image carry a GitHub artifact attestation from the release workflow. Why there is no signing key: [ADR 0012](adr/0012-release-integrity.md).
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
It builds `0.5.0-beta-dry` into `dist/` (both archives, `get.sh`, `SHA256SUMS`, both images loaded into the local Docker, nothing pushed), checks the checksums and that a changed byte fails them, checks the attestation subject list (`scripts/release/subjects.sh`), looks for tool names in the unpacked archives, then runs the image check and the install test on the host's architecture. It needs Docker. Attestations need GitHub's OIDC token, so only the release workflow makes them.

`VERSION=<version> scripts/release/build.sh` builds the release files alone; `scripts/release/verify.sh dist` checks them.
## The workflows
- `.github/workflows/ci.yml` runs `make check-full` on every push to `main` and every pull request. It can only read the repository.
- `.github/workflows/release.yml` runs when a `v*` tag is pushed. It builds the release files, runs the install test on the amd64 archive, pushes `ghcr.io/pdyrust/karvan:<version>` (plus `<version>-amd64` and `<version>-arm64`, the two images it joins), attests every file and image, checks the attestations with `gh attestation verify`, and creates the GitHub release with the archives, `get.sh` and `SHA256SUMS`. A tag with a `-`, such as `v0.5.0-beta`, makes a pre-release. The release notes are the tag's section of `CHANGELOG.md` (`## 0.5.0-beta`); without that section the workflow stops before it pushes anything.
- Run it by hand from the Actions tab (Release, Run workflow) to try it: with `dry_run` on, the default, it builds, tests and attests the files, builds both images without pushing them, and makes no release. The run's summary lists the attested files and digests.
- `make ci-lint` checks both workflows with actionlint and every shell script with shellcheck, without pushing.
Every action in the workflows is pinned by its full commit SHA. No secret is stored: the workflows use the token GitHub gives each run.
After the first release, open the `karvan` package on GitHub (your profile, Packages, Package settings) and check that its visibility is public. A new package can start out private, and then `docker pull` asks for a login.

## Release notes for v0.5.0-beta (draft)

The workflow takes the notes from the `## 0.5.0-beta` section of `CHANGELOG.md`, so that section is the draft: its first line says beta. Before the tag:

- the release dry run on GitHub (Actions, Release, Run workflow) passes,
- `docs/benchmarks.md` holds a passing bridge run on a real server,
- the owner has read every Persian string.
