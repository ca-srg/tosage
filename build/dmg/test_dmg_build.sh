#!/bin/bash

# tosage DMG Build Test Script
# Tests the complete DMG creation pipeline

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

print_test() {
    echo -e "${BLUE}[TEST]${NC} $1"
}

# Variables
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$SCRIPT_DIR"
TEST_RESULTS=()
FAILED_TESTS=0

# Function to run a test
run_test() {
    local TEST_NAME="$1"
    local TEST_CMD="$2"
    
    print_test "$TEST_NAME"
    
    if eval "$TEST_CMD"; then
        print_status "$TEST_NAME passed"
        TEST_RESULTS+=("✓ $TEST_NAME")
    else
        print_error "$TEST_NAME failed"
        TEST_RESULTS+=("✗ $TEST_NAME")
        ((FAILED_TESTS++))
    fi
    echo ""
}

# Main test execution
main() {
    echo "tosage DMG Build Test Suite"
    echo "==========================="
    echo ""
    
    # 1. Clean environment
    print_info "Cleaning build environment..."
    make clean >/dev/null 2>&1 || true
    
    # 2. Check dependencies
    run_test "Dependency Check" "./scripts/check-dmg-deps.sh"
    
    # 3. Build binaries
    print_info "Building binaries..."
    if ! make build-darwin >/dev/null 2>&1; then
        print_error "Failed to build binaries"
        exit 1
    fi
    print_status "Binaries built successfully"
    echo ""
    
    # 4. Test app bundle creation
    run_test "App Bundle Creation (ARM64)" "ARCH=arm64 ./build/dmg/create-app-bundle.sh >/dev/null 2>&1"
    
    # Verify app bundle structure
    if [ -d "build/dmg/tosage.app" ]; then
        run_test "App Bundle Structure" "test -f 'build/dmg/tosage.app/Contents/Info.plist' && test -f 'build/dmg/tosage.app/Contents/MacOS/tosage'"
        run_test "App Bundle Icon" "test -f 'build/dmg/tosage.app/Contents/Resources/app.icns'"
    fi
    
    # 5. Test DMG creation
    run_test "DMG Creation (ARM64)" "ARCH=arm64 ./build/dmg/create-dmg.sh >/dev/null 2>&1"
    
    # Find created DMG
    DMG_FILE=$(find . -name "*.dmg" -maxdepth 1 | head -1)
    
    if [ -n "$DMG_FILE" ]; then
        print_info "Created DMG: $DMG_FILE"
        
        # 6. Test DMG verification
        run_test "DMG Verification" "./scripts/verify-dmg.sh '$DMG_FILE' >/dev/null 2>&1"
        
        # 7. Test DMG mounting
        print_test "DMG Mounting"
        if MOUNT_POINT=$(hdiutil attach "$DMG_FILE" -nobrowse -noverify -noautoopen | grep "Volumes" | cut -f3-); then
            print_status "DMG mounted successfully"
            
            # Check contents
            run_test "DMG Contents Check" "test -d '$MOUNT_POINT/tosage.app' && test -L '$MOUNT_POINT/Applications'"
            
            # Unmount
            hdiutil detach "$MOUNT_POINT" -quiet
            print_status "DMG unmounted successfully"
        else
            print_error "Failed to mount DMG"
            ((FAILED_TESTS++))
        fi
        echo ""
    else
        print_error "No DMG file created"
        ((FAILED_TESTS++))
    fi
    
    # 8. Test code signing (if certificate available)
    if security find-identity -v -p codesigning | grep -q "Developer ID"; then
        run_test "Code Signing" "./scripts/sign-and-notarize.sh >/dev/null 2>&1"
    else
        print_warning "Skipping code signing test (no certificate found)"
    fi
    
    # 9. Test Makefile targets
    print_info "Testing Makefile targets..."
    run_test "Make dmg-clean" "make dmg-clean >/dev/null 2>&1"
    run_test "Make dmg-arm64" "make dmg-arm64 >/dev/null 2>&1"
    
    # Find new DMG
    NEW_DMG=$(find . -name "*.dmg" -maxdepth 1 -newer "$0" | head -1)
    if [ -n "$NEW_DMG" ]; then
        run_test "Make dmg-verify" "make dmg-verify >/dev/null 2>&1"
    fi
    
    # 10. Test error handling
    print_info "Testing error handling..."
    
    # Test with missing icon
    mv assets/icon.png assets/icon.png.bak 2>/dev/null || true
    run_test "Error Handling - Missing Icon" "! ARCH=arm64 ./build/dmg/create-app-bundle.sh >/dev/null 2>&1"
    mv assets/icon.png.bak assets/icon.png 2>/dev/null || true
    
    # Summary
    echo ""
    echo "Test Summary"
    echo "============"
    for result in "${TEST_RESULTS[@]}"; do
        echo "$result"
    done
    
    echo ""
    if [ $FAILED_TESTS -eq 0 ]; then
        print_status "All tests passed!"
        
        # Show final DMG info
        if [ -n "$NEW_DMG" ]; then
            echo ""
            print_info "Final DMG: $NEW_DMG"
            print_info "Size: $(du -h "$NEW_DMG" | cut -f1)"
        fi
        
        exit 0
    else
        print_error "$FAILED_TESTS test(s) failed"
        exit 1
    fi
}

# Run main function
main "$@"