#!/bin/sh
# Builds Karvan's image from this tree (IMAGE=<tag> checks one already
# built) and checks it in throwaway containers and volumes: only the binary
# on the base, root, init prints the sign-in details once, the panel binds
# port 80 and turns healthy, and a test panel stops and starts through the
# Docker socket (D26 to D29). Needs Docker; nothing touches the host's
# Karvan paths or ports.
set -eu
ROOT=$(cd "$(dirname "$0")/../.." && pwd)
NAME=karvan-dockertest
PANEL=karvan-smoke-panel
case $(uname -m) in
x86_64 | amd64) ARCH=amd64 ;;
aarch64 | arm64) ARCH=arm64 ;;
*) echo "no Karvan build for $(uname -m)" >&2 && exit 1 ;;
esac
work=$(mktemp -d)
compose() { docker compose -f "$work/opt/$PANEL/docker-compose.yml" -p "$PANEL" "$@"; }
cleanup() {
	docker rm -f "$NAME" >/dev/null 2>&1 || true
	[ ! -f "$work/opt/$PANEL/docker-compose.yml" ] || compose down -t 1 >/dev/null 2>&1 || true
	docker volume rm "$NAME-data" "$NAME-etc" >/dev/null 2>&1 || true
	rm -rf "$work"
}
trap cleanup EXIT
trap 'exit 1' INT TERM
fail() {
	echo "FAIL: $*" >&2
	exit 1
}
step() { echo "== $*"; }
# files lists the regular files of image $1.
files() {
	id=$(docker create "$1" x)
	docker export "$id" | tar -t | grep -v '/$' | sort
	docker rm "$id" >/dev/null
}
# karvan runs a Karvan command in a new container on the test volumes.
karvan() { docker run --rm -v "$NAME-data:/var/lib/karvan" -v "$NAME-etc:/etc/karvan" "$IMAGE" "$@"; }

if [ -z "${IMAGE:-}" ]; then
	step "build the image from this tree"
	mkdir -p "$work/ctx/linux_$ARCH"
	make -s -C "$ROOT" build-go VERSION=0.0.0-docker BIN="$work/ctx/linux_$ARCH/karvan"
	IMAGE=karvan-dockertest:smoke
	docker buildx build --load -q -t "$IMAGE" --build-arg VERSION=0.0.0-docker \
		-f "$ROOT/packaging/docker/Dockerfile" "$work/ctx" >/dev/null
fi

step "only the binary on the base, as root"
base=$(sed -n 's/^FROM //p' "$ROOT/packaging/docker/Dockerfile")
files "$base" >"$work/base.txt"
files "$IMAGE" >"$work/image.txt"
extra=$(comm -13 "$work/base.txt" "$work/image.txt")
[ "$extra" = "usr/local/bin/karvan" ] || fail "files beyond the base: $extra"
user=$(docker image inspect -f '{{.Config.User}}' "$IMAGE")
[ "$user" = "0:0" ] || fail "runs as '$user'"
echo "ok: base + usr/local/bin/karvan, user $user, $(docker image inspect -f '{{.Size}}' "$IMAGE") bytes"

step "init prints the sign-in details once"
docker volume create "$NAME-data" >/dev/null
docker volume create "$NAME-etc" >/dev/null
karvan init -port 80 >"$work/init.txt" 2>&1 || fail "init: $(grep -v -i password "$work/init.txt")"
grep -q "Username" "$work/init.txt" || fail "init printed no sign-in details"
addr=$(sed -n 's|.*Address *\([a-z]*://\)<server address>:80/\([^/]*\)/.*|\1 \2|p' "$work/init.txt")
[ -n "$addr" ] || fail "no address on port 80"
scheme=${addr% *} path=${addr#* }
if karvan init -port 80 >"$work/init2.txt" 2>&1; then fail "a second init ran"; fi
! grep -q "Username" "$work/init2.txt" || fail "the sign-in details were shown twice"
echo "ok: second init refused: $(head -n 1 "$work/init2.txt")"

step "serve binds port 80 and turns healthy"
mkdir -p "$work/opt/$PANEL" "$work/smoke"
: >"$work/opt/$PANEL/.env"
cat >"$work/opt/$PANEL/docker-compose.yml" <<EOF
services:
  marzban:
    image: gozargah/marzban:v0.8.4
    entrypoint: ["sleep", "infinity"]
    network_mode: none
EOF
compose up -d --quiet-pull >/dev/null 2>&1 || fail "test panel did not start"
docker run -d --name "$NAME" -v "$NAME-data:/var/lib/karvan" -v "$NAME-etc:/etc/karvan" \
	-v "$work/opt:/opt:ro" -v /var/run/docker.sock:/var/run/docker.sock "$IMAGE" >/dev/null
status=
for _ in $(seq 60); do
	status=$(docker inspect -f '{{.State.Health.Status}}' "$NAME")
	[ "$status" = healthy ] && break
	sleep 2
done
[ "$status" = healthy ] || fail "health is '$status': $(docker logs --tail 5 "$NAME" 2>&1)"
ip=$(docker inspect -f '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' "$NAME")
# init's self-signed certificate: -k, the fingerprint is not the point here.
code=$(curl -sk -o /dev/null -w '%{http_code}' "$scheme$ip:80/$path/")
[ "$code" = 200 ] || fail "sign-in page on port 80 answered $code"
code=$(curl -sk -o /dev/null -w '%{http_code}' "$scheme$ip:80/wrong/")
[ "$code" = 404 ] || fail "a wrong path answered $code"
docker exec "$NAME" karvan health
echo "ok: healthy, port 80 (${scheme%://}) serves the sign-in page and 404 elsewhere"

step "a test panel stops and starts through the Docker socket"
(cd "$ROOT" && CGO_ENABLED=0 go test -c -tags dockertest -o "$work/smoke/panelctl.test" ./internal/panelctl/)
docker run --rm --entrypoint /smoke/panelctl.test -v "$work/smoke:/smoke:ro" -v "$work/opt:/opt:ro" \
	-v /var/run/docker.sock:/var/run/docker.sock -e KARVAN_SMOKE_PANEL="/opt/$PANEL" "$IMAGE" \
	-test.run TestSocketSmoke -test.v || fail "socket stop and start"
[ "$(docker inspect -f '{{.State.Running}}' "$PANEL-marzban-1")" = true ] || fail "the test panel is not running"
echo "PASS"
