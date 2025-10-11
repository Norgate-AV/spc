# Makefile for spc - Crestron SIMPL+ Compiler Wrapper

SHELL = /bin/bash

# Variables
APP_NAME := spc.exe
GO_MODULE := github.com/Norgate-AV/spc
BUILD_DIR = bin
BINARY = $(BUILD_DIR)/$(APP_NAME)

# Build configuration
DIST_DIR := dist
SRC_DIR := .

# Go build settings
CGO_ENABLED := 0

# Version information (from git tags and commit)
VERSION = dev
COMMIT = unknown
BUILD_TIME = unknown

# Optimization flags for smallest possible binary
GCFLAGS := -gcflags="all=-l"
ASMFLAGS := -asmflags="all=-trimpath=$(PWD)"
LDFLAGS_BASE := -s -w -buildid= -X github.com/Norgate-AV/spc/internal/version.Version=$(VERSION) -X github.com/Norgate-AV/spc/internal/version.Commit=$(COMMIT) -X github.com/Norgate-AV/spc/internal/version.BuildTime=$(BUILD_TIME)
LDFLAGS := -ldflags "$(LDFLAGS_BASE)"
LDFLAGS_RELEASE := -ldflags "$(LDFLAGS_BASE) -extldflags '-static'"

# Build tags for optimized builds
BUILD_TAGS := netgo osusergo

# Additional optimization settings
GOOS_BUILD := $(if $(GOOS),$(GOOS),$(shell go env GOOS))
GOARCH_BUILD := $(if $(GOARCH),$(GOARCH),$(shell go env GOARCH))

.PHONY: build clean test fmt vet help lint all deps ci goreleaser-check goreleaser-test

# Build the binary into the build directory
build: $(BUILD_DIR)
	@CGO_ENABLED=$(CGO_ENABLED) go build $(LDFLAGS) -tags "$(BUILD_TAGS)" -o $(BINARY) ./$(SRC_DIR)

$(BUILD_DIR):
	@mkdir -p $(BUILD_DIR)

# Clean build artifacts
clean:
	@rm -rf $(BUILD_DIR) $(DIST_DIR) coverage.out coverage.html

# Run tests
test:
	@go test ./...

test-coverage:
	@go test -race -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@go tool cover -func=coverage.out | tail -1

# Format Go code
fmt:
	@go fmt ./...

# Run go vet
vet:
	@go vet ./...

# Run both fmt and vet
lint: fmt vet
	@go tool golangci-lint run

# Build and run tests
all: clean test build

deps:
	@go mod download

ci: deps lint test build

goreleaser-check:
	@go tool goreleaser check || echo "⚠️  goreleaser not found"

goreleaser-test:
	@./scripts/test-goreleaser.sh

# Show help
help:
	@echo "Available targets:"
	@echo "  build    - Build the spc binary into $(BUILD_DIR)/ directory"
	@echo "  clean    - Remove build artifacts"
	@echo "  test     - Run Go tests"
	@echo "  fmt      - Format Go code"
	@echo "  vet      - Run go vet"
	@echo "  lint     - Run fmt and vet"
	@echo "  all      - Build and run tests"
	@echo "  help     - Show this help"
