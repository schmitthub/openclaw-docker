.DEFAULT_GOAL := help

.PHONY: help build build-all run version test vet lint fmt tidy check \
	licenses licenses-check release-dry-run clean

BIN_DIR ?= ./bin
BINARY ?= openclaw-docker
MAIN_PACKAGE ?= .
DIST_DIR ?= ./dist
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
DATE ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS ?= -s -w -X github.com/schmitthub/openclaw-docker/internal/build.Version=$(VERSION) -X github.com/schmitthub/openclaw-docker/internal/build.Date=$(DATE)

help:
	@echo "OpenClaw Docker CLI - developer and CI tasks"
	@echo ""
	@echo "Build and run:"
	@echo "  build            Build local CLI binary to $(BIN_DIR)/$(BINARY)"
	@echo "  build-all        Build release binaries for darwin/linux amd64/arm64"
	@echo "  run              Run CLI from source (go run . --help)"
	@echo "  version          Print resolved build version and metadata"
	@echo ""
	@echo "Quality gates:"
	@echo "  test             Run unit tests"
	@echo "  vet              Run go vet"
	@echo "  lint             Run golangci-lint"
	@echo "  fmt              Run gofmt on all Go files"
	@echo "  tidy             Run go mod tidy"
	@echo "  check            Run test + vet + lint"
	@echo ""
	@echo "Release and housekeeping:"
	@echo "  release-dry-run  Run goreleaser snapshot build"
	@echo "  licenses-check   Validate required license docs exist"
	@echo "  clean            Remove build artifacts"

build:
	@mkdir -p $(BIN_DIR)
	go build -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/$(BINARY) $(MAIN_PACKAGE)

build-all:
	@mkdir -p $(DIST_DIR)
	@set -e; \
	for os in darwin linux; do \
		for arch in amd64 arm64; do \
			out="$(DIST_DIR)/$(BINARY)-$$os-$$arch"; \
			echo "Building $$out"; \
			GOOS=$$os GOARCH=$$arch CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o "$$out" $(MAIN_PACKAGE); \
		done; \
	done

run:
	go run . --help

version:
	@echo "VERSION=$(VERSION)"
	@echo "DATE=$(DATE)"

test:
	go test ./...

vet:
	go vet ./...

lint:
	golangci-lint run --config .golangci.yml

fmt:
	@gofmt -w $$(find . -type f -name '*.go' -not -path './vendor/*')

tidy:
	go mod tidy

check: test vet lint

licenses:
	@echo "Validating license documentation..."
	@test -f LICENSE || (echo "ERROR: LICENSE is missing" >&2; exit 1)
	@test -f README.md || (echo "ERROR: README.md is missing" >&2; exit 1)
	@echo "License documentation is present."

licenses-check: licenses

release-dry-run:
	goreleaser release --snapshot --clean

clean:
	rm -rf $(BIN_DIR) $(DIST_DIR) coverage.out
