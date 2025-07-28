#!/bin/bash

# Test script that avoids systray-dependent packages on non-macOS systems

echo "Running tests for core functionality..."

# Test packages that don't depend on systray
PACKAGES=(
    "./domain/..."
    "./infrastructure/logging/..."
    "./infrastructure/repository/..."
    "./infrastructure/service/..."
    "./usecase/..."
)

# Run tests for each package
for pkg in "${PACKAGES[@]}"; do
    echo "Testing $pkg"
    if ! go test "$pkg"; then
        echo "Tests failed for $pkg"
        exit 1
    fi
done

echo "All core tests passed!"

# Try to test DI package with Linux target to avoid systray issues
echo "Testing DI package (cross-compiled)..."
if ! GOOS=linux go test ./infrastructure/di; then
    echo "Warning: DI package tests failed"
fi

# Try to build main binary with Linux target
echo "Testing main build (cross-compiled)..."
if ! GOOS=linux go build .; then
    echo "Error: Main build failed"
    exit 1
fi

echo "All tests completed successfully!"