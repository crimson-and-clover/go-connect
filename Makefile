.PHONY: all build clean test lint install release

# Binary name
BINARY_NAME=go-connect
CMD_PATH=./cmd/go-connect

# Build directory
BUILD_DIR=build

# Version info
VERSION?=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME=$(shell date -u '+%Y-%m-%d_%H:%M:%S')
LDFLAGS=-ldflags "-s -w -X main.version=$(VERSION) -X main.buildTime=$(BUILD_TIME)"

# Default target
all: build

# Build for current platform
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) $(CMD_PATH)
	@echo "Built: $(BUILD_DIR)/$(BINARY_NAME)"

# Clean build artifacts
clean:
	@echo "Cleaning..."
	@rm -rf $(BUILD_DIR) dist/
	@go clean -cache
	@echo "Cleaned"

# Run tests
test:
	@echo "Running tests..."
	go test -v ./...

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

# Run linter
lint:
	@echo "Running linter..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run ./...; \
	else \
		echo "golangci-lint not installed. Install with:"; \
		echo "  curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $$(go env GOPATH)/bin"; \
		exit 1; \
	fi

# Install locally
install: build
	@echo "Installing $(BINARY_NAME)..."
	@cp $(BUILD_DIR)/$(BINARY_NAME) $(GOPATH)/bin/ || cp $(BUILD_DIR)/$(BINARY_NAME) ~/go/bin/
	@echo "Installed to GOPATH/bin"

# Run the application
run: build
	$(BUILD_DIR)/$(BINARY_NAME)

# Development mode - build and run
dev:
	go run $(CMD_PATH)

# Build for all platforms
release:
	@echo "Building releases..."
	@mkdir -p $(BUILD_DIR)

	@echo "  Linux AMD64..."
	@GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 $(CMD_PATH)

	@echo "  Linux ARM64..."
	@GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64 $(CMD_PATH)

	@echo "  Windows AMD64..."
	@GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe $(CMD_PATH)

	@echo "  Windows ARM64..."
	@GOOS=windows GOARCH=arm64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-windows-arm64.exe $(CMD_PATH)

	@echo "  macOS AMD64..."
	@GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 $(CMD_PATH)

	@echo "  macOS ARM64..."
	@GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 $(CMD_PATH)

	@echo "  FreeBSD AMD64..."
	@GOOS=freebsd GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-freebsd-amd64 $(CMD_PATH)

	@echo "Done! Binaries in $(BUILD_DIR)/"

# Generate checksums for releases
checksums:
	@cd $(BUILD_DIR) && sha256sum $(BINARY_NAME)-* > checksums.txt
	@echo "Checksums generated: $(BUILD_DIR)/checksums.txt"

# Format code
fmt:
	@echo "Formatting code..."
	go fmt ./...

# Download dependencies
deps:
	@echo "Downloading dependencies..."
	go mod download
	go mod tidy

# Check for vulnerabilities
vuln:
	@echo "Checking for vulnerabilities..."
	@if command -v govulncheck >/dev/null 2>&1; then \
		govulncheck ./...; \
	else \
		echo "govulncheck not installed. Install with:"; \
		echo "  go install golang.org/x/vuln/cmd/govulncheck@latest"; \
		exit 1; \
	fi

# Show help
help:
	@echo "Available targets:"
	@echo "  make build         - Build for current platform"
	@echo "  make clean         - Clean build artifacts"
	@echo "  make test          - Run tests"
	@echo "  make test-coverage - Run tests with coverage report"
	@echo "  make lint          - Run linter"
	@echo "  make install       - Install to GOPATH/bin"
	@echo "  make run           - Build and run"
	@echo "  make dev           - Run in development mode"
	@echo "  make release       - Build for all platforms"
	@echo "  make checksums     - Generate checksums for releases"
	@echo "  make fmt           - Format code"
	@echo "  make deps          - Download and tidy dependencies"
	@echo "  make vuln          - Check for vulnerabilities"
