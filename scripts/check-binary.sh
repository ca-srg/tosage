#!/bin/bash

# Check binary dependencies and architecture

set -e

if [ $# -eq 0 ]; then
    echo "Usage: $0 <path-to-binary-or-app>"
    echo ""
    echo "This script checks binary dependencies and architecture."
    exit 1
fi

TARGET="$1"

# If it's an app bundle, extract the binary path
if [ -d "$TARGET" ] && [[ "$TARGET" == *.app ]]; then
    APP_NAME=$(basename "$TARGET" .app)
    BINARY_PATH="$TARGET/Contents/MacOS/$APP_NAME"
    echo "Checking app bundle: $TARGET"
    echo "Binary path: $BINARY_PATH"
else
    BINARY_PATH="$TARGET"
    echo "Checking binary: $BINARY_PATH"
fi

if [ ! -f "$BINARY_PATH" ]; then
    echo "Error: Binary not found at $BINARY_PATH"
    exit 1
fi

echo ""
echo "=== File Information ==="
file "$BINARY_PATH"

echo ""
echo "=== Architecture ==="
lipo -info "$BINARY_PATH" 2>/dev/null || echo "lipo not available or single architecture"

echo ""
echo "=== Dependencies ==="
otool -L "$BINARY_PATH" | head -20

echo ""
echo "=== Load Commands ==="
otool -l "$BINARY_PATH" | grep -A 5 "LC_VERSION_MIN_MACOSX\|LC_BUILD_VERSION" | head -20

echo ""
echo "=== Linked Frameworks ==="
otool -L "$BINARY_PATH" | grep -E "(framework|dylib)" || echo "No frameworks or dylibs found"

echo ""
echo "=== Code Signature ==="
codesign -dvv "$BINARY_PATH" 2>&1 | grep -E "(Signature|Identifier|Format)" || echo "No signature found"

# Check for missing dependencies
echo ""
echo "=== Checking for missing dependencies ==="
MISSING=false
while IFS= read -r line; do
    if [[ "$line" =~ ^[[:space:]]+(/.+)[[:space:]]\( ]]; then
        dep="${BASH_REMATCH[1]}"
        if [ ! -f "$dep" ] && [ ! -d "$dep" ]; then
            echo "Missing: $dep"
            MISSING=true
        fi
    fi
done < <(otool -L "$BINARY_PATH")

if [ "$MISSING" = false ]; then
    echo "All dependencies found"
fi

echo ""
echo "Done!"