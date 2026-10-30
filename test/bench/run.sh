#!/bin/sh
# The bridge benchmark of D37, safe to run next to other services:
#   karvan-bench-server  the bridge and a fake panel   --cpus=1 --memory=1g
#   karvan-bench-load    the load generator            --cpus=1 --memory=1g
# Both run on a private internal Docker network: no host port is
# published, the host network is not used, and nothing can reach out.
# Each container's root file system is read only. The only host folder
# is /tmp/karvan-bench: the test binary (mounted read only) and the
# server's database (mounted as its /tmp). The containers, the network,
# /tmp/karvan-bench and the base image (when this run pulled it) are
# removed at the end, also on Ctrl-C. The server stops by itself after
# 60 minutes at the latest.
#
# Usage: sh run.sh [path/to/bench.test]
#   KARVAN_BENCH_TOKENS  stored tokens, 1000 to 1000000 (default 1000000)
#   KARVAN_BENCH_SCALE   share of each load run, (0, 1] (default 1)
#   KARVAN_BENCH_SMALL_HOST=1 skips the 4 core check (a test on a laptop)
set -eu

src=${1:-./bench.test}
image=gcr.io/distroless/static-debian12@sha256:d75cdd72874d4790092fcb1b058493ecf6bb5bf2b2b897045b00ff01d91843f2
work=/tmp/karvan-bench
net=karvan-bench-net
server=karvan-bench-server
load=karvan-bench-load

# Preflight: 4 cores and 2.5 GB available, so 2 cores and the rest of the
# memory stay with the services already on this server.
cores=$(nproc)
avail_mb=$(awk '/^MemAvailable:/ { print int($2 / 1024) }' /proc/meminfo)
if { [ "$cores" -lt 4 ] && [ "${KARVAN_BENCH_SMALL_HOST:-}" != 1 ]; } || [ "$avail_mb" -lt 2560 ]; then
	echo "refusing to run: needs 4 cores and 2560 MB available, found $cores cores and $avail_mb MB" >&2
	exit 2
fi
[ -f "$src" ] || { echo "no test binary at $src" >&2; exit 2; }
for name in "$server" "$load"; do
	if docker ps -a --format '{{.Names}}' | grep -qx "$name"; then
		echo "refusing to run: a container named $name already exists" >&2
		exit 2
	fi
done
if docker network inspect "$net" >/dev/null 2>&1; then
	echo "refusing to run: a network named $net already exists" >&2
	exit 2
fi
[ ! -e "$work" ] || { echo "refusing to run: $work already exists" >&2; exit 2; }

pulled=0
docker image inspect "$image" >/dev/null 2>&1 || pulled=1
cleanup() {
	docker rm -f "$server" "$load" >/dev/null 2>&1 || true
	docker network rm "$net" >/dev/null 2>&1 || true
	[ "$pulled" -eq 0 ] || docker image rm "$image" >/dev/null 2>&1 || true
	rm -rf "$work"
}
trap cleanup EXIT
trap 'echo "stopped, cleaning up" >&2; exit 130' INT TERM

mkdir -m 0700 "$work" "$work/data"
install -m 0555 "$src" "$work/bench.test"
user="$(id -u):$(id -g)" # the files in $work stay this user's, so cleanup can remove them

echo "host: $cores cores, $avail_mb MB available, load average $(cut -d' ' -f1-3 /proc/loadavg)"
echo "host CPU:$(grep -m1 'model name' /proc/cpuinfo | cut -d: -f2)"
start=$(date +%s)
docker network create --internal "$net" >/dev/null
docker run --detach --name "$server" --network "$net" --cpus=1 --memory=1g --read-only --user "$user" \
	-e KARVAN_BENCH_TOKENS="${KARVAN_BENCH_TOKENS:-}" \
	-v "$work/bench.test:/bench.test:ro" -v "$work/data:/tmp" "$image" \
	/bench.test -test.run '^TestBridgeServer$' -test.v -test.timeout 70m >/dev/null
echo "storing ${KARVAN_BENCH_TOKENS:-1000000} tokens, then the load runs"
status=0
docker run --rm --name "$load" --network "$net" --cpus=1 --memory=1g --read-only --user "$user" \
	-e KARVAN_BENCH_SERVER="$server" -e KARVAN_BENCH_TOKENS="${KARVAN_BENCH_TOKENS:-}" \
	-e KARVAN_BENCH_SCALE="${KARVAN_BENCH_SCALE:-}" \
	-v "$work/bench.test:/bench.test:ro" "$image" \
	/bench.test -test.run '^TestBridgeLoad$' -test.v -test.timeout 70m >"$work/load.log" 2>&1 &
wait $! || status=1 # wait, unlike a foreground command, lets Ctrl-C run the trap at once
grep -vE '^=== ' "$work/load.log" || true
docker wait "$server" >/dev/null 2>&1 || true
echo "--- server"
docker logs "$server" 2>&1 | grep -E 'stored|ready after|peak|FAIL|panic' || true
echo "--- took $(($(date +%s) - start)) s"
cleanup
if [ "$status" -eq 0 ]; then echo "RESULT: PASS"; else echo "RESULT: FAIL"; fi
exit "$status"
