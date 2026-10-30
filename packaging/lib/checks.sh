# Checks that run before install.sh changes anything (D17).
# shellcheck shell=sh

# arch_name prints the release name of this machine's architecture, or
# nothing when Karvan has no build for it.
arch_name() {
	case $(uname -m) in
	x86_64 | amd64) echo amd64 ;;
	aarch64 | arm64) echo arm64 ;;
	esac
}

# need_tool adds a line to problems when the command is missing.
need_tool() {
	command -v "$1" >/dev/null 2>&1 && return 0
	problems="${problems}  $1 is missing: install the $2 package
"
}

# check_system stops with one line per problem when this machine cannot
# run the archive built for $1 (amd64 or arm64).
check_system() {
	problems=
	[ "$(id -u)" = 0 ] || problems="${problems}  run it as root, for example with sudo
"
	[ "$(uname -s)" = Linux ] || problems="${problems}  Karvan runs on Linux only
"
	if [ ! -d /run/systemd/system ]; then
		problems="${problems}  systemd is not running: Karvan's service needs it
"
	fi
	machine=$(arch_name)
	if [ -z "$machine" ]; then
		problems="${problems}  this machine is $(uname -m): Karvan has builds for amd64 and arm64 only
"
	elif [ "$machine" != "$1" ]; then
		problems="${problems}  this archive is for $1, but this machine is $machine: download the $machine archive
"
	fi
	need_tool systemctl systemd
	need_tool sha256sum coreutils
	need_tool tar tar
	if [ -n "$problems" ]; then
		fail "Karvan was not installed; nothing was changed:"
		printf '%s' "$problems" | sed 's/^  /    /' >&2
		exit 1
	fi
}
