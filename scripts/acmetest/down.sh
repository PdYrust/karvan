#!/usr/bin/env bash
# Stops what up.sh started.
set -euo pipefail
cd "$(dirname "$0")"
docker compose -f compose.yml down --remove-orphans
