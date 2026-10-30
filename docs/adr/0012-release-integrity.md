# ADR 0012: Release integrity with checksums and GitHub artifact attestations
## Status
Accepted, 2026-10-29
## Context
Karvan runs as root and is installed with a one-liner (`get.sh`) or pulled as an image from `ghcr.io`. A user needs a way to tell that an archive or an image is the one this repository built, and the one-liner must refuse a broken or changed download. The owner wants nothing to store, rotate or leak: no signing key and no repository secret (owner answer Q2). The repository is public.
## Decision
- Every release has `SHA256SUMS` over its two archives, made by `scripts/release/build.sh`; the release carries only those three files, and `get.sh` lives inside each archive and on `main` (`packaging/get.sh`), not as a release file. `get.sh` downloads `SHA256SUMS` from the same release and checks the archive with `sha256sum` before anything is unpacked; a missing line or a mismatch stops with nothing installed. `scripts/release/verify.sh` makes the same check on a release folder.
- The release workflow attests both archives, `SHA256SUMS` and the image digest of each architecture with `actions/attest-build-provenance`: keyless Sigstore signing with the workflow's OIDC token (`id-token: write`, `attestations: write`), the image attestation pushed to the registry next to the image. The subject list comes from `scripts/release/subjects.sh`, which fails when any subject is missing.
- Users check `SHA256SUMS` (the one-liner does it for them) and may also run `gh attestation verify` (docs/release.md), which proves the file was built by this repository's release workflow at a given commit.
- The archives are packed the same way every time (sorted names, root owner, fixed modes, the commit time, gzip without a name or time), so a rebuild of the same commit can be compared.
## Alternatives considered
- **A GPG or minisign key.** A secret in the repository settings that has to be kept, rotated and published, and a key that users must fetch from somewhere they trust. Rejected by the owner.
- **cosign with a stored key.** The same key problem as above. Keyless cosign is what the attestation action already does, without an extra tool.
- **Checksums only.** They catch a broken download, but whoever can change the release files can change `SHA256SUMS` too. Kept as the first check, not the only one.
## Consequences
- Nothing secret exists to leak. Trust rests on GitHub's OIDC identity of the release workflow and on Sigstore's public log.
- An attestation needs GitHub's OIDC token, so it cannot be made on a developer machine. `make release-dry` builds both archives, `SHA256SUMS` and both images, checks the checksums (and that one changed byte fails them) and checks the subject list; the attestation step itself is checked by `make ci-lint` and, before the first tag, by `gh attestation verify` on a `workflow_dispatch` dry run.
- `get.sh` checks the checksum but not the attestation, since `gh` is rarely installed on a server. The attestation check is one optional command for those who want it.
