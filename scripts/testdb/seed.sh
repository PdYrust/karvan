#!/usr/bin/env bash
# Builds the destination schema fixtures with each panel's own migrations:
# every supported database gets a fresh database, the official panel image
# runs `alembic upgrade head` against it, and the result is dumped to
# internal/panel/<panel>db/testdata/<revision>.<dialect>.sql.
# Needs Docker and the databases from test/db/compose.yml.
set -euo pipefail

cd "$(git rev-parse --show-toplevel)"

MARZBAN_IMAGE=gozargah/marzban:v0.8.4
MARZBAN_REV=2b231de97dc3
# Each PasarGuard version Karvan knows: its image and its alembic head.
PASARGUARD=(
  pasarguard/panel:v5.4.1=48a6bcb8bba1
  pasarguard/panel:v5.5.1@sha256:16765ab9838197ce10b9431e72721b23276c66375a9d481ce8c2777b035576c3=c8d2e4f6a1b3
)
COMPOSE=(docker compose -f test/db/compose.yml)

work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT
chmod 777 "$work"

"${COMPOSE[@]}" up -d --wait >/dev/null

# migrate <image> <url>: run the panel's migrations against url.
migrate() {
  local image=$1 url=$2
  local entry=(--entrypoint alembic)
  local args=(upgrade head)
  case $image in pasarguard/*)
    entry=(--entrypoint python)
    args=(-m alembic upgrade head)
    ;;
  esac
  docker run --rm --network host --user "$(id -u):$(id -g)" -w /code -v "$work:/data" \
    -e SQLALCHEMY_DATABASE_URL="$url" "${entry[@]}" "$image" "${args[@]}" >/dev/null 2>"$work/migrate.log" ||
    { cat "$work/migrate.log" >&2; exit 1; }
}

# Dump commands strip lines that change on every run (dates, versions) and
# the key material the panels generate on first migration (the jwt secret,
# Marzban's self-signed TLS key). Tests insert their own.
no_keys() { grep -vE "^INSERT INTO (public\\.)?\`?(tls|jwt)\`? "; }

dump_sqlite() { sqlite3 "$work/$1" .dump | grep -v '^INSERT INTO sqlite_stat1' | no_keys; }

dump_mysql() {
  local svc=$1 tool=$2 db=$3
  "${COMPOSE[@]}" exec -T "$svc" "$tool" -uroot -ptest --skip-dump-date --no-tablespaces \
    --skip-comments --skip-extended-insert "$db" 2>/dev/null |
    sed -E 's/ AUTO_INCREMENT=[0-9]+//' | no_keys
}

# The timescale image puts the extension in every new database; it is not
# the panel's, so it stays out of the fixture.
dump_pg() {
  local svc=$1 db=$2
  "${COMPOSE[@]}" exec -T "$svc" pg_dump -U postgres --no-owner --no-privileges --inserts \
    --exclude-extension=timescaledb "$db" 2>/dev/null |
    grep -vE '^-- Dumped (from|by)|^\\(un)?restrict ' | no_keys
}

fresh_mysql() { "${COMPOSE[@]}" exec -T "$1" "$2" -uroot -ptest -e "DROP DATABASE IF EXISTS seed; CREATE DATABASE seed" 2>/dev/null; }
fresh_pg() { "${COMPOSE[@]}" exec -T "$1" psql -qU postgres -c "DROP DATABASE IF EXISTS seed" -c "CREATE DATABASE seed" >/dev/null; }

out_m=internal/panel/marzbandb/testdata
out_p=internal/panel/pasarguarddb/testdata
mkdir -p "$out_m" "$out_p"

echo "marzban sqlite"
migrate "$MARZBAN_IMAGE" "sqlite:////data/m.sqlite3"
dump_sqlite m.sqlite3 >"$out_m/$MARZBAN_REV.sqlite.sql"

for pair in mysql:mysql:mysqldump:33061 mariadb:mariadb:mariadb-dump:33062; do
  IFS=: read -r svc client tool port <<<"$pair"
  echo "marzban $svc"
  fresh_mysql "$svc" "$client"
  migrate "$MARZBAN_IMAGE" "mysql+pymysql://root:test@127.0.0.1:$port/seed"
  dump_mysql "$svc" "$tool" seed >"$out_m/$MARZBAN_REV.$svc.sql"
done

for version in "${PASARGUARD[@]}"; do
  image=${version%=*}
  rev=${version##*=}
  echo "pasarguard $rev sqlite"
  rm -f "$work/p.sqlite3"
  migrate "$image" "sqlite+aiosqlite:////data/p.sqlite3"
  dump_sqlite p.sqlite3 >"$out_p/$rev.sqlite.sql"
  for pair in mysql:mysql:mysqldump:33061 mariadb:mariadb:mariadb-dump:33062; do
    IFS=: read -r svc client tool port <<<"$pair"
    echo "pasarguard $rev $svc"
    fresh_mysql "$svc" "$client"
    migrate "$image" "mysql+asyncmy://root:test@127.0.0.1:$port/seed"
    dump_mysql "$svc" "$tool" seed >"$out_p/$rev.$svc.sql"
  done
  for pair in postgres:54321 timescale:54322; do
    IFS=: read -r svc port <<<"$pair"
    echo "pasarguard $rev $svc"
    fresh_pg "$svc"
    migrate "$image" "postgresql+asyncpg://postgres:test@127.0.0.1:$port/seed"
    dump_pg "$svc" seed >"$out_p/$rev.$svc.sql"
  done
done

echo "fixtures written"
