#!/bin/sh
# Installs, updates or removes Karvan. Run it from the repository the way
# the README shows:
#   sudo bash -c "$(curl -fsSL <raw get.sh address>)" @ install [--version V]
#        [--https none|ip|domain] [--domain NAME] [--email ADDR] [--port N] [--yes]
#   ... @ update [--version V] | @ uninstall [--purge]
# A bare run means install. With no version it takes the newest release,
# pre-releases included. The download is checked against SHA256SUMS and on
# any failure nothing is installed. The options after the version go to
# the archive's install.sh. karvan update runs the installed copy with
# --update, and with --archive FILE to use a release archive that is
# already on this machine.
set -eu

BASE_URL=https://github.com/PdYrust/karvan/releases/download
# With no version the newest release listed here is taken (GitHub's latest
# skips pre-releases).
RELEASES_URL="https://api.github.com/repos/PdYrust/karvan/releases?per_page=1"
# The release build writes its version here; --version overrides it, and an
# update without --version ignores it.
VERSION=
INSTALLED=/usr/local/lib/karvan/current/install.sh

# The same look as lib/ui.sh, which is not unpacked yet.
if [ -t 1 ] && [ -z "${NO_COLOR:-}" ] && [ "${TERM:-}" != dumb ]; then
	UI_TTY=1
	C_BOLD=$(printf '\033[1m') C_DIM=$(printf '\033[2m') C_OFF=$(printf '\033[0m')
	C_OK=$(printf '\033[32m') C_ERR=$(printf '\033[31m')
else
	UI_TTY=0 C_BOLD='' C_DIM='' C_OFF='' C_OK='' C_ERR=''
fi
case ${LC_ALL:-${LC_CTYPE:-${LANG:-}}} in
*UTF-8* | *utf-8* | *UTF8* | *utf8*) M_OK='✓' M_ERR='✗' ;;
*) M_OK='ok' M_ERR='x ' ;;
esac
header() {
	KARVAN_HEADED=1
	export KARVAN_HEADED
	printf '%s%s%s\n' "$C_BOLD" "$*" "$C_OFF"
}
ok() { printf '  %s%s%s %s\n' "$C_OK" "$M_OK" "$C_OFF" "$*"; }
die() {
	printf '  %s%s%s %s\n' "$C_ERR" "$M_ERR" "$C_OFF" "$1" >&2
	shift
	for line in "$@"; do
		printf '    %s\n' "$line" >&2
	done
	exit 1
}
# spin runs a command with a spinner and label $1 on a terminal.
spin() {
	label=$1
	shift
	if [ "$UI_TTY" = 0 ]; then
		"$@"
		return
	fi
	"$@" &
	pid=$!
	i=0
	while kill -0 "$pid" 2>/dev/null; do
		case $i in
		0) f='|' ;; 1) f='/' ;; 2) f='-' ;; *) f="\\" ;;
		esac
		printf '\r  %s%s%s %s' "$C_DIM" "$f" "$C_OFF" "$label" >&2
		i=$(((i + 1) % 4))
		sleep 0.1
	done
	rc=0
	wait "$pid" || rc=$?
	printf '\r\033[K' >&2
	return "$rc"
}

[ "$(id -u)" = 0 ] || die "Run it as root." "For example: sudo bash -c \"\$(curl -fsSL <get.sh address>)\" @ install"
case $(uname -m) in
x86_64 | amd64) arch=amd64 ;;
aarch64 | arm64) arch=arm64 ;;
*) die "This machine is $(uname -m): Karvan has builds for amd64 and arm64 only." ;;
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
	[ -f "$INSTALLED" ] || die "Karvan is not installed here."
	exec sh "$(readlink -f "$INSTALLED")" --uninstall "$@"
	;;
-*) ;;
*) die "Unknown command $1." "Use install, update or uninstall." ;;
esac
while [ $# -gt 0 ]; do
	case $1 in
	--version)
		[ $# -ge 2 ] || die "--version needs a version, for example 0.5.0-beta."
		VERSION=${2#v}
		pinned=1
		shift
		;;
	--archive)
		[ $# -ge 2 ] || die "--archive needs a release archive."
		archive=$2
		shift
		;;
	--update) update=--update ;;
	*) break ;;
	esac
	shift
done

for tool in curl sha256sum tar; do
	command -v "$tool" >/dev/null 2>&1 || die "$tool is missing." "Install it first, for example: apt install curl coreutils tar"
done
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT
trap 'exit 1' INT TERM

current=$(readlink /usr/local/lib/karvan/current 2>/dev/null || true)
# action names the run for the header.
action() {
	if [ -z "$update" ]; then
		echo install
	elif [ -n "$current" ]; then
		echo "update from $current"
	else
		echo update
	fi
}
if [ -n "$archive" ]; then
	file=$(basename "$archive")
	name=${file%.tar.gz}
	case $name in
	karvan_?*_linux_"$arch") ;;
	*) die "$file is not a Karvan release archive for $arch; nothing was installed." ;;
	esac
	VERSION=${name#karvan_}
	VERSION=${VERSION%_linux_*}
	header "Karvan $VERSION, $(action)"
	sums=$(dirname "$archive")/SHA256SUMS
	if [ -f "$sums" ]; then
		awk -v f="$file" '$2 == f { n++ } END { exit n != 1 }' "$sums" || die "SHA256SUMS does not list $file; nothing was installed."
		(cd "$(dirname "$archive")" && sha256sum -c --ignore-missing --quiet SHA256SUMS) || die "$file does not match SHA256SUMS; nothing was installed."
		ok "Checked $file against SHA256SUMS"
	else
		ok "Using $file (no SHA256SUMS next to it, so it is not checked)"
	fi
	cp "$archive" "$tmp/$file"
else
	if [ -z "$pinned" ] && { [ -n "$update" ] || [ -z "$VERSION" ]; }; then
		VERSION=$(spin "Looking for the newest release" curl -fsSL --retry 3 "$RELEASES_URL" | sed -n 's/^ *"tag_name": *"v\{0,1\}\([^"]*\)".*/\1/p' | head -n 1)
		[ -n "$VERSION" ] || die "Could not find the newest Karvan release." "Check this server's connection to github.com, or pick one with --version."
	fi
	case $VERSION in
	[0-9]*[!0-9A-Za-z.+-]* | [!0-9]*) die "Not a version: $VERSION" ;;
	esac
	name=karvan_${VERSION}_linux_${arch}
	header "Karvan $VERSION, $(action)"
	spin "Downloading $name.tar.gz" curl -fsSL --retry 3 -o "$tmp/SHA256SUMS" "$BASE_URL/v$VERSION/SHA256SUMS" ||
		die "Could not download SHA256SUMS for $VERSION." "Check that release $VERSION exists and this server can reach github.com."
	spin "Downloading $name.tar.gz" curl -fsSL --retry 3 -o "$tmp/$name.tar.gz" "$BASE_URL/v$VERSION/$name.tar.gz" ||
		die "Could not download $name.tar.gz." "Check that release $VERSION has a build for $arch."
	ok "Downloaded $name.tar.gz"
	awk -v f="$name.tar.gz" '$2 == f { n++ } END { exit n != 1 }' "$tmp/SHA256SUMS" || die "SHA256SUMS does not list $name.tar.gz; nothing was installed."
	(cd "$tmp" && sha256sum -c --ignore-missing --quiet SHA256SUMS) || die "The download does not match SHA256SUMS; nothing was installed." "Try again; if it fails again, report it."
	ok "Checked it against SHA256SUMS"
fi
tar -xzf "$tmp/$name.tar.gz" -C "$tmp"
[ -f "$tmp/$name/install.sh" ] || die "The archive has no $name/install.sh; nothing was installed."
sh "$tmp/$name/install.sh" $update "$@"
