#!/bin/bash

# DMG Diagnostic Script
# Helps diagnose DMG compatibility issues

set -e

echo "DMG Diagnostic Tool"
echo "=================="
echo ""

# Check if DMG file is provided
if [ $# -eq 0 ]; then
    echo "Usage: $0 <dmg-file>"
    exit 1
fi

DMG_FILE="$1"

if [ ! -f "$DMG_FILE" ]; then
    echo "Error: DMG file not found: $DMG_FILE"
    exit 1
fi

echo "Analyzing: $DMG_FILE"
echo "Size: $(du -h "$DMG_FILE" | cut -f1)"
echo ""

# Get file type
echo "File type:"
file "$DMG_FILE"
echo ""

# Get DMG format info
echo "DMG format info:"
hdiutil imageinfo "$DMG_FILE" | grep -E "(Format:|Class:|Checksum Type:)" || true
echo ""

# Try to verify
echo "Verification:"
if hdiutil verify "$DMG_FILE" 2>&1; then
    echo "✓ Verification passed"
else
    echo "✗ Verification failed"
fi
echo ""

# Try to attach
echo "Mount test:"
if MOUNT_POINT=$(hdiutil attach -nobrowse -noverify "$DMG_FILE" 2>&1 | grep "Volumes" | cut -f3-); then
    echo "✓ Mount successful at: $MOUNT_POINT"
    
    # List contents
    echo ""
    echo "Contents:"
    ls -la "$MOUNT_POINT" || true
    
    # Detach
    hdiutil detach "$MOUNT_POINT" -quiet || true
else
    echo "✗ Mount failed"
fi

echo ""
echo "Diagnostic complete"