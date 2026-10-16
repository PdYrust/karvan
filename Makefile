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
.PHONY: build build-go web web-deps test lint lint-web fmt floor cover check check-web check-full vuln audit-web clean test-db testdb-seed testdb-down

build: web build-go

build-go:
	CGO_ENABLED=0 go build -trimpath -ldflags "$(LDFLAGS)" -o $(BIN) ./cmd/karvan

web-deps:
	$(PNPM) install --frozen-lockfile

web: web-deps
	$(PNPM) run build
	$(PNPM) run size

test:
	go test -race ./...

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
	$(TESTDB_ENV) go test -tags dbtest ./internal/panel/... ./internal/migration/... ./internal/discovery/...
# Rebuild the schema fixtures with the panels' own migrations.
testdb-seed:
	./scripts/testdb/seed.sh
testdb-down:
	$(TESTDB) down
# Full gate: before a phase ends or a release.
check-full: check test test-db vuln audit-web web

clean:
	rm -rf bin coverage.out web/dist/assets web/dist/.vite web/dist/*.html web/dist/*.js web/dist/*.svg
