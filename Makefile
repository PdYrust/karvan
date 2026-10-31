PKG      := github.com/PdYrust/karvan
BIN      := bin/karvan
VERSION  ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT   ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
DATE     ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS  := -s -w \
	-X $(PKG)/internal/buildinfo.Version=$(VERSION) \
	-X $(PKG)/internal/buildinfo.Commit=$(COMMIT) \
	-X $(PKG)/internal/buildinfo.Date=$(DATE)

GOLANGCI ?= golangci-lint
PNPM     ?= pnpm --dir web

TESTDB   := docker compose -f test/db/compose.yml
TESTDB_ENV := KARVAN_TEST_MYSQL='root:test@tcp(127.0.0.1:33061)/' \
	KARVAN_TEST_MARIADB='root:test@tcp(127.0.0.1:33062)/' \
	KARVAN_TEST_POSTGRES='postgres://postgres:test@127.0.0.1:54321/postgres' \
	KARVAN_TEST_TIMESCALE='postgres://postgres:test@127.0.0.1:54322/postgres'
.PHONY: build build-go web web-deps test lint lint-web fmt floor cover check check-web check-full vuln audit-web clean test-db testdb-seed testdb-down test-panels testpanels-down test-acme bench bench-bridge bench-bridge-bin bench-api test-install release-dry ci-lint

build: web build-go

build-go:
	CGO_ENABLED=0 go build -trimpath -ldflags "$(LDFLAGS)" -o $(BIN) ./cmd/karvan

web-deps:
	$(PNPM) install --frozen-lockfile

web: web-deps
	$(PNPM) run build
	$(PNPM) run size

# The run tests write SQLite through modernc, which takes about 9 minutes under
# the race detector on a 2-core host; the default 10 minute limit is too close.
test:
	go test -race -timeout 30m ./...

lint:
	$(GOLANGCI) run ./...

fmt:
	$(GOLANGCI) fmt ./...
	$(PNPM) run format

floor:
	./scripts/check-floor.sh

cover:
	./scripts/check-coverage.sh

check-web:
	$(PNPM) run check

vuln:
	govulncheck ./...

audit-web:
	$(PNPM) audit --audit-level high

# Fast gate: run after every change. cover runs the tests once with coverage.
check: floor lint cover check-web

# Destination tests on every database server (needs Docker).
test-db:
	$(TESTDB) up -d --wait
	$(TESTDB_ENV) go test -tags dbtest ./internal/panel/... ./internal/migration/... ./internal/discovery/... ./internal/safemode/...
# Rebuild the schema fixtures with the panels' own migrations.
testdb-seed:
	./scripts/testdb/seed.sh
testdb-down:
	$(TESTDB) down
# Reload, login and API read-back against running Marzban and PasarGuard
# (needs Docker; starts fresh panels every time).
test-panels:
	./scripts/testpanels/up.sh
	go test -tags paneltest -count=1 -p 1 ./internal/migration/reload/ ./internal/bridge/ ./internal/panelctl/
# Let's Encrypt orders against Pebble by HTTP-01 and DNS-01 (needs Docker;
# uses ports 5002, 8053, 8055, 14000 and 15000).
test-acme:
	./scripts/acmetest/up.sh
	go test -tags acmetest -count=1 -run TestPebble ./internal/certorder/; s=$$?; ./scripts/acmetest/down.sh; exit $$s
# The engine at 100k users, then check, resolve and execute at 2000 admins
# (the 2000 admin run takes about 20 minutes on a 2-core host).
bench:
	go test -run x -bench . -benchtime 3x ./internal/migration/...
	go test -tags benchtest -count=1 -timeout 60m -v -run TestBenchEngine ./internal/migration/run/
# The bridge with a million stored tokens under open-loop load: the bridge
# and a fake panel in one container of 1 CPU and 1 GB, the load generator
# in another, on a private Docker network (needs Docker; about 20 minutes).
# bench-bridge-bin builds the two files to run it on another server
# (docs/benchmarks.md).
bench-bridge-bin:
	CGO_ENABLED=0 go test -c -tags benchtest -o bin/bench.test ./test/bench/
	install -m 0755 test/bench/run.sh bin/run.sh
bench-bridge: bench-bridge-bin
	sh bin/run.sh bin/bench.test
# The HTTP API at 50 clients against a running Karvan (KARVAN_BENCH_URL,
# KARVAN_BENCH_USER and KARVAN_BENCH_PASSWORD).
bench-api:
	go test -tags benchtest -count=1 -v -run TestAPILatency ./test/bench/
# Install, update, roll back and remove Karvan in a clean Debian 12 system with systemd
# (needs Docker). ARCHIVE=<release archive> tests that archive instead of
# one packed from this tree.
test-install:
	./scripts/installtest/run.sh
# Build, check and test the release files locally without pushing anything
# (needs Docker; VERSION defaults to 0.5.0-beta-dry, output in dist/).
release-dry:
	./scripts/release/dry.sh
# Check the GitHub workflows (actionlint, which runs shellcheck on their
# steps) and every tracked shell script (needs Docker).
ACTIONLINT := rhysd/actionlint:1.7.7@sha256:887a259a5a534f3c4f36cb02dca341673c6089431057242cdc931e9f133147e9
ci-lint:
	docker run --rm -v "$(CURDIR):/repo:ro" -w /repo $(ACTIONLINT) -color
	shellcheck $$(git ls-files '*.sh')
testpanels-down:
	for d in /tmp/karvan-panels/karvan-test-*; do [ -f $$d/docker-compose.yml ] && docker compose -f $$d/docker-compose.yml -p $$(basename $$d) down; done; true
# Full gate: before a phase ends or a release.
check-full: check test test-db test-panels test-acme vuln audit-web web

clean:
	rm -rf bin coverage.out web/dist/assets web/dist/.vite web/dist/*.html web/dist/*.js web/dist/*.svg
