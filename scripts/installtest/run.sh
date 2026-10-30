#!/bin/sh
# Installs, updates, rolls back and removes Karvan in a clean Debian 12
# system with systemd the way an operator would, from a local copy of the
# release files, and checks the result (D36). Needs Docker. ARCHIVE=<release
# archive> tests that archive; otherwise one is packed from a fresh build of
# this tree. The update builds are made from the same archive.
set -eu
ROOT=$(cd "$(dirname "$0")/../.." && pwd)
IMAGE=karvan-installtest:debian12
NAME=karvan-installtest
PORT=8099
case $(uname -m) in
x86_64 | amd64) ARCH=amd64 OTHER=arm64 ;;
aarch64 | arm64) ARCH=arm64 OTHER=amd64 ;;
*) echo "no Karvan build for $(uname -m)" >&2 && exit 1 ;;
esac
# The update builds: V2 is the same binary under a new version, BAD never
# gets healthy and V3 claims a newer store schema.
V2=0.0.1-test
BAD=0.0.2-bad
V3=0.0.3-test
work=$(mktemp -d)
cleanup() {
	[ -n "${KEEP:-}" ] || docker rm -f "$NAME" >/dev/null 2>&1 || true
	rm -rf "$work"
}
trap cleanup EXIT
trap 'exit 1' INT TERM
fail() {
	echo "FAIL: $*" >&2
	exit 1
}
step() { echo "== $*"; }
# sys runs a command inside the test system as root.
sys() { docker exec "$NAME" sh -c "$1"; }
# pack builds the binary and lays out the release archive of D16.
pack() {
	VERSION=${VERSION:-0.0.0-test}
	dir=$work/karvan_${VERSION}_linux_$ARCH
	mkdir -p "$dir/lib"
	make -s -C "$ROOT" build-go VERSION="$VERSION" BIN="$dir/karvan"
	cp "$ROOT"/packaging/install.sh "$ROOT"/packaging/get.sh "$ROOT"/packaging/karvan.service "$dir/"
	cp "$ROOT"/packaging/lib/*.sh "$dir/lib/"
	cp "$ROOT/README.md" "$ROOT/LICENSE" "$dir/"
	tar -czf "$work/karvan_${VERSION}_linux_$ARCH.tar.gz" -C "$work" "karvan_${VERSION}_linux_$ARCH"
	rm -rf "$dir"
	ARCHIVE=$work/karvan_${VERSION}_linux_$ARCH.tar.gz
}
# derive packs the archive of version $1 from the unpacked base archive,
# with the binary replaced by file $2 when given.
derive() {
	dir=$work/karvan_$1_linux_$ARCH
	cp -a "$work/base/karvan_${VERSION}_linux_$ARCH" "$dir"
	[ -z "${2:-}" ] || install -m 0755 "$2" "$dir/karvan"
	tar -czf "$dir.tar.gz" -C "$work" "karvan_$1_linux_$ARCH"
	rm -rf "$dir"
}
# release_tree puts archive $3 of version $2, its SHA256SUMS and a get.sh
# pointed at the local server under /srv/release/<$1>/download/v<$2>/ in the
# test system. A tree named bad gets a SHA256SUMS that does not match.
release_tree() {
	tree=$work/release/$1/download/v$2
	mkdir -p "$tree"
	cp "$3" "$tree/"
	(cd "$tree" && sha256sum "$(basename "$3")" >SHA256SUMS)
	if [ "$1" = bad ]; then
		awk '{ printf "%064d  %s\n", 0, $2 }' "$tree/SHA256SUMS" >"$tree/sums" && mv "$tree/sums" "$tree/SHA256SUMS"
	fi
	local_get "$1" "$2" >"$tree/get.sh"
}
# local_get prints get.sh pointed at the local server's tree $1, with
# version $2 stamped in (none: --update asks the local release list).
local_get() {
	sed -e "s|^BASE_URL=.*|BASE_URL=http://127.0.0.1:$PORT/$1/download|" \
		-e "s|^RELEASES_URL=.*|RELEASES_URL=http://127.0.0.1:$PORT/releases.json|" \
		-e "s|^VERSION=.*|VERSION=$2|" "$ROOT/packaging/get.sh"
}
if [ -z "${ARCHIVE:-}" ]; then
	step "pack an archive from this tree"
	pack
else
	name=$(basename "$ARCHIVE" .tar.gz)
	VERSION=${name#karvan_}
	VERSION=${VERSION%_linux_*}
fi
name=karvan_${VERSION}_linux_$ARCH
mkdir -p "$work/base"
tar -xzf "$ARCHIVE" -C "$work/base"
cat >"$work/bad-karvan" <<EOF
#!/bin/sh
case \$1 in
version) echo '{"version":"$BAD","commit":"","date":"","schema":999}' ;;
serve) : >/var/lib/karvan/bad-ran && exec sleep 600 ;;
*) exit 1 ;;
esac
EOF
cat >"$work/v3-karvan" <<'EOF'
#!/bin/sh
if [ "${1:-}" = version ]; then
	/srv/karvan.real "$@" | sed 's/"schema":[0-9]*/"schema":999/'
	exit
fi
exec /srv/karvan.real "$@"
EOF
derive "$V2"
derive "$BAD" "$work/bad-karvan"
derive "$V3" "$work/v3-karvan"
step "start the test system"
docker build -q -t "$IMAGE" "$ROOT/scripts/installtest" >/dev/null
docker rm -f "$NAME" >/dev/null 2>&1 || true
docker run -d --name "$NAME" --tmpfs /run --tmpfs /run/lock \
	-v /sys/fs/cgroup:/sys/fs/cgroup:rw --cgroupns=host "$IMAGE" >/dev/null
sys 'systemctl is-system-running --wait >/dev/null 2>&1 || true'
release_tree good "$VERSION" "$ARCHIVE"
release_tree good "$V2" "$work/karvan_${V2}_linux_$ARCH.tar.gz"
local_get good '' >"$work/release/get-latest.sh"
printf '[\n  {\n    "tag_name": "v%s",\n    "prerelease": true\n  }\n]\n' "$V2" >"$work/release/releases.json"
release_tree bad "$VERSION" "$ARCHIVE"
docker cp "$work/release" "$NAME:/srv/" >/dev/null
docker cp "$ARCHIVE" "$NAME:/srv/" >/dev/null
sys 'mkdir -p /srv/updates/v3'
docker cp "$work/karvan_${BAD}_linux_$ARCH.tar.gz" "$NAME:/srv/updates/" >/dev/null
docker cp "$work/karvan_${V3}_linux_$ARCH.tar.gz" "$NAME:/srv/updates/v3/" >/dev/null
sys "cd /srv/updates/v3 && sha256sum karvan_${V3}_linux_$ARCH.tar.gz >SHA256SUMS"
docker cp "$work/base/$name/karvan" "$NAME:/srv/karvan.real" >/dev/null
sys "busybox httpd -p 127.0.0.1:$PORT -h /srv/release"
sys 'mkdir -p /opt/marzban /var/lib/marzban /root/cert && echo SQLALCHEMY_DATABASE_URL=sqlite:////var/lib/marzban/db.sqlite3 >/opt/marzban/.env && : >/var/lib/marzban/db.sqlite3'
# nothing_installed fails when an install that should have stopped left
# anything behind.
nothing_installed() {
	sys '! [ -e /usr/local/lib/karvan ] && ! [ -e /usr/local/bin/karvan ] && ! [ -e /var/lib/karvan ] && ! [ -e /etc/karvan ] && ! [ -e /etc/systemd/system/karvan.service ]' ||
		fail "$1 left files behind"
}

# expect_stop runs $2 in the test system, which must fail with output that
# contains $3 and change nothing.
expect_stop() {
	if out=$(sys "$2" 2>&1); then
		fail "$1 did not stop"
	fi
	case $out in
	*"$3"*) ;;
	*) fail "$1 said: $out" ;;
	esac
	nothing_installed "$1"
	echo "ok   $1"
}

step "checks stop before any change"
expect_stop "wrong checksum" "curl -fsSL http://127.0.0.1:$PORT/bad/download/v$VERSION/get.sh | sh" "does not match SHA256SUMS"
sys "mkdir -p /srv/unpacked && tar -xzf /srv/$name.tar.gz -C /srv/unpacked"
expect_stop "not root" "su -s /bin/sh nobody -c 'sh /srv/unpacked/$name/install.sh --yes'" "run it as root"
expect_stop "missing tar" "mv /usr/bin/tar /usr/bin/tar.off; sh /srv/unpacked/$name/install.sh --yes; s=\$?; mv /usr/bin/tar.off /usr/bin/tar; exit \$s" "install the tar package"
sys "cp -a /srv/unpacked/$name /srv/unpacked/karvan_${VERSION}_linux_$OTHER"
expect_stop "wrong architecture" "sh /srv/unpacked/karvan_${VERSION}_linux_$OTHER/install.sh --yes" "this archive is for $OTHER"

# signin.sh signs in with the details the install printed (kept inside the
# test system, never shown here) and prints the HTTP status.
cat >"$work/signin.sh" <<'EOF'
#!/bin/sh
set -eu
out=/root/install.out
port=$(sed -n 's|^  Address   https://<server address>:\([0-9]*\)/.*|\1|p' "$out")
path=$(sed -n 's|^  Address   https://<server address>:[0-9]*/\([^/]*\)/$|\1|p' "$out")
user=$(sed -n 's/^  Username  //p' "$out")
pass=$(sed -n 's/^  Password  //p' "$out")
base=https://127.0.0.1:$port/$path/api/v1
jar=$(mktemp)
token=$(curl -sk -c "$jar" "$base/session" | sed -n 's/.*"csrfToken":"\([^"]*\)".*/\1/p')
printf '{"username":"%s","password":"%s"}' "$user" "$pass" |
	curl -sk -o /dev/null -w '%{http_code}' -b "$jar" -H "X-CSRF-Token: $token" \
		-H "Origin: https://127.0.0.1:$port" -H 'Content-Type: application/json' --data-binary @- "$base/session"
rm -f "$jar"
EOF
docker cp "$work/signin.sh" "$NAME:/srv/signin.sh" >/dev/null

# show_install prints the install output without the password.
show_install() { sys 'grep -v "Password  " /root/install.out' >&2 || true; }

# one_liner installs the README's way, from the get.sh with no version
# stamped in, with the arguments after @ install in $2.
one_liner() {
	sys "bash -c \"\$(curl -fsSL http://127.0.0.1:$PORT/get-latest.sh)\" @ install $2 >$1 2>&1" || {
		sys "grep -v 'Password  ' $1" >&2
		fail "the one-liner install"
	}
}

expect_mode() {
	got=$(sys "stat -c '%a %U' $1")
	[ "$got" = "$2 root" ] || fail "$1 is $got, want $2 root"
}

signed_in() {
	code=$(sys 'sh /srv/signin.sh')
	[ "$code" = 200 ] || fail "sign-in answered $code"
}

step "install a pinned version with the one-liner"
one_liner /root/install.out "--version $VERSION"
sys 'grep -q "^  Password  " /root/install.out' || { show_install && fail "init printed no password"; }
sys 'systemctl is-active --quiet karvan' || fail "the service is not active"
pid=$(sys 'systemctl show -p MainPID --value karvan')
user=$(sys "stat -c %U /proc/$pid")
[ "$user" = root ] || fail "the service runs as $user"
expect_mode /var/lib/karvan 700
expect_mode /etc/karvan 700
expect_mode /etc/karvan/config.json 600
expect_mode /etc/karvan/secret.key 600
[ "$(sys 'readlink /usr/local/lib/karvan/current')" = "$VERSION" ] || fail "current does not point at $VERSION"
sys 'karvan health' || fail "karvan health"
signed_in
echo "ok   installed, running as root, modes as D18, sign-in answers"

# on_version fails unless current points at $1, previous at $2 (- for
# none), the service is healthy and the first install's sign-in works.
on_version() {
	[ "$(sys 'readlink /usr/local/lib/karvan/current')" = "$1" ] || fail "current is not $1"
	prev=$(sys 'readlink /usr/local/lib/karvan/previous || echo -')
	[ "$prev" = "$2" ] || fail "previous is $prev, want $2"
	sys 'systemctl is-active --quiet karvan && karvan health >/dev/null' || fail "$1 is not healthy"
	signed_in
}
# data_kept fails unless the test data marker is there and every file in
# $1 (space separated, may be empty) is gone from the data folder.
data_kept() {
	sys 'grep -qx before-updates /var/lib/karvan/test-data' || fail "the test data is gone"
	for f in $1; do
		sys "! [ -e /var/lib/karvan/$f ]" || fail "$f is still in the data folder"
	done
}
# count prints the number of folders in $1.
count() { sys "find $1 -mindepth 1 -maxdepth 1 -type d | wc -l"; }
step "update to the newest release"
sys 'echo before-updates >/var/lib/karvan/test-data'
sys "bash -c \"\$(curl -fsSL http://127.0.0.1:$PORT/get-latest.sh)\" @ update" >/dev/null || fail "the update to $V2"
on_version "$V2" "$VERSION"
data_kept ''
[ "$(count /var/lib/karvan-backups)" = 1 ] || fail "the update made no backup"
expect_mode /var/lib/karvan-backups 700
echo "ok   updated to $V2 with a backup, data intact"
step "a failed update goes back by itself"
if out=$(sys "karvan update --archive /srv/updates/karvan_${BAD}_linux_$ARCH.tar.gz" 2>&1); then
	fail "the unhealthy update did not report a failure"
fi
case $out in
*"going back to $V2"*"is running again"*) ;;
*) fail "the failed update said: $out" ;;
esac
on_version "$V2" "$VERSION"
data_kept bad-ran
sys "! [ -e /usr/local/lib/karvan/$BAD ]" || fail "the failed version folder is still there"
echo "ok   unhealthy $BAD rolled back to $V2, backup restored, data intact"
step "rollback"
sys 'karvan rollback' >/dev/null || fail "karvan rollback"
on_version "$VERSION" "$V2"
data_kept ''
echo "ok   rolled back to $VERSION, data intact"
step "rollback across a schema change"
sys "karvan update --archive /srv/updates/v3/karvan_${V3}_linux_$ARCH.tar.gz" >/dev/null || fail "the update to $V3"
on_version "$V3" "$VERSION"
[ "$(count /usr/local/lib/karvan)" = 2 ] || fail "more than two version folders are kept"
[ "$(count /var/lib/karvan-backups)" = 2 ] || fail "more than two backups are kept"
sys ': >/var/lib/karvan/after-update'
if out=$(sys 'karvan rollback' 2>&1); then
	fail "the rollback across a schema change did not ask"
fi
# stdout and stderr come back through docker exec in either order.
case $out in
*"restores the backup taken at 20"*" UTC; changes made since then are lost"*) ;;
*) fail "the rollback said: $out" ;;
esac
case $out in
*"run it again with --yes"*) ;;
*) fail "the rollback said: $out" ;;
esac
on_version "$V3" "$VERSION"
sys '[ -e /var/lib/karvan/after-update ]' || fail "the rollback that stopped changed the data"
sys 'karvan rollback --yes' >/dev/null || fail "karvan rollback --yes"
on_version "$VERSION" "$V3"
data_kept after-update
echo "ok   the rollback named the backup time, stopped without --yes and restored the backup with it"
step "uninstall keeps the data"
sys 'karvan uninstall' >/dev/null || fail "karvan uninstall"
sys '! [ -e /etc/systemd/system/karvan.service ] && ! [ -e /usr/local/lib/karvan ] && ! [ -e /usr/local/bin/karvan ]' || fail "uninstall left Karvan's files"
sys '[ -f /var/lib/karvan/karvan.db ] && [ -f /etc/karvan/secret.key ] && [ -f /etc/karvan/config.json ] && [ -d /var/lib/karvan-backups ]' || fail "uninstall removed the data"
echo "ok   uninstalled, data, config and backups kept"
step "reinstall the newest release reads the same data"
one_liner /root/reinstall.out ''
[ "$(sys 'readlink /usr/local/lib/karvan/current')" = "$V2" ] || fail "the bare install did not take the newest release $V2"
sys 'grep -q "Found the data" /root/reinstall.out && ! grep -q "Password  " /root/reinstall.out' || fail "reinstall set Karvan up again"
sys 'systemctl is-active --quiet karvan' || fail "the service is not active after the reinstall"
signed_in
data_kept ''
echo "ok   reinstalled $V2 (newest, pre-release), the old sign-in details still work"
step "purge"
if sys 'echo delete | karvan uninstall --purge' >/dev/null 2>&1; then
	fail "purge went on without the typed words"
fi
sys '[ -e /usr/local/bin/karvan ] && [ -f /var/lib/karvan/karvan.db ]' || fail "purge without the typed words removed something"
sys "echo delete karvan data | bash -c \"\$(curl -fsSL http://127.0.0.1:$PORT/get-latest.sh)\" @ uninstall --purge" >/dev/null 2>&1 || fail "@ uninstall --purge"
nothing_installed "purge"
sys '! [ -e /var/lib/karvan-backups ]' || fail "purge kept the backups"
echo "ok   purge asked for the words, then removed Karvan, its data, config and backups"
echo "PASS install test ($VERSION, $ARCH)"
