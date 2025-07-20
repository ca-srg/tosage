#!/bin/bash

# tosage DMG Creation Script
# Creates a macOS DMG installer for tosage

set -e

# Variables
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
BUILD_DIR="$SCRIPT_DIR"
APP_NAME="tosage"
BUNDLE_NAME="${APP_NAME}.app"
BUNDLE_PATH="$BUILD_DIR/$BUNDLE_NAME"
DMG_NAME="${APP_NAME}.dmg"
VOLUME_NAME="$APP_NAME"
BACKGROUND_IMG="$BUILD_DIR/resources/background.png"
VOLUME_ICON="$BUILD_DIR/resources/app.icns"

# Architecture (default to arm64)
ARCH="${ARCH:-arm64}"

# Get version from git tag or default
VERSION=$(cd "$PROJECT_ROOT" && git describe --tags --always --dirty 2>/dev/null || echo "1.0.0")

# Output DMG path
OUTPUT_DMG="$PROJECT_ROOT/${APP_NAME}-${VERSION}-darwin-${ARCH}.dmg"
TEMP_DMG="$BUILD_DIR/temp.dmg"
STAGING_DIR="$BUILD_DIR/dmg-staging"

echo "Creating DMG for $APP_NAME $VERSION ($ARCH)..."

# Check required tools
if ! command -v hdiutil >/dev/null 2>&1; then
    echo "Error: hdiutil command not found. This is required for DMG creation."
    exit 1
fi

if ! command -v osascript >/dev/null 2>&1; then
    echo "Error: osascript command not found. This is required for DMG window setup."
    exit 1
fi

# Check if app bundle exists
if [ ! -d "$BUNDLE_PATH" ]; then
    echo "App bundle not found at $BUNDLE_PATH"
    echo "Running app bundle creation script..."
    if ! "$BUILD_DIR/create-app-bundle.sh"; then
        echo "Error: Failed to create app bundle"
        exit 1
    fi
fi

# Clean up any existing files
echo "Cleaning up existing files..."
rm -f "$TEMP_DMG" "$OUTPUT_DMG"
rm -rf "$STAGING_DIR"

# Create staging directory
echo "Creating staging directory..."
mkdir -p "$STAGING_DIR"

# Copy app bundle to staging
echo "Copying app bundle to staging..."
cp -R "$BUNDLE_PATH" "$STAGING_DIR/"

# Create Applications symlink
echo "Creating Applications symlink..."
ln -s /Applications "$STAGING_DIR/Applications"

# Copy background image to staging if it exists
if [ -f "$BACKGROUND_IMG" ]; then
    echo "Copying background image to staging..."
    mkdir -p "$STAGING_DIR/.background"
    cp "$BACKGROUND_IMG" "$STAGING_DIR/.background/background.png"
fi

# Calculate required DMG size
echo "Calculating required DMG size..."
STAGING_SIZE_MB=$(du -sm "$STAGING_DIR" | cut -f1)
DMG_SIZE_MB=$((STAGING_SIZE_MB * 2 + 50))  # Double the size plus 50MB buffer
echo "Staging size: ${STAGING_SIZE_MB}MB, DMG size: ${DMG_SIZE_MB}MB"

# Create temporary DMG
echo "Creating temporary DMG..."
if ! hdiutil create -srcfolder "$STAGING_DIR" -volname "$VOLUME_NAME" -fs APFS \
    -format UDRW -size ${DMG_SIZE_MB}m "$TEMP_DMG"; then
    echo "Error: Failed to create temporary DMG"
    echo "Attempting with HFS+ as fallback..."
    # Fallback to HFS+ for older macOS versions
    if ! hdiutil create -srcfolder "$STAGING_DIR" -volname "$VOLUME_NAME" -fs HFS+ \
        -format UDRW -size ${DMG_SIZE_MB}m "$TEMP_DMG"; then
        echo "Error: Failed to create temporary DMG with both APFS and HFS+"
        rm -rf "$STAGING_DIR"
        exit 1
    fi
fi

# Mount the temporary DMG
echo "Mounting temporary DMG..."
DEVICE=$(hdiutil attach -readwrite -noverify -noautoopen "$TEMP_DMG" | \
    egrep '^/dev/' | sed 1q | awk '{print $1}')
MOUNT_POINT="/Volumes/$VOLUME_NAME"

# Wait for mount
sleep 2

# Set up the DMG window properties
echo "Setting up DMG window properties..."

# Background image is already in staging, no need to copy again

# Create .DS_Store file with window settings using AppleScript
echo "Configuring DMG window..."
osascript <<EOF
tell application "Finder"
    tell disk "$VOLUME_NAME"
        open
        set current view of container window to icon view
        set toolbar visible of container window to false
        set statusbar visible of container window to false
        set the bounds of container window to {200, 100, 840, 500}
        set viewOptions to the icon view options of container window
        set arrangement of viewOptions to not arranged
        set icon size of viewOptions to 128
        
        -- Set background if available
        try
            set background picture of viewOptions to file ".background:background.png"
        end try
        
        -- Position items
        set position of item "$BUNDLE_NAME" of container window to {160, 200}
        set position of item "Applications" of container window to {480, 200}
        
        close
        open
        update without registering applications
        delay 2
    end tell
end tell
EOF

# Set volume icon if available
if [ -f "$VOLUME_ICON" ]; then
    echo "Setting volume icon..."
    cp "$VOLUME_ICON" "$MOUNT_POINT/.VolumeIcon.icns"
    if command -v SetFile >/dev/null 2>&1; then
        SetFile -a C "$MOUNT_POINT"
    else
        echo "Warning: SetFile not available, skipping custom icon attribute"
    fi
fi

# Hide background folder
if [ -d "$MOUNT_POINT/.background" ]; then
    if command -v SetFile >/dev/null 2>&1; then
        SetFile -a V "$MOUNT_POINT/.background"
    else
        echo "Warning: SetFile not available, skipping background folder hiding"
    fi
fi

# Sync
sync

# Wait for Finder to finish
echo "Waiting for Finder to finish..."
sleep 5

# Unmount the DMG with retry logic
echo "Unmounting temporary DMG..."
MAX_RETRIES=5
RETRY_COUNT=0

while [ $RETRY_COUNT -lt $MAX_RETRIES ]; do
    if hdiutil detach "$DEVICE" -force 2>/dev/null; then
        echo "Successfully unmounted DMG"
        break
    else
        RETRY_COUNT=$((RETRY_COUNT + 1))
        if [ $RETRY_COUNT -lt $MAX_RETRIES ]; then
            echo "Failed to unmount, retrying in 3 seconds... (attempt $RETRY_COUNT/$MAX_RETRIES)"
            sleep 3
        else
            echo "Error: Failed to unmount DMG after $MAX_RETRIES attempts"
            exit 1
        fi
    fi
done

# Ensure all filesystem operations are complete
echo "Syncing filesystem..."
sync
sleep 2

# Convert to compressed DMG
echo "Creating final DMG..."
echo "Converting temp DMG to compressed format..."
if ! hdiutil convert "$TEMP_DMG" -format UDZO -imagekey zlib-level=6 -o "$OUTPUT_DMG"; then
    echo "Error: Failed to create final DMG with UDZO format"
    echo "Attempting with UDBZ format as fallback..."
    # Try UDBZ (bzip2) compression as alternative
    if ! hdiutil convert "$TEMP_DMG" -format UDBZ -o "$OUTPUT_DMG"; then
        echo "Error: Failed to create final DMG with both UDZO and UDBZ formats"
        rm -f "$TEMP_DMG"
        rm -rf "$STAGING_DIR"
        exit 1
    fi
fi

# Verify the output DMG was created
if [ ! -f "$OUTPUT_DMG" ]; then
    echo "Error: Output DMG not found at $OUTPUT_DMG"
    exit 1
fi

# Code signing for DMG
if [ -n "$CODESIGN_IDENTITY" ]; then
    echo ""
    echo "Signing DMG with identity: $CODESIGN_IDENTITY"
    
    # Sign the DMG
    if codesign --force --sign "$CODESIGN_IDENTITY" "$OUTPUT_DMG"; then
        echo "DMG signed successfully"
        
        # Verify signature
        echo "Verifying DMG signature..."
        codesign --verify --verbose "$OUTPUT_DMG"
    else
        echo "Warning: Failed to sign DMG"
    fi
else
    echo ""
    echo "Skipping DMG signing (CODESIGN_IDENTITY not set)"
fi

# Notarization process (API Key method)
if [ "$NOTARIZE" = "true" ]; then
    # Check CI environment and notarization requirements
    IS_CI=false
    if [ -n "$CI" ] || [ -n "$GITHUB_ACTIONS" ]; then
        IS_CI=true
    fi
    
    # Verify required notarization credentials
    if [ -z "$API_KEY_ID" ] || [ -z "$API_KEY_PATH" ] || [ -z "$API_ISSUER" ]; then
        echo ""
        echo "ERROR: Notarization is enabled but required credentials are missing"
        echo ""
        echo "Missing credentials:"
        [ -z "$API_KEY_ID" ] && echo "  - API_KEY_ID is not set"
        [ -z "$API_KEY_PATH" ] && echo "  - API_KEY_PATH is not set"
        [ -z "$API_ISSUER" ] && echo "  - API_ISSUER is not set"
        echo ""
        echo "To enable notarization, please set:"
        echo "  export API_KEY_ID='your-api-key-id'"
        echo "  export API_KEY_PATH='path/to/AuthKey_XXXXXX.p8'"
        echo "  export API_ISSUER='your-issuer-id'"
        echo ""
        if [ "$IS_CI" = true ]; then
            echo "In CI environment, notarization is required when NOTARIZE=true"
            exit 1
        else
            echo "Skipping notarization due to missing credentials"
        fi
    elif [ ! -f "$API_KEY_PATH" ]; then
        echo ""
        echo "ERROR: API key file not found at: $API_KEY_PATH"
        echo "Please ensure the .p8 file exists at the specified path"
        exit 1
    else
    echo ""
    echo "Starting notarization process..."
    echo "API_KEY_ID: $API_KEY_ID"
    echo "API_KEY_PATH: $API_KEY_PATH"
    echo "API_ISSUER: $API_ISSUER"
    
    # Verify API key file exists and is readable
    if [ ! -r "$API_KEY_PATH" ]; then
        echo "ERROR: Cannot read API key file at: $API_KEY_PATH"
        ls -la "$(dirname "$API_KEY_PATH")" 2>/dev/null || echo "Directory does not exist"
        exit 1
    fi
    
    echo "API key file verified at: $API_KEY_PATH"
    echo "File size: $(wc -c < "$API_KEY_PATH") bytes"
    
    # Additional API key validation
    echo ""
    echo "Validating API key file format..."
    if grep -q "BEGIN PRIVATE KEY" "$API_KEY_PATH" && grep -q "END PRIVATE KEY" "$API_KEY_PATH"; then
        echo "✓ API key file has correct PEM format"
    else
        echo "❌ ERROR: API key file does not appear to be in correct PEM format"
        echo "Expected format: -----BEGIN PRIVATE KEY----- ... -----END PRIVATE KEY-----"
        exit 1
    fi
    
    # Check if notarytool is available
    if ! command -v xcrun >/dev/null 2>&1; then
        echo "Error: xcrun not found. Xcode command line tools required for notarization."
        exit 1
    fi
    
    # Verify notarytool is available
    echo "Testing notarytool availability..."
    NOTARYTOOL_VERSION=$(xcrun notarytool --version 2>&1)
    NOTARYTOOL_VERSION_EXIT=$?
    echo "notarytool version output: $NOTARYTOOL_VERSION"
    echo "notarytool version exit code: $NOTARYTOOL_VERSION_EXIT"
    
    if [ $NOTARYTOOL_VERSION_EXIT -ne 0 ]; then
        echo "ERROR: notarytool not available or returned error"
        echo "Xcode version:"
        xcodebuild -version || echo "xcodebuild not found"
        echo ""
        echo "Trying xcrun --find notarytool:"
        xcrun --find notarytool || echo "notarytool not found"
        exit 1
    fi
    
    # Verify required credentials format
    echo ""
    echo "Verifying credential formats..."
    
    # Check API_KEY_ID format (should be alphanumeric characters)
    if ! echo "$API_KEY_ID" | grep -qE '^[A-Z0-9]+$'; then
        echo "WARNING: API_KEY_ID format may be invalid"
        echo "Expected: Uppercase alphanumeric characters (e.g., ABC123DEF4 or 6886SMKC2ACA)"
        echo "Actual value: $API_KEY_ID"
        echo "Actual length: ${#API_KEY_ID}"
    else
        echo "✓ API_KEY_ID format looks correct (${#API_KEY_ID} characters)"
    fi
    
    # Check API_ISSUER format (should be a UUID)
    if ! echo "$API_ISSUER" | grep -qE '^[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}$'; then
        echo "WARNING: API_ISSUER format may be invalid"
        echo "Expected: UUID format (e.g., 12345678-1234-1234-1234-123456789abc)"
    else
        echo "✓ API_ISSUER format looks correct"
    fi
    
    echo ""
    echo "Notarization setup complete"
    
    # Create a temporary directory for notarization
    NOTARIZE_DIR="$BUILD_DIR/notarize-temp"
    mkdir -p "$NOTARIZE_DIR"
    
    # Notarize the app bundle first
    echo "Notarizing app bundle..."
    APP_ZIP="$NOTARIZE_DIR/tosage-app.zip"
    
    # Create ZIP of app bundle for notarization
    echo "Creating ZIP for notarization..."
    ditto -c -k --sequesterRsrc --keepParent "$BUNDLE_PATH" "$APP_ZIP"
    
    # Submit for notarization
    echo "Submitting app bundle for notarization..."
    echo "Command: xcrun notarytool submit \"$APP_ZIP\" --key-id \"[REDACTED]\" --key \"$API_KEY_PATH\" --issuer \"[REDACTED]\" --wait"
    
    # Debug: Check if we're in the right directory
    echo "Current working directory: $PWD"
    echo "APP_ZIP exists: $([ -f "$APP_ZIP" ] && echo "yes" || echo "no")"
    echo "APP_ZIP size: $(wc -c < "$APP_ZIP" 2>/dev/null || echo "N/A") bytes"
    
    # Additional debugging for credentials
    echo ""
    echo "Credential debugging (without exposing secrets):"
    echo "  API_KEY_ID length: ${#API_KEY_ID}"
    echo "  API_KEY_ID contains only alphanumeric: $(echo "$API_KEY_ID" | grep -qE '^[A-Z0-9]+$' && echo "yes" || echo "no")"
    echo "  API_KEY_ID first 4 chars: ${API_KEY_ID:0:4}..."
    echo "  API_KEY_ID last 4 chars: ...${API_KEY_ID: -4}"
    echo "  API_ISSUER is valid UUID: $(echo "$API_ISSUER" | grep -qE '^[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}$' && echo "yes" || echo "no")"
    
    # Check for potential GitHub masking issues
    if [ "${API_KEY_ID}" = "***" ] || [ "${API_ISSUER}" = "***" ]; then
        echo ""
        echo "ERROR: GitHub secrets masking detected!"
        echo "The API credentials have been replaced with '***'"
        echo "This is a known issue with GitHub Actions when secrets are logged."
        echo ""
        echo "Please ensure:"
        echo "  1. Secrets are properly set in GitHub repository settings"
        echo "  2. Secret names match exactly (case-sensitive)"
        echo "  3. No spaces or special characters in secret values"
        exit 1
    fi
    
    # Run notarytool and capture exit code
    echo ""
    echo "Starting notarization submission..."
    echo "  File: $APP_ZIP"
    echo "  Size: $(du -h "$APP_ZIP" | cut -f1)"
    echo ""
    
    set +e  # Temporarily disable exit on error
    
    # Try running notarytool (timeout command may not be available on macOS)
    if command -v timeout >/dev/null 2>&1; then
        NOTARYTOOL_OUTPUT=$(timeout 300 xcrun notarytool submit "$APP_ZIP" \
            --key-id "$API_KEY_ID" \
            --key "$API_KEY_PATH" \
            --issuer "$API_ISSUER" \
            --wait --verbose 2>&1)
        NOTARYTOOL_EXIT_CODE=$?
        
        # Check if timeout occurred
        if [ $NOTARYTOOL_EXIT_CODE -eq 124 ]; then
            echo "ERROR: notarytool timed out after 5 minutes"
            NOTARYTOOL_EXIT_CODE=133
        fi
    else
        # Run without timeout on macOS
        NOTARYTOOL_OUTPUT=$(xcrun notarytool submit "$APP_ZIP" \
            --key-id "$API_KEY_ID" \
            --key "$API_KEY_PATH" \
            --issuer "$API_ISSUER" \
            --wait --verbose 2>&1)
        NOTARYTOOL_EXIT_CODE=$?
    fi
    
    set -e  # Re-enable exit on error
    
    echo "Notarytool exit code: $NOTARYTOOL_EXIT_CODE"
    echo "Notarytool output:"
    echo "$NOTARYTOOL_OUTPUT"
    
    if [ $NOTARYTOOL_EXIT_CODE -ne 0 ]; then
        echo "ERROR: notarytool command failed with exit code $NOTARYTOOL_EXIT_CODE"
        echo ""
        echo "Common exit codes:"
        echo "  69 - Service unavailable"
        echo "  133 - Killed by signal (often authentication failure)"
        echo "  1 - General error"
        echo ""
        echo "Possible causes:"
        echo "  - Invalid API credentials (check API_KEY_ID, API_ISSUER)"
        echo "  - API key file format issues (should be .p8 format)"
        echo "  - API key permissions (needs Developer ID Application access)"
        echo "  - Network connectivity problems"
        echo "  - Apple notary service issues"
        echo ""
        echo "Debug information:"
        echo "  API_KEY_ID length: ${#API_KEY_ID}"
        echo "  API_ISSUER length: ${#API_ISSUER}"
        echo "  API_KEY_PATH exists: $([ -f "$API_KEY_PATH" ] && echo "yes" || echo "no")"
        echo "  API_KEY_PATH size: $(wc -c < "$API_KEY_PATH" 2>/dev/null || echo "N/A") bytes"
        echo "  Current directory: $PWD"
        
        # Try to show more details about the API key file
        if [ -f "$API_KEY_PATH" ]; then
            echo ""
            echo "API key file first line:"
            head -1 "$API_KEY_PATH" | sed 's/\(......\).*/\1.../' # Show only first 6 chars
            echo "API key file lines: $(wc -l < "$API_KEY_PATH")"
        fi
        
        # Check if it's a JWT parsing issue
        echo ""
        echo "JWT generation appears to have failed. This can happen when:"
        echo "  - The .p8 file is corrupted or incomplete"
        echo "  - The API key has been revoked"
        echo "  - The API_KEY_ID doesn't match the .p8 file"
        echo "  - The bundle ID is not associated with the Developer ID certificate"
        echo "  - The API key doesn't have 'Developer Relations' access"
        echo ""
        echo "To verify your API key in App Store Connect:"
        echo "  1. Go to https://appstoreconnect.apple.com/access/api"
        echo "  2. Find your key ID: $API_KEY_ID"
        echo "  3. Ensure it has 'Admin' or 'Developer' role"
        echo "  4. Check the key hasn't expired or been revoked"
        
        rm -rf "$NOTARIZE_DIR"
        exit 1
    fi
    
    SUBMISSION_ID=$(echo "$NOTARYTOOL_OUTPUT" | grep -E "id: [a-f0-9-]+" | head -1 | awk '{print $2}')
    
    if [ -n "$SUBMISSION_ID" ]; then
        echo "Notarization submission ID: $SUBMISSION_ID"
        
        # Check notarization status
        echo "Waiting for notarization to complete..."
        NOTARIZE_STATUS=$(xcrun notarytool info "$SUBMISSION_ID" \
            --key-id "$API_KEY_ID" \
            --key "$API_KEY_PATH" \
            --issuer "$API_ISSUER" 2>&1 | grep -E "status:" | awk '{print $2}')
        
        if [ "$NOTARIZE_STATUS" = "Accepted" ]; then
            echo "App bundle notarization successful!"
            
            # Staple the notarization ticket to the app
            echo "Stapling notarization ticket to app bundle..."
            if xcrun stapler staple "$BUNDLE_PATH"; then
                echo "Successfully stapled ticket to app bundle"
            else
                echo "ERROR: Failed to staple ticket to app bundle"
                rm -rf "$NOTARIZE_DIR"
                exit 1
            fi
        else
            echo "ERROR: App bundle notarization failed or not accepted"
            echo "Status: $NOTARIZE_STATUS"
            
            # Get notarization log for debugging
            echo "Fetching notarization log..."
            xcrun notarytool log "$SUBMISSION_ID" \
                --key-id "$API_KEY_ID" \
                --key "$API_KEY_PATH" \
                --issuer "$API_ISSUER" || true
            
            rm -rf "$NOTARIZE_DIR"
            exit 1
        fi
    else
        echo "ERROR: Failed to submit app bundle for notarization"
        echo "Please check:"
        echo "  - API_KEY_ID is correct"
        echo "  - API_KEY_PATH points to valid .p8 file"
        echo "  - API_ISSUER is correct"
        echo "  - The certificate is valid for notarization"
        rm -rf "$NOTARIZE_DIR"
        exit 1
    fi
    
    # Now notarize the DMG
    echo ""
    echo "Notarizing DMG..."
    
    # Submit DMG for notarization
    echo "Submitting DMG for notarization..."
    echo "Command: xcrun notarytool submit \"$OUTPUT_DMG\" --key-id \"$API_KEY_ID\" --key \"$API_KEY_PATH\" --issuer \"$API_ISSUER\" --wait"
    
    set +e  # Temporarily disable exit on error
    DMG_NOTARYTOOL_OUTPUT=$(xcrun notarytool submit "$OUTPUT_DMG" \
        --key-id "$API_KEY_ID" \
        --key "$API_KEY_PATH" \
        --issuer "$API_ISSUER" \
        --wait --verbose 2>&1)
    DMG_NOTARYTOOL_EXIT_CODE=$?
    set -e  # Re-enable exit on error
    
    echo "DMG Notarytool exit code: $DMG_NOTARYTOOL_EXIT_CODE"
    echo "DMG Notarytool output:"
    echo "$DMG_NOTARYTOOL_OUTPUT"
    
    if [ $DMG_NOTARYTOOL_EXIT_CODE -ne 0 ]; then
        echo "ERROR: DMG notarytool command failed with exit code $DMG_NOTARYTOOL_EXIT_CODE"
        rm -rf "$NOTARIZE_DIR"
        exit 1
    fi
    
    DMG_SUBMISSION_ID=$(echo "$DMG_NOTARYTOOL_OUTPUT" | grep -E "id: [a-f0-9-]+" | head -1 | awk '{print $2}')
    
    if [ -n "$DMG_SUBMISSION_ID" ]; then
        echo "DMG notarization submission ID: $DMG_SUBMISSION_ID"
        
        # Check notarization status
        echo "Waiting for DMG notarization to complete..."
        DMG_NOTARIZE_STATUS=$(xcrun notarytool info "$DMG_SUBMISSION_ID" \
            --key-id "$API_KEY_ID" \
            --key "$API_KEY_PATH" \
            --issuer "$API_ISSUER" 2>&1 | grep -E "status:" | awk '{print $2}')
        
        if [ "$DMG_NOTARIZE_STATUS" = "Accepted" ]; then
            echo "DMG notarization successful!"
            
            # Staple the notarization ticket to the DMG
            echo "Stapling notarization ticket to DMG..."
            if xcrun stapler staple "$OUTPUT_DMG"; then
                echo "Successfully stapled ticket to DMG"
            else
                echo "ERROR: Failed to staple ticket to DMG"
                rm -rf "$NOTARIZE_DIR"
                exit 1
            fi
        else
            echo "ERROR: DMG notarization failed or not accepted"
            echo "Status: $DMG_NOTARIZE_STATUS"
            
            # Get notarization log for debugging
            echo "Fetching DMG notarization log..."
            xcrun notarytool log "$DMG_SUBMISSION_ID" \
                --key-id "$API_KEY_ID" \
                --key "$API_KEY_PATH" \
                --issuer "$API_ISSUER" || true
            
            rm -rf "$NOTARIZE_DIR"
            exit 1
        fi
    else
        echo "ERROR: Failed to submit DMG for notarization"
        rm -rf "$NOTARIZE_DIR"
        exit 1
    fi
    
    # Clean up notarization temp directory
    rm -rf "$NOTARIZE_DIR"
    
    echo ""
    echo "Notarization process complete"
    fi  # End of credential check
else
    echo ""
    echo "Skipping notarization (NOTARIZE not set to 'true')"
    echo "To enable notarization, set:"
    echo '  export NOTARIZE="true"'
    echo '  export API_KEY_ID="your-key-id"'
    echo '  export API_KEY_PATH="/path/to/AuthKey_XXXXX.p8"'
    echo '  export API_ISSUER="your-issuer-id"'
fi

# Clean up
echo "Cleaning up..."
rm -f "$TEMP_DMG"
rm -rf "$STAGING_DIR"

# Display result
echo ""
echo "DMG created successfully!"
echo "  Path: $OUTPUT_DMG"
echo "  Size: $(du -h "$OUTPUT_DMG" | cut -f1)"

# Verify DMG
echo ""
echo "Verifying DMG..."
hdiutil verify "$OUTPUT_DMG"

echo ""
echo "DMG creation complete!"