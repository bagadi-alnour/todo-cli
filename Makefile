# Todo CLI Makefile

# Variables
BINARY_NAME=todo
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "0.1.0")
BUILD_DATE=$(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS=-ldflags "-X github.com/bagadi-alnour/todo-cli/cmd.Version=$(VERSION) -X github.com/bagadi-alnour/todo-cli/cmd.BuildDate=$(BUILD_DATE)"

# Go related variables
GOBASE=$(shell pwd)
GOBIN=$(GOBASE)/bin
GOCMD=go
# Use a throwaway Go build cache (keeps runs reproducible-ish without persisting between builds)
GOCACHE ?= $(TMPDIR)/todo-go-cache
export GOCACHE

# Build the binary
.PHONY: build
build:
	@echo "Building $(BINARY_NAME)..."
	$(GOCMD) build $(LDFLAGS) -o $(BINARY_NAME) ./cmd/todo

# Build for all platforms
.PHONY: build-all
build-all: build-linux build-darwin build-windows

.PHONY: build-linux
build-linux:
	@echo "Building for Linux..."
	GOOS=linux GOARCH=amd64 $(GOCMD) build $(LDFLAGS) -o $(BINARY_NAME)-linux-amd64 ./cmd/todo
	GOOS=linux GOARCH=arm64 $(GOCMD) build $(LDFLAGS) -o $(BINARY_NAME)-linux-arm64 ./cmd/todo

.PHONY: build-darwin
build-darwin:
	@echo "Building for macOS..."
	GOOS=darwin GOARCH=amd64 $(GOCMD) build $(LDFLAGS) -o $(BINARY_NAME)-darwin-amd64 ./cmd/todo
	GOOS=darwin GOARCH=arm64 $(GOCMD) build $(LDFLAGS) -o $(BINARY_NAME)-darwin-arm64 ./cmd/todo

.PHONY: build-windows
build-windows:
	@echo "Building for Windows..."
	GOOS=windows GOARCH=amd64 $(GOCMD) build $(LDFLAGS) -o $(BINARY_NAME)-windows-amd64.exe ./cmd/todo

# Install locally
.PHONY: install
install: build
	@echo "Installing $(BINARY_NAME) to ~/bin..."
	@mkdir -p ~/bin
	cp $(BINARY_NAME) ~/bin/$(BINARY_NAME)
	@echo "Make sure ~/bin is in your PATH"

# Install globally (requires sudo)
.PHONY: install-global
install-global: build
	@echo "Installing $(BINARY_NAME) to /usr/local/bin..."
	sudo cp $(BINARY_NAME) /usr/local/bin/$(BINARY_NAME)

# Run tests
.PHONY: test
test:
	$(GOCMD) test -v ./...

# Run tests with coverage
.PHONY: test-coverage
test-coverage:
	$(GOCMD) test -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Format code
.PHONY: fmt
fmt:
	$(GOCMD) fmt ./...

# Lint code
.PHONY: lint
lint:
	@if command -v golangci-lint >/dev/null; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not installed. Run: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
	fi

# Clean build artifacts
.PHONY: clean
clean:
	@echo "Cleaning..."
	rm -f $(BINARY_NAME)
	rm -f $(BINARY_NAME)-*
	rm -f coverage.out coverage.html
	$(GOCMD) clean

# Run the application
.PHONY: run
run: build
	./$(BINARY_NAME)

# Development: watch and rebuild on changes
.PHONY: dev
dev:
	@if command -v air >/dev/null; then \
		air; \
	else \
		echo "air not installed. Run: go install github.com/cosmtrek/air@latest"; \
		echo "Falling back to simple run..."; \
		$(MAKE) run; \
	fi

# Generate shell completions
.PHONY: completions
completions: build
	@mkdir -p completions
	./$(BINARY_NAME) completion bash > completions/$(BINARY_NAME).bash
	./$(BINARY_NAME) completion zsh > completions/_$(BINARY_NAME)
	./$(BINARY_NAME) completion fish > completions/$(BINARY_NAME).fish
	@echo "Completions generated in completions/"

# Show help
.PHONY: help
help:
	@echo "Todo CLI Makefile"
	@echo ""
	@echo "Usage:"
	@echo "  make build           Build the binary"
	@echo "  make build-all       Build for all platforms"
	@echo "  make install         Install to ~/bin"
	@echo "  make install-global  Install to /usr/local/bin (requires sudo)"
	@echo "  make test            Run tests"
	@echo "  make test-coverage   Run tests with coverage report"
	@echo "  make fmt             Format code"
	@echo "  make lint            Run linter"
	@echo "  make clean           Clean build artifacts"
	@echo "  make run             Build and run"
	@echo "  make dev             Development mode with hot reload"
	@echo "  make completions     Generate shell completions"
	@echo ""
	@echo "Variables:"
	@echo "  VERSION=$(VERSION)"
	@echo "  BUILD_DATE=$(BUILD_DATE)"

.DEFAULT_GOAL := build
