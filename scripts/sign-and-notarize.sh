#!/bin/bash

# tosage Code Signing and Notarization Script
# Signs the app bundle and DMG with Developer ID certificate

set -e

# Variables
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
BUILD_DIR="$PROJECT_ROOT/build/dmg"
APP_NAME="tosage"
BUNDLE_NAME="${APP_NAME}.app"
BUNDLE_PATH="$BUILD_DIR/$BUNDLE_NAME"

# Signing identity (can be overridden by environment variable)
SIGNING_IDENTITY="${SIGNING_IDENTITY:-Developer ID Application}"
TEAM_ID="${TEAM_ID:-}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Function to print colored output
print_status() {
    echo -e "${GREEN}[✓]${NC} $1"
}

print_error() {
    echo -e "${RED}[✗]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[!]${NC} $1"
}

# Function to check if certificate exists
check_certificate() {
    echo "Checking for signing certificate..."
    
    if security find-identity -v -p codesigning | grep -q "$SIGNING_IDENTITY"; then
        print_status "Found signing certificate: $SIGNING_IDENTITY"
        
        # Extract team ID if not provided
        if [ -z "$TEAM_ID" ]; then
            TEAM_ID=$(security find-identity -v -p codesigning | grep "$SIGNING_IDENTITY" | head -1 | sed 's/.*(\(.*\)).*/\1/')
            print_status "Detected Team ID: $TEAM_ID"
        fi
        
        return 0
    else
        print_warning "Signing certificate not found: $SIGNING_IDENTITY"
        print_warning "Available certificates:"
        security find-identity -v -p codesigning | grep -E "Developer ID|Apple Development" || echo "  None found"
        return 1
    fi
}

# Function to sign a file
sign_file() {
    local FILE_PATH="$1"
    local FILE_TYPE="$2"
    
    echo ""
    echo "Signing $FILE_TYPE: $(basename "$FILE_PATH")"
    
    if [ ! -e "$FILE_PATH" ]; then
        print_error "$FILE_TYPE not found at: $FILE_PATH"
        return 1
    fi
    
    # Sign with hardened runtime and timestamp
    if codesign --force --sign "$SIGNING_IDENTITY" \
        --options runtime \
        --timestamp \
        --entitlements "$SCRIPT_DIR/entitlements.plist" \
        "$FILE_PATH" 2>/dev/null || \
       codesign --force --sign "$SIGNING_IDENTITY" \
        --options runtime \
        --timestamp \
        "$FILE_PATH"; then
        print_status "$FILE_TYPE signed successfully"
        
        # Verify signature
        if codesign --verify --deep --strict "$FILE_PATH"; then
            print_status "Signature verified"
        else
            print_error "Signature verification failed"
            return 1
        fi
    else
        print_error "Failed to sign $FILE_TYPE"
        return 1
    fi
}

# Function to notarize
notarize_dmg() {
    local DMG_PATH="$1"
    local APPLE_ID="${APPLE_ID:-}"
    local APP_PASSWORD="${APP_PASSWORD:-}"
    
    if [ -z "$APPLE_ID" ] || [ -z "$APP_PASSWORD" ]; then
        print_warning "Notarization skipped: APPLE_ID or APP_PASSWORD not set"
        return 0
    fi
    
    echo ""
    echo "Notarizing DMG..."
    
    # Submit for notarization
    local NOTARIZE_OUTPUT=$(xcrun notarytool submit "$DMG_PATH" \
        --apple-id "$APPLE_ID" \
        --password "$APP_PASSWORD" \
        --team-id "$TEAM_ID" \
        --wait 2>&1)
    
    if echo "$NOTARIZE_OUTPUT" | grep -q "status: Accepted"; then
        print_status "Notarization successful"
        
        # Staple the notarization
        if xcrun stapler staple "$DMG_PATH"; then
            print_status "Notarization stapled to DMG"
        else
            print_warning "Failed to staple notarization"
        fi
    else
        print_error "Notarization failed"
        echo "$NOTARIZE_OUTPUT"
        return 1
    fi
}

# Main execution
main() {
    echo "tosage Code Signing and Notarization"
    echo "===================================="
    
    # Check for certificate
    if ! check_certificate; then
        print_warning "Proceeding without code signing"
        print_warning "The app will require user approval on first launch"
        exit 0
    fi
    
    # Create entitlements file if it doesn't exist
    if [ ! -f "$SCRIPT_DIR/entitlements.plist" ]; then
        echo "Creating entitlements file..."
        cat > "$SCRIPT_DIR/entitlements.plist" <<EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>com.apple.security.cs.allow-jit</key>
    <false/>
    <key>com.apple.security.cs.allow-unsigned-executable-memory</key>
    <false/>
    <key>com.apple.security.cs.disable-library-validation</key>
    <false/>
    <key>com.apple.security.device.audio-input</key>
    <false/>
    <key>com.apple.security.device.camera</key>
    <false/>
    <key>com.apple.security.files.user-selected.read-only</key>
    <true/>
    <key>com.apple.security.network.client</key>
    <true/>
</dict>
</plist>
EOF
    fi
    
    # Sign the app bundle
    if [ -d "$BUNDLE_PATH" ]; then
        # Sign all frameworks and dylibs first (if any)
        find "$BUNDLE_PATH" -name "*.dylib" -o -name "*.framework" | while read -r lib; do
            sign_file "$lib" "Library"
        done
        
        # Sign the main executable
        sign_file "$BUNDLE_PATH/Contents/MacOS/$APP_NAME" "Executable"
        
        # Sign the entire bundle
        sign_file "$BUNDLE_PATH" "App Bundle"
    else
        print_warning "App bundle not found at: $BUNDLE_PATH"
    fi
    
    # Find and sign DMG files
    echo ""
    echo "Looking for DMG files to sign..."
    
    find "$PROJECT_ROOT" -name "*.dmg" -maxdepth 1 | while read -r dmg; do
        sign_file "$dmg" "DMG"
        
        # Notarize if credentials are available
        notarize_dmg "$dmg"
    done
    
    echo ""
    print_status "Code signing process completed"
    
    # Final validation
    echo ""
    echo "Validation Summary:"
    echo "==================="
    
    if [ -d "$BUNDLE_PATH" ]; then
        echo "App Bundle:"
        spctl -a -t exec -vv "$BUNDLE_PATH" 2>&1 || print_warning "Gatekeeper validation failed for app bundle"
    fi
    
    find "$PROJECT_ROOT" -name "*.dmg" -maxdepth 1 | while read -r dmg; do
        echo ""
        echo "DMG: $(basename "$dmg")"
        spctl -a -t open --context context:primary-signature -vv "$dmg" 2>&1 || print_warning "Gatekeeper validation failed for DMG"
    done
}

# Run main function
main "$@"