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

.PHONY: build build-go web web-deps test lint lint-web fmt floor cover check check-web check-full vuln audit-web clean

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

# Full gate: before a phase ends or a release.
check-full: check test vuln audit-web web

clean:
	rm -rf bin coverage.out web/dist/assets web/dist/.vite web/dist/*.html web/dist/*.js web/dist/*.svg
