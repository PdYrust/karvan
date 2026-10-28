#!/usr/bin/env bash
# Starts Pebble and its challenge server for `make test-acme` and waits
# until both answer; copies Pebble's API root to $ACMETEST_DIR so the test
# can trust it. Needs Docker (run it through sg docker -c when the user is
# not in the docker group).
#   ACME directory: https://127.0.0.1:14000/dir, roots on :15000
#   DNS: 127.0.0.1:8053, TXT records set on http://127.0.0.1:8055
set -euo pipefail
cd "$(dirname "$0")"
dir=${ACMETEST_DIR:-/tmp/karvan-acmetest}
docker compose -f compose.yml down --remove-orphans >/dev/null 2>&1 || true
docker compose -f compose.yml up -d
mkdir -p "$dir"
docker compose -f compose.yml cp pebble:/test/certs/pebble.minica.pem "$dir/pebble.minica.pem"
for _ in $(seq 1 60); do
  if curl -fsS --cacert "$dir/pebble.minica.pem" https://127.0.0.1:14000/dir >/dev/null 2>&1 &&
    curl -fsS -X POST -d '{"host":"up.karvan.test."}' http://127.0.0.1:8055/clear-txt >/dev/null 2>&1; then
    echo "pebble up"
    exit 0
  fi
  sleep 1
done
echo "pebble did not start" >&2
docker compose -f compose.yml logs >&2
exit 1
