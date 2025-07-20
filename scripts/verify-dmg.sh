#!/bin/bash

# tosage DMG Verification Script
# Verifies DMG integrity, signature, and notarization

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
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

print_info() {
    echo -e "${BLUE}[i]${NC} $1"
}

# Function to verify DMG
verify_dmg() {
    local DMG_PATH="$1"
    local ERRORS=0
    
    echo "DMG Verification Report"
    echo "======================="
    echo "File: $(basename "$DMG_PATH")"
    echo "Size: $(du -h "$DMG_PATH" | cut -f1)"
    echo ""
    
    # 1. Check file exists
    if [ ! -f "$DMG_PATH" ]; then
        print_error "DMG file not found: $DMG_PATH"
        return 1
    fi
    
    # 2. Verify DMG structure
    echo "Checking DMG structure..."
    if hdiutil verify "$DMG_PATH" >/dev/null 2>&1; then
        print_status "DMG structure is valid"
    else
        print_error "DMG structure verification failed"
        ((ERRORS++))
    fi
    
    # 3. Check code signature
    echo ""
    echo "Checking code signature..."
    if codesign -dvv "$DMG_PATH" 2>&1 | grep -q "Signature="; then
        print_status "DMG is signed"
        
        # Get signing details
        SIGNER=$(codesign -dvv "$DMG_PATH" 2>&1 | grep "Authority=" | head -1 | cut -d= -f2)
        print_info "Signed by: $SIGNER"
        
        # Verify signature
        if codesign --verify --deep --strict "$DMG_PATH" 2>&1; then
            print_status "Signature is valid"
        else
            print_error "Signature verification failed"
            ((ERRORS++))
        fi
    else
        print_warning "DMG is not signed"
    fi
    
    # 4. Check notarization
    echo ""
    echo "Checking notarization..."
    if xcrun stapler validate "$DMG_PATH" 2>&1 | grep -q "validate worked"; then
        print_status "DMG is notarized"
    else
        print_warning "DMG is not notarized"
    fi
    
    # 5. Check Gatekeeper
    echo ""
    echo "Checking Gatekeeper acceptance..."
    SPCTL_OUTPUT=$(spctl -a -t open --context context:primary-signature -vv "$DMG_PATH" 2>&1)
    if echo "$SPCTL_OUTPUT" | grep -q "accepted"; then
        print_status "Gatekeeper accepts this DMG"
    else
        print_warning "Gatekeeper may block this DMG"
        print_info "Output: $SPCTL_OUTPUT"
    fi
    
    # 6. Mount and verify contents
    echo ""
    echo "Checking DMG contents..."
    MOUNT_POINT=$(hdiutil attach "$DMG_PATH" -nobrowse -noverify -noautoopen | grep "Volumes" | cut -f3-)
    
    if [ -n "$MOUNT_POINT" ]; then
        print_status "DMG mounted successfully at: $MOUNT_POINT"
        
        # Check for app bundle
        APP_BUNDLE=$(find "$MOUNT_POINT" -name "*.app" -maxdepth 1 | head -1)
        if [ -n "$APP_BUNDLE" ]; then
            print_status "Found app bundle: $(basename "$APP_BUNDLE")"
            
            # Verify app bundle signature
            if codesign --verify --deep --strict "$APP_BUNDLE" 2>&1; then
                print_status "App bundle signature is valid"
            else
                print_error "App bundle signature verification failed"
                ((ERRORS++))
            fi
            
            # Check app bundle structure
            if [ -f "$APP_BUNDLE/Contents/Info.plist" ]; then
                print_status "App bundle structure is valid"
                
                # Extract version info
                VERSION=$(/usr/libexec/PlistBuddy -c "Print :CFBundleShortVersionString" "$APP_BUNDLE/Contents/Info.plist" 2>/dev/null || echo "Unknown")
                print_info "App version: $VERSION"
            else
                print_error "App bundle structure is invalid"
                ((ERRORS++))
            fi
        else
            print_error "No app bundle found in DMG"
            ((ERRORS++))
        fi
        
        # Check for Applications symlink
        if [ -L "$MOUNT_POINT/Applications" ]; then
            print_status "Applications symlink exists"
        else
            print_warning "Applications symlink not found"
        fi
        
        # Unmount
        hdiutil detach "$MOUNT_POINT" -quiet
    else
        print_error "Failed to mount DMG"
        ((ERRORS++))
    fi
    
    # Summary
    echo ""
    echo "Summary"
    echo "======="
    if [ $ERRORS -eq 0 ]; then
        print_status "All checks passed!"
        return 0
    else
        print_error "Found $ERRORS error(s)"
        return 1
    fi
}

# Main execution
main() {
    if [ $# -eq 0 ]; then
        # Find all DMG files in current directory
        DMG_FILES=$(find . -name "*.dmg" -maxdepth 1 2>/dev/null)
        
        if [ -z "$DMG_FILES" ]; then
            echo "Usage: $0 [dmg-file]"
            echo "   or: $0 (to verify all DMG files in current directory)"
            echo ""
            echo "No DMG files found in current directory"
            exit 1
        fi
        
        # Verify each DMG
        for dmg in $DMG_FILES; do
            verify_dmg "$dmg"
            echo ""
            echo "---"
            echo ""
        done
    else
        # Verify specific DMG
        verify_dmg "$1"
    fi
}

# Run main function
main "$@"