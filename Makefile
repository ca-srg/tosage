# tosage Makefile
# Build configuration for tosage CLI and daemon (Darwin/macOS only)

# Variables
BINARY_NAME := tosage
VERSION := $(shell git describe --tags --always 2>/dev/null || echo "dev")
BUILD_TIME := $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')
GO_VERSION := $(shell go version | awk '{print $$3}')

# Code signing and notarization variables (optional)
# Set these environment variables to enable signing and notarization:
# CODESIGN_IDENTITY - Developer ID certificate (e.g., "Developer ID Application: Your Name (TEAMID)")
# NOTARIZE - Set to "true" to enable notarization
# API_KEY_ID - App Store Connect API Key ID
# API_KEY_PATH - Path to .p8 key file
# API_ISSUER - Issuer ID from App Store Connect

# Build flags
LDFLAGS := -ldflags "-w -s -X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME)"
BUILD_TAGS := darwin
BUILD_FLAGS := -tags "$(BUILD_TAGS)"

# Default target
.DEFAULT_GOAL := build

# Build targets
.PHONY: all
all: clean test build

.PHONY: build
build:
	@echo "Building $(BINARY_NAME) for macOS..."
	@go build $(BUILD_FLAGS) $(LDFLAGS) -o $(BINARY_NAME) .
	@echo "Build complete: ./$(BINARY_NAME)"

.PHONY: build-darwin
build-darwin:
	@echo "Building $(BINARY_NAME) for Darwin (macOS)..."
	@echo "Note: Cross-compilation with CGO is not supported. Building for current architecture only."
	@CURRENT_ARCH=$$(uname -m); \
	if [ "$$CURRENT_ARCH" = "arm64" ]; then \
		echo "Building for ARM64..."; \
		CGO_ENABLED=1 GOOS=darwin GOARCH=arm64 go build -tags "darwin" $(LDFLAGS) -o $(BINARY_NAME)-darwin-arm64 .; \
	else \
		echo "Error: Only ARM64 architecture is supported"; \
		exit 1; \
	fi
	@echo "Darwin build complete"

.PHONY: build-all
build-all: build-darwin
	@echo "All platform builds complete"

# Installation targets
.PHONY: install
install: build
	@echo "Installing $(BINARY_NAME) to /usr/local/bin..."
	@sudo cp $(BINARY_NAME) /usr/local/bin/
	@echo "Installation complete"

.PHONY: install-daemon
install-daemon: build
	@echo "Installing $(BINARY_NAME) daemon for macOS..."
	@sudo cp $(BINARY_NAME) /usr/local/bin/
	@echo "Creating LaunchAgent..."
	@mkdir -p ~/Library/LaunchAgents
	@echo '<?xml version="1.0" encoding="UTF-8"?>' > ~/Library/LaunchAgents/com.tosage.daemon.plist
	@echo '<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">' >> ~/Library/LaunchAgents/com.tosage.daemon.plist
	@echo '<plist version="1.0">' >> ~/Library/LaunchAgents/com.tosage.daemon.plist
	@echo '<dict>' >> ~/Library/LaunchAgents/com.tosage.daemon.plist
	@echo '    <key>Label</key>' >> ~/Library/LaunchAgents/com.tosage.daemon.plist
	@echo '    <string>com.tosage.daemon</string>' >> ~/Library/LaunchAgents/com.tosage.daemon.plist
	@echo '    <key>ProgramArguments</key>' >> ~/Library/LaunchAgents/com.tosage.daemon.plist
	@echo '    <array>' >> ~/Library/LaunchAgents/com.tosage.daemon.plist
	@echo '        <string>/usr/local/bin/tosage</string>' >> ~/Library/LaunchAgents/com.tosage.daemon.plist
	@echo '        <string>--daemon</string>' >> ~/Library/LaunchAgents/com.tosage.daemon.plist
	@echo '    </array>' >> ~/Library/LaunchAgents/com.tosage.daemon.plist
	@echo '    <key>RunAtLoad</key>' >> ~/Library/LaunchAgents/com.tosage.daemon.plist
	@echo '    <true/>' >> ~/Library/LaunchAgents/com.tosage.daemon.plist
	@echo '    <key>KeepAlive</key>' >> ~/Library/LaunchAgents/com.tosage.daemon.plist
	@echo '    <false/>' >> ~/Library/LaunchAgents/com.tosage.daemon.plist
	@echo '    <key>StandardOutPath</key>' >> ~/Library/LaunchAgents/com.tosage.daemon.plist
	@echo '    <string>/tmp/tosage.log</string>' >> ~/Library/LaunchAgents/com.tosage.daemon.plist
	@echo '    <key>StandardErrorPath</key>' >> ~/Library/LaunchAgents/com.tosage.daemon.plist
	@echo '    <string>/tmp/tosage.error.log</string>' >> ~/Library/LaunchAgents/com.tosage.daemon.plist
	@echo '</dict>' >> ~/Library/LaunchAgents/com.tosage.daemon.plist
	@echo '</plist>' >> ~/Library/LaunchAgents/com.tosage.daemon.plist
	@echo "Loading daemon..."
	@launchctl load ~/Library/LaunchAgents/com.tosage.daemon.plist
	@echo "Daemon installation complete"

.PHONY: uninstall-daemon
uninstall-daemon:
	@echo "Uninstalling $(BINARY_NAME) daemon..."
	@launchctl unload ~/Library/LaunchAgents/com.tosage.daemon.plist 2>/dev/null || true
	@rm -f ~/Library/LaunchAgents/com.tosage.daemon.plist
	@echo "Daemon uninstalled"

# Development targets
.PHONY: run
run:
	@go run $(BUILD_FLAGS) . $(ARGS)

.PHONY: run-daemon
run-daemon:
	@go run $(BUILD_FLAGS) . --daemon

.PHONY: run-cli
run-cli:
	@go run $(BUILD_FLAGS) . --cli

# Testing targets
.PHONY: test
test:
	@echo "Running tests..."
	@go test $(BUILD_FLAGS) -v ./...

.PHONY: test-integration
test-integration:
	@echo "Running integration tests..."
	@go test $(BUILD_FLAGS) -v -tags=integration ./...

.PHONY: test-coverage
test-coverage:
	@echo "Running tests with coverage..."
	@go test $(BUILD_FLAGS) -v -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Code quality targets
.PHONY: lint
lint:
	@echo "Running linter..."
	@golangci-lint run

.PHONY: fmt
fmt:
	@echo "Formatting code..."
	@go fmt ./...

.PHONY: vet
vet:
	@echo "Running go vet..."
	@go vet $(BUILD_FLAGS) ./...

.PHONY: check
check: fmt vet lint test
	@echo "All checks passed"

# Dependency management
.PHONY: deps
deps:
	@echo "Downloading dependencies..."
	@go mod download

.PHONY: tidy
tidy:
	@echo "Tidying dependencies..."
	@go mod tidy

.PHONY: vendor
vendor:
	@echo "Vendoring dependencies..."
	@go mod vendor

# DMG targets
.PHONY: dmg-check
dmg-check:
	@./scripts/check-dmg-deps.sh

.PHONY: app-bundle-arm64
app-bundle-arm64: build-darwin dmg-check
	@echo "Creating macOS app bundle for ARM64..."
	@ARCH=arm64 ./build/dmg/create-app-bundle.sh

.PHONY: dmg-arm64
dmg-arm64: app-bundle-arm64
	@echo "Creating DMG for ARM64..."
	@echo "Environment: CODESIGN_IDENTITY=$${CODESIGN_IDENTITY:-[NOT SET]}"
	@ARCH=arm64 ./build/dmg/create-dmg.sh || { \
		echo "DMG creation failed with exit code: $$?"; \
		echo "Current directory contents:"; \
		ls -la; \
		echo "build/dmg contents:"; \
		ls -la build/dmg/ 2>/dev/null || echo "build/dmg not found"; \
		exit 1; \
	}

.PHONY: dmg-all
dmg-all: dmg-arm64
	@echo "All DMG builds complete"

# Signed and notarized DMG targets
.PHONY: dmg-signed-arm64
dmg-signed-arm64:
	@if [ -z "$(CODESIGN_IDENTITY)" ]; then \
		echo "Error: CODESIGN_IDENTITY is not set"; \
		echo "Set it with: export CODESIGN_IDENTITY=\"Developer ID Application: Your Name (TEAMID)\""; \
		exit 1; \
	fi
	@echo "Creating signed DMG for ARM64..."
	@$(MAKE) app-bundle-arm64
	@ARCH=arm64 ./build/dmg/create-dmg.sh

.PHONY: dmg-notarized-arm64
dmg-notarized-arm64:
	@if [ -z "$(CODESIGN_IDENTITY)" ] || [ -z "$(API_KEY_ID)" ] || [ -z "$(API_KEY_PATH)" ] || [ -z "$(API_ISSUER)" ]; then \
		echo "Error: Required environment variables for notarization are not set"; \
		echo "Required variables:"; \
		echo "  CODESIGN_IDENTITY - Developer ID certificate"; \
		echo "  API_KEY_ID - App Store Connect API Key ID"; \
		echo "  API_KEY_PATH - Path to .p8 key file"; \
		echo "  API_ISSUER - Issuer ID from App Store Connect"; \
		exit 1; \
	fi
	@echo "Creating signed and notarized DMG for ARM64..."
	@NOTARIZE=true $(MAKE) dmg-signed-arm64

.PHONY: dmg-notarized-all
dmg-notarized-all: dmg-notarized-arm64
	@echo "All signed and notarized DMG builds complete"

.PHONY: dmg-verify
dmg-verify:
	@echo "Verifying DMG files..."
	@./scripts/verify-dmg.sh

.PHONY: dmg-clean
dmg-clean:
	@echo "Cleaning DMG build artifacts..."
	@rm -rf build/dmg/*.app
	@rm -rf build/dmg/dmg-staging
	@rm -f build/dmg/temp.dmg
	@rm -f *.dmg
	@echo "DMG artifacts cleaned"

# Release targets
.PHONY: release
release: build-all dmg-all
#release: check build-all dmg-all
	@echo "Creating release $(VERSION)..."
	@mkdir -p dist
	@cp $(BINARY_NAME)-darwin-* dist/
	@cd dist && tar -czf $(BINARY_NAME)-$(VERSION)-darwin-arm64.tar.gz $(BINARY_NAME)-darwin-arm64
	@echo "Copying DMG files to dist..."
	@if ls *.dmg 1>/dev/null 2>&1; then \
		cp *.dmg dist/; \
		echo "DMG files copied to dist/"; \
		ls -la dist/*.dmg; \
	else \
		echo "Warning: No DMG files found to copy"; \
	fi
	@echo "Release artifacts created in dist/"

# Clean targets
.PHONY: clean
clean: dmg-clean
	@echo "Cleaning build artifacts..."
	@rm -f $(BINARY_NAME) $(BINARY_NAME)-*
	@rm -rf dist/
	@rm -f coverage.out coverage.html
	@echo "Clean complete"

.PHONY: clean-all
clean-all: clean
	@echo "Cleaning all generated files..."
	@go clean -cache -testcache -modcache
	@echo "Deep clean complete"

# Help target
.PHONY: help
help:
	@echo "tosage Makefile (macOS only)"
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@echo "Build targets:"
	@echo "  build          - Build for current platform (macOS)"
	@echo "  build-darwin   - Build for macOS (arm64 only)"
	@echo "  build-all      - Build for all architectures"
	@echo ""
	@echo "DMG targets:"
	@echo "  app-bundle-arm64 - Create macOS app bundle for ARM64"
	@echo "  dmg-arm64      - Create DMG installer for ARM64 (unsigned)"
	@echo "  dmg-all        - Create DMG installers for all architectures (unsigned)"
	@echo "  dmg-signed-arm64 - Create signed DMG for ARM64 (requires CODESIGN_IDENTITY)"
	@echo "  dmg-notarized-arm64 - Create signed and notarized DMG for ARM64"
	@echo "  dmg-notarized-all - Create signed and notarized DMGs for all architectures"
	@echo "  dmg-verify     - Verify DMG files"
	@echo "  dmg-clean      - Clean DMG build artifacts"
	@echo ""
	@echo "Installation targets:"
	@echo "  install        - Install binary to /usr/local/bin"
	@echo "  install-daemon - Install as macOS daemon"
	@echo "  uninstall-daemon - Remove macOS daemon"
	@echo ""
	@echo "Development targets:"
	@echo "  run            - Run the application"
	@echo "  run-daemon     - Run in daemon mode"
	@echo "  run-cli        - Run in CLI mode"
	@echo ""
	@echo "Testing targets:"
	@echo "  test           - Run all tests"
	@echo "  test-integration - Run integration tests"
	@echo "  test-coverage  - Run tests with coverage report"
	@echo "  benchmark      - Run benchmarks"
	@echo ""
	@echo "Code quality targets:"
	@echo "  lint           - Run linter"
	@echo "  fmt            - Format code"
	@echo "  vet            - Run go vet"
	@echo "  check          - Run all quality checks"
	@echo ""
	@echo "Other targets:"
	@echo "  deps           - Download dependencies"
	@echo "  tidy           - Tidy dependencies"
	@echo "  vendor         - Vendor dependencies"
	@echo "  release        - Create release artifacts"
	@echo "  clean          - Remove build artifacts"
	@echo "  clean-all      - Deep clean including caches"
	@echo "  help           - Show this help message"
	@echo ""
	@echo "Code Signing and Notarization:"
	@echo "  To sign and notarize DMGs, set these environment variables:"
	@echo "  CODESIGN_IDENTITY - Developer ID certificate"
	@echo "  NOTARIZE=true     - Enable notarization"
	@echo "  API_KEY_ID        - App Store Connect API Key ID"
	@echo "  API_KEY_PATH      - Path to .p8 key file"
	@echo "  API_ISSUER        - Issuer ID from App Store Connect"