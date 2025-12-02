# Bombardino Makefile
# Build variables
APP_NAME := bombardino
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "v0.2.0-beta")
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS := -ldflags "-X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.buildTime=$(BUILD_TIME)"

# Go variables
GOCMD := go
GOBUILD := $(GOCMD) build
GOCLEAN := $(GOCMD) clean
GOTEST := $(GOCMD) test
GOGET := $(GOCMD) get
GOMOD := $(GOCMD) mod
GOFMT := $(GOCMD) fmt

# Directories
BINARY_DIR := bin
CMD_DIR := cmd/bombardino
DIST_DIR := dist
MCP_DIR := mcp

# Binary names
BINARY_UNIX := $(BINARY_DIR)/$(APP_NAME)
BINARY_WINDOWS := $(BINARY_DIR)/$(APP_NAME).exe
BINARY_DARWIN := $(BINARY_DIR)/$(APP_NAME)-darwin
BINARY_LINUX := $(BINARY_DIR)/$(APP_NAME)-linux

# Default target
.PHONY: all
all: clean test build

# Help target
.PHONY: help
help: ## Show this help message
	@echo "Bombardino Build System"
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

# Build targets
.PHONY: build
build: build-go build-mcp ## Build Go binary and MCP server

.PHONY: build-go
build-go: ## Build the Go binary for current platform
	@echo "Building $(APP_NAME) $(VERSION)..."
	@mkdir -p $(BINARY_DIR)
	$(GOBUILD) $(LDFLAGS) -o $(BINARY_UNIX) ./$(CMD_DIR)
	@echo "✅ Build complete: $(BINARY_UNIX)"

.PHONY: build-mcp
build-mcp: ## Build the MCP server
	@echo "Building MCP server..."
	@cd $(MCP_DIR) && npm install --silent && npm run build
	@echo "✅ MCP server build complete: $(MCP_DIR)/dist/"

.PHONY: build-all
build-all: clean build-linux build-darwin build-windows ## Build binaries for all platforms
	@echo "✅ All platform builds complete"

.PHONY: build-linux
build-linux: ## Build binary for Linux
	@echo "Building for Linux..."
	@mkdir -p $(BINARY_DIR)
	GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_LINUX) ./$(CMD_DIR)

.PHONY: build-darwin
build-darwin: ## Build binary for macOS
	@echo "Building for macOS..."
	@mkdir -p $(BINARY_DIR)
	GOOS=darwin GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_DARWIN) ./$(CMD_DIR)

.PHONY: build-windows
build-windows: ## Build binary for Windows
	@echo "Building for Windows..."
	@mkdir -p $(BINARY_DIR)
	GOOS=windows GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_WINDOWS) ./$(CMD_DIR)

# Development targets
.PHONY: run
run: build ## Build and run the application
	./$(BINARY_UNIX) -version

.PHONY: run-example
run-example: build ## Build and run with example config
	./$(BINARY_UNIX) -config=examples/example-config.json -workers=5

# Test targets
.PHONY: test
test: ## Run all tests
	@echo "Running tests..."
	$(GOTEST) -v ./...

.PHONY: test-coverage
test-coverage: ## Run tests with coverage report
	@echo "Running tests with coverage..."
	$(GOTEST) -v -race -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "✅ Coverage report generated: coverage.html"

.PHONY: test-short
test-short: ## Run only short tests
	$(GOTEST) -v -short ./...

.PHONY: bench
bench: ## Run benchmarks
	$(GOTEST) -v -bench=. -benchmem ./...

# Code quality targets
.PHONY: fmt
fmt: ## Format Go code
	@echo "Formatting code..."
	$(GOFMT) ./...

.PHONY: vet
vet: ## Run go vet
	@echo "Running go vet..."
	$(GOCMD) vet ./...

.PHONY: lint
lint: ## Run golangci-lint (requires golangci-lint to be installed)
	@echo "Running golangci-lint..."
	@which golangci-lint > /dev/null || (echo "❌ golangci-lint not installed. Run: make install-tools" && exit 1)
	golangci-lint run

.PHONY: check
check: fmt vet test ## Run all code quality checks

# Dependency management
.PHONY: deps
deps: ## Download dependencies
	@echo "Downloading dependencies..."
	$(GOMOD) download
	$(GOMOD) tidy

.PHONY: deps-update
deps-update: ## Update dependencies
	@echo "Updating dependencies..."
	$(GOMOD) get -u ./...
	$(GOMOD) tidy

# Installation targets
.PHONY: install
install: build ## Install binary to $GOPATH/bin
	@echo "Installing $(APP_NAME)..."
	cp $(BINARY_UNIX) $(GOPATH)/bin/$(APP_NAME)
	@echo "✅ Installed to $(GOPATH)/bin/$(APP_NAME)"

.PHONY: install-tools
install-tools: ## Install development tools
	@echo "Installing development tools..."
	$(GOGET) github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Release targets
.PHONY: release
release: clean test build-all ## Prepare release (build all platforms)
	@echo "Preparing release $(VERSION)..."
	@mkdir -p $(DIST_DIR)
	@cp $(BINARY_LINUX) $(DIST_DIR)/$(APP_NAME)-$(VERSION)-linux-amd64
	@cp $(BINARY_DARWIN) $(DIST_DIR)/$(APP_NAME)-$(VERSION)-darwin-amd64
	@cp $(BINARY_WINDOWS) $(DIST_DIR)/$(APP_NAME)-$(VERSION)-windows-amd64.exe
	@cp README.md $(DIST_DIR)/
	@cp LICENSE $(DIST_DIR)/
	@cp CHANGELOG.md $(DIST_DIR)/
	@echo "✅ Release artifacts ready in $(DIST_DIR)/"

.PHONY: release-zip
release-zip: release ## Create release zip archives
	@echo "Creating release archives..."
	@cd $(DIST_DIR) && zip $(APP_NAME)-$(VERSION)-linux-amd64.zip $(APP_NAME)-$(VERSION)-linux-amd64 README.md LICENSE CHANGELOG.md
	@cd $(DIST_DIR) && zip $(APP_NAME)-$(VERSION)-darwin-amd64.zip $(APP_NAME)-$(VERSION)-darwin-amd64 README.md LICENSE CHANGELOG.md
	@cd $(DIST_DIR) && zip $(APP_NAME)-$(VERSION)-windows-amd64.zip $(APP_NAME)-$(VERSION)-windows-amd64.exe README.md LICENSE CHANGELOG.md
	@echo "✅ Release archives created"

# Docker targets (for future use)
.PHONY: docker-build
docker-build: ## Build Docker image
	@echo "Docker support coming in future release..."

# Cleanup targets
.PHONY: clean
clean: ## Clean build artifacts
	@echo "Cleaning build artifacts..."
	$(GOCLEAN)
	@rm -rf $(BINARY_DIR)
	@rm -rf $(DIST_DIR)
	@rm -f coverage.out coverage.html
	@rm -rf $(MCP_DIR)/dist

.PHONY: clean-all
clean-all: clean ## Clean all artifacts including dependencies
	$(GOMOD) clean -cache
	@rm -rf $(MCP_DIR)/node_modules

.PHONY: clean-mcp
clean-mcp: ## Clean MCP server artifacts
	@echo "Cleaning MCP server..."
	@rm -rf $(MCP_DIR)/dist
	@rm -rf $(MCP_DIR)/node_modules

# Info targets
.PHONY: version
version: ## Show version information
	@echo "Version: $(VERSION)"
	@echo "Commit:  $(COMMIT)"
	@echo "Built:   $(BUILD_TIME)"

.PHONY: info
info: ## Show build information
	@echo "App Name:    $(APP_NAME)"
	@echo "Version:     $(VERSION)"
	@echo "Commit:      $(COMMIT)"
	@echo "Build Time:  $(BUILD_TIME)"
	@echo "Go Version:  $(shell $(GOCMD) version)"
	@echo "Binary Dir:  $(BINARY_DIR)"
	@echo "Dist Dir:    $(DIST_DIR)"