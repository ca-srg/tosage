#!/bin/bash

# tosage Installation Script
# This script helps install tosage and bypass Gatekeeper warnings

set -e

echo "tosage Installation Script"
echo "========================="
echo ""

# Check if running on macOS
if [[ "$OSTYPE" != "darwin"* ]]; then
    echo "Error: This script is for macOS only"
    exit 1
fi

# Find DMG file
DMG_FILE=""
if [ -f "$1" ]; then
    DMG_FILE="$1"
else
    # Look for DMG in current directory
    DMG_FILE=$(find . -name "tosage*.dmg" -maxdepth 1 | head -1)
    if [ -z "$DMG_FILE" ]; then
        echo "Error: No DMG file found"
        echo ""
        echo "Usage: $0 [path-to-dmg]"
        echo ""
        echo "Please provide the path to the tosage DMG file"
        exit 1
    fi
fi

echo "Installing from: $DMG_FILE"
echo ""

# Mount DMG
echo "Mounting DMG..."
MOUNT_POINT=$(hdiutil attach "$DMG_FILE" -nobrowse | grep "Volumes" | cut -f3-)
if [ -z "$MOUNT_POINT" ]; then
    echo "Error: Failed to mount DMG"
    exit 1
fi

echo "Mounted at: $MOUNT_POINT"

# Check if app exists in DMG
APP_PATH="$MOUNT_POINT/tosage.app"
if [ ! -d "$APP_PATH" ]; then
    echo "Error: tosage.app not found in DMG"
    hdiutil detach "$MOUNT_POINT" -quiet
    exit 1
fi

# Copy to Applications
echo ""
echo "Copying tosage.app to Applications..."
if [ -d "/Applications/tosage.app" ]; then
    echo "Existing installation found. Backing up..."
    sudo mv "/Applications/tosage.app" "/Applications/tosage.app.backup.$(date +%Y%m%d%H%M%S)"
fi

sudo cp -R "$APP_PATH" "/Applications/"

# Remove quarantine attributes
echo "Removing security restrictions..."
sudo xattr -cr "/Applications/tosage.app"

# Re-sign with ad-hoc signature
echo "Applying ad-hoc signature..."
sudo codesign --force --sign - "/Applications/tosage.app"

# Unmount DMG
echo "Unmounting DMG..."
hdiutil detach "$MOUNT_POINT" -quiet

# Verify installation
if [ -d "/Applications/tosage.app" ]; then
    echo ""
    echo "✅ Installation complete!"
    echo ""
    echo "tosage has been installed to /Applications/tosage.app"
    echo ""
    echo "You can now:"
    echo "1. Open tosage from Finder > Applications"
    echo "2. Or run from Terminal: /Applications/tosage.app/Contents/MacOS/tosage"
    echo ""
    echo "First run will configure the application and create ~/.config/tosage/config.json"
else
    echo ""
    echo "❌ Installation failed"
    exit 1
fi