# tosage - Claude Code Token Usage Tool

A simple Go command-line tool that outputs today's Claude Code token usage count in JST (Japan Standard Time).

## Features

- Outputs token usage from JST 00:00 to current time as a single number
- Automatically finds Claude Code data directories
- Returns `0` if no usage today

## Installation

### Prerequisites

- Go 1.21 or higher

### Build from Source

```bash
# Clone the repository
git clone https://github.com/ca-srg/tosage.git
cd tosage

# Build the binary
go build -o tosage

# Or use make
make build
```

## macOS App Bundle and DMG Creation

This project includes tools to create macOS app bundles and DMG installers.

### Makefile Targets

#### App Bundle Targets

##### `app-bundle-arm64` / `app-bundle-amd64`
**Purpose**: Creates a macOS app bundle (.app)

1. **Binary Build**: Executes `build-darwin` to create Go binary
2. **Dependency Check**: Runs `dmg-check` to verify required tools
3. **App Bundle Creation**: Executes `create-app-bundle.sh` to create:
   - `tosage.app/Contents/MacOS/tosage` - Executable file
   - `tosage.app/Contents/Info.plist` - App metadata
   - `tosage.app/Contents/Resources/app.icns` - App icon
   - `tosage.app/Contents/PkgInfo` - App type information

#### DMG Targets

##### `dmg-arm64` / `dmg-amd64`
**Purpose**: Creates unsigned DMG installers

1. Creates app bundle (executes `app-bundle-*`)
2. Runs `create-dmg.sh` to create DMG:
   - Includes app bundle in DMG
   - Adds symlink to `/Applications`
   - Sets background image and window layout
   - Output: `tosage-{version}-darwin-{arch}.dmg`

##### `dmg-signed-arm64` / `dmg-signed-amd64`
**Purpose**: Creates signed DMGs

- Requires `CODESIGN_IDENTITY` environment variable
- Adds code signature to app bundle and DMG

##### `dmg-notarized-arm64` / `dmg-notarized-amd64`
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

## Usage

Simply run the command to get today's token count:

```bash
./tosage
```

Output example:
```
123456
```

If there's no usage today, it outputs:
```
0
```

## Data Sources

The tool automatically searches for Claude Code data in these locations:

- `~/.config/claude/projects/` (new default location)
- `~/.claude/projects/` (legacy location)
- macOS: `~/Library/Application Support/claude/projects/`
- Windows: `%LOCALAPPDATA%\claude\projects\`

## GitHub Actions Setup

For maintainers who want to build signed releases, see [GitHub Secrets Setup Guide](GITHUB_SECRETS_SETUP.md) for required configuration.

## License

MIT License