#!/bin/sh
# Prints the attestation subjects of the release in DIR (default dist/), one
# per line: the path of both archives and SHA256SUMS, then the image
# of each architecture as <name>@sha256:<digest>, read from DIR/image-digests
# (lines "linux/<arch> <name>@sha256:<digest>", written by the image build).
# Fails, printing nothing, when any subject is missing (D33).
set -eu
DIR=${1:-$(cd "$(dirname "$0")/../.." && pwd)/dist}
fail() {
	echo "subjects: $*" >&2
	exit 1
}
files=
for arch in amd64 arm64; do
	set -- "$DIR"/karvan_*_linux_"$arch".tar.gz
	{ [ $# = 1 ] && [ -f "$1" ]; } || fail "want one $arch archive in $DIR"
	files="$files$1
"
done
[ -f "$DIR/SHA256SUMS" ] || fail "no SHA256SUMS in $DIR"
files="$files$DIR/SHA256SUMS
"
[ -f "$DIR/image-digests" ] || fail "no image-digests in $DIR"
images=
for arch in amd64 arm64; do
	ref=$(awk -v p="linux/$arch" '$1 == p { print $2 }' "$DIR/image-digests")
	echo "$ref" | grep -Eqx '[a-z0-9./_-]+@sha256:[0-9a-f]{64}' || fail "no image digest for linux/$arch"
	images="$images$ref
"
done
printf '%s%s' "$files" "$images"
