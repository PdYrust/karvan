#!/usr/bin/env bash
# Statement coverage gate for the core Go packages listed in CONSTRAINTS.md.
set -euo pipefail

min=80
core="config secret store auth audit httpapi httpapi/middleware httpapi/handlers settings"

cd "$(git rev-parse --show-toplevel)"
out=$(go test -cover ./internal/... 2>&1) || { echo "$out"; exit 1; }

status=0
for p in $core; do
  [ -d "internal/$p" ] || continue
  line=$(echo "$out" | grep -E "github.com/PdYrust/karvan/internal/$p[[:space:]]" || true)
  pct=$(echo "$line" | grep -oE 'coverage: [0-9.]+%' | grep -oE '[0-9.]+' || echo 0)
  if awk -v p="$pct" -v m="$min" 'BEGIN { exit !(p < m) }'; then
    echo "coverage: internal/$p is ${pct}%, needs ${min}%"
    status=1
  else
    echo "coverage: internal/$p ${pct}%"
  fi
done
exit $status
