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

.PHONY: build build-go test lint fmt floor cover check check-full vuln clean

build: build-go

build-go:
	CGO_ENABLED=0 go build -trimpath -ldflags "$(LDFLAGS)" -o $(BIN) ./cmd/karvan

test:
	go test -race ./...

lint:
	$(GOLANGCI) run ./...

fmt:
	$(GOLANGCI) fmt ./...

floor:
	./scripts/check-floor.sh

cover:
	./scripts/check-coverage.sh

vuln:
	govulncheck ./...

# Fast gate: run after every change. cover runs the tests once with coverage.
check: floor lint cover

# Full gate: before a phase ends or a release.
check-full: check test vuln

clean:
	rm -rf bin coverage.out
