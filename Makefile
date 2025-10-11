# Makefile for spc - Crestron SIMPL+ Compiler Wrapper

.PHONY: build clean test fmt vet help

# Build the binary into the build directory
build:
	mkdir -p build
	go build -o build/spc.exe

# Clean build artifacts
clean:
	rm -rf build

# Run tests
test:
	go test ./...

# Format Go code
fmt:
	go fmt ./...

# Run go vet
vet:
	go vet ./...

# Run both fmt and vet
lint: fmt vet

# Build and run tests
all: build test

# Show help
help:
	@echo "Available targets:"
	@echo "  build    - Build the spc binary into build/ directory"
	@echo "  clean    - Remove build artifacts"
	@echo "  test     - Run Go tests"
	@echo "  fmt      - Format Go code"
	@echo "  vet      - Run go vet"
	@echo "  lint     - Run fmt and vet"
	@echo "  all      - Build and run tests"
	@echo "  help     - Show this help"
