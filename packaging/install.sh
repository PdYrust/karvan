#!/bin/sh
# Installs or updates Karvan from this archive folder, goes back to the
# previous version, or removes Karvan. Uses only the files next to it. Run
# it as root.
# shellcheck source-path=SCRIPTDIR
set -eu

HERE=$(cd "$(dirname "$0")" && pwd)
# shellcheck source=lib/checks.sh
. "$HERE/lib/checks.sh"
# shellcheck source=lib/layout.sh
. "$HERE/lib/layout.sh"
# shellcheck source=lib/backup.sh
. "$HERE/lib/backup.sh"

usage() {
	cat <<-EOF
		Usage: install.sh [--port N] [--yes]
		       install.sh --update
		       install.sh --rollback [--yes]
		       install.sh --uninstall [--purge]

		  --port N      serve the panel on port N (default: a free port from 20000 to 60000)
		  --yes         do not ask: take the proposed port, or restore the backup on a rollback
		  --update      update an installed Karvan to the version of this folder
		  --rollback    go back to the previous version
		  --uninstall   remove Karvan and keep its data, config and backups
		  --purge       with --uninstall, also delete them (asks you to type delete karvan data)
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

# read_archive sets version and arch from the archive folder's name.
read_archive() {
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
}
do_install() {
	read_archive
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

# start_healthy starts the service without waiting on systemd, so a start
# that hangs is caught by wait_healthy. The old heartbeat goes first, so
# only the new process can pass karvan health.
start_healthy() {
	rm -f "$DATA_DIR/run/heartbeat"
	systemctl start --no-block karvan.service && wait_healthy
}
# do_update follows D23: back up with the service stopped, switch, start,
# and switch back when the new version is not healthy in 60 seconds.
do_update() {
	read_archive
	check_system "$arch"
	if [ ! -e "$LIB_DIR/current" ] || [ ! -f "$UNIT" ]; then
		die "Karvan is not installed. Install it with install.sh"
	fi
	old=$(version_of current)
	if [ "$version" = "$old" ]; then
		echo "Karvan $version is already installed."
		return 0
	fi
	"$HERE/karvan" version -json >/dev/null 2>&1 || die "the karvan binary of $version does not run here; nothing was changed"
	echo "Updating Karvan $old to $version..."
	# The new folder goes in while the old version still runs.
	install_version "$HERE" "$version"
	systemctl stop karvan.service
	if ! backup=$(take_backup "$old"); then
		prune_versions
		systemctl start karvan.service || true
		die "could not copy $DATA_DIR to $BACKUP_DIR (is the disk full?); Karvan $old was not changed"
	fi
	echo "Saved a copy of $DATA_DIR in $BACKUP_DIR/$backup"
	switch_current "$version"
	put_unit
	if start_healthy; then
		set_previous "$old"
		prune_versions
		echo "Karvan $version is running."
		return 0
	fi
	echo "Karvan $version did not start cleanly; going back to $old." >&2
	systemctl stop karvan.service || true
	new_schema=$(schema_of "$version")
	old_schema=$(schema_of "$old")
	switch_current "$old"
	put_unit
	# The new binary may have changed the data before it failed.
	if [ "${new_schema:-0}" -gt "${old_schema:-0}" ]; then
		restore_backup "$backup"
	fi
	prune_versions
	start_healthy || die "Karvan $old did not start again either. See: journalctl -u karvan"
	die "Karvan $old is running again. See why $version failed: journalctl -u karvan"
}
# do_rollback follows D24: an old binary never starts on data it cannot
# read, so a schema change restores the backup, after asking.
do_rollback() {
	[ "$(id -u)" = 0 ] || die "run it as root, for example with sudo"
	cur=$(version_of current)
	prev=$(version_of previous)
	if [ -z "$prev" ] || [ ! -x "$LIB_DIR/$prev/karvan" ]; then
		die "there is no earlier version to go back to"
	fi
	prev_schema=$(schema_of "$prev")
	[ -n "$prev_schema" ] || die "the karvan binary of $prev does not run; nothing was changed"
	cur_schema=$(schema_of "$cur")
	restore=
	if [ "${cur_schema:-0}" -gt "$prev_schema" ]; then
		restore=$(backup_of "$prev")
		[ -n "$restore" ] || die "Karvan $prev cannot read the data of $cur and there is no backup from $prev; nothing was changed"
		when=$(backup_time "$restore")
		echo "Karvan $prev cannot read the data of $cur. Going back restores the backup taken at $when; changes made since then are lost."
		if [ "$yes" = 0 ]; then
			can_ask || die "Nothing was changed. To go back and restore that backup, run it again with --yes"
			printf 'Restore the backup taken at %s? [y/N]: ' "$when" >/dev/tty
			read -r answer </dev/tty || answer=
			case $answer in
			y | Y | yes) ;;
			*) die "Nothing was changed." ;;
			esac
		fi
	fi
	systemctl stop karvan.service
	switch_current "$prev"
	set_previous "$cur"
	put_unit
	if [ -n "$restore" ]; then
		restore_backup "$restore"
	fi
	start_healthy || die "Karvan $prev is in place but not healthy. See: journalctl -u karvan"
	echo "Karvan $prev is running. Karvan update or karvan rollback brings back $cur."
}
do_uninstall() {
	[ "$(id -u)" = 0 ] || die "run it as root, for example with sudo"
	if [ "$purge" = 1 ]; then
		printf 'This also deletes %s, %s and %s.\nType delete karvan data to go on: ' "$DATA_DIR" "$ETC_DIR" "$BACKUP_DIR" >&2
		read -r answer || answer=
		[ "$answer" = "delete karvan data" ] || die "Nothing was removed."
	fi
	if [ -f "$UNIT" ]; then
		systemctl disable --now karvan.service || true
	fi
	remove_path "$UNIT"
	systemctl daemon-reload 2>/dev/null || true
	remove_path "$BIN_LINK"
	remove_path "$LIB_DIR"
	if [ "$purge" = 1 ]; then
		remove_path "$DATA_DIR"
		remove_path "$ETC_DIR"
		remove_path "$BACKUP_DIR"
		echo "Karvan, its data, config and backups are removed."
		return 0
	fi
	echo "Karvan is removed. Its data, config and backups are still in $DATA_DIR, $ETC_DIR and $BACKUP_DIR."
}

port=
yes=0
purge=0
action=install
while [ $# -gt 0 ]; do
	case $1 in
	--port)
		[ $# -ge 2 ] || die "--port needs a number"
		port=$2
		shift
		;;
	--yes) yes=1 ;;
	--update) action=update ;;
	--rollback) action=rollback ;;
	--uninstall) action=uninstall ;;
	--purge) purge=1 ;;
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

[ "$purge" = 0 ] || [ "$action" = uninstall ] || die "--purge goes with --uninstall"
case $action in
install) do_install ;;
update) do_update ;;
rollback) do_rollback ;;
uninstall) do_uninstall ;;
esac
