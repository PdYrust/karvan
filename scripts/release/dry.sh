#!/bin/sh
# The local release dry run (D35): builds VERSION (default 0.5.0-beta-dry)
# into dist/, checks SHA256SUMS and that a changed byte fails the check,
# builds the image for both architectures with --load (nothing is pushed),
# checks the attestation subject list, looks for tool names in the unpacked
# archives, then runs the image check and the install test on the host's
# architecture. Needs Docker. The attestations themselves need GitHub's
# OIDC token and are made by the release workflow only.
set -eu
ROOT=$(cd "$(dirname "$0")/../.." && pwd)
VERSION=${VERSION:-0.5.0-beta-dry}
export VERSION
OUT=$ROOT/dist
export OUT
REL=$ROOT/scripts/release
work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT
trap 'exit 1' INT TERM
fail() {
	echo "FAIL: $*" >&2
	exit 1
}
step() { echo "== $*"; }
case $(uname -m) in
x86_64 | amd64) host=amd64 ;;
aarch64 | arm64) host=arm64 ;;
*) fail "no Karvan build for $(uname -m)" ;;
esac
step "build $VERSION"
"$REL/build.sh"
step "check SHA256SUMS"
"$REL/verify.sh" "$OUT"
cp "$OUT"/*.tar.gz "$OUT/get.sh" "$OUT/SHA256SUMS" "$work/"
f=$work/karvan_${VERSION}_linux_amd64.tar.gz
b=$(od -An -tu1 -j 100 -N 1 "$f" | tr -d ' ')
printf '%b' "\\0$(printf '%03o' $(((b + 1) % 256)))" | dd of="$f" bs=1 seek=100 conv=notrunc status=none
if "$REL/verify.sh" "$work" >/dev/null 2>&1; then fail "verify.sh accepted an archive with a changed byte"; fi
echo "ok: a changed byte in $(basename "$f") fails the check"
step "build the images (no push)"
: >"$OUT/image-digests"
for arch in amd64 arm64; do
	tag=karvan-release-dry:$VERSION-$arch
	docker buildx build --load -q --platform "linux/$arch" -t "$tag" --build-arg VERSION="$VERSION" \
		-f "$ROOT/packaging/docker/Dockerfile" "$OUT/image" >/dev/null
	[ "$(docker image inspect -f '{{.Architecture}}' "$tag")" = "$arch" ] || fail "$tag is not $arch"
	echo "linux/$arch karvan-release-dry@$(docker image inspect -f '{{.Id}}' "$tag")" >>"$OUT/image-digests"
done
cat "$OUT/image-digests"
step "check the attestation subjects"
"$REL/subjects.sh" "$OUT" >"$work/subjects"
cat "$work/subjects"
{
	ls "$OUT"/karvan_*_linux_*.tar.gz
	echo "$OUT/SHA256SUMS"
	echo "$OUT/get.sh"
	awk '{ print $2 }' "$OUT/image-digests"
} | sort >"$work/want"
sort "$work/subjects" | diff "$work/want" - || fail "the subject list differs from the release files"
echo "ok: every archive, SHA256SUMS, get.sh and both image digests"
step "no tool names in the archives"
mkdir "$work/unpacked"
for a in "$OUT"/*.tar.gz; do tar -xzf "$a" -C "$work/unpacked"; done
# The same names the floor check bans in the tree.
names=$(sed -n "s/^report ai-reference '\([^']*\)'.*/\1/p" "$ROOT/scripts/check-floor.sh")
[ -n "$names" ] || fail "no name list in scripts/check-floor.sh"
if grep -rnaE "$names" "$work/unpacked" "$OUT/get.sh"; then fail "tool names in the release files"; fi
echo "ok: none"
step "image check on linux/$host"
IMAGE=karvan-release-dry:$VERSION-$host "$ROOT/scripts/dockertest/run.sh"
step "install test on linux/$host"
ARCHIVE=$OUT/karvan_${VERSION}_linux_$host.tar.gz "$ROOT/scripts/installtest/run.sh"
echo "release dry run: PASS"
