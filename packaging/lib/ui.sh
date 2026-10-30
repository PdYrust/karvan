# How the installer's scripts talk: a header, one line per step, a
# spinner while a step runs. Colour and the spinner only on a terminal and
# when NO_COLOR is unset; plain text otherwise. get.sh has its own copy of
# the part it needs, since it runs before any archive is unpacked.
# shellcheck shell=sh

if [ -t 1 ] && [ -z "${NO_COLOR:-}" ] && [ "${TERM:-}" != dumb ]; then
	UI_TTY=1
	C_BOLD=$(printf '\033[1m') C_DIM=$(printf '\033[2m') C_OFF=$(printf '\033[0m')
	C_OK=$(printf '\033[32m') C_WARN=$(printf '\033[33m') C_ERR=$(printf '\033[31m')
else
	UI_TTY=0 C_BOLD='' C_DIM='' C_OFF='' C_OK='' C_WARN='' C_ERR=''
fi
case ${LC_ALL:-${LC_CTYPE:-${LANG:-}}} in
*UTF-8* | *utf-8* | *UTF8* | *utf8*) M_OK='✓' M_WARN='!' M_ERR='✗' M_RULE='─' ;;
*) M_OK='ok' M_WARN='! ' M_ERR='x ' M_RULE='-' ;;
esac

# header prints the one header line, once per run.
header() {
	[ -z "${KARVAN_HEADED:-}" ] || return 0
	KARVAN_HEADED=1
	export KARVAN_HEADED
	printf '%s%s%s\n' "$C_BOLD" "$*" "$C_OFF"
}
ok() { printf '  %s%s%s %s\n' "$C_OK" "$M_OK" "$C_OFF" "$*"; }
warn() { printf '  %s%s%s %s\n' "$C_WARN" "$M_WARN" "$C_OFF" "$*"; }
detail() { printf '    %s%s%s\n' "$C_DIM" "$*" "$C_OFF"; }
# fail prints an error line on stderr; the lines after the first say what
# to do.
fail() {
	printf '  %s%s%s %s\n' "$C_ERR" "$M_ERR" "$C_OFF" "$1" >&2
	shift
	for line in "$@"; do
		printf '    %s\n' "$line" >&2
	done
}
die() {
	fail "$@"
	exit 1
}

# spin runs a command while a spinner and label $1 turn on stderr, when
# there is a terminal, and returns the command's status. The command's
# own output goes where spin's goes, so $(spin ...) captures it.
spin() {
	label=$1
	shift
	if [ "$UI_TTY" = 0 ]; then
		"$@"
		return
	fi
	"$@" &
	spin_pid=$!
	spin_i=0
	while kill -0 "$spin_pid" 2>/dev/null; do
		case $spin_i in
		0) f='|' ;; 1) f='/' ;; 2) f='-' ;; *) f="\\" ;;
		esac
		printf '\r  %s%s%s %s' "$C_DIM" "$f" "$C_OFF" "$label" >&2
		spin_i=$(((spin_i + 1) % 4))
		sleep 0.1
	done
	spin_rc=0
	wait "$spin_pid" || spin_rc=$?
	printf '\r\033[K' >&2
	return "$spin_rc"
}

# rule prints a line under a ruled block's width.
rule() {
	line=
	i=0
	while [ "$i" -lt 60 ]; do
		line=$line$M_RULE
		i=$((i + 1))
	done
	printf '  %s%s%s\n' "$C_DIM" "$line" "$C_OFF"
}

# field prints one label and value of the result block, the value bold.
field() { printf '  %-12s%s%s%s\n' "$1" "$C_BOLD" "$2" "$C_OFF"; }

# ask prints question $1 with default $2 on the terminal and prints the
# answer, the default when it is empty.
ask() {
	if [ -n "$2" ]; then
		printf '%s %s[%s]%s: ' "$1" "$C_DIM" "$2" "$C_OFF" >/dev/tty
	else
		printf '%s: ' "$1" >/dev/tty
	fi
	read -r ask_answer </dev/tty || ask_answer=
	printf '%s\n' "${ask_answer:-$2}"
}
