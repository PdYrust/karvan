#!/bin/sh
# Installs Karvan in a clean Debian 12 system with systemd the way an
# operator would, from a local copy of the release files, and checks the
# result (D36). Needs Docker. ARCHIVE=<release archive> tests that archive;
# otherwise one is packed from a fresh build of this tree.
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

# release_tree puts the archive, its SHA256SUMS and a get.sh pointed at the local
# server under /srv/release/<tree>/download/v<version>/ in the test system.
# A tree named bad gets a SHA256SUMS that does not match.
release_tree() {
	tree=$work/release/$1/download/v$VERSION
	mkdir -p "$tree"
	cp "$ARCHIVE" "$tree/"
	(cd "$tree" && sha256sum "$(basename "$ARCHIVE")" >SHA256SUMS)
	if [ "$1" = bad ]; then
		awk '{ printf "%064d  %s\n", 0, $2 }' "$tree/SHA256SUMS" >"$tree/sums" && mv "$tree/sums" "$tree/SHA256SUMS"
	fi
	sed -e "s|^BASE_URL=.*|BASE_URL=http://127.0.0.1:$PORT/$1/download|" \
		-e "s|^VERSION=.*|VERSION=$VERSION|" "$ROOT/packaging/get.sh" >"$tree/get.sh"
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

step "start the test system"
docker build -q -t "$IMAGE" "$ROOT/scripts/installtest" >/dev/null
docker rm -f "$NAME" >/dev/null 2>&1 || true
docker run -d --name "$NAME" --tmpfs /run --tmpfs /run/lock \
	-v /sys/fs/cgroup:/sys/fs/cgroup:rw --cgroupns=host "$IMAGE" >/dev/null
sys 'systemctl is-system-running --wait >/dev/null 2>&1 || true'
release_tree good
release_tree bad
docker cp "$work/release" "$NAME:/srv/" >/dev/null
docker cp "$ARCHIVE" "$NAME:/srv/" >/dev/null
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

one_liner() {
	sys "curl -fsSL http://127.0.0.1:$PORT/good/download/v$VERSION/get.sh | sh >$1 2>&1" || {
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

step "install with the one-liner"
one_liner /root/install.out
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

step "uninstall keeps the data"
sys 'karvan uninstall' >/dev/null || fail "karvan uninstall"
sys '! [ -e /etc/systemd/system/karvan.service ] && ! [ -e /usr/local/lib/karvan ] && ! [ -e /usr/local/bin/karvan ]' || fail "uninstall left Karvan's files"
sys '[ -f /var/lib/karvan/karvan.db ] && [ -f /etc/karvan/secret.key ] && [ -f /etc/karvan/config.json ]' || fail "uninstall removed the data"
echo "ok   uninstalled, data and config kept"

step "reinstall reads the same data"
one_liner /root/reinstall.out
sys 'grep -q "Found the data" /root/reinstall.out && ! grep -q "Password  " /root/reinstall.out' || fail "reinstall set Karvan up again"
sys 'systemctl is-active --quiet karvan' || fail "the service is not active after the reinstall"
signed_in
echo "ok   reinstalled, the old sign-in details still work"
echo "PASS install test ($VERSION, $ARCH)"
