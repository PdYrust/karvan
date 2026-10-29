#!/bin/sh
# Downloads a Karvan release from GitHub, checks it against SHA256SUMS and
# runs its install.sh with the remaining arguments. On any failure nothing
# is installed.
set -eu

BASE_URL=https://github.com/PdYrust/karvan/releases/download
# The release build writes its version here; --version overrides it.
VERSION=

die() {
	echo "$*" >&2
	exit 1
}

[ "$(id -u)" = 0 ] || die "run it as root, for example: curl -fsSL <get.sh address> | sudo sh"
for tool in curl sha256sum tar; do
	command -v "$tool" >/dev/null 2>&1 || die "$tool is missing: install it first (packages curl, coreutils, tar)"
done
case $(uname -m) in
x86_64 | amd64) arch=amd64 ;;
aarch64 | arm64) arch=arm64 ;;
*) die "this machine is $(uname -m): Karvan has builds for amd64 and arm64 only" ;;
esac

if [ "${1:-}" = --version ]; then
	[ $# -ge 2 ] || die "--version needs a version, for example 0.5.0-beta"
	VERSION=${2#v}
	shift 2
fi
case $VERSION in
'') die "no version given: run it with --version <version>" ;;
[0-9]*[!0-9A-Za-z.+-]* | [!0-9]*) die "not a version: $VERSION" ;;
esac

name=karvan_${VERSION}_linux_${arch}
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT
trap 'exit 1' INT TERM

echo "Downloading Karvan $VERSION for $arch..."
curl -fsSL --retry 3 -o "$tmp/SHA256SUMS" "$BASE_URL/v$VERSION/SHA256SUMS" || die "could not download SHA256SUMS for $VERSION"
curl -fsSL --retry 3 -o "$tmp/$name.tar.gz" "$BASE_URL/v$VERSION/$name.tar.gz" || die "could not download $name.tar.gz"
awk -v f="$name.tar.gz" '$2 == f { n++ } END { exit n != 1 }' "$tmp/SHA256SUMS" || die "SHA256SUMS does not list $name.tar.gz; nothing was installed"
(cd "$tmp" && sha256sum -c --ignore-missing --quiet SHA256SUMS) || die "the download does not match SHA256SUMS; nothing was installed"

tar -xzf "$tmp/$name.tar.gz" -C "$tmp"
[ -f "$tmp/$name/install.sh" ] || die "the archive has no $name/install.sh; nothing was installed"
sh "$tmp/$name/install.sh" "$@"
