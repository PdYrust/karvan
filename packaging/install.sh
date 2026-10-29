#!/bin/sh
# Installs Karvan from this archive folder, or removes it (--uninstall).
# Uses only the files next to it. Run it as root.
# shellcheck source-path=SCRIPTDIR
set -eu

HERE=$(cd "$(dirname "$0")" && pwd)
# shellcheck source=lib/checks.sh
. "$HERE/lib/checks.sh"
# shellcheck source=lib/layout.sh
. "$HERE/lib/layout.sh"

usage() {
	cat <<-EOF
		Usage: install.sh [--port N] [--yes]
		       install.sh --uninstall

		  --port N      serve the panel on port N (default: a free port from 20000 to 60000)
		  --yes         do not ask; take the proposed port
		  --uninstall   remove Karvan and keep its data and config
	EOF
}

die() {
	echo "$*" >&2
	exit 1
}

# can_ask is true when there is a terminal to ask on, even when the script
# itself arrives through a pipe.
can_ask() {
	[ "$yes" = 0 ] && (: </dev/tty) 2>/dev/null
}

choose_port() {
	if [ -z "$port" ]; then
		port=$("$HERE/karvan" ports -free)
		if can_ask; then
			printf 'Panel port [%s]: ' "$port" >/dev/tty
			read -r answer </dev/tty || answer=
			[ -n "$answer" ] && port=$answer
		fi
	fi
	case $port in
	'' | *[!0-9]* | 0*) die "the port must be a number from 1 to 65535" ;;
	esac
	[ "$port" -le 65535 ] || die "the port must be a number from 1 to 65535"
}

do_install() {
	name=$(basename "$HERE")
	case $name in
	karvan_?*_linux_amd64 | karvan_?*_linux_arm64) ;;
	*) die "run install.sh from the unpacked karvan_<version>_linux_<arch> folder" ;;
	esac
	arch=${name##*_}
	version=${name%_linux_*}
	version=${version#karvan_}
	# The version names a folder that is replaced, so it must be a plain name.
	case $version in
	[0-9]*[!0-9A-Za-z.+-]* | [!0-9]*) die "the archive folder has a bad version: $version" ;;
	esac
	check_system "$arch"
	# Without the unit, an earlier install stopped part way: do it again.
	if [ -e "$LIB_DIR/current" ] && [ -f "$UNIT" ]; then
		die "Karvan is already installed. Update it with: karvan update"
	fi
	fresh=1
	[ -f "$DATA_DIR/karvan.db" ] && fresh=0
	[ "$fresh" = 1 ] && choose_port

	install_version "$HERE" "$version"
	switch_current "$version"
	make_dirs
	if [ "$fresh" = 1 ]; then
		"$BIN_LINK" init -port "$port"
	else
		echo "Found the data of an earlier install in $DATA_DIR; Karvan keeps using it."
	fi
	install_unit
	wait_healthy || die "Karvan $version is installed but not healthy. See: journalctl -u karvan"
	echo "Karvan $version is running. If a firewall is on, open the panel port in it."
}

# wait_healthy gives the service up to 60 seconds to report in.
wait_healthy() {
	i=0
	while [ "$i" -lt 60 ]; do
		systemctl is-active --quiet karvan.service && "$BIN_LINK" health >/dev/null 2>&1 && return 0
		sleep 1
		i=$((i + 1))
	done
	return 1
}

do_uninstall() {
	[ "$(id -u)" = 0 ] || die "run it as root, for example with sudo"
	if [ -f "$UNIT" ]; then
		systemctl disable --now karvan.service || true
	fi
	remove_path "$UNIT"
	systemctl daemon-reload 2>/dev/null || true
	remove_path "$BIN_LINK"
	remove_path "$LIB_DIR"
	echo "Karvan is removed. Its data and config are still in $DATA_DIR and $ETC_DIR."
}

port=
yes=0
action=install
while [ $# -gt 0 ]; do
	case $1 in
	--port)
		[ $# -ge 2 ] || die "--port needs a number"
		port=$2
		shift
		;;
	--yes) yes=1 ;;
	--uninstall) action=uninstall ;;
	-h | --help)
		usage
		exit 0
		;;
	*)
		usage >&2
		exit 2
		;;
	esac
	shift
done

case $action in
install) do_install ;;
uninstall) do_uninstall ;;
esac
