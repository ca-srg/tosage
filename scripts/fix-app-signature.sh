#!/bin/bash

# Fix macOS app signature and Gatekeeper issues
# This script removes invalid signatures and allows the app to run

set -e

if [ $# -eq 0 ]; then
    echo "Usage: $0 <path-to-app>"
    echo ""
    echo "This script fixes apps that show 'is damaged and can't be opened' error"
    echo "by removing invalid signatures and clearing quarantine attributes."
    exit 1
fi

APP_PATH="$1"

if [ ! -d "$APP_PATH" ]; then
    echo "Error: App not found: $APP_PATH"
    exit 1
fi

echo "Fixing app: $APP_PATH"
echo ""

# Check current signature
echo "Current signature status:"
codesign -dvv "$APP_PATH" 2>&1 || true
echo ""

# Remove the signature completely
echo "Removing invalid signature..."
codesign --remove-signature "$APP_PATH"

# Remove extended attributes including quarantine
echo "Removing quarantine attributes..."
xattr -cr "$APP_PATH"

# Re-sign with ad-hoc signature (no identity required)
echo "Re-signing with ad-hoc signature..."
codesign --force --deep --sign - "$APP_PATH"

# Verify the new signature
echo ""
echo "New signature status:"
codesign -dvv "$APP_PATH" 2>&1 || true

# Check with Gatekeeper
echo ""
echo "Gatekeeper check:"
spctl -a -vvv "$APP_PATH" 2>&1 || echo "Note: Gatekeeper may still block unsigned apps"

echo ""
echo "Fix complete!"
echo ""
echo "To run the app:"
echo "1. Right-click on the app and select 'Open'"
echo "2. Click 'Open' in the security dialog"
echo ""
echo "Or disable Gatekeeper temporarily:"
echo "  sudo spctl --master-disable"
echo "  (Remember to re-enable it: sudo spctl --master-enable)"