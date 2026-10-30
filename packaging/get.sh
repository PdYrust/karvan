#!/bin/sh
# Installs, updates or removes Karvan. Run it from the repository the way
# the README shows:
#   sudo bash -c "$(curl -fsSL <raw get.sh address>)" @ install [--version V]
#   ... @ update [--version V] | @ uninstall [--purge]
# A bare run means install. With no version it takes the newest release,
# pre-releases included. The download is checked against SHA256SUMS and on
# any failure nothing is installed. The copy in a release has that release's
# version stamped in, so piping it to sh installs that version. karvan update
# runs the installed copy with --update, and with --archive FILE to use a
# release archive that is already on this machine.
set -eu

BASE_URL=https://github.com/PdYrust/karvan/releases/download
# With no version the newest release listed here is taken (GitHub's latest
# skips pre-releases).
RELEASES_URL="https://api.github.com/repos/PdYrust/karvan/releases?per_page=1"
# The release build writes its version here; --version overrides it, and an
# update without --version ignores it.
VERSION=
INSTALLED=/usr/local/lib/karvan/current/install.sh

die() {
	echo "$*" >&2
	exit 1
}

[ "$(id -u)" = 0 ] || die "run it as root, for example: sudo bash -c \"\$(curl -fsSL <get.sh address>)\" @ install"
case $(uname -m) in
x86_64 | amd64) arch=amd64 ;;
aarch64 | arm64) arch=arm64 ;;
*) die "this machine is $(uname -m): Karvan has builds for amd64 and arm64 only" ;;
esac
update=
archive=
pinned=
case ${1:-install} in
install) [ $# -eq 0 ] || shift ;;
update)
	update=--update
	shift
	;;
uninstall)
	shift
	[ -f "$INSTALLED" ] || die "Karvan is not installed here"
	exec sh "$(readlink -f "$INSTALLED")" --uninstall "$@"
	;;
-*) ;;
*) die "unknown command $1: use install, update or uninstall" ;;
esac
while [ $# -gt 0 ]; do
	case $1 in
	--version)
		[ $# -ge 2 ] || die "--version needs a version, for example 0.5.0-beta"
		VERSION=${2#v}
		pinned=1
		shift
		;;
	--archive)
		[ $# -ge 2 ] || die "--archive needs a release archive"
		archive=$2
		shift
		;;
	--update) update=--update ;;
	*) break ;;
	esac
	shift
done
for tool in curl sha256sum tar; do
	command -v "$tool" >/dev/null 2>&1 || die "$tool is missing: install it first (packages curl, coreutils, tar)"
done
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT
trap 'exit 1' INT TERM
if [ -n "$archive" ]; then
	file=$(basename "$archive")
	name=${file%.tar.gz}
	case $name in
	karvan_?*_linux_"$arch") ;;
	*) die "$file is not a Karvan release archive for $arch; nothing was installed" ;;
	esac
	sums=$(dirname "$archive")/SHA256SUMS
	if [ -f "$sums" ]; then
		awk -v f="$file" '$2 == f { n++ } END { exit n != 1 }' "$sums" || die "SHA256SUMS does not list $file; nothing was installed"
		(cd "$(dirname "$archive")" && sha256sum -c --ignore-missing --quiet SHA256SUMS) || die "$file does not match SHA256SUMS; nothing was installed"
	else
		echo "No SHA256SUMS next to $file: it is not checked."
	fi
	cp "$archive" "$tmp/$file"
else
	if [ -z "$pinned" ] && { [ -n "$update" ] || [ -z "$VERSION" ]; }; then
		VERSION=$(curl -fsSL --retry 3 "$RELEASES_URL" | sed -n 's/^ *"tag_name": *"v\{0,1\}\([^"]*\)".*/\1/p' | head -n 1)
		[ -n "$VERSION" ] || die "could not find the newest Karvan release"
	fi
	case $VERSION in
	[0-9]*[!0-9A-Za-z.+-]* | [!0-9]*) die "not a version: $VERSION" ;;
	esac
	name=karvan_${VERSION}_linux_${arch}
	echo "Downloading Karvan $VERSION for $arch..."
	curl -fsSL --retry 3 -o "$tmp/SHA256SUMS" "$BASE_URL/v$VERSION/SHA256SUMS" || die "could not download SHA256SUMS for $VERSION"
	curl -fsSL --retry 3 -o "$tmp/$name.tar.gz" "$BASE_URL/v$VERSION/$name.tar.gz" || die "could not download $name.tar.gz"
	awk -v f="$name.tar.gz" '$2 == f { n++ } END { exit n != 1 }' "$tmp/SHA256SUMS" || die "SHA256SUMS does not list $name.tar.gz; nothing was installed"
	(cd "$tmp" && sha256sum -c --ignore-missing --quiet SHA256SUMS) || die "the download does not match SHA256SUMS; nothing was installed"
fi
tar -xzf "$tmp/$name.tar.gz" -C "$tmp"
[ -f "$tmp/$name/install.sh" ] || die "the archive has no $name/install.sh; nothing was installed"
sh "$tmp/$name/install.sh" $update "$@"
