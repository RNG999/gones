# Makefile for gones - Go NES Emulator

# Build variables
BINARY_NAME=gones
VERSION ?= $(shell git describe --tags --exact-match 2>/dev/null || echo "dev")
GIT_COMMIT=$(shell git rev-parse HEAD 2>/dev/null || echo "unknown")
BUILD_TIME=$(shell date -u +%Y-%m-%dT%H:%M:%SZ)
BUILD_USER=$(shell whoami)@$(shell hostname)

# Go build flags
LDFLAGS=-ldflags "-X gones/internal/version.Version=$(VERSION) \
                 -X gones/internal/version.GitCommit=$(GIT_COMMIT) \
                 -X gones/internal/version.BuildTime=$(BUILD_TIME) \
                 -X gones/internal/version.BuildUser=$(BUILD_USER)"

# Default target
.PHONY: all
all: build

# Build the main binary
.PHONY: build
build:
	@echo "Building $(BINARY_NAME) version $(VERSION)..."
	go build $(LDFLAGS) -o $(BINARY_NAME) ./cmd/gones

# Build with race detection
.PHONY: build-race
build-race:
	@echo "Building $(BINARY_NAME) with race detection..."
	go build -race $(LDFLAGS) -o $(BINARY_NAME) ./cmd/gones

# Install the binary
.PHONY: install
install:
	@echo "Installing $(BINARY_NAME)..."
	go install $(LDFLAGS) ./cmd/gones

# Clean build artifacts
.PHONY: clean
clean:
	@echo "Cleaning build artifacts..."
	rm -f $(BINARY_NAME)
	go clean ./...

# Run tests
.PHONY: test
test:
	@echo "Running tests..."
	go test ./internal/cpu -v
	go test ./internal/ppu -v
	go test ./internal/memory -v
	go test ./internal/cartridge -v

# Run tests with coverage
.PHONY: test-coverage
test-coverage:
	@echo "Running tests with coverage..."
	go test -coverprofile=coverage.out ./internal/...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Run performance tests
.PHONY: test-performance
test-performance:
	@echo "Running performance tests..."
	go test -bench=. -benchmem ./internal/...

# Run integration tests
.PHONY: test-integration
test-integration:
	@echo "Running integration tests..."
	go test ./test/integration/... -v

# Format code
.PHONY: fmt
fmt:
	@echo "Formatting code..."
	go fmt ./...

# Lint code
.PHONY: lint
lint:
	@echo "Linting code..."
	golangci-lint run ./...

# Vet code
.PHONY: vet
vet:
	@echo "Vetting code..."
	go vet ./...

# Check dependencies
.PHONY: deps
deps:
	@echo "Checking dependencies..."
	go mod verify
	go mod tidy

# Update dependencies
.PHONY: update-deps
update-deps:
	@echo "Updating dependencies..."
	go get -u ./...
	go mod tidy

# Show version information
.PHONY: version
version:
	@echo "Version: $(VERSION)"
	@echo "Git Commit: $(GIT_COMMIT)"
	@echo "Build Time: $(BUILD_TIME)"
	@echo "Build User: $(BUILD_USER)"

# Development build with debug info
.PHONY: build-dev
build-dev:
	@echo "Building development version..."
	go build -gcflags="all=-N -l" $(LDFLAGS) -o $(BINARY_NAME) ./cmd/gones

# Release build with optimizations
.PHONY: build-release
build-release:
	@echo "Building release version..."
	CGO_ENABLED=0 go build -a -installsuffix cgo $(LDFLAGS) -o $(BINARY_NAME) ./cmd/gones
	strip $(BINARY_NAME) 2>/dev/null || true

# Cross-compile for different platforms
.PHONY: build-cross
build-cross:
	@echo "Cross-compiling for multiple platforms..."
	@mkdir -p dist
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o dist/$(BINARY_NAME)-linux-amd64 ./cmd/gones
	GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o dist/$(BINARY_NAME)-linux-arm64 ./cmd/gones
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o dist/$(BINARY_NAME)-darwin-amd64 ./cmd/gones
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o dist/$(BINARY_NAME)-darwin-arm64 ./cmd/gones
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o dist/$(BINARY_NAME)-windows-amd64.exe ./cmd/gones

# Docker build
.PHONY: docker-build
docker-build:
	@echo "Building Docker image..."
	docker build -t gones:$(VERSION) .

# Run with sample ROM (if available)
.PHONY: run
run: build
	@echo "Running $(BINARY_NAME)..."
	@if [ -f roms/sample.nes ]; then \
		./$(BINARY_NAME) -rom roms/sample.nes; \
	else \
		./$(BINARY_NAME); \
	fi

# Run with debug flags
.PHONY: run-debug
run-debug: build-dev
	@echo "Running $(BINARY_NAME) in debug mode..."
	@if [ -f roms/sample.nes ]; then \
		./$(BINARY_NAME) -rom roms/sample.nes -debug; \
	else \
		./$(BINARY_NAME) -debug; \
	fi

# Quick development cycle
.PHONY: dev
dev: fmt vet build

# Full check before commit
.PHONY: check
check: fmt vet lint test

# Help target
.PHONY: help
help:
	@echo "Available targets:"
	@echo "  build          - Build the main binary"
	@echo "  build-dev      - Build with debug information"
	@echo "  build-release  - Build optimized release binary"
	@echo "  build-race     - Build with race detection"
	@echo "  build-cross    - Cross-compile for multiple platforms"
	@echo "  install        - Install the binary"
	@echo "  clean          - Clean build artifacts"
	@echo "  test           - Run core tests"
	@echo "  test-coverage  - Run tests with coverage report"
	@echo "  test-performance - Run performance benchmarks"
	@echo "  test-integration - Run integration tests"
	@echo "  fmt            - Format code"
	@echo "  lint           - Lint code"
	@echo "  vet            - Vet code"
	@echo "  deps           - Check dependencies"
	@echo "  update-deps    - Update dependencies"
	@echo "  version        - Show version information"
	@echo "  run            - Build and run emulator"
	@echo "  run-debug      - Build and run in debug mode"
	@echo "  dev            - Quick development cycle (fmt + vet + build)"
	@echo "  check          - Full check before commit"
	@echo "  docker-build   - Build Docker image"
	@echo "  help           - Show this help"