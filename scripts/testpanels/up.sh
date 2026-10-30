#!/usr/bin/env bash
# Starts Marzban 0.8.4 and PasarGuard 5.5.1 on SQLite for `make test-panels`
# and waits until both answer. Each panel is laid out like the official
# install scripts do it: its own folder with .env and docker-compose.yml,
# started as the compose project named after the folder, so Karvan stops
# and starts it the way it would on a real server. Needs Docker.
#   Marzban:    http://127.0.0.1:18000, webhook to 127.0.0.1:18099
#   PasarGuard: http://127.0.0.1:18001
set -euo pipefail
cd "$(git rev-parse --show-toplevel)"
PANELS_DIR=${PANELS_DIR:-/tmp/karvan-panels}
m=$PANELS_DIR/karvan-test-marzban
p=$PANELS_DIR/karvan-test-pasarguard

down() {
  if [ -f "$1/docker-compose.yml" ]; then
    docker compose -f "$1/docker-compose.yml" -p "$(basename "$1")" down --remove-orphans >/dev/null 2>&1 || true
  fi
}
down "$m"
down "$p"
rm -rf "$PANELS_DIR"
mkdir -p "$m" "$p"

# compose <dir> <service> <image>: the panel's compose file. Host network
# like a real install; it runs as the calling user so the test can write
# its SQLite file; the folder has the same path inside and outside.
compose() {
  cat >"$1/docker-compose.yml" <<EOF
services:
  $2:
    image: $3
    restart: "no"
    network_mode: host
    user: "$(id -u):$(id -g)"
    env_file: $1/.env
    environment:
      HOME: $1
    volumes: ["$1:$1"]
EOF
}
compose "$m" marzban gozargah/marzban:v0.8.4
compose "$p" pasarguard pasarguard/panel:v5.5.1@sha256:16765ab9838197ce10b9431e72721b23276c66375a9d481ce8c2777b035576c3

# The inbound tags match migrationtest.XrayInbounds; all listen on loopback.
cat >"$m/xray_config.json" <<'EOF'
{
  "log": {"loglevel": "warning"},
  "inbounds": [
    {"tag": "VLESS TCP", "listen": "127.0.0.1", "port": 18441, "protocol": "vless",
     "settings": {"clients": [], "decryption": "none"}, "streamSettings": {"network": "tcp"}},
    {"tag": "VMess WS", "listen": "127.0.0.1", "port": 18442, "protocol": "vmess",
     "settings": {"clients": []}, "streamSettings": {"network": "ws", "wsSettings": {"path": "/vm"}}},
    {"tag": "Trojan WS", "listen": "127.0.0.1", "port": 18443, "protocol": "trojan",
     "settings": {"clients": []}, "streamSettings": {"network": "ws", "wsSettings": {"path": "/tr"}}},
    {"tag": "Shadowsocks TCP", "listen": "127.0.0.1", "port": 18444, "protocol": "shadowsocks",
     "settings": {"clients": [], "network": "tcp,udp"}}
  ],
  "outbounds": [{"protocol": "freedom", "tag": "DIRECT"}]
}
EOF
cat >"$m/.env" <<EOF
UVICORN_HOST = "127.0.0.1"
UVICORN_PORT = 18000
SQLALCHEMY_DATABASE_URL = "sqlite:///$m/db.sqlite3"
XRAY_JSON = "$m/xray_config.json"
XRAY_SUBSCRIPTION_URL_PREFIX = "https://sub.example.test"
WEBHOOK_ADDRESS = "http://127.0.0.1:18099/marzban"
WEBHOOK_SECRET = "karvan-test"
JOB_SEND_NOTIFICATIONS_INTERVAL = 1
EOF
cat >"$p/.env" <<EOF
UVICORN_HOST = "127.0.0.1"
UVICORN_PORT = 18001
SQLALCHEMY_DATABASE_URL = "sqlite+aiosqlite:///$p/db.sqlite3"
EOF

for d in "$m" "$p"; do
  docker compose -f "$d/docker-compose.yml" -p "$(basename "$d")" up -d >/dev/null
done
for url in http://127.0.0.1:18000/openapi.json http://127.0.0.1:18001/health; do
  for _ in $(seq 1 90); do
    curl -fsS -o /dev/null "$url" 2>/dev/null && continue 2
    sleep 1
  done
  echo "panel at $url did not start" >&2
  for d in "$m" "$p"; do docker compose -f "$d/docker-compose.yml" -p "$(basename "$d")" logs --tail 40 >&2; done
  exit 1
done
echo "panels up in $PANELS_DIR"
