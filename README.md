# tosage

[🇯🇵 日本語版](./README_JA.md)

<p align="center">
  <img src="assets/icon.png" alt="tosage logo" width="256" height="256">
</p>

A Go application that tracks Claude Code and Cursor token usage and sends metrics to Prometheus. It can run in CLI mode (outputs today's token count) or daemon mode (system tray application with periodic metrics sending).

## Features

- **Token Usage Tracking**: Monitors token usage from both Claude Code and Cursor
- **Prometheus Integration**: Sends metrics via remote write API
- **Dual Mode Operation**: CLI mode for quick checks, daemon mode for continuous monitoring
- **macOS System Tray**: Native system tray support for daemon mode
- **Automatic Data Discovery**: Finds Claude Code data across multiple locations
- **Cursor API Integration**: Fetches premium request usage and pricing information

## Installation

### Pre-built Binaries

Download the latest release from [GitHub Releases](https://github.com/ca-srg/tosage/releases).

### From Source

```bash
git clone https://github.com/ca-srg/tosage.git
cd tosage
make build
```

## Configuration

```bash
# 1. Run application to generate config.json

# 2. Modify config.json
$ cat ~/.config/tosage/config.json
{
  "prometheus": {
    "remote_write_url": "https://<prometheus_url>/api/prom/push",
    "username": "",
    "password": ""
  },
  "logging": {
    "promtail": {
      "url": "https://<logs_url>",
      "username": "",
      "password": ""
    }
  }
}

# 3. Run again
```

## Usage

### CLI Mode

Outputs today's token count:

```bash
tosage
```

### Daemon Mode

Runs as a system tray application with periodic metrics sending:

```bash
tosage -d
```

## Building

### Requirements

#### Build Requirements

- Go 1.21 or higher
- macOS (for daemon mode)
- Make

#### Runtime Requirements

- Prometheus Remote Write API endpoint for metrics collection
- Grafana Loki (optional) for log aggregation via Promtail

### Build Commands

```bash
# Build for current platform
make build

# Build macOS ARM64 binary
make build-darwin

# Build app bundle for macOS
make app-bundle-arm64

# Build DMG installer
make dmg-arm64

# Run all checks (fmt, vet, lint, test)
make check
```

### macOS App Bundle and DMG Creation

#### App Bundle Targets

##### `app-bundle-arm64`
**Purpose**: Creates a macOS app bundle (.app)

1. **Binary Build**: Executes `build-darwin` to create Go binary
2. **Dependency Check**: Runs `dmg-check` to verify required tools
3. **App Bundle Creation**: Executes `create-app-bundle.sh` to create:
   - `tosage.app/Contents/MacOS/tosage` - Executable file
   - `tosage.app/Contents/Info.plist` - App metadata
   - `tosage.app/Contents/Resources/app.icns` - App icon
   - `tosage.app/Contents/PkgInfo` - App type information

#### DMG Targets

##### `dmg-arm64`
**Purpose**: Creates unsigned DMG installers

1. Creates app bundle (executes `app-bundle-*`)
2. Runs `create-dmg.sh` to create DMG:
   - Includes app bundle in DMG
   - Adds symlink to `/Applications`
   - Sets background image and window layout
   - Output: `tosage-{version}-darwin-{arch}.dmg`

##### `dmg-signed-arm64`
**Purpose**: Creates signed DMGs

- Requires `CODESIGN_IDENTITY` environment variable
- Adds code signature to app bundle and DMG

##### `dmg-notarized-arm64`
**Purpose**: Creates signed and notarized DMGs

- Adds Apple notarization in addition to signing
- Allows installation without Gatekeeper warnings

### Build Process Flow

```
Go Source Code
    ↓ (go build)
Executable Binary
    ↓ (create-app-bundle.sh)
.app Bundle
    ↓ (create-dmg.sh)
.dmg Installer
    ↓ (codesign + notarization)
Distributable DMG
```

### Usage Examples

#### Create unsigned DMG:
```bash
make dmg-arm64
```

#### Create signed DMG:
```bash
export CODESIGN_IDENTITY="Developer ID Application: Your Name (TEAMID)"
make dmg-signed-arm64
```

#### Create signed and notarized DMG:
```bash
export CODESIGN_IDENTITY="Developer ID Application: Your Name (TEAMID)"
export API_KEY_ID="your-key-id"
export API_KEY_PATH="/path/to/AuthKey_XXXXX.p8"
export API_ISSUER="your-issuer-id"
make dmg-notarized-arm64
```

#### Create for all architectures:
```bash
make dmg-notarized-all
```

## Architecture

The project follows Clean Architecture with clear separation of concerns:

### Domain Layer
- **Entities**: Core business entities (Claude Code entries, Cursor usage data)
- **Repository Interfaces**: Abstractions for data access
- **Domain Errors**: Business logic specific errors

### Infrastructure Layer
- **Configuration**: Application settings management
- **Dependency Injection**: IoC container for clean dependency management
- **Logging**: Multiple logger implementations (debug, promtail)
- **Repository Implementations**: 
  - Cursor API client for usage data
  - SQLite database for Cursor token history
  - JSONL reader for Claude Code data
  - Prometheus remote write client

### Use Case Layer
- **Services**: Business logic implementation
  - Claude Code data processing
  - Cursor API integration and token tracking
  - Metrics collection and sending
  - Application status tracking

### Interface Layer
- **Controllers**: Application entry points
  - CLI controller for command-line interface
  - Daemon controller for background service
  - System tray controller for UI

## Data Sources

### Claude Code
Searches for data in:
- `~/.config/claude/projects/` (new default)
- `~/.claude/projects/` (legacy)
- `~/Library/Application Support/claude/projects/` (macOS)

### Cursor
Uses Cursor API to fetch:
- Premium (GPT-4) request usage
- Usage-based pricing information
- Team membership status

## Notes

- macOS only (uses CGO for system tray)
- Time calculations use JST (Asia/Tokyo) timezone
- Configuration file: `~/.config/tosage/config.json`

## TODO

- [ ] Add Vertex AI token usage tracking
- [ ] Add Amazon Bedrock token usage tracking

## GitHub Actions Setup

For maintainers who want to build signed releases, see [GitHub Secrets Setup Guide](GITHUB_SECRETS_SETUP.md) for required configuration.

## License

MIT License