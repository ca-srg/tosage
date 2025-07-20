#!/bin/bash

# Test DMG creation locally to debug GitHub Actions issues
# This script simulates the DMG creation process with various settings

set -e

echo "DMG Creation Test Script"
echo "======================="
echo ""

# Check system info
echo "System Information:"
sw_vers
echo ""
echo "Disk space:"
df -h
echo ""

# Test directory
TEST_DIR="./dmg-test"
rm -rf "$TEST_DIR"
mkdir -p "$TEST_DIR"

# Create test app bundle
TEST_APP="$TEST_DIR/TestApp.app"
mkdir -p "$TEST_APP/Contents/MacOS"
echo '#!/bin/bash' > "$TEST_APP/Contents/MacOS/test"
echo 'echo "Test app"' >> "$TEST_APP/Contents/MacOS/test"
chmod +x "$TEST_APP/Contents/MacOS/test"

# Test 1: HFS+ with high compression
echo ""
echo "Test 1: HFS+ with zlib-level=9"
echo "-------------------------------"
TEMP_DMG="$TEST_DIR/test1-temp.dmg"
FINAL_DMG="$TEST_DIR/test1-final.dmg"

if hdiutil create -srcfolder "$TEST_APP" -volname "Test1" -fs HFS+ \
    -format UDRW -size 50m "$TEMP_DMG"; then
    echo "✓ Temp DMG created"
    
    if hdiutil convert "$TEMP_DMG" -format UDZO -imagekey zlib-level=9 -o "$FINAL_DMG"; then
        echo "✓ Final DMG created with high compression"
        hdiutil verify "$FINAL_DMG" && echo "✓ DMG verification passed" || echo "✗ DMG verification failed"
    else
        echo "✗ Failed to convert with high compression"
    fi
else
    echo "✗ Failed to create temp DMG"
fi

# Test 2: HFS+ with medium compression
echo ""
echo "Test 2: HFS+ with zlib-level=6"
echo "-------------------------------"
TEMP_DMG="$TEST_DIR/test2-temp.dmg"
FINAL_DMG="$TEST_DIR/test2-final.dmg"

if hdiutil create -srcfolder "$TEST_APP" -volname "Test2" -fs HFS+ \
    -format UDRW -size 50m "$TEMP_DMG"; then
    echo "✓ Temp DMG created"
    
    if hdiutil convert "$TEMP_DMG" -format UDZO -imagekey zlib-level=6 -o "$FINAL_DMG"; then
        echo "✓ Final DMG created with medium compression"
        hdiutil verify "$FINAL_DMG" && echo "✓ DMG verification passed" || echo "✗ DMG verification failed"
    else
        echo "✗ Failed to convert with medium compression"
    fi
else
    echo "✗ Failed to create temp DMG"
fi

# Test 3: APFS with medium compression
echo ""
echo "Test 3: APFS with zlib-level=6"
echo "-------------------------------"
TEMP_DMG="$TEST_DIR/test3-temp.dmg"
FINAL_DMG="$TEST_DIR/test3-final.dmg"

if hdiutil create -srcfolder "$TEST_APP" -volname "Test3" -fs APFS \
    -format UDRW -size 50m "$TEMP_DMG"; then
    echo "✓ Temp DMG created with APFS"
    
    if hdiutil convert "$TEMP_DMG" -format UDZO -imagekey zlib-level=6 -o "$FINAL_DMG"; then
        echo "✓ Final DMG created with medium compression"
        hdiutil verify "$FINAL_DMG" && echo "✓ DMG verification passed" || echo "✗ DMG verification failed"
    else
        echo "✗ Failed to convert with medium compression"
    fi
else
    echo "✗ Failed to create temp DMG with APFS"
fi

# Test 4: UDBZ format (bzip2 compression)
echo ""
echo "Test 4: HFS+ with UDBZ format"
echo "------------------------------"
TEMP_DMG="$TEST_DIR/test4-temp.dmg"
FINAL_DMG="$TEST_DIR/test4-final.dmg"

if hdiutil create -srcfolder "$TEST_APP" -volname "Test4" -fs HFS+ \
    -format UDRW -size 50m "$TEMP_DMG"; then
    echo "✓ Temp DMG created"
    
    if hdiutil convert "$TEMP_DMG" -format UDBZ -o "$FINAL_DMG"; then
        echo "✓ Final DMG created with UDBZ format"
        hdiutil verify "$FINAL_DMG" && echo "✓ DMG verification passed" || echo "✗ DMG verification failed"
    else
        echo "✗ Failed to convert with UDBZ format"
    fi
else
    echo "✗ Failed to create temp DMG"
fi

# Summary
echo ""
echo "Summary of created DMGs:"
echo "------------------------"
ls -lh "$TEST_DIR"/*.dmg 2>/dev/null || echo "No DMG files created"

# Clean up
echo ""
read -p "Clean up test files? (y/n) " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    rm -rf "$TEST_DIR"
    echo "Test files cleaned up"
else
    echo "Test files kept in: $TEST_DIR"
fi