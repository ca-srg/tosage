#!/bin/bash

# Fix DMG extended attributes
# This script restores the necessary extended attributes for DMG files
# that were downloaded from GitHub Actions or other sources that strip xattrs

set -e

if [ $# -eq 0 ]; then
    echo "Usage: $0 <dmg-file>"
    echo ""
    echo "This script fixes DMG files that show as 'data' instead of disk images"
    echo "by restoring the necessary extended attributes."
    exit 1
fi

DMG_FILE="$1"

if [ ! -f "$DMG_FILE" ]; then
    echo "Error: File not found: $DMG_FILE"
    exit 1
fi

echo "Fixing extended attributes for: $DMG_FILE"

# Check current file type
echo ""
echo "Current file type:"
file "$DMG_FILE"

# Check current attributes
echo ""
echo "Current extended attributes:"
xattr -l "$DMG_FILE" 2>/dev/null || echo "No extended attributes found"

# Add the FinderInfo attribute to identify as disk image
echo ""
echo "Adding disk image attributes..."

# Create the FinderInfo data (identifies as disk image)
# The FinderInfo for disk images needs to be exactly 32 bytes
# Type: 'devi' (device), Creator: 'ddsk' (disk), followed by 24 zero bytes
xattr -wx com.apple.FinderInfo "64 65 76 69 64 64 73 6B 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00" "$DMG_FILE"

# Verify the change
echo ""
echo "Fixed file type:"
file "$DMG_FILE"

echo ""
echo "New extended attributes:"
xattr -l "$DMG_FILE"

# Test mounting
echo ""
echo "Testing mount..."
if hdiutil attach -nobrowse -noverify "$DMG_FILE" >/dev/null 2>&1; then
    echo "✓ DMG can be mounted successfully"
    # Find and unmount
    MOUNT_POINT=$(mount | grep "$DMG_FILE" | awk '{print $3}')
    if [ -n "$MOUNT_POINT" ]; then
        hdiutil detach "$MOUNT_POINT" -quiet
    fi
else
    echo "✗ Failed to mount DMG"
    echo "The file may have other issues beyond extended attributes"
fi

echo ""
echo "Done! The DMG file should now be recognized by macOS."
echo "Try double-clicking it in Finder."