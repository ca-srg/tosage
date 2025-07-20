#!/bin/bash

# tosage DMG Dependencies Check Script
# Verifies all required tools and dependencies for DMG creation

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

# Variables
ERRORS=0
WARNINGS=0

echo "DMG Build Dependencies Check"
echo "============================"
echo ""

# 1. Check OS
echo "Checking operating system..."
if [[ "$OSTYPE" == "darwin"* ]]; then
    print_status "Running on macOS ($(sw_vers -productVersion))"
else
    print_error "This script requires macOS"
    ((ERRORS++))
fi

# 2. Check required command line tools
echo ""
echo "Checking required tools..."

# hdiutil
if command -v hdiutil >/dev/null 2>&1; then
    print_status "hdiutil found"
else
    print_error "hdiutil not found (required for DMG creation)"
    ((ERRORS++))
fi

# codesign
if command -v codesign >/dev/null 2>&1; then
    print_status "codesign found"
else
    print_error "codesign not found (required for code signing)"
    ((ERRORS++))
fi

# spctl
if command -v spctl >/dev/null 2>&1; then
    print_status "spctl found"
else
    print_error "spctl not found (required for Gatekeeper validation)"
    ((ERRORS++))
fi

# xcrun
if command -v xcrun >/dev/null 2>&1; then
    print_status "xcrun found"
    
    # Check for notarytool
    if xcrun --find notarytool >/dev/null 2>&1; then
        print_status "notarytool found"
    else
        print_warning "notarytool not found (required for notarization)"
        print_info "Install Xcode or Xcode Command Line Tools"
        ((WARNINGS++))
    fi
    
    # Check for stapler
    if xcrun --find stapler >/dev/null 2>&1; then
        print_status "stapler found"
    else
        print_warning "stapler not found (required for notarization)"
        ((WARNINGS++))
    fi
else
    print_error "xcrun not found (Xcode Command Line Tools required)"
    ((ERRORS++))
fi

# iconutil
if command -v iconutil >/dev/null 2>&1; then
    print_status "iconutil found"
else
    print_error "iconutil not found (required for icon conversion)"
    ((ERRORS++))
fi

# sips
if command -v sips >/dev/null 2>&1; then
    print_status "sips found"
else
    print_error "sips not found (required for image processing)"
    ((ERRORS++))
fi

# SetFile
if command -v SetFile >/dev/null 2>&1; then
    print_status "SetFile found"
else
    print_warning "SetFile not found (optional for DMG customization)"
    print_info "Install Xcode Command Line Tools if needed"
    ((WARNINGS++))
fi

# osascript
if command -v osascript >/dev/null 2>&1; then
    print_status "osascript found"
else
    print_error "osascript not found (required for DMG window setup)"
    ((ERRORS++))
fi

# 3. Check optional tools
echo ""
echo "Checking optional tools..."

# ImageMagick (convert)
if command -v convert >/dev/null 2>&1; then
    print_status "ImageMagick found"
else
    print_warning "ImageMagick not found (optional for SVG to PNG conversion)"
    print_info "Install with: brew install imagemagick"
    ((WARNINGS++))
fi

# 4. Check Go installation
echo ""
echo "Checking Go installation..."
if command -v go >/dev/null 2>&1; then
    GO_VERSION=$(go version | awk '{print $3}')
    print_status "Go found ($GO_VERSION)"
else
    print_error "Go not found (required for building binaries)"
    ((ERRORS++))
fi

# 5. Check code signing identity
echo ""
echo "Checking code signing identity..."
if security find-identity -v -p codesigning | grep -q "Developer ID"; then
    print_status "Developer ID certificate found"
    security find-identity -v -p codesigning | grep "Developer ID" | head -5 | while read line; do
        print_info "  $line"
    done
else
    print_warning "No Developer ID certificate found"
    print_info "DMG can be created but won't be signed"
    print_info "Users will see security warnings when opening the DMG"
    ((WARNINGS++))
fi

# 6. Check project structure
echo ""
echo "Checking project structure..."

if [ -d "build/dmg" ]; then
    print_status "DMG build directory exists"
else
    print_error "DMG build directory not found (build/dmg)"
    ((ERRORS++))
fi

if [ -f "build/dmg/create-app-bundle.sh" ]; then
    print_status "App bundle creation script found"
else
    print_error "App bundle creation script not found"
    ((ERRORS++))
fi

if [ -f "build/dmg/create-dmg.sh" ]; then
    print_status "DMG creation script found"
else
    print_error "DMG creation script not found"
    ((ERRORS++))
fi

if [ -f "assets/icon_black.png" ]; then
    print_status "Icon file found"
else
    print_error "Icon file not found (assets/icon_black.png)"
    ((ERRORS++))
fi

# 7. Check permissions
echo ""
echo "Checking file permissions..."

if [ -x "build/dmg/create-app-bundle.sh" ]; then
    print_status "App bundle script is executable"
else
    print_warning "App bundle script is not executable"
    print_info "Run: chmod +x build/dmg/create-app-bundle.sh"
    ((WARNINGS++))
fi

if [ -x "build/dmg/create-dmg.sh" ]; then
    print_status "DMG creation script is executable"
else
    print_warning "DMG creation script is not executable"
    print_info "Run: chmod +x build/dmg/create-dmg.sh"
    ((WARNINGS++))
fi

# Summary
echo ""
echo "Summary"
echo "======="

if [ $ERRORS -eq 0 ] && [ $WARNINGS -eq 0 ]; then
    print_status "All checks passed! Ready to build DMG."
    exit 0
elif [ $ERRORS -eq 0 ]; then
    print_warning "Found $WARNINGS warning(s)"
    print_info "DMG creation should work but some features may be limited"
    exit 0
else
    print_error "Found $ERRORS error(s) and $WARNINGS warning(s)"
    print_info "Please fix the errors before attempting to build DMG"
    exit 1
fi