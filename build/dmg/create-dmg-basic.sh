#!/bin/bash

# Ultra-basic DMG Creation Script for maximum compatibility
# Uses the simplest possible method to create DMGs

set -e

# Variables
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
BUILD_DIR="$SCRIPT_DIR"
APP_NAME="tosage"
BUNDLE_NAME="${APP_NAME}.app"
BUNDLE_PATH="$BUILD_DIR/$BUNDLE_NAME"

# Architecture (default to arm64)
ARCH="${ARCH:-arm64}"

# Get version from git tag or default
VERSION=$(cd "$PROJECT_ROOT" && git describe --tags --always --dirty 2>/dev/null || echo "1.0.0")

# Output DMG path
OUTPUT_DMG="$PROJECT_ROOT/${APP_NAME}-${VERSION}-darwin-${ARCH}.dmg"
TEMP_DIR="/tmp/tosage-dmg-$$"

echo "Creating basic DMG for $APP_NAME $VERSION ($ARCH)..."
echo "Using ultra-compatible settings for CI environment"

# Check if app bundle exists
if [ ! -d "$BUNDLE_PATH" ]; then
    echo "Error: App bundle not found at $BUNDLE_PATH"
    exit 1
fi

# Clean up any existing files
echo "Cleaning up..."
rm -f "$OUTPUT_DMG"
rm -rf "$TEMP_DIR"

# Create temp directory
echo "Creating temp directory..."
mkdir -p "$TEMP_DIR"

# Copy app to temp directory
echo "Copying app bundle..."
cp -R "$BUNDLE_PATH" "$TEMP_DIR/"

# Create Applications symlink
ln -s /Applications "$TEMP_DIR/Applications"

# Create DMG using the absolute simplest method
echo "Creating DMG (basic method)..."
hdiutil create \
    -volname "$APP_NAME" \
    -srcfolder "$TEMP_DIR" \
    -fs HFS+ \
    -format UDRO \
    "$OUTPUT_DMG"

# Clean up
rm -rf "$TEMP_DIR"

# Show info
echo ""
echo "DMG created: $OUTPUT_DMG"
echo "Size: $(du -h "$OUTPUT_DMG" | cut -f1)"

# Basic verification
echo ""
echo "Verifying DMG..."
hdiutil verify "$OUTPUT_DMG" || echo "Warning: Verification failed"

echo ""
echo "Done!"