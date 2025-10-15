# Makefile for spc - Crestron SIMPL+ Compiler Wrapper

SHELL := C:/Program Files/Git/usr/bin/bash.exe

# Variables
APP_NAME := spc.exe
GO_MODULE := github.com/Norgate-AV/spc
BUILD_DIR = bin
BINARY = $(BUILD_DIR)/$(APP_NAME)

# Build configuration
DIST_DIR := dist
SRC_DIR := .
COVERAGE_DIR := .coverage

# Go build settings
CGO_ENABLED := 0

# Version information (from git tags and commit)
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
BUILD_TIME := $(shell git log -1 --format=%cI 2>/dev/null || echo unknown)

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

.PHONY: build build-release clean test test-coverage fmt vet help lint all deps ci run install

# Build the binary into the build directory
build: $(BUILD_DIR)
	@CGO_ENABLED=$(CGO_ENABLED) go build $(LDFLAGS) -tags "$(BUILD_TAGS)" -o $(BINARY) ./$(SRC_DIR)

# Build optimized release binary
build-release: $(BUILD_DIR)
	@CGO_ENABLED=$(CGO_ENABLED) go build $(LDFLAGS_RELEASE) -tags "$(BUILD_TAGS)" -trimpath -o $(BINARY) ./$(SRC_DIR)

$(BUILD_DIR):
	@mkdir -p $(BUILD_DIR)

# Run the application
run: build
	@$(BINARY)

# Install to GOPATH/bin
install:
	@CGO_ENABLED=$(CGO_ENABLED) go install $(LDFLAGS) -tags "$(BUILD_TAGS)" .

# Clean build artifacts
clean:
	@rm -rf $(BUILD_DIR) $(DIST_DIR) $(COVERAGE_DIR)

# Run tests
test:
	@go test ./... -v

# Run tests with coverage
test-coverage:
	@mkdir -p $(COVERAGE_DIR)
	@go test -coverprofile=$(COVERAGE_DIR)/coverage.out -covermode=atomic ./internal/...
	@go tool cover -html=$(COVERAGE_DIR)/coverage.out -o $(COVERAGE_DIR)/coverage.html
	@go tool cover -func=$(COVERAGE_DIR)/coverage.out | tail -1
	@echo "Coverage report: $(COVERAGE_DIR)/coverage.html"

# Format Go code
fmt:
	@go fmt ./...

# Run go vet
vet:
	@go vet ./...

# Run both fmt and vet
lint: fmt vet
	@golangci-lint run 2>/dev/null || echo "⚠️  golangci-lint not installed"

# Build and run tests
all: clean test build

deps:
	@go mod download

ci: deps lint test build

# Show help
help:
	@echo "Available targets:"
	@echo "  build         - Build the spc binary into $(BUILD_DIR)/ directory"
	@echo "  build-release - Build optimized release binary"
	@echo "  run           - Build and run the application"
	@echo "  install       - Install to GOPATH/bin"
	@echo "  clean         - Remove build artifacts"
	@echo "  test          - Run Go tests"
	@echo "  test-coverage - Run tests with coverage report"
	@echo "  fmt           - Format Go code"
	@echo "  vet           - Run go vet"
	@echo "  lint          - Run fmt, vet, and golangci-lint"
	@echo "  deps          - Download dependencies"
	@echo "  ci            - Run CI pipeline (deps, lint, test, build)"
	@echo "  all           - Clean, test, and build"
	@echo "  help          - Show this help"
