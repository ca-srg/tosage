#!/bin/bash

# tosage App Bundle Creation Script
# Creates a macOS app bundle for tosage

set -e

# Variables
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
BUILD_DIR="$SCRIPT_DIR"
APP_NAME="tosage"
BUNDLE_NAME="${APP_NAME}.app"
BUNDLE_PATH="$BUILD_DIR/$BUNDLE_NAME"
BINARY_NAME="tosage"
ICON_PNG="$PROJECT_ROOT/assets/icon_black.png"
ICNS_FILE="$BUILD_DIR/resources/app.icns"

# Get version from git tag or default
VERSION=$(cd "$PROJECT_ROOT" && git describe --tags --always --dirty 2>/dev/null || echo "1.0.0")
BUILD_TIME=$(date -u '+%Y-%m-%dT%H:%M:%SZ')

# Architecture (default to arm64)
ARCH="${ARCH:-arm64}"

echo "Creating macOS app bundle for $APP_NAME $VERSION ($ARCH)..."

# Clean existing bundle
if [ -d "$BUNDLE_PATH" ]; then
    echo "Removing existing bundle..."
    rm -rf "$BUNDLE_PATH"
fi

# Create bundle structure
echo "Creating bundle structure..."
mkdir -p "$BUNDLE_PATH/Contents/MacOS"
mkdir -p "$BUNDLE_PATH/Contents/Resources"

# Create Info.plist
echo "Creating Info.plist..."
cat > "$BUNDLE_PATH/Contents/Info.plist" <<EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>CFBundleExecutable</key>
    <string>$BINARY_NAME</string>
    <key>CFBundleIdentifier</key>
    <string>com.tosage.app</string>
    <key>CFBundleName</key>
    <string>$APP_NAME</string>
    <key>CFBundleDisplayName</key>
    <string>$APP_NAME</string>
    <key>CFBundleVersion</key>
    <string>$VERSION</string>
    <key>CFBundleShortVersionString</key>
    <string>$VERSION</string>
    <key>CFBundlePackageType</key>
    <string>APPL</string>
    <key>CFBundleSignature</key>
    <string>????</string>
    <key>CFBundleIconFile</key>
    <string>app</string>
    <key>LSMinimumSystemVersion</key>
    <string>11.0</string>
    <key>LSUIElement</key>
    <false/>
    <key>NSHighResolutionCapable</key>
    <true/>
    <key>NSSupportsAutomaticGraphicsSwitching</key>
    <true/>
    <key>CFBundleInfoDictionaryVersion</key>
    <string>6.0</string>
    <key>NSHumanReadableCopyright</key>
    <string>Copyright © $(date +%Y) tosage. All rights reserved.</string>
    <key>LSEnvironment</key>
    <dict>
        <key>TOSAGE_DAEMON_ENABLED</key>
        <string>true</string>
    </dict>
</dict>
</plist>
EOF

# Create icon file (.icns) from PNG
if [ -f "$ICON_PNG" ]; then
    echo "Creating .icns file from PNG..."
    
    # Check if sips is available
    if ! command -v sips >/dev/null 2>&1; then
        echo "Warning: sips command not found. Skipping icon creation."
        # Use default icon instead
        if [ -f "$BUILD_DIR/resources/app.icns" ]; then
            echo "Using pre-built icon file"
            cp "$BUILD_DIR/resources/app.icns" "$ICNS_FILE"
        else
            echo "No icon will be set for the app bundle"
        fi
        return 0
    fi
    
    # Check if iconutil is available
    if ! command -v iconutil >/dev/null 2>&1; then
        echo "Warning: iconutil command not found. Skipping icon creation."
        # Use default icon instead
        if [ -f "$BUILD_DIR/resources/app.icns" ]; then
            echo "Using pre-built icon file"
            cp "$BUILD_DIR/resources/app.icns" "$ICNS_FILE"
        else
            echo "No icon will be set for the app bundle"
        fi
        return 0
    fi
    
    # Create temporary iconset directory
    ICONSET_DIR="$BUILD_DIR/tosage.iconset"
    mkdir -p "$ICONSET_DIR"
    
    # Generate icon sizes
    echo "Generating icon sizes..."
    for size in 16 32 128 256 512; do
        size2x=$((size * 2))
        if ! sips -z $size $size "$ICON_PNG" --out "$ICONSET_DIR/icon_${size}x${size}.png" >/dev/null 2>&1; then
            echo "Warning: Failed to generate ${size}x${size} icon"
        fi
        if [ $size -le 256 ]; then
            if ! sips -z $size2x $size2x "$ICON_PNG" --out "$ICONSET_DIR/icon_${size}x${size}@2x.png" >/dev/null 2>&1; then
                echo "Warning: Failed to generate ${size}x${size}@2x icon"
            fi
        fi
    done
    
    # Create icns file
    if iconutil -c icns "$ICONSET_DIR" -o "$ICNS_FILE" 2>/dev/null; then
        echo "Icon file created successfully"
    else
        echo "Warning: Failed to create .icns file"
        rm -rf "$ICONSET_DIR"
        # Continue without icon
    fi
    
    # Copy to app bundle if icon was created
    if [ -f "$ICNS_FILE" ]; then
        cp "$ICNS_FILE" "$BUNDLE_PATH/Contents/Resources/app.icns"
    fi
    
    # Clean up
    rm -rf "$ICONSET_DIR"
else
    echo "Warning: Icon file not found at $ICON_PNG"
    echo "App bundle will be created without custom icon"
    # Check if pre-built icon exists
    if [ -f "$BUILD_DIR/resources/app.icns" ]; then
        echo "Using pre-built icon file"
        cp "$BUILD_DIR/resources/app.icns" "$BUNDLE_PATH/Contents/Resources/app.icns"
    fi
fi

# Build binary if it doesn't exist
BINARY_PATH="$PROJECT_ROOT/$BINARY_NAME-darwin-$ARCH"
if [ ! -f "$BINARY_PATH" ]; then
    echo "Binary not found at $BINARY_PATH, building..."
    cd "$PROJECT_ROOT"
    
    # Check if current architecture matches target
    CURRENT_ARCH=$(uname -m)
    if [ "$CURRENT_ARCH" = "arm64" ] && [ "$ARCH" = "arm64" ]; then
        CGO_ENABLED=1 GOOS=darwin GOARCH=$ARCH go build -tags darwin -ldflags "-w -s -X main.Version=$VERSION -X main.BuildTime=$BUILD_TIME" -o "$BINARY_PATH" .
    elif [ "$CURRENT_ARCH" = "x86_64" ] && [ "$ARCH" = "amd64" ]; then
        CGO_ENABLED=1 GOOS=darwin GOARCH=$ARCH go build -tags darwin -ldflags "-w -s -X main.Version=$VERSION -X main.BuildTime=$BUILD_TIME" -o "$BINARY_PATH" .
    else
        echo "Error: Cannot cross-compile with CGO enabled"
        echo "Current architecture: $CURRENT_ARCH"
        echo "Target architecture: $ARCH"
        echo "Please build on the target architecture or use a pre-built binary"
        exit 1
    fi
fi

# Copy binary to bundle
echo "Copying binary to bundle..."
cp "$BINARY_PATH" "$BUNDLE_PATH/Contents/MacOS/$BINARY_NAME"

# Set executable permissions
chmod +x "$BUNDLE_PATH/Contents/MacOS/$BINARY_NAME"

# Create PkgInfo file
echo "APPL????" > "$BUNDLE_PATH/Contents/PkgInfo"

# Check if running in CI environment
IS_CI=false
if [ -n "$CI" ] || [ -n "$GITHUB_ACTIONS" ]; then
    IS_CI=true
fi

# Code signing
if [ -n "$CODESIGN_IDENTITY" ] && [ "$CODESIGN_IDENTITY" != "-" ]; then
    echo ""
    echo "Signing app bundle with identity: $CODESIGN_IDENTITY"
    
    # Sign the app bundle
    ENTITLEMENTS_PATH="$PROJECT_ROOT/scripts/entitlements.plist"
    
    # Check if entitlements file exists
    if [ ! -f "$ENTITLEMENTS_PATH" ]; then
        echo "ERROR: Entitlements file not found at $ENTITLEMENTS_PATH"
        echo "Please ensure scripts/entitlements.plist exists in the project root"
        exit 1
    fi
    
    if codesign --force --sign "$CODESIGN_IDENTITY" \
        --options runtime \
        --entitlements "$ENTITLEMENTS_PATH" \
        "$BUNDLE_PATH"
    then
        echo "App bundle signed successfully"
        
        # Verify signature
        echo "Verifying signature..."
        codesign --verify --verbose "$BUNDLE_PATH"
        
        # Check with spctl (Gatekeeper)
        echo "Checking with Gatekeeper..."
        spctl -a -vvv "$BUNDLE_PATH" || echo "Note: spctl check may fail until notarized"
    else
        echo "ERROR: Failed to sign app bundle"
        if [ "$IS_CI" = true ]; then
            echo "Code signing is required in CI environment"
            exit 1
        fi
    fi
else
    if [ "$IS_CI" = true ]; then
        echo ""
        echo "❌ ERROR: Code signing is required in CI environment"
        echo ""
        echo "CODESIGN_IDENTITY is not set or is invalid"
        echo ""
        echo "Please ensure the following GitHub secrets are configured:"
        echo "- MACOS_CERTIFICATE: Base64 encoded .p12 certificate"
        echo "- MACOS_CERTIFICATE_PWD: Certificate password"
        echo "- CODESIGN_IDENTITY: Developer ID Application identity"
        echo ""
        echo "Example CODESIGN_IDENTITY format:"
        echo "  Developer ID Application: Your Name (TEAMID)"
        exit 1
    else
        echo ""
        echo "Creating ad-hoc signature for app bundle..."
        # Sign with ad-hoc signature (no identity)
        if codesign --force --sign - "$BUNDLE_PATH"; then
            echo "App bundle signed with ad-hoc signature"
            
            # Verify signature
            echo "Verifying signature..."
            codesign --verify --verbose "$BUNDLE_PATH"
        else
            echo "Warning: Failed to create ad-hoc signature"
        fi
        
        echo ""
        echo "Note: The app is not signed with a Developer ID."
        echo "Users will need to right-click and select 'Open' to run it."
    fi
fi

echo ""
echo "App bundle created successfully at: $BUNDLE_PATH"

# Verify bundle structure
echo ""
echo "Bundle structure:"
find "$BUNDLE_PATH" -type f | sort

echo ""
echo "Bundle info:"
echo "  Name: $APP_NAME"
echo "  Version: $VERSION"
echo "  Architecture: $ARCH"
echo "  Bundle ID: com.tosage.app"
echo "  Path: $BUNDLE_PATH"