#!/usr/bin/env bash
# Starts Marzban 0.8.4 and PasarGuard 5.4.1 on SQLite for `make test-panels`
# and waits until both answer. Each gets a fresh folder under PANELS_DIR
# with its ENV, Xray config and database. Needs Docker.
#   Marzban:    http://127.0.0.1:18000, webhook to 127.0.0.1:18099
#   PasarGuard: http://127.0.0.1:18001
set -euo pipefail
cd "$(git rev-parse --show-toplevel)"
export PANELS_DIR=${PANELS_DIR:-/tmp/karvan-panels}
PANELS_UID=$(id -u)
PANELS_GID=$(id -g)
export PANELS_UID PANELS_GID
COMPOSE=(docker compose -f test/panels/compose.yml)

docker compose -p karvan-testpanels down --remove-orphans >/dev/null 2>&1 || true
rm -rf "$PANELS_DIR"
mkdir -p "$PANELS_DIR/marzban" "$PANELS_DIR/pasarguard"
m=$PANELS_DIR/marzban
p=$PANELS_DIR/pasarguard

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

"${COMPOSE[@]}" up -d >/dev/null
for url in http://127.0.0.1:18000/openapi.json http://127.0.0.1:18001/health; do
  for _ in $(seq 1 90); do
    curl -fsS -o /dev/null "$url" 2>/dev/null && continue 2
    sleep 1
  done
  echo "panel at $url did not start" >&2
  "${COMPOSE[@]}" logs --tail 40 >&2
  exit 1
done
echo "panels up in $PANELS_DIR"
