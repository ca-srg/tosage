# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

tosage is a Go application that tracks both Claude Code and Cursor token usage and sends metrics to Prometheus. It can run in CLI mode (outputs today's token count) or daemon mode (system tray application with periodic metrics sending).

## Key Commands

### Build
```bash
# Build for current platform (macOS)
make build

# Build for macOS ARM64
make build-darwin

# Build app bundle for macOS
make app-bundle-arm64

# Build DMG installer
make dmg-arm64
```

### Test
```bash
# Run all tests
make test

# Run tests with coverage
make test-coverage

# Run integration tests
make test-integration
```

### Code Quality
```bash
# Run linter (golangci-lint)
make lint

# Format code
make fmt

# Run go vet
make vet

# Run all checks (fmt, vet, lint, test)
make check
```

### Development
```bash
# Run in CLI mode
make run-cli

# Run in daemon mode
make run-daemon

# Install as system daemon
make install-daemon

# Clean build artifacts
make clean
```

## Architecture

The project follows Clean Architecture with clear separation of concerns:

### Domain Layer (`domain/`)
- **entity/**: Core business entities
  - `cc_entry.go`: Claude Code usage entry
  - `cursor_usage.go`: Cursor usage data (premium requests, usage-based pricing)
- **repository/**: Repository interfaces
- **errors.go**: Domain-specific errors
- **logger.go**: Logger interface

### Infrastructure Layer (`infrastructure/`)
- **config/**: Application configuration
- **di/**: Dependency injection container
- **logging/**: Logger implementations (debug, promtail)
- **repository/**: Repository implementations
  - `cursor_api_repository.go`: Cursor API client for usage data
  - `cursor_db_repository.go`: SQLite database for Cursor token history
  - `jsonl_cc_repository.go`: Claude Code data file reader
  - `prometheus_metrics_repository.go`: Prometheus remote write

### Use Case Layer (`usecase/`)
- **interface/**: Service interfaces
- **impl/**: Service implementations
  - `cc_service_impl.go`: Claude Code data processing
  - `cursor_service_impl.go`: Cursor API integration and token tracking
  - `metrics_service_impl.go`: Metrics collection and sending
  - `status_service_impl.go`: Application status tracking

### Interface Layer (`interface/`)
- **controller/**: Application controllers
  - `cli_controller.go`: CLI mode controller
  - `daemon_controller.go`: Daemon mode controller (macOS)
  - `systray_controller.go`: System tray UI (macOS)

## Key Functionality

### Token Usage Tracking
The application tracks token usage from two sources:

1. **Claude Code**: Searches for data in:
   - `~/.config/claude/projects/` (new default)
   - `~/.claude/projects/` (legacy)
   - `~/Library/Application Support/claude/projects/` (macOS)

2. **Cursor**: Uses Cursor API to fetch:
   - Premium (GPT-4) request usage
   - Usage-based pricing information
   - Team membership status

### Metrics Sending
When configured, sends metrics to Prometheus via remote write API:
- Claude Code token count
- Cursor token count (aggregated from database)
- Timestamp in JST
- Configurable interval (default from config)
- TODO: Vertex AI usage
- TODO: Bedrock usage

## Important Notes

- macOS only application (uses CGO for system tray)
- Time calculations use JST (Asia/Tokyo) timezone
- Daemon mode requires macOS system tray support
- Configuration file: `~/.config/tosage/config.json`