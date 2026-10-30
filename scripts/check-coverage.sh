#!/usr/bin/env bash
# Statement coverage gate for the core Go packages listed below.
set -euo pipefail

min=80
core="config secret store auth audit settings httpapi httpapi/middleware httpapi/handlers httpapi/respond"

cd "$(git rev-parse --show-toplevel)"
# One run gives both the test result and the coverage numbers. The race
# detector runs in `make test` (full gate), because argon2id under -race
# doubles the time of the fast gate.
out=$(go test -cover ./... 2>&1) || { echo "$out"; exit 1; }
echo "$out" | grep -E '^(ok|FAIL|---)' || true

status=0
for p in $core; do
  [ -d "internal/$p" ] || continue
  line=$(echo "$out" | grep -E "github.com/PdYrust/karvan/internal/${p}[[:space:]]" || true)
  pct=$(echo "$line" | grep -oE 'coverage: [0-9.]+%' | grep -oE '[0-9.]+' || echo 0)
  if awk -v p="$pct" -v m="$min" 'BEGIN { exit !(p < m) }'; then
    echo "coverage: internal/$p is ${pct}%, needs ${min}%"
    status=1
  else
    echo "coverage: internal/$p ${pct}%"
  fi
done
exit $status
