# This server's address for the result, the domain check and the
# firewall hint.
# shellcheck shell=sh

# IP_URL answers with the address the internet sees this server on.
IP_URL=${KARVAN_IP_URL:-https://api.ipify.org}

# is_ipv4 reports whether $1 is a dotted IPv4 address.
is_ipv4() {
	case $1 in
	'' | *[!0-9.]* | *..* | .* | *.) return 1 ;;
	esac
	old_ifs=$IFS
	IFS=.
	# shellcheck disable=SC2086 # split on the dots on purpose
	set -- $1
	IFS=$old_ifs
	[ $# -eq 4 ] || return 1
	for part in "$@"; do
		[ "${#part}" -le 3 ] && [ "$part" -le 255 ] || return 1
	done
}

# public_host prints the address to show in the panel's address: what an
# address service sees, else the source address of the route out, else
# the host name.
public_host() {
	ip=$(curl -fsS -4 --max-time 4 "$IP_URL" 2>/dev/null | tr -d ' \r\n') || ip=
	if is_ipv4 "$ip"; then
		echo "$ip"
		return
	fi
	ip=$(ip route get 1.1.1.1 2>/dev/null | sed -n 's/.* src \([0-9.]*\).*/\1/p' | head -n 1)
	if is_ipv4 "$ip"; then
		echo "$ip"
		return
	fi
	hostname -f 2>/dev/null || hostname
}

# points_here reports whether domain $1 resolves to address $2 or to one
# of this server's own addresses.
points_here() {
	got=$(getent ahosts "$1" 2>/dev/null | awk '{ print $1 }' | sort -u)
	[ -n "$got" ] || return 1
	mine=$(printf '%s\n' "$2"; ip -o addr show 2>/dev/null | awk '{ sub("/.*", "", $4); print $4 }')
	for a in $got; do
		printf '%s\n' "$mine" | grep -qxF "$a" && return 0
	done
	return 1
}

# firewall_hint prints the command that opens TCP ports $@ in the active
# firewall, or nothing when neither ufw nor firewalld is on.
firewall_hint() {
	if command -v ufw >/dev/null 2>&1 && ufw status 2>/dev/null | grep -q '^Status: active'; then
		cmd=
		for p in "$@"; do
			cmd="${cmd:+$cmd && }ufw allow $p/tcp"
		done
		echo "ufw is on. Open the panel with: $cmd"
	elif command -v firewall-cmd >/dev/null 2>&1 && [ "$(firewall-cmd --state 2>/dev/null)" = running ]; then
		cmd=
		for p in "$@"; do
			cmd="$cmd --add-port=$p/tcp"
		done
		echo "firewalld is on. Open the panel with: firewall-cmd --permanent$cmd && firewall-cmd --reload"
	fi
}
