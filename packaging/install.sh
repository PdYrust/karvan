#!/bin/sh
# Installs or updates Karvan from this archive folder, goes back to the
# previous version, or removes Karvan. Uses only the files next to it. Run
# it as root.
# shellcheck source-path=SCRIPTDIR
set -eu

HERE=$(cd "$(dirname "$0")" && pwd)
# shellcheck source=lib/ui.sh
. "$HERE/lib/ui.sh"
# shellcheck source=lib/checks.sh
. "$HERE/lib/checks.sh"
# shellcheck source=lib/host.sh
. "$HERE/lib/host.sh"
# shellcheck source=lib/layout.sh
. "$HERE/lib/layout.sh"
# shellcheck source=lib/backup.sh
. "$HERE/lib/backup.sh"

usage() {
	cat <<-EOF
		Usage: install.sh [--https none|ip|domain] [--domain NAME] [--email ADDR] [--port N] [--yes]
		       install.sh --update
		       install.sh --rollback [--yes]
		       install.sh --uninstall [--purge]

		  --https none    plain HTTP, no certificate
		  --https ip      HTTPS on this server's IP with Karvan's own certificate (default)
		  --https domain  HTTPS on --domain NAME with a Let's Encrypt certificate, checked
		                  on port 80; --email ADDR is an optional contact for Let's Encrypt
		  --port N        serve the panel on port N (default: a free port from 20000 to 60000)
		  --yes           do not ask: take the defaults, or restore the backup on a rollback
		  --update        update an installed Karvan to the version of this folder
		  --rollback      go back to the previous version
		  --uninstall     remove Karvan and keep its data, config and backups
		  --purge         with --uninstall, also delete them (asks you to type delete karvan data)
	EOF
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
			port=$(ask "Panel port" "$port")
		fi
	fi
	case $port in
	'' | *[!0-9]* | 0*) die "The port must be a number from 1 to 65535." ;;
	esac
	[ "$port" -le 65535 ] || die "The port must be a number from 1 to 65535."
}

# choose_https sets https (none, ip or domain), and domain and email for
# a domain, from the flags or the menu.
choose_https() {
	if [ -z "$https" ] && [ -n "$domain" ]; then
		https=domain
	fi
	if [ -z "$https" ] && can_ask; then
		{
			echo
			echo "How should the panel be served?"
			echo "  1) Plain HTTP, no certificate"
			echo "  2) HTTPS on this server's IP, with Karvan's own certificate"
			echo "  3) HTTPS on a domain, with a Let's Encrypt certificate"
		} >/dev/tty
		case $(ask "Choose" 2) in
		1) https=none ;;
		2) https=ip ;;
		3) https=domain ;;
		*) die "Choose 1, 2 or 3." ;;
		esac
	fi
	https=${https:-ip}
	case $https in
	none | ip)
		[ -z "$domain$email" ] || die "--domain and --email go with --https domain."
		return 0
		;;
	domain) ;;
	*) die "--https must be none, ip or domain." ;;
	esac
	if [ -z "$domain" ] && can_ask; then
		domain=$(ask "Domain name, for example panel.example.com" "")
	fi
	if [ -z "$email" ] && can_ask; then
		email=$(ask "Email for Let's Encrypt (optional)" "")
	fi
	domain=$(printf '%s' "$domain" | tr '[:upper:]' '[:lower:]')
	case $domain in
	*[!a-z0-9.-]* | .* | *. | *..* | -* | *-) die "${domain} is not a domain name." "Use a name like panel.example.com." ;;
	*?.?*) ;;
	*) die "--https domain needs a domain name." "Use --domain panel.example.com, or --https ip." ;;
	esac
	case $email in
	*[!A-Za-z0-9.@+_-]* | *@*@* | @* | *@) die "$email is not an email address." ;;
	'' | *?@?*.?*) ;;
	*) die "$email is not an email address." ;;
	esac
}

# read_archive sets version and arch from the archive folder's name.
read_archive() {
	name=$(basename "$HERE")
	case $name in
	karvan_?*_linux_amd64 | karvan_?*_linux_arm64) ;;
	*) die "Run install.sh from the unpacked karvan_<version>_linux_<arch> folder." ;;
	esac
	arch=${name##*_}
	version=${name%_linux_*}
	version=${version#karvan_}
	# The version names a folder that is replaced, so it must be a plain name.
	case $version in
	[0-9]*[!0-9A-Za-z.+-]* | [!0-9]*) die "The archive folder has a bad version: $version" ;;
	esac
}

do_install() {
	read_archive
	header "Karvan $version, install"
	check_system "$arch"
	# Without the unit, an earlier install stopped part way: do it again.
	if [ -e "$LIB_DIR/current" ] && [ -f "$UNIT" ]; then
		die "Karvan is already installed." "To update it, run: karvan update"
	fi
	ok "Checked this system ($arch, systemd)"
	fresh=1
	[ -f "$DATA_DIR/karvan.db" ] && fresh=0
	if [ "$fresh" = 1 ]; then
		choose_https
		choose_port
		host=$(spin "Looking up this server's address" public_host)
		ok "This server's address is $host"
	fi
	install_version "$HERE" "$version"
	switch_current "$version"
	make_dirs
	ok "Installed Karvan $version in $LIB_DIR"
	creds=
	if [ "$fresh" = 1 ]; then
		set_up
	else
		warn "Found the data of an earlier install in $DATA_DIR; Karvan keeps using it."
	fi
	put_unit
	systemctl enable --quiet karvan.service
	spin "Starting Karvan" start_healthy || die "Karvan $version is installed but not healthy." "See why: journalctl -u karvan"
	ok "Started karvan.service"
	[ -z "$creds" ] || show_result
}

# set_up runs karvan init with the HTTPS choice and keeps its output, the
# sign-in details, in memory for show_result.
set_up() {
	flags="-tls self"
	label="Setting up the panel"
	case $https in
	none) flags="-tls off" ;;
	domain)
		if points_here "$domain" "$host"; then
			flags="-tls domain -domain $domain${email:+ -email $email}"
			label="Getting a Let's Encrypt certificate for $domain"
		else
			warn "$domain does not point at this server ($host), so Let's Encrypt cannot check it."
			detail "The panel uses Karvan's own certificate. Point the domain here, then get its certificate on the Domains page."
		fi
		;;
	esac
	# shellcheck disable=SC2086 # flags are separate words, checked by choose_https
	creds=$(spin "$label" "$BIN_LINK" init -installer -log-level error -port "$port" -host "$host" $flags) ||
		die "Karvan could not set up the panel." "See the error above. Nothing is running yet; to try again, run: karvan uninstall --purge, then install."
	notice=$(cred notice)
	if [ -n "$notice" ]; then
		warn "$notice"
	fi
	case $(cred scheme):$(cred fingerprint) in
	http:*) ok "Set up the panel on plain HTTP" ;;
	https:) ok "Set up the panel on HTTPS with a Let's Encrypt certificate for $domain" ;;
	*) ok "Set up the panel on HTTPS with Karvan's own certificate" ;;
	esac
}

# cred prints value $1 of the init output.
cred() { printf '%s\n' "$creds" | sed -n "s/^$1=//p"; }

# show_result prints the sign-in details once, then what to check.
show_result() {
	scheme=$(cred scheme)
	addr="$scheme://$(cred host):$(cred port)/$(cred path)/"
	fp=$(cred fingerprint)
	echo
	echo "  Save these now; they are shown only once."
	rule
	field Address "$addr"
	field Port "$(cred port)"
	field Path "/$(cred path)/"
	field Username "$(cred username)"
	field Password "$(cred password)"
	[ -z "$fp" ] || field Fingerprint "$fp"
	rule
	creds=
	if [ "$scheme" = http ]; then
		warn "Plain HTTP: the password crosses the network unencrypted. Turn on HTTPS in Settings when you can."
	elif [ -n "$fp" ]; then
		detail "The browser warns about Karvan's own certificate. Go on only if it shows the SHA-256 fingerprint above."
	fi
	hint=$(firewall_hint "$port")
	if [ -n "$hint" ]; then
		warn "$hint"
	else
		detail "If the server has a cloud firewall, open TCP port $port in it."
	fi
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

# do_update: back up with the service stopped, switch, start,
# and switch back when the new version is not healthy in 60 seconds.
do_update() {
	read_archive
	old=$(version_of current)
	header "Karvan $version, update${old:+ from $old}"
	check_system "$arch"
	if [ ! -e "$LIB_DIR/current" ] || [ ! -f "$UNIT" ]; then
		die "Karvan is not installed." "To install it, run install.sh without --update."
	fi
	if [ "$version" = "$old" ]; then
		ok "Karvan $version is already installed."
		return 0
	fi
	"$HERE/karvan" version -json >/dev/null 2>&1 || die "The karvan binary of $version does not run here; nothing was changed."
	# The new folder goes in while the old version still runs.
	install_version "$HERE" "$version"
	ok "Unpacked Karvan $version next to $old"
	systemctl stop karvan.service
	if ! backup=$(spin "Saving a copy of $DATA_DIR" take_backup "$old"); then
		prune_versions
		systemctl start karvan.service || true
		die "Could not copy $DATA_DIR to $BACKUP_DIR; Karvan $old was not changed." "Check the free disk space: df -h $BACKUP_DIR"
	fi
	ok "Saved a copy of $DATA_DIR in $BACKUP_DIR/$backup"
	switch_current "$version"
	put_unit
	if spin "Starting Karvan $version" start_healthy; then
		set_previous "$old"
		prune_versions
		ok "Karvan $version is running"
		detail "To go back to $old: karvan rollback"
		return 0
	fi
	fail "Karvan $version did not start cleanly; going back to $old."
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
	spin "Starting Karvan $old" start_healthy || die "Karvan $old did not start again either." "See why: journalctl -u karvan"
	die "Karvan $old is running again." "See why $version failed: journalctl -u karvan"
}

# do_rollback: an old binary never starts on data it cannot
# read, so a schema change restores the backup, after asking.
do_rollback() {
	[ "$(id -u)" = 0 ] || die "Run it as root, for example with sudo."
	cur=$(version_of current)
	prev=$(version_of previous)
	header "Karvan ${prev:-?}, rollback from $cur"
	if [ -z "$prev" ] || [ ! -x "$LIB_DIR/$prev/karvan" ]; then
		die "There is no earlier version to go back to."
	fi
	prev_schema=$(schema_of "$prev")
	[ -n "$prev_schema" ] || die "The karvan binary of $prev does not run; nothing was changed."
	cur_schema=$(schema_of "$cur")
	restore=
	if [ "${cur_schema:-0}" -gt "$prev_schema" ]; then
		restore=$(backup_of "$prev")
		[ -n "$restore" ] || die "Karvan $prev cannot read the data of $cur and there is no backup from $prev; nothing was changed."
		when=$(backup_time "$restore")
		warn "Karvan $prev cannot read the data of $cur. Going back restores the backup taken at $when; changes made since then are lost."
		if [ "$yes" = 0 ]; then
			can_ask || die "Nothing was changed." "To go back and restore that backup, run it again with --yes"
			case $(ask "Restore the backup taken at $when? (y/N)" n) in
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
		ok "Restored the backup taken at $when"
	fi
	spin "Starting Karvan $prev" start_healthy || die "Karvan $prev is in place but not healthy." "See why: journalctl -u karvan"
	ok "Karvan $prev is running"
	detail "karvan update or karvan rollback brings back $cur."
}

do_uninstall() {
	[ "$(id -u)" = 0 ] || die "Run it as root, for example with sudo."
	header "Karvan $(version_of current), uninstall"
	if [ "$purge" = 1 ]; then
		warn "This also deletes $DATA_DIR, $ETC_DIR and $BACKUP_DIR."
		printf 'Type delete karvan data to go on: ' >&2
		read -r answer || answer=
		[ "$answer" = "delete karvan data" ] || die "Nothing was removed."
	fi
	if [ -f "$UNIT" ]; then
		systemctl disable --quiet --now karvan.service || true
		ok "Stopped karvan.service"
	fi
	remove_path "$UNIT"
	systemctl daemon-reload 2>/dev/null || true
	remove_path "$BIN_LINK"
	remove_path "$LIB_DIR"
	if [ "$purge" = 1 ]; then
		remove_path "$DATA_DIR"
		remove_path "$ETC_DIR"
		remove_path "$BACKUP_DIR"
		ok "Removed Karvan, its data, config and backups"
		return 0
	fi
	ok "Removed Karvan"
	detail "Its data, config and backups are still in $DATA_DIR, $ETC_DIR and $BACKUP_DIR."
}

port=
https=
domain=
email=
yes=0
purge=0
action=install
# need_value stops when flag $1 has no value after it ($2 is $#).
need_value() { [ "$2" -ge 2 ] || die "$1 needs a value."; }
while [ $# -gt 0 ]; do
	case $1 in
	--port)
		need_value "$1" $#
		port=$2
		shift
		;;
	--https)
		need_value "$1" $#
		https=$2
		shift
		;;
	--domain)
		need_value "$1" $#
		domain=$2
		shift
		;;
	--email)
		need_value "$1" $#
		email=$2
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
[ "$purge" = 0 ] || [ "$action" = uninstall ] || die "--purge goes with --uninstall."

case $action in
install) do_install ;;
update) do_update ;;
rollback) do_rollback ;;
uninstall) do_uninstall ;;
esac
