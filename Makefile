MODULE  := github.com/magifd2/stail
BINARY  := stail
BIN_DIR := bin

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -ldflags "-s -w -X main.version=$(VERSION)"

# Allow callers to override GOCACHE / GOMODCACHE for environments where the
# default cache directory is not writable (e.g. sandboxes, CI containers).
# Example: make build GOCACHE=/tmp/go-cache GOMODCACHE=/tmp/gopath/pkg/mod
GOCACHE    ?= $(HOME)/.cache/go-build
GOMODCACHE ?= $(shell go env GOPATH)/pkg/mod

export GOCACHE
export GOMODCACHE

PLATFORMS := \
	darwin/amd64 \
	darwin/arm64 \
	linux/amd64 \
	linux/arm64 \
	windows/amd64

.PHONY: build build-all test lint clean tidy help

## build: Build binary for the current OS/Arch → ./bin/stail
build:
	@mkdir -p $(BIN_DIR)
	go build $(LDFLAGS) -o $(BIN_DIR)/$(BINARY) .

## build-all: Cross-compile for all target platforms
build-all:
	@mkdir -p $(BIN_DIR)
	$(foreach platform,$(PLATFORMS),$(call build_platform,$(platform)))

define build_platform
	$(eval OS   := $(word 1,$(subst /, ,$(1))))
	$(eval ARCH := $(word 2,$(subst /, ,$(1))))
	$(eval EXT  := $(if $(filter windows,$(OS)),.exe,))
	$(eval OUT  := $(BIN_DIR)/$(BINARY)-$(OS)-$(ARCH)$(EXT))
	@echo "Building $(OUT)..."
	GOOS=$(OS) GOARCH=$(ARCH) go build $(LDFLAGS) -o $(OUT) .

endef

## test: Run all unit tests
test:
	go test -v -race -count=1 ./...

## lint: Run golangci-lint (requires golangci-lint to be installed)
lint:
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run ./...; \
	else \
		echo "golangci-lint not found, skipping"; \
	fi

## tidy: Tidy go modules
tidy:
	go mod tidy

## clean: Remove build artifacts
clean:
	rm -rf $(BIN_DIR)

## help: Show available targets
help:
	@grep -E '^## ' $(MAKEFILE_LIST) | sed 's/## //'
