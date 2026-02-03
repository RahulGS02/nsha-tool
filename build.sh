#!/bin/bash

# NSHA Build Script
# Builds executables for multiple platforms

set -e

VERSION="1.0.0"
APP_NAME="nsha"
BUILD_DIR="build"

echo "╔═══════════════════════════════════════════════════════════╗"
echo "║           Building NSHA - Null SHA Fixer                  ║"
echo "║                   Version: $VERSION                        ║"
echo "╚═══════════════════════════════════════════════════════════╝"
echo ""

# Clean build directory
echo "[CLEAN] Cleaning build directory..."
rm -rf $BUILD_DIR
mkdir -p $BUILD_DIR

# Build flags for smaller binaries
LDFLAGS="-s -w -X main.Version=$VERSION"

echo ""
echo "[BUILD] Building executables..."
echo ""

# Windows AMD64
echo "  [BUILD] Building for Windows (amd64)..."
GOOS=windows GOARCH=amd64 go build -ldflags="$LDFLAGS" -o $BUILD_DIR/${APP_NAME}-windows-amd64.exe
echo "     [OK] $BUILD_DIR/${APP_NAME}-windows-amd64.exe"

# Windows ARM64
echo "  [BUILD] Building for Windows (arm64)..."
GOOS=windows GOARCH=arm64 go build -ldflags="$LDFLAGS" -o $BUILD_DIR/${APP_NAME}-windows-arm64.exe
echo "     [OK] $BUILD_DIR/${APP_NAME}-windows-arm64.exe"

# Linux AMD64
echo "  [BUILD] Building for Linux (amd64)..."
GOOS=linux GOARCH=amd64 go build -ldflags="$LDFLAGS" -o $BUILD_DIR/${APP_NAME}-linux-amd64
echo "     [OK] $BUILD_DIR/${APP_NAME}-linux-amd64"

# Linux ARM64
echo "  [BUILD] Building for Linux (arm64)..."
GOOS=linux GOARCH=arm64 go build -ldflags="$LDFLAGS" -o $BUILD_DIR/${APP_NAME}-linux-arm64
echo "     [OK] $BUILD_DIR/${APP_NAME}-linux-arm64"

# macOS AMD64
echo "  [BUILD] Building for macOS (amd64)..."
GOOS=darwin GOARCH=amd64 go build -ldflags="$LDFLAGS" -o $BUILD_DIR/${APP_NAME}-darwin-amd64
echo "     [OK] $BUILD_DIR/${APP_NAME}-darwin-amd64"

# macOS ARM64 (Apple Silicon)
echo "  [BUILD] Building for macOS (arm64)..."
GOOS=darwin GOARCH=arm64 go build -ldflags="$LDFLAGS" -o $BUILD_DIR/${APP_NAME}-darwin-arm64
echo "     [OK] $BUILD_DIR/${APP_NAME}-darwin-arm64"

echo ""
echo "[SUCCESS] Build complete!"
echo ""
echo "[INFO] Executables are in the '$BUILD_DIR' directory:"
ls -lh $BUILD_DIR/
echo ""
echo "[INFO] To install locally, copy the appropriate binary to your PATH"
echo "   Example: sudo cp $BUILD_DIR/${APP_NAME}-linux-amd64 /usr/local/bin/nsha"

