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
